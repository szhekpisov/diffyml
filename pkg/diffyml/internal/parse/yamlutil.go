package parse

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
	"gopkg.in/yaml.v3"
)

// MarshalStructuredYAML marshals structured types to a YAML string.
func MarshalStructuredYAML(val interface{}) (string, bool) {
	if om := types.ToOrderedMap(val); om != nil {
		val = om
	}
	switch val.(type) {
	case *types.OrderedMap, []interface{}:
		node := ValueToYAMLNode(val)
		out, err := yaml.Marshal(node)
		if err == nil {
			return string(out), true
		}
	}
	return "", false
}

// OrderedMapToGeneric converts an OrderedMap to a yaml.Node.
func OrderedMapToGeneric(om *types.OrderedMap) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	node.Content = make([]*yaml.Node, 0)
	for _, key := range om.Keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
		valNode := ValueToYAMLNode(om.Values[key])
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node
}

// ValueToYAMLNode converts a Go value to a yaml.Node for serialization.
func ValueToYAMLNode(val interface{}) *yaml.Node {
	if om := types.ToOrderedMap(val); om != nil {
		val = om
	}
	switch v := val.(type) {
	case *types.OrderedMap:
		return OrderedMapToGeneric(v)
	case []interface{}:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		node.Content = make([]*yaml.Node, 0, len(v))
		for _, item := range v {
			node.Content = append(node.Content, ValueToYAMLNode(item))
		}
		return node
	default:
		n := &yaml.Node{}
		_ = n.Encode(val)
		return n
	}
}
