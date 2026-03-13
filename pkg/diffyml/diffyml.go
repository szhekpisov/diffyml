package diffyml

import (
	"cmp"
	"fmt"
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
	// Path is the dot-notation path to the changed value (e.g., "some.yaml.structure.name").
	Path string
	// Type indicates the kind of change (added, removed, modified, order changed).
	Type DiffType
	// From is the original value (nil for additions).
	From any
	// To is the new value (nil for removals).
	To any
	// DocumentIndex indicates which document in a multi-document YAML file (0-based).
	DocumentIndex int
}

// Options configures the comparison behavior.
type Options struct {
	// IgnoreOrderChanges ignores list order differences when true.
	IgnoreOrderChanges bool
	// IgnoreWhitespaceChanges ignores leading/trailing whitespace differences when true.
	IgnoreWhitespaceChanges bool
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

// registerPath registers a path in the pathOrder map if not already present.
func registerPath(pathOrder map[string]int, index *int, prefix string) {
	if prefix == "" {
		return
	}
	if _, exists := pathOrder[prefix]; !exists {
		pathOrder[prefix] = *index
		*index++
	}
}

// extractPathsFromValue recursively extracts path ordering from a parsed YAML value.
func extractPathsFromValue(prefix string, val any, opts *Options, pathOrder map[string]int, index *int) {
	switch v := val.(type) {
	case *OrderedMap:
		registerPath(pathOrder, index, prefix)
		for _, key := range v.Keys {
			childPath := key
			if prefix != "" {
				childPath = prefix + "." + key
			}
			extractPathsFromValue(childPath, v.Values[key], opts, pathOrder, index)
		}
	case map[string]any:
		registerPath(pathOrder, index, prefix)
		var keys []string
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPath := key
			if prefix != "" {
				childPath = prefix + "." + key
			}
			extractPathsFromValue(childPath, v[key], opts, pathOrder, index)
		}
	case []any:
		registerPath(pathOrder, index, prefix)
		for i, item := range v {
			var childPath string
			if id := getIdentifier(item, opts); isComparableIdentifier(id) {
				childPath = prefix + "." + fmt.Sprint(id)
			} else {
				childPath = prefix + "." + strconv.Itoa(i)
			}
			extractPathsFromValue(childPath, item, opts, pathOrder, index)
		}
	default:
		registerPath(pathOrder, index, prefix)
	}
}

// sortDiffs sorts differences by path for consistent output.
// Sorts root-level additions first, then by path depth and alphabetically.
// extractPathOrder extracts the order of all paths from parsed documents
func extractPathOrder(fromDocs, toDocs []any, opts *Options) map[string]int {
	pathOrder := make(map[string]int)
	index := 0

	for _, doc := range fromDocs {
		extractPathsFromValue("", doc, opts, pathOrder, &index)
	}
	for _, doc := range toDocs {
		extractPathsFromValue("", doc, opts, pathOrder, &index)
	}

	return pathOrder
}

// hasNumericPathSuffix checks if a path ends with a numeric suffix after a dot (e.g., ".0", ".1").
func hasNumericPathSuffix(path string) bool {
	lastDot := strings.LastIndex(path, ".")
	if lastDot < 0 || lastDot >= len(path)-1 {
		return false
	}
	suffix := path[lastDot+1:]
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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

// isListEntry checks if a difference represents a list entry
// This matches the logic in formatter.go
func isListEntryDiff(diff Difference) bool {
	path := diff.Path

	// Check for bracket notation [0], [1], etc.
	if len(path) > 0 && path[len(path)-1] == ']' {
		return true
	}

	if hasNumericPathSuffix(path) {
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
func compareByRootOrder(pathI, pathJ string, pathOrder map[string]int) (int, bool) {
	rootI := pathI
	rootJ := pathJ
	if dotIdx := strings.Index(pathI, "."); dotIdx != -1 {
		rootI = pathI[:dotIdx]
	}
	if dotIdx := strings.Index(pathJ, "."); dotIdx != -1 {
		rootJ = pathJ[:dotIdx]
	}

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
func compareByExactOrParentOrder(pathI, pathJ string, pathOrder map[string]int, findParentOrder func(string) (int, bool)) int {
	// First try exact path match
	orderI, okI := pathOrder[pathI]
	orderJ, okJ := pathOrder[pathJ]
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
	if c := cmp.Compare(strings.Count(pathI, "."), strings.Count(pathJ, ".")); c != 0 {
		return c
	}

	return cmp.Compare(pathI, pathJ)
}

func sortDiffsWithOrder(diffs []Difference, pathOrder map[string]int) {
	findParentOrder := func(path string) (int, bool) {
		for {
			if order, ok := pathOrder[path]; ok {
				return order, true
			}
			lastDot := strings.LastIndex(path, ".")
			if lastDot == -1 {
				break
			}
			path = path[:lastDot]
		}
		return 0, false
	}

	slices.SortStableFunc(diffs, func(diffI, diffJ Difference) int {
		pathI := diffI.Path
		pathJ := diffJ.Path

		if c, decided := compareByRootOrder(pathI, pathJ, pathOrder); decided {
			return c
		}

		return compareByExactOrParentOrder(pathI, pathJ, pathOrder, findParentOrder)
	})
}
