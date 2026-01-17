// Package diffyml provides YAML diff functionality for comparing YAML documents.
package diffyml

import (
	"fmt"
	"sort"
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
	From interface{}
	// To is the new value (nil for removals).
	To interface{}
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

// sortDiffs sorts differences by path for consistent output.
// Sorts root-level additions first, then by path depth and alphabetically.
// extractPathOrder extracts the order of all paths from parsed documents
func extractPathOrder(fromDocs, toDocs []interface{}, opts *Options) map[string]int {
	pathOrder := make(map[string]int)
	index := 0

	var extractFromValue func(prefix string, val interface{})
	extractFromValue = func(prefix string, val interface{}) {
		switch v := val.(type) {
		case *OrderedMap:
			// Register the prefix itself
			if prefix != "" {
				if _, exists := pathOrder[prefix]; !exists {
					pathOrder[prefix] = index
					index++
				}
			}
			// Then recurse into children in order
			for _, key := range v.Keys {
				childPath := key
				if prefix != "" {
					childPath = prefix + "." + key
				}
				extractFromValue(childPath, v.Values[key])
			}
		case map[string]interface{}:
			// Register the prefix itself
			if prefix != "" {
				if _, exists := pathOrder[prefix]; !exists {
					pathOrder[prefix] = index
					index++
				}
			}
			// For regular maps, sort keys alphabetically for deterministic order
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
				extractFromValue(childPath, v[key])
			}
		case []interface{}:
			// Register the prefix itself
			if prefix != "" {
				if _, exists := pathOrder[prefix]; !exists {
					pathOrder[prefix] = index
					index++
				}
			}
			// For lists, process items by identifier if available
			for i, item := range v {
				var childPath string
				if id := getIdentifier(item, opts); isComparableIdentifier(id) {
					childPath = fmt.Sprintf("%s.%v", prefix, id)
				} else {
					childPath = fmt.Sprintf("%s.%d", prefix, i)
				}
				extractFromValue(childPath, item)
			}
		default:
			// Scalar - just register the path
			if prefix != "" {
				if _, exists := pathOrder[prefix]; !exists {
					pathOrder[prefix] = index
					index++
				}
			}
		}
	}

	// Extract from both documents to cover all keys
	for _, doc := range fromDocs {
		extractFromValue("", doc)
	}
	for _, doc := range toDocs {
		extractFromValue("", doc)
	}

	return pathOrder
}

// isListEntry checks if a difference represents a list entry
// This matches the logic in formatter.go
func isListEntryDiff(diff Difference) bool {
	path := diff.Path

	// Check for bracket notation [0], [1], etc.
	if len(path) > 0 && path[len(path)-1] == ']' {
		return true
	}

	// Check for dot notation .0, .1, etc. (path ends with .digit)
	if len(path) > 1 {
		lastDot := strings.LastIndex(path, ".")
		if lastDot >= 0 && lastDot < len(path)-1 {
			suffix := path[lastDot+1:]
			// Check if suffix is all digits
			isDigit := true
			for _, c := range suffix {
				if c < '0' || c > '9' {
					isDigit = false
					break
				}
			}
			if isDigit {
				return true
			}
		}
	}

	// Check if the value is a map with identifier fields (typical for list items)
	var val interface{}
	if diff.To != nil {
		val = diff.To
	} else {
		val = diff.From
	}

	// Check OrderedMap
	if om, ok := val.(*OrderedMap); ok {
		if _, hasName := om.Values["name"]; hasName {
			return true
		}
		if _, hasID := om.Values["id"]; hasID {
			return true
		}
	}

	// Check regular map
	if m, ok := val.(map[string]interface{}); ok {
		if _, hasName := m["name"]; hasName {
			return true
		}
		if _, hasID := m["id"]; hasID {
			return true
		}
	}

	return false
}

func sortDiffsWithOrder(diffs []Difference, pathOrder map[string]int) {
	sort.SliceStable(diffs, func(i, j int) bool {
		pathI := diffs[i].Path
		pathJ := diffs[j].Path
		diffI := diffs[i]
		diffJ := diffs[j]

		// Root-level additions (will be displayed as "(root level)") always come first
		// These are DiffAdded with no dots in path and not list entries
		isRootAddI := diffI.Type == DiffAdded && !strings.Contains(pathI, ".") && !isListEntryDiff(diffI)
		isRootAddJ := diffJ.Type == DiffAdded && !strings.Contains(pathJ, ".") && !isListEntryDiff(diffJ)

		if isRootAddI && !isRootAddJ {
			return true
		}
		if !isRootAddI && isRootAddJ {
			return false
		}

		// Extract root component (first segment before dot)
		rootI := pathI
		rootJ := pathJ
		if dotIdx := strings.Index(pathI, "."); dotIdx != -1 {
			rootI = pathI[:dotIdx]
		}
		if dotIdx := strings.Index(pathJ, "."); dotIdx != -1 {
			rootJ = pathJ[:dotIdx]
		}

		// Group by root component using document order
		if rootI != rootJ {
			orderI, okI := pathOrder[rootI]
			orderJ, okJ := pathOrder[rootJ]
			if okI && okJ {
				return orderI < orderJ
			}
			return rootI < rootJ // Fallback to alphabetical
		}

		// Within same root component, use path order from document
		// First try exact path match
		orderI, okI := pathOrder[pathI]
		orderJ, okJ := pathOrder[pathJ]
		if okI && okJ {
			return orderI < orderJ
		}

		// If one has order and other doesn't, prefer the one with order
		if okI && !okJ {
			return true
		}
		if !okI && okJ {
			return false
		}

		// Try to find a parent path that has order
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

		parentOrderI, okI := findParentOrder(pathI)
		parentOrderJ, okJ := findParentOrder(pathJ)
		if okI && okJ && parentOrderI != parentOrderJ {
			return parentOrderI < parentOrderJ
		}

		// Within same parent, sort by depth first
		depthI := strings.Count(pathI, ".")
		depthJ := strings.Count(pathJ, ".")
		if depthI != depthJ {
			return depthI < depthJ
		}

		// Then sort alphabetically as last resort
		return pathI < pathJ
	})
}
