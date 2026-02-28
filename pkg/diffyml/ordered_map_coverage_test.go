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

	t.Run("alias node cycle detection", func(t *testing.T) {
		// Self-referencing alias: should return nil instead of infinite recursion.
		alias := &yaml.Node{Kind: yaml.AliasNode}
		alias.Alias = alias
		got := nodeToInterface(alias)
		if got != nil {
			t.Errorf("expected nil for cyclic alias, got %v", got)
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

func TestNodeToInterface_MappingOddContent(t *testing.T) {
	// ordered_map.go:75 — `i+1 < len(node.Content)` → `<= len(node.Content)`
	// If mutated, accessing node.Content[i+1] when i+1 == len would panic.
	// We create a MappingNode with an odd number of Content entries.
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key1"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "val1"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "orphanKey"},
			// Missing value node — odd content count
		},
	}

	// Should not panic
	result := nodeToInterface(node)
	om, ok := result.(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap, got %T", result)
	}

	// Should have only 1 key ("key1") since "orphanKey" has no pair
	if len(om.Keys) != 1 {
		t.Errorf("expected 1 key for odd content, got %d: %v", len(om.Keys), om.Keys)
	}
	if om.Keys[0] != "key1" {
		t.Errorf("expected key 'key1', got %q", om.Keys[0])
	}
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
