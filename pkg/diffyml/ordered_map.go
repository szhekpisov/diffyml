// ordered_map.go - OrderedMap preserves YAML field order during parsing.
//
// Standard Go maps do not preserve insertion order, but YAML documents
// have a defined key order. OrderedMap stores both Keys (in order) and
// Values (for fast lookup) so that formatters can reproduce the original
// field ordering.
package diffyml

import (
	"bytes"
	"errors"
	"io"
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
		if errors.Is(err, io.EOF) {
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
	// Start with nil seen map; only allocate if we encounter an alias.
	return nodeToInterfaceImpl(node, nil)
}

// nodeToInterfaceImpl is the recursive implementation of nodeToInterface.
// The seen set tracks alias targets to detect cycles (e.g. an anchor referencing itself).
// It is nil until the first AliasNode is encountered, avoiding allocation in the common case.
func nodeToInterfaceImpl(node *yaml.Node, seen map[*yaml.Node]bool) any {
	if node == nil {
		return nil
	}

	// A document node wraps a single content node.
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil
		}
		return nodeToInterfaceImpl(node.Content[0], seen)
	}

	switch node.Kind {
	case yaml.MappingNode:
		nKeys := len(node.Content) / 2
		om := &OrderedMap{
			Keys:   make([]string, 0, nKeys),
			Values: make(map[string]any, nKeys),
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if key == "<<" {
				// YAML merge key: merge the referenced map's entries
				merged := nodeToInterfaceImpl(node.Content[i+1], seen)
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
			val := nodeToInterfaceImpl(node.Content[i+1], seen)
			om.Keys = append(om.Keys, key)
			om.Values[key] = val
		}
		return om

	case yaml.SequenceNode:
		list := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			list = append(list, nodeToInterfaceImpl(child, seen))
		}
		return list

	case yaml.ScalarNode:
		return resolveScalar(node)

	case yaml.AliasNode:
		if seen == nil {
			seen = make(map[*yaml.Node]bool)
		}
		if seen[node.Alias] {
			return nil // break cycle
		}
		seen[node.Alias] = true
		result := nodeToInterfaceImpl(node.Alias, seen)
		delete(seen, node.Alias)
		return result

	default:
		return nil
	}
}

// resolveScalar converts a scalar yaml.Node into the appropriate Go type.
// Handles common types directly without going through the yaml.v3 decoder
// for better performance on hot paths.
func resolveScalar(node *yaml.Node) any {
	tag := node.Tag
	value := node.Value

	switch tag {
	case "!!str":
		return value
	case "!!int":
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	case "!!float":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	case "!!bool":
		switch strings.ToLower(value) {
		case "true":
			return true
		case "false":
			return false
		}
	case "!!null":
		return nil
	}

	// Fallback to yaml.v3 decoder for uncommon types (timestamps, binary, etc.)
	var val any
	if err := node.Decode(&val); err != nil {
		return value
	}
	return val
}
