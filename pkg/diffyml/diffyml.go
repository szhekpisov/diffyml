package diffyml

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
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
	// DiffUnchanged indicates a value equal between both documents. Only emitted
	// in inverse mode (Options.Unchanged); the normal comparison never reports it.
	DiffUnchanged
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
	// DocumentKind is the Kubernetes "kind" of the document (e.g., "Secret", "ConfigMap").
	// Empty for non-K8s documents or when detection is disabled. Used by sensitive value
	// masking to identify Secret resources without parsing DocumentName, since apiVersion
	// can itself contain "/" (e.g., "apps/v1").
	DocumentKind string
	// listEntry marks a collapsed DiffUnchanged entry whose immediate container
	// is a sequence (inverse mode only). The normal added/removed path infers
	// list-vs-map from the value shape via hasIdentifierField, but inverse mode
	// emits raw collapsed values, so the walk records the container kind directly
	// for isListEntryDiff. Unexported: only the inverse walk sets it.
	listEntry bool
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
	// Unchanged inverts the report: instead of differences, emit the keys/values
	// that are equal between both documents (the "inverse diff"). Equal subtrees
	// collapse to a single entry at the highest equal node.
	Unchanged bool
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

	// Parse both YAML inputs into per-document *yaml.Node trees. Nodes flow
	// end-to-end through chroot, extractPathOrder, and compareDocs;
	// nodeToInterface materialization is deferred to Difference.From/To
	// emission sites.
	fromNodes, err := parse(from)
	if err != nil {
		return nil, err
	}
	toNodes, err := parse(to)
	if err != nil {
		return nil, err
	}

	if opts.Swap {
		fromNodes, toNodes = toNodes, fromNodes
	}

	// Apply chroot on the node trees so post-chroot output keeps source-line
	// info and matches extractPathOrder's view exactly.
	if opts.Chroot != "" {
		fromNodes, err = applyChrootToDocs(fromNodes, opts.Chroot, opts.ChrootListToDocuments)
		if err != nil {
			return nil, err
		}
		toNodes, err = applyChrootToDocs(toNodes, opts.Chroot, opts.ChrootListToDocuments)
		if err != nil {
			return nil, err
		}
	} else {
		if opts.ChrootFrom != "" {
			fromNodes, err = applyChrootToDocs(fromNodes, opts.ChrootFrom, opts.ChrootListToDocuments)
			if err != nil {
				return nil, err
			}
		}
		if opts.ChrootTo != "" {
			toNodes, err = applyChrootToDocs(toNodes, opts.ChrootTo, opts.ChrootListToDocuments)
			if err != nil {
				return nil, err
			}
		}
	}

	// Compare documents and sort results. In inverse mode, collect equal
	// values instead of differences; both paths feed the same sort/filter/
	// format pipeline downstream.
	pathOrder := extractPathOrder(fromNodes, toNodes, opts)
	var diffs []Difference
	if opts.Unchanged {
		diffs = collectUnchangedDocs(fromNodes, toNodes, opts)
	} else {
		diffs = compareDocs(fromNodes, toNodes, opts)
	}
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
	// aliasSeen tracks alias targets currently being walked to break cycles
	// (e.g. an anchor whose subtree contains an alias back to itself).
	// Lazily initialised on first encounter.
	aliasSeen map[*yaml.Node]bool
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

// walk recursively extracts path ordering from a *yaml.Node tree using the
// incremental path buffer. After resolveMergeKeys runs at parse time, "<<" keys
// are gone from MappingNode contents, so MappingNode iteration is straight
// source-order. Aliases are dereferenced inline with a per-walk cycle break
// (mirroring nodeToInterfaceImpl's seen-set), so a cyclic anchor terminates
// instead of recursing forever.
func (w *pathWalker) walk(n *yaml.Node) {
	if n == nil {
		w.register()
		return
	}
	switch n.Kind {
	case yaml.DocumentNode:
		// DocumentNodes only appear at the root, where w.buf is empty and
		// w.register() is a no-op; skip the call and let the recursion guard
		// on empty Content handle the no-content case.
		if len(n.Content) > 0 {
			w.walk(n.Content[0])
		}
	case yaml.AliasNode:
		target := n.Alias
		if target == nil {
			return
		}
		// aliasSeen is allocated lazily on first alias encounter; a read on
		// the nil map returns the zero value (false), so the cycle check runs
		// unconditionally before allocation.
		if w.aliasSeen[target] {
			return
		}
		if w.aliasSeen == nil {
			w.aliasSeen = make(map[*yaml.Node]bool)
		}
		w.aliasSeen[target] = true
		w.walk(target)
		delete(w.aliasSeen, target)
	case yaml.MappingNode:
		w.register()
		for i := 0; i+1 < len(n.Content); i += 2 {
			w.push(n.Content[i].Value)
			w.walk(n.Content[i+1])
			w.pop()
		}
	case yaml.SequenceNode:
		w.register()
		for i, item := range n.Content {
			var seg string
			if id := getIdentifierNode(item, w.opts); isComparableIdentifier(id) {
				seg = sprintIdentifier(id)
			} else {
				seg = strconv.Itoa(i)
			}
			w.push(seg)
			w.walk(item)
			w.pop()
		}
	default:
		// ScalarNode and any unknown kind: just register the current path.
		w.register()
	}
}

// extractPathOrder extracts the order of all paths from parsed documents
// (per-document *yaml.Node trees, already merge-resolved by parseNodes).
func extractPathOrder(fromNodes, toNodes []*yaml.Node, opts *Options) map[string]int {
	w := pathWalker{
		pathOrder: make(map[string]int),
		opts:      opts,
		buf:       make([]byte, 0, 256),
		lengths:   make([]int, 0, 16),
	}

	for _, n := range fromNodes {
		w.walk(n)
	}
	for _, n := range toNodes {
		w.walk(n)
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
	// Inverse mode emits raw collapsed values, so the hasIdentifierField shape
	// heuristic below would misfire on map subtrees whose value carries a
	// name/id key. The inverse walk records the real container kind instead.
	if diff.Type == DiffUnchanged {
		return diff.listEntry
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

// compareByExactOrParentOrderCached compares two paths within the same root using exact or parent order, with pre-computed path strings.
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
