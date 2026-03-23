package diffyml

import (
	"cmp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// DiffType represents the kind of difference detected between YAML documents.
type DiffType int

const (
	// DiffAdded indicates a new entry was added in the target document.
	DiffAdded DiffType = iota
	// DiffRemoved indicates an entry was removed from the source document.
	DiffRemoved
	// DiffModified indicates a value was changed between documents.
	DiffModified
	// DiffOrderChanged indicates list order changed (when not ignoring order).
	DiffOrderChanged
)

// Difference represents a single change between two YAML documents.
type Difference struct {
	// Path is the structured path to the changed value (e.g., DiffPath{"some", "yaml", "structure", "name"}).
	Path DiffPath
	// Type indicates the kind of change (added, removed, modified, order changed).
	Type DiffType
	// From is the original value (nil for additions).
	From any
	// To is the new value (nil for removals).
	To any
	// DocumentIndex indicates which document in a multi-document YAML file (0-based).
	DocumentIndex int
	// DocumentName is a human-readable label for the document (e.g., K8s resource display name).
	// Empty for non-K8s documents or when detection is disabled.
	DocumentName string
}

// Options configures the comparison behavior.
type Options struct {
	// IgnoreOrderChanges ignores list order differences when true.
	IgnoreOrderChanges bool
	// IgnoreWhitespaceChanges ignores leading/trailing whitespace differences when true.
	IgnoreWhitespaceChanges bool
	// FormatStrings canonicalizes embedded JSON strings before comparison.
	// When true, if both values parse as valid JSON, formatting-only differences are ignored.
	FormatStrings bool
	// IgnoreValueChanges excludes value changes from the report when true.
	IgnoreValueChanges bool
	// DetectKubernetes enables Kubernetes resource structure detection.
	DetectKubernetes bool
	// DetectRenames enables document-level rename detection for Kubernetes resources.
	DetectRenames bool
	// IgnoreApiVersion omits apiVersion from K8s resource identifiers when matching.
	IgnoreApiVersion bool
	// AdditionalIdentifiers specifies additional fields to use as identifiers in named entry lists.
	AdditionalIdentifiers []string
	// NoCertInspection disables x509 certificate inspection, comparing as raw text.
	NoCertInspection bool
	// Swap reverses the from/to comparison.
	Swap bool
	// Chroot changes the comparison root to this path for both files.
	Chroot string
	// ChrootFrom changes only the 'from' file's root to this path.
	ChrootFrom string
	// ChrootTo changes only the 'to' file's root to this path.
	ChrootTo string
	// ChrootListToDocuments treats list items as separate documents when chroot points to a list.
	ChrootListToDocuments bool
}

// Compare compares two YAML documents and returns the differences.
// The from and to parameters should contain valid YAML content.
// If opts is nil, default options are used.
func Compare(from, to []byte, opts *Options) ([]Difference, error) {
	if opts == nil {
		opts = &Options{}
	}

	// Parse both YAML documents
	fromDocs, err := parse(from)
	if err != nil {
		return nil, err
	}
	toDocs, err := parse(to)
	if err != nil {
		return nil, err
	}

	// Apply swap if requested
	if opts.Swap {
		fromDocs, toDocs = toDocs, fromDocs
	}

	// Apply chroot if specified
	if opts.Chroot != "" {
		fromDocs, err = applyChrootToDocs(fromDocs, opts.Chroot, opts.ChrootListToDocuments)
		if err != nil {
			return nil, err
		}
		toDocs, err = applyChrootToDocs(toDocs, opts.Chroot, opts.ChrootListToDocuments)
		if err != nil {
			return nil, err
		}
	} else {
		if opts.ChrootFrom != "" {
			fromDocs, err = applyChrootToDocs(fromDocs, opts.ChrootFrom, opts.ChrootListToDocuments)
			if err != nil {
				return nil, err
			}
		}
		if opts.ChrootTo != "" {
			toDocs, err = applyChrootToDocs(toDocs, opts.ChrootTo, opts.ChrootListToDocuments)
			if err != nil {
				return nil, err
			}
		}
	}

	// Compare documents and sort results
	pathOrder := extractPathOrder(fromDocs, toDocs, opts)
	diffs := compareDocs(fromDocs, toDocs, opts)
	sortDiffsWithOrder(diffs, pathOrder)

	return diffs, nil
}

// pathWalker holds state for extractPathOrder to avoid per-node DiffPath and String allocations.
// It maintains an incremental byte buffer that is pushed/popped as we descend/ascend,
// avoiding full path reconstruction on every node.
type pathWalker struct {
	pathOrder map[string]int
	index     int
	opts      *Options
	// buf is the incrementally-built path string as bytes.
	buf []byte
	// lengths tracks buf length before each push for efficient pop.
	lengths []int
}

// push appends a segment to the running path buffer.
func (w *pathWalker) push(seg string) {
	w.lengths = append(w.lengths, len(w.buf))
	switch {
	case strings.Contains(seg, "."):
		w.buf = append(w.buf, '[')
		w.buf = append(w.buf, seg...)
		w.buf = append(w.buf, ']')
	case len(w.buf) > 0 && (len(seg) == 0 || seg[0] != '['):
		w.buf = append(w.buf, '.')
		w.buf = append(w.buf, seg...)
	default:
		w.buf = append(w.buf, seg...)
	}
}

// pop restores the path buffer to the state before the last push.
func (w *pathWalker) pop() {
	n := len(w.lengths) - 1
	w.buf = w.buf[:w.lengths[n]]
	w.lengths = w.lengths[:n]
}

// register registers the current path in pathOrder if not already present.
func (w *pathWalker) register() {
	if len(w.buf) == 0 {
		return
	}
	key := string(w.buf)
	if _, exists := w.pathOrder[key]; !exists {
		w.pathOrder[key] = w.index
		w.index++
	}
}

// walk recursively extracts path ordering from a parsed YAML value using an incremental path buffer.
func (w *pathWalker) walk(val any) {
	switch v := val.(type) {
	case *OrderedMap:
		w.register()
		for _, key := range v.Keys {
			w.push(key)
			w.walk(v.Values[key])
			w.pop()
		}
	case map[string]any:
		w.register()
		var keys []string
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			w.push(key)
			w.walk(v[key])
			w.pop()
		}
	case []any:
		w.register()
		for i, item := range v {
			var seg string
			if id := getIdentifier(item, w.opts); isComparableIdentifier(id) {
				seg = sprintIdentifier(id)
			} else {
				seg = strconv.Itoa(i)
			}
			w.push(seg)
			w.walk(item)
			w.pop()
		}
	default:
		w.register()
	}
}

