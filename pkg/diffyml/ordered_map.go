// ordered_map.go - OrderedMap preserves YAML field order during parsing.
//
// Standard Go maps do not preserve insertion order, but YAML documents
// have a defined key order. OrderedMap stores both Keys (in order) and
// Values (for fast lookup) so that formatters can reproduce the original
// field ordering.
package diffyml

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// OrderedMap is a map that preserves insertion order of keys.
type OrderedMap struct {
	Keys   []string
	Values map[string]interface{}
}

// NewOrderedMap creates an empty OrderedMap.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Keys:   nil,
		Values: make(map[string]interface{}),
	}
}

// String implements fmt.Stringer so that %v produces readable inline YAML
// instead of Go's default struct representation.
func (m *OrderedMap) String() string {
	parts := make([]string, 0, len(m.Keys))
	for _, key := range m.Keys {
		parts = append(parts, fmt.Sprintf("%s: %v", key, m.Values[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// ParseWithOrder parses YAML content into documents using OrderedMap for mappings
// so that field order from the source document is preserved.
func ParseWithOrder(content []byte) ([]interface{}, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var docs []interface{}

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
func nodeToInterface(node *yaml.Node) interface{} {
	return nodeToInterfaceWithCycleDetection(node, make(map[*yaml.Node]bool))
}

// nodeToInterfaceWithCycleDetection is the recursive implementation of nodeToInterface.
// The seen set tracks alias targets to detect cycles (e.g. an anchor referencing itself).
func nodeToInterfaceWithCycleDetection(node *yaml.Node, seen map[*yaml.Node]bool) interface{} {
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
		list := make([]interface{}, 0, len(node.Content))
		for _, child := range node.Content {
			list = append(list, nodeToInterfaceWithCycleDetection(child, seen))
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
func resolveScalar(node *yaml.Node) interface{} {
	// Use yaml.v3's own resolution by decoding into interface{}
	var val interface{}
	if err := node.Decode(&val); err != nil {
		return node.Value
	}
	return val
}
