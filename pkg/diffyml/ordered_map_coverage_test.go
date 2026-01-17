package diffyml

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNodeToInterface_EdgeCases(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		got := nodeToInterface(nil)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("empty document node", func(t *testing.T) {
		node := &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: nil,
		}
		got := nodeToInterface(node)
		if got != nil {
			t.Errorf("expected nil for empty document node, got %v", got)
		}
	})

	t.Run("alias node", func(t *testing.T) {
		target := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "aliased-value",
		}
		alias := &yaml.Node{
			Kind:  yaml.AliasNode,
			Alias: target,
		}
		got := nodeToInterface(alias)
		if got != "aliased-value" {
			t.Errorf("expected 'aliased-value', got %v", got)
		}
	})

	t.Run("unknown kind", func(t *testing.T) {
		node := &yaml.Node{
			Kind: 0, // invalid/unknown kind
		}
		got := nodeToInterface(node)
		if got != nil {
			t.Errorf("expected nil for unknown kind, got %v", got)
		}
	})
}

func TestResolveScalar_DecodeError(t *testing.T) {
	// Construct a scalar node with a tag that will cause Decode to fail
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!binary",
		Value: "not-valid-base64!!!",
	}
	got := resolveScalar(node)
	// When Decode fails, resolveScalar falls back to node.Value
	if got != "not-valid-base64!!!" {
		t.Errorf("expected fallback to node.Value %q, got %v", node.Value, got)
	}
}
