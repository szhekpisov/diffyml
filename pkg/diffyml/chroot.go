// chroot.go - Path navigation to focus comparison on YAML subsections.
//
// Allows comparing only specific parts of YAML documents using dot-notation paths.
// Supports array indexing (e.g., "items[0].name") and separate paths for from/to files.
// Key functions: applyChroot, applyChrootToDocs.
//
// Operates on *yaml.Node trees so the post-chroot output keeps source-line info
// and feeds the rest of the node pipeline (extractPathOrder, comparator)
// without re-deriving anything.
package diffyml

import (
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
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

// navigateToPath navigates to the specified dot-notation path within a parsed
// YAML node tree. Path format: "level1.level2.key" or "items[0].name" for list
// access. Returns the node at the path, or a *ChrootError if the path doesn't
// exist or traverses a type that can't be descended.
//
// A leading DocumentNode is unwrapped automatically so callers can pass either
// a DocumentNode straight from parse() or a sub-node. AliasNodes encountered
// mid-traversal are dereferenced.
func navigateToPath(doc *yaml.Node, path string) (*yaml.Node, error) {
	segments, err := parsePath(path)
	if err != nil {
		return nil, &ChrootError{
			Path:    path,
			Message: err.Error(),
		}
	}
	current := unwrapDocOrAlias(doc)

	for _, seg := range segments {
		current = unwrapDocOrAlias(current)
		if seg.isIndex {
			if current == nil || current.Kind != yaml.SequenceNode {
				return nil, &ChrootError{
					Path:    path,
					Message: fmt.Sprintf("expected list at %q, got %T", seg.key, nodeToInterface(current)),
				}
			}
			if seg.index < 0 || seg.index >= len(current.Content) {
				return nil, &ChrootError{
					Path:    path,
					Message: fmt.Sprintf("index %d out of bounds (list has %d items)", seg.index, len(current.Content)),
				}
			}
			current = current.Content[seg.index]
			continue
		}

		// Map key access.
		if current == nil || current.Kind != yaml.MappingNode {
			return nil, &ChrootError{
				Path:    path,
				Message: fmt.Sprintf("expected map at %q, got %T", seg.key, nodeToInterface(current)),
			}
		}
		val := lookupMappingValueNode(current, seg.key)
		if val == nil {
			return nil, &ChrootError{
				Path:    path,
				Message: fmt.Sprintf("key %q not found", seg.key),
			}
		}
		current = val
	}

	return current, nil
}

// unwrapDocOrAlias strips a leading DocumentNode wrapper or follows an alias
// chain to its target, so callers can hand any node shape to navigateToPath.
// Cyclic alias chains resolve to nil via resolveAlias.
func unwrapDocOrAlias(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.DocumentNode {
		if len(n.Content) == 0 {
			return nil
		}
		n = n.Content[0]
	}
	if n != nil && n.Kind == yaml.AliasNode {
		n = resolveAlias(n)
	}
	return n
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
	parts, err := splitPath(path)
	if err != nil {
		return nil, err
	}

	for _, part := range parts {
		// Check for bracket notation: key[index]
		if idx := strings.Index(part, "["); idx >= 0 {
			key := part[:idx]
			indexStr := part[idx+1 : len(part)-1]
			// indexStr must be the only bracketed segment; reject extra
			// brackets either inside (e.g. "key[0][1]") or trailing past ']'
			// (e.g. "key[1]extra", where the trailing chars push ']' into indexStr).
			if strings.ContainsAny(indexStr, "[]") {
				return nil, fmt.Errorf("invalid list index syntax %q", part)
			}
			if indexStr == "" {
				return nil, fmt.Errorf("empty list index in %q", part)
			}

			if key != "" {
				// First add the key segment
				segments = append(segments, pathSegment{key: key})
			}

			// Then add the index segment (numeric) or quoted map key (non-numeric)
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				// Non-numeric bracket content is a quoted map key (e.g., [helm.sh/chart])
				segments = append(segments, pathSegment{key: indexStr})
			} else {
				segments = append(segments, pathSegment{index: index, isIndex: true})
			}
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

// applyChroot applies chroot path scoping to a single document node. When
// listToDocuments is true and the chrooted node is a SequenceNode, the list's
// children are returned as separate document slots; otherwise the chrooted
// node is wrapped as a single-document slice.
func applyChroot(doc *yaml.Node, path string, listToDocuments bool) ([]*yaml.Node, error) {
	if path == "" {
		return []*yaml.Node{doc}, nil
	}

	result, err := navigateToPath(doc, path)
	if err != nil {
		return nil, err
	}

	if listToDocuments {
		target := unwrapDocOrAlias(result)
		if target != nil && target.Kind == yaml.SequenceNode {
			expanded := make([]*yaml.Node, len(target.Content))
			copy(expanded, target.Content)
			return expanded, nil
		}
	}

	return []*yaml.Node{result}, nil
}

// applyChrootToDocs applies chroot to multiple parsed documents.
func applyChrootToDocs(docs []*yaml.Node, path string, listToDocuments bool) ([]*yaml.Node, error) {
	var result []*yaml.Node
	for _, doc := range docs {
		chrootDocs, err := applyChroot(doc, path, listToDocuments)
		if err != nil {
			return nil, err
		}
		result = append(result, chrootDocs...)
	}

	return result, nil
}
