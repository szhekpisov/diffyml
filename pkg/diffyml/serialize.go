// serialize.go - YAML serialization utilities for structured types.
//
// Converts OrderedMap, map[string]any, and []any to yaml.Node trees
// for serialization. Used by formatValue (formatter.go), serializeDocument
// (rename.go), and SerializeValue (summarizer.go).
package diffyml

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// marshalStructuredYAML marshals structured types (*OrderedMap, map[string]any,
// []any) to a YAML string. Returns the YAML string and true on success,
// or ("", false) if val is not a structured type or marshaling fails.
func marshalStructuredYAML(val any) (string, bool) {
	switch val.(type) {
	case *OrderedMap, map[string]any, []any:
		node := valueToYAMLNode(val)
		out, err := yaml.Marshal(node)
		if err == nil {
			return string(out), true
		}
	}
	return "", false
}

// orderedMapToGeneric converts an OrderedMap to a yaml.v3-serializable
// structure that preserves key order.
func orderedMapToGeneric(om *OrderedMap) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	node.Content = make([]*yaml.Node, 0)
	for _, key := range om.Keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
		valNode := valueToYAMLNode(om.Values[key])
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node
}

// valueToYAMLNode converts a Go value to a yaml.Node for serialization.
// Recursively handles *OrderedMap, []any, and map[string]any
// so that nested structured values are serialized as proper YAML.
func valueToYAMLNode(val any) *yaml.Node {
	switch v := val.(type) {
	case *OrderedMap:
		return orderedMapToGeneric(v)
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		node.Content = make([]*yaml.Node, 0, len(v))
		for _, item := range v {
			node.Content = append(node.Content, valueToYAMLNode(item))
		}
		return node
	case map[string]any:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		node.Content = make([]*yaml.Node, 0)
		for _, k := range sortedMapKeys(v) {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k, Tag: "!!str"}
			valNode := valueToYAMLNode(v[k])
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node
	default:
		n := &yaml.Node{}
		_ = n.Encode(val)
		return n
	}
}

// sortedMapKeys returns the keys of a map[string]any in sorted order.
func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SerializeValue serializes a Difference.From or Difference.To value into a
// human-readable string for prompt inclusion.
func SerializeValue(val any) string {
	if val == nil {
		return "<none>"
	}

	if s, ok := marshalStructuredYAML(val); ok {
		return strings.TrimRight(s, "\n")
	}
	return fmt.Sprintf("%v", val)
}
