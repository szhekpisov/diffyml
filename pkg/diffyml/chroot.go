// chroot.go - Path navigation to focus comparison on YAML subsections.
package diffyml

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

type ChrootError = types.ChrootError

func navigateToPath(doc interface{}, path string) (interface{}, error) {
	return compare.NavigateToPath(doc, path)
}

func applyChroot(doc interface{}, path string, listToDocuments bool) ([]interface{}, error) {
	return compare.ApplyChroot(doc, path, listToDocuments)
}

// pathSegment represents a single segment in a path.
// Kept locally so that tests in package diffyml can access the unexported key field.
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

	parts, err := splitPath(path)
	if err != nil {
		return nil, err
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		if idx := strings.Index(part, "["); idx >= 0 {
			if strings.Count(part, "[") != 1 || strings.Count(part, "]") != 1 || !strings.HasSuffix(part, "]") {
				return nil, fmt.Errorf("invalid list index syntax %q", part)
			}
			key := part[:idx]
			indexStr := part[idx+1 : len(part)-1]
			if indexStr == "" {
				return nil, fmt.Errorf("empty list index in %q", part)
			}

			if key != "" {
				segments = append(segments, pathSegment{key: key})
			}

			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid list index %q", indexStr)
			}
			segments = append(segments, pathSegment{index: index, isIndex: true})
		} else {
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
