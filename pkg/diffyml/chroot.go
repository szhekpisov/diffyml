// chroot.go - Path navigation to focus comparison on YAML subsections.
//
// Allows comparing only specific parts of YAML documents using dot-notation paths.
// Supports array indexing (e.g., "items[0].name") and separate paths for from/to files.
// Key functions: ApplyChroot(), applyChrootToDocs().
package diffyml

import (
	"fmt"
	"strconv"
	"strings"
)

// ChrootError represents an error navigating to a chroot path.
type ChrootError struct {
	Path    string
	Message string
}

// Error implements the error interface.
func (e *ChrootError) Error() string {
	return fmt.Sprintf("chroot path %q: %s", e.Path, e.Message)
}

// navigateToPath navigates to the specified dot-notation path within a document.
// Path format: "level1.level2.key" or "items[0].name" for list access.
// Returns the value at the path, or an error if path doesn't exist.
func navigateToPath(doc interface{}, path string) (interface{}, error) {
	if path == "" {
		return doc, nil
	}

	// Parse and navigate path segments
	segments, err := parsePath(path)
	if err != nil {
		return nil, &ChrootError{
			Path:    path,
			Message: err.Error(),
		}
	}
	current := doc

	for _, seg := range segments {
		if seg.isIndex {
			// Array index access
			list, ok := current.([]interface{})
			if !ok {
				return nil, &ChrootError{
					Path:    path,
					Message: fmt.Sprintf("expected list at %q, got %T", seg.key, current),
				}
			}
			if seg.index < 0 || seg.index >= len(list) {
				return nil, &ChrootError{
					Path:    path,
					Message: fmt.Sprintf("index %d out of bounds (list has %d items)", seg.index, len(list)),
				}
			}
			current = list[seg.index]
		} else {
			// Map key access - support both OrderedMap and regular map
			var val interface{}
			var exists bool

			switch m := current.(type) {
			case *OrderedMap:
				val, exists = m.Values[seg.key]
			case map[string]interface{}:
				val, exists = m[seg.key]
			default:
				return nil, &ChrootError{
					Path:    path,
					Message: fmt.Sprintf("expected map at %q, got %T", seg.key, current),
				}
			}

			if !exists {
				return nil, &ChrootError{
					Path:    path,
					Message: fmt.Sprintf("key %q not found", seg.key),
				}
			}
			current = val
		}
	}

	return current, nil
}

// pathSegment represents a single segment in a path.
type pathSegment struct {
	key     string // The key name (for maps)
	index   int    // The index (for lists)
	isIndex bool   // True if this segment is a list index
}

// parsePath parses a dot-notation path with optional index accessors.
// Examples: "foo.bar", "items[0]", "data[0].name"
func parsePath(path string) ([]pathSegment, error) {
	var segments []pathSegment
	if path == "" {
		return segments, nil
	}

	// Split by dots, but handle bracket notation
	parts, err := splitPath(path)
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
				segments = append(segments, pathSegment{key: key})
			}

			// Then add the index segment
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid list index %q", indexStr)
			}
			segments = append(segments, pathSegment{index: index, isIndex: true})
		} else {
			// Simple key
			segments = append(segments, pathSegment{key: part})
		}
	}

	return segments, nil
}

// splitPath splits a path by dots, preserving bracket notation.
func splitPath(path string) ([]string, error) {
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

// applyChroot applies chroot path scoping to a document.
// If listToDocuments is true and the path points to a list,
// each list item is returned as a separate document.
func applyChroot(doc interface{}, path string, listToDocuments bool) ([]interface{}, error) {
	if path == "" {
		return []interface{}{doc}, nil
	}

	result, err := navigateToPath(doc, path)
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

// applyChrootToDocs applies chroot to multiple documents.
func applyChrootToDocs(docs []interface{}, path string, listToDocuments bool) ([]interface{}, error) {
	if path == "" {
		return docs, nil
	}

	var result []interface{}
	for _, doc := range docs {
		chrootDocs, err := applyChroot(doc, path, listToDocuments)
		if err != nil {
			return nil, err
		}
		result = append(result, chrootDocs...)
	}

	return result, nil
}
