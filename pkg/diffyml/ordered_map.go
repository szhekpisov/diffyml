// ordered_map.go - OrderedMap preserves YAML field order during parsing.
//
// Standard Go maps do not preserve insertion order, but YAML documents
// have a defined key order. OrderedMap stores both Keys (in order) and
// Values (for fast lookup) so that formatters can reproduce the original
// field ordering.
package diffyml

import (
	"bytes"
	"io"
	"math"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// OrderedMap is a map that preserves insertion order of keys.
type OrderedMap struct {
	Keys   []string
	Values map[string]any
}

// NewOrderedMap creates an empty OrderedMap.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Keys:   nil,
		Values: make(map[string]any),
	}
}

// ParseWithOrder parses YAML content into documents using OrderedMap for mappings
// so that field order from the source document is preserved.
func ParseWithOrder(content []byte) ([]any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var docs []any

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, wrapParseError(err)
		}
		docs = append(docs, nodeToInterface(&node))
	}

	return docs, nil
}

// nodeToInterface converts a yaml.Node tree into Go values,
// using *OrderedMap for mapping nodes to preserve key order.
func nodeToInterface(node *yaml.Node) any {
	return nodeToInterfaceWithCycleDetection(node, make(map[*yaml.Node]bool))
}

// nodeToInterfaceWithCycleDetection is the recursive implementation of nodeToInterface.
// The seen set tracks alias targets to detect cycles (e.g. an anchor referencing itself).
func nodeToInterfaceWithCycleDetection(node *yaml.Node, seen map[*yaml.Node]bool) any {
	if node == nil {
		return nil
	}

	// A document node wraps a single content node.
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil
		}
		return nodeToInterfaceWithCycleDetection(node.Content[0], seen)
	}

	switch node.Kind {
	case yaml.MappingNode:
		om := NewOrderedMap()
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if key == "<<" {
				// YAML merge key: merge the referenced map's entries
				merged := nodeToInterfaceWithCycleDetection(node.Content[i+1], seen)
				if mergedMap, ok := merged.(*OrderedMap); ok {
					for _, mk := range mergedMap.Keys {
						if _, exists := om.Values[mk]; !exists {
							om.Keys = append(om.Keys, mk)
							om.Values[mk] = mergedMap.Values[mk]
						}
					}
				}
				continue
			}
			val := nodeToInterfaceWithCycleDetection(node.Content[i+1], seen)
			om.Keys = append(om.Keys, key)
			om.Values[key] = val
		}
		return om

	case yaml.SequenceNode:
		list := make([]any, len(node.Content))
		for i, child := range node.Content {
			list[i] = nodeToInterfaceWithCycleDetection(child, seen)
		}
		return list

	case yaml.ScalarNode:
		return resolveScalar(node)

	case yaml.AliasNode:
		if seen[node.Alias] {
			return nil // break cycle
		}
		seen[node.Alias] = true
		result := nodeToInterfaceWithCycleDetection(node.Alias, seen)
		delete(seen, node.Alias)
		return result

	default:
		return nil
	}
}

// resolveScalar converts a scalar yaml.Node into the appropriate Go type.
// Uses direct tag-based resolution instead of node.Decode for performance.
func resolveScalar(node *yaml.Node) any {
	tag := node.Tag
	value := node.Value

	// Resolve implicit tags (untagged scalars)
	if tag == "" || tag == "!" {
		return resolveUntaggedScalar(node, value)
	}

	switch tag {
	case "!!str":
		return value
	case "!!int":
		if i, err := strconv.ParseInt(value, 0, 64); err == nil {
			if i >= math.MinInt && i <= math.MaxInt {
				return int(i)
			}
			return i
		}
		if ui, err := strconv.ParseUint(value, 0, 64); err == nil {
			return ui
		}
		return value
	case "!!float":
		switch strings.ToLower(value) {
		case ".inf", "+.inf":
			return math.Inf(1)
		case "-.inf":
			return math.Inf(-1)
		case ".nan":
			return math.NaN()
		default:
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				return f
			}
			return value
		}
	case "!!bool":
		switch strings.ToLower(value) {
		case "true", "yes", "on":
			return true
		case "false", "no", "off":
			return false
		default:
			return value
		}
	case "!!null":
		return nil
	case "!!timestamp":
		// Decode to time.Time for proper type comparison
		var val any
		if err := node.Decode(&val); err == nil {
			return val
		}
		return value
	case "!!binary":
		return value
	default:
		// Unknown tag on a scalar node — Decode always succeeds and returns the
		// raw string, so just return the value directly.
		return value
	}
}

// looksLikeTimestamp checks if a value might be a YAML timestamp (YYYY-MM-DD...).
// This is a fast heuristic to avoid falling back to node.Decode for most strings.
func looksLikeTimestamp(value string) bool {
	// Minimum: YYYY-MM-DD = 10 chars
	if len(value) < 10 {
		return false
	}
	// Must start with 4 digits, then a hyphen
	return value[0] >= '0' && value[0] <= '9' &&
		value[1] >= '0' && value[1] <= '9' &&
		value[2] >= '0' && value[2] <= '9' &&
		value[3] >= '0' && value[3] <= '9' &&
		value[4] == '-'
}

// resolveUntaggedScalar resolves an untagged YAML scalar value to its Go type.
func resolveUntaggedScalar(node *yaml.Node, value string) any {
	// Null
	switch strings.ToLower(value) {
	case "", "~", "null":
		return nil
	}

	// Bool
	switch strings.ToLower(value) {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	}

	// Integer (try common bases)
	if i, err := strconv.ParseInt(value, 0, 64); err == nil {
		if i >= math.MinInt && i <= math.MaxInt {
			return int(i)
		}
		return i
	}

	// Float
	switch strings.ToLower(value) {
	case ".inf", "+.inf":
		return math.Inf(1)
	case "-.inf":
		return math.Inf(-1)
	case ".nan":
		return math.NaN()
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		// Only return float if it looks like a float (has dot or e/E)
		for _, c := range value {
			if c == '.' || c == 'e' || c == 'E' {
				return f
			}
		}
	}

	// Timestamp — fall back to node.Decode for correctness (rare path)
	if looksLikeTimestamp(value) {
		var val any
		if err := node.Decode(&val); err == nil {
			return val
		}
	}

	// Default: string
	return value
}