// extractPathOrder extracts the order of all paths from parsed documents.
func extractPathOrder(fromDocs, toDocs []any, opts *Options) map[string]int {
	w := pathWalker{
		pathOrder: make(map[string]int),
		opts:      opts,
		buf:       make([]byte, 0, 256),
		lengths:   make([]int, 0, 16),
	}

	for _, doc := range fromDocs {
		w.walk(doc)
	}
	for _, doc := range toDocs {
		w.walk(doc)
	}

	return w.pathOrder
}

// hasIdentifierField checks if a value is a map containing "name" or "id" fields.
func hasIdentifierField(val any) bool {
	if om, ok := val.(*OrderedMap); ok {
		if _, hasName := om.Values["name"]; hasName {
			return true
		}
		if _, hasID := om.Values["id"]; hasID {
			return true
		}
	}
	if m, ok := val.(map[string]any); ok {
		if _, hasName := m["name"]; hasName {
			return true
		}
		if _, hasID := m["id"]; hasID {
			return true
		}
	}
	return false
}

// isListEntryDiff checks if a difference represents a list entry.
func isListEntryDiff(diff Difference) bool {
	// Check if last segment is numeric (list index)
	if diff.Path.HasNumericLast() {
		return true
	}
	// Check for bracket notation [0], [1], etc. (bare doc index)
	if diff.Path.IsBareDocIndex() {
		return true
	}
	// Check if the value is a map with identifier fields
	var val any
	if diff.To != nil {
		val = diff.To
	} else {
		val = diff.From
	}
	return hasIdentifierField(val)
}

