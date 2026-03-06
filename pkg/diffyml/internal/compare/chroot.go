// chroot.go - Path navigation to focus comparison on YAML subsections.
//
// Allows comparing only specific parts of YAML documents using dot-notation paths.
// Supports array indexing (e.g., "items[0].name") and separate paths for from/to files.
// Key functions: ApplyChroot(), ApplyChrootToDocs().
package compare

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// NavigateToPath navigates to the specified dot-notation path within a document.
// Path format: "level1.level2.key" or "items[0].name" for list access.
// Returns the value at the path, or an error if path doesn't exist.
func NavigateToPath(doc interface{}, path string) (interface{}, error) {
	if path == "" {
		return doc, nil
	}

	// Parse and navigate path segments
	segments, err := ParsePath(path)
	if err != nil {
		return nil, &types.ChrootError{
			Path:    path,
			Message: err.Error(),
		}
	}
	current := doc

	for _, seg := range segments {
		if seg.IsIndex {
			// Array index access
			list, ok := current.([]interface{})
			if !ok {
				return nil, &types.ChrootError{
					Path:    path,
					Message: fmt.Sprintf("expected list at %q, got %T", seg.Key, current),
				}
			}
			if seg.Index < 0 || seg.Index >= len(list) {
				return nil, &types.ChrootError{
					Path:    path,
					Message: fmt.Sprintf("index %d out of bounds (list has %d items)", seg.Index, len(list)),
				}
			}
			current = list[seg.Index]
		} else {
			// Map key access
			om := types.ToOrderedMap(current)
			if om == nil {
				return nil, &types.ChrootError{
					Path:    path,
					Message: fmt.Sprintf("expected map at %q, got %T", seg.Key, current),
				}
			}

			val, exists := om.Values[seg.Key]
			if !exists {
				return nil, &types.ChrootError{
					Path:    path,
					Message: fmt.Sprintf("key %q not found", seg.Key),
				}
			}
			current = val
		}
	}

	return current, nil
}

// PathSegment represents a single segment in a path.
type PathSegment struct {
	Key     string // The key name (for maps)
	Index   int    // The index (for lists)
	IsIndex bool   // True if this segment is a list index
}

// ParsePath parses a dot-notation path with optional index accessors.
// Examples: "foo.bar", "items[0]", "data[0].name"
func ParsePath(path string) ([]PathSegment, error) {
	var segments []PathSegment
	if path == "" {
		return segments, nil
	}

	// Split by dots, but handle bracket notation
	parts, err := SplitPath(path)
	if err != nil {
		return nil, err
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Check for bracket notation: key[index]
		if idx := strings.Index(part, "["); idx >= 0 {
			if strings.Count(part, "[") != 1 || strings.Count(part, "]") != 1 || !strings.HasSuffix(part, "]") {
				return nil, fmt.Errorf("invalid list index syntax %q", part)
			}
			// Has index accessor
			key := part[:idx]
			indexStr := part[idx+1 : len(part)-1] // Remove [ and ]
			if indexStr == "" {
				return nil, fmt.Errorf("empty list index in %q", part)
			}

			if key != "" {
				// First add the key segment
				segments = append(segments, PathSegment{Key: key})
			}

			// Then add the index segment
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid list index %q", indexStr)
			}
			segments = append(segments, PathSegment{Index: index, IsIndex: true})
		} else {
			// Simple key
			segments = append(segments, PathSegment{Key: part})
		}
	}

	return segments, nil
}

// SplitPath splits a path by dots, preserving bracket notation.
func SplitPath(path string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inBracket := false

	for _, r := range path {
		switch r {
		case '.':
			if !inBracket {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
				continue
			}
		case '[':
			if inBracket {
				return nil, fmt.Errorf("invalid path syntax %q", path)
			}
			inBracket = true
		case ']':
			if !inBracket {
				return nil, fmt.Errorf("invalid path syntax %q", path)
			}
			inBracket = false
		}
		current.WriteRune(r)
	}
	if inBracket {
		return nil, fmt.Errorf("invalid path syntax %q", path)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}

// ApplyChroot applies chroot path scoping to a document.
// If listToDocuments is true and the path points to a list,
// each list item is returned as a separate document.
func ApplyChroot(doc interface{}, path string, listToDocuments bool) ([]interface{}, error) {
	if path == "" {
		return []interface{}{doc}, nil
	}

	result, err := NavigateToPath(doc, path)
	if err != nil {
		return nil, err
	}

	// Check if result is a list and listToDocuments is enabled
	if list, ok := result.([]interface{}); ok && listToDocuments {
		return list, nil
	}

	// Return as single document
	return []interface{}{result}, nil
}

// ApplyChrootToDocs applies chroot to multiple documents.
func ApplyChrootToDocs(docs []interface{}, path string, listToDocuments bool) ([]interface{}, error) {
	if path == "" {
		return docs, nil
	}

	var result []interface{}
	for _, doc := range docs {
		chrootDocs, err := ApplyChroot(doc, path, listToDocuments)
		if err != nil {
			return nil, err
		}
		result = append(result, chrootDocs...)
	}

	return result, nil
}