// compareByRootOrder compares two diff paths by their root component's document order.
// Returns (comparison result, true) if roots differ and can be compared, (0, false) otherwise.
func compareByRootOrder(pathI, pathJ DiffPath, pathOrder map[string]int) (int, bool) {
	rootI := pathI.Root()
	rootJ := pathJ.Root()

	if rootI == rootJ {
		return 0, false
	}

	orderI, okI := pathOrder[rootI]
	orderJ, okJ := pathOrder[rootJ]
	if okI && okJ {
		return cmp.Compare(orderI, orderJ), true
	}
	return cmp.Compare(rootI, rootJ), true
}

// compareByExactOrParentOrder compares two paths within the same root using exact or parent order.
func compareByExactOrParentOrder(pathI, pathJ DiffPath, pathOrder map[string]int, findParentOrder func(DiffPath) (int, bool)) int {
	// First try exact path match
	orderI, okI := pathOrder[pathI.String()]
	orderJ, okJ := pathOrder[pathJ.String()]
	if okI && okJ {
		return cmp.Compare(orderI, orderJ)
	}
	if okI && !okJ {
		return -1
	}
	if !okI && okJ {
		return 1
	}

	parentOrderI, okI := findParentOrder(pathI)
	parentOrderJ, okJ := findParentOrder(pathJ)
	if okI && okJ {
		if c := cmp.Compare(parentOrderI, parentOrderJ); c != 0 {
			return c
		}
	}

	// Within same parent, sort by depth first
	if c := cmp.Compare(pathI.Depth(), pathJ.Depth()); c != 0 {
		return c
	}

	return cmp.Compare(pathI.String(), pathJ.String())
}

func sortDiffsWithOrder(diffs []Difference, pathOrder map[string]int) {
	if len(diffs) == 0 {
		return
	}

	// Pre-compute path strings to avoid repeated String() calls during O(n log n) comparisons.
	pathStrs := make([]string, len(diffs))
	for i := range diffs {
		pathStrs[i] = diffs[i].Path.String()
	}

	findParentOrder := func(path DiffPath) (int, bool) {
		for !path.IsEmpty() {
			if order, ok := pathOrder[path.String()]; ok {
				return order, true
			}
			path = path.Parent()
		}
		return 0, false
	}

	// Sort indices to keep pathStrs aligned with diffs.
	indices := make([]int, len(diffs))
	for i := range indices {
		indices[i] = i
	}

	slices.SortStableFunc(indices, func(i, j int) int {
		pathI := diffs[i].Path
		pathJ := diffs[j].Path

		if c, decided := compareByRootOrder(pathI, pathJ, pathOrder); decided {
			return c
		}

		return compareByExactOrParentOrderCached(pathStrs[i], pathStrs[j], pathI, pathJ, pathOrder, findParentOrder)
	})

	// Reorder diffs according to sorted indices.
	sorted := make([]Difference, len(diffs))
	for i, idx := range indices {
		sorted[i] = diffs[idx]
	}
	copy(diffs, sorted)
}

// compareByExactOrParentOrderCached is like compareByExactOrParentOrder but uses pre-computed path strings.
func compareByExactOrParentOrderCached(strI, strJ string, pathI, pathJ DiffPath, pathOrder map[string]int, findParentOrder func(DiffPath) (int, bool)) int {
	orderI, okI := pathOrder[strI]
	orderJ, okJ := pathOrder[strJ]
	if okI && okJ {
		return cmp.Compare(orderI, orderJ)
	}
	if okI && !okJ {
		return -1
	}
	if !okI && okJ {
		return 1
	}

	parentOrderI, okI := findParentOrder(pathI)
	parentOrderJ, okJ := findParentOrder(pathJ)
	if okI && okJ {
		if c := cmp.Compare(parentOrderI, parentOrderJ); c != 0 {
			return c
		}
	}

	if c := cmp.Compare(pathI.Depth(), pathJ.Depth()); c != 0 {
		return c
	}

	return cmp.Compare(strI, strJ)
}
