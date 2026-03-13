package diffyml

import (
	"math"
	"testing"
	"time"

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

func TestResolveScalar_AllTags(t *testing.T) {
	tests := []struct {
		name  string
		tag   string
		value string
		check func(t *testing.T, got any)
	}{
		{"str", "!!str", "hello", func(t *testing.T, got any) {
			if got != "hello" {
				t.Errorf("expected 'hello', got %v", got)
			}
		}},
		{"int", "!!int", "42", func(t *testing.T, got any) {
			if got != 42 {
				t.Errorf("expected 42, got %v (%T)", got, got)
			}
		}},
		{"int hex", "!!int", "0xFF", func(t *testing.T, got any) {
			if got != 255 {
				t.Errorf("expected 255, got %v", got)
			}
		}},
		{"int invalid", "!!int", "not-a-number", func(t *testing.T, got any) {
			if got != "not-a-number" {
				t.Errorf("expected fallback string, got %v", got)
			}
		}},
		{"int uint64", "!!int", "18446744073709551615", func(t *testing.T, got any) {
			if _, ok := got.(uint64); !ok {
				t.Errorf("expected uint64, got %T: %v", got, got)
			}
		}},
		{"float", "!!float", "3.14", func(t *testing.T, got any) {
			if got != 3.14 {
				t.Errorf("expected 3.14, got %v", got)
			}
		}},
		{"float inf", "!!float", ".inf", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsInf(f, 1) {
				t.Errorf("expected +Inf, got %v", got)
			}
		}},
		{"float +inf", "!!float", "+.inf", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsInf(f, 1) {
				t.Errorf("expected +Inf, got %v", got)
			}
		}},
		{"float -inf", "!!float", "-.inf", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsInf(f, -1) {
				t.Errorf("expected -Inf, got %v", got)
			}
		}},
		{"float nan", "!!float", ".nan", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsNaN(f) {
				t.Errorf("expected NaN, got %v", got)
			}
		}},
		{"float invalid", "!!float", "not-a-float", func(t *testing.T, got any) {
			if got != "not-a-float" {
				t.Errorf("expected fallback string, got %v", got)
			}
		}},
		{"bool true", "!!bool", "true", func(t *testing.T, got any) {
			if got != true {
				t.Errorf("expected true, got %v", got)
			}
		}},
		{"bool yes", "!!bool", "yes", func(t *testing.T, got any) {
			if got != true {
				t.Errorf("expected true, got %v", got)
			}
		}},
		{"bool on", "!!bool", "on", func(t *testing.T, got any) {
			if got != true {
				t.Errorf("expected true, got %v", got)
			}
		}},
		{"bool false", "!!bool", "false", func(t *testing.T, got any) {
			if got != false {
				t.Errorf("expected false, got %v", got)
			}
		}},
		{"bool no", "!!bool", "no", func(t *testing.T, got any) {
			if got != false {
				t.Errorf("expected false, got %v", got)
			}
		}},
		{"bool off", "!!bool", "off", func(t *testing.T, got any) {
			if got != false {
				t.Errorf("expected false, got %v", got)
			}
		}},
		{"bool invalid", "!!bool", "maybe", func(t *testing.T, got any) {
			if got != "maybe" {
				t.Errorf("expected fallback string, got %v", got)
			}
		}},
		{"null", "!!null", "null", func(t *testing.T, got any) {
			if got != nil {
				t.Errorf("expected nil, got %v", got)
			}
		}},
		{"timestamp", "!!timestamp", "2020-01-01", func(t *testing.T, got any) {
			if _, ok := got.(time.Time); !ok {
				t.Errorf("expected time.Time, got %T: %v", got, got)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &yaml.Node{Kind: yaml.ScalarNode, Tag: tt.tag, Value: tt.value}
			got := resolveScalar(node)
			tt.check(t, got)
		})
	}
}

func TestResolveScalar_UnknownTag(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!custom", Value: "something"}
	got := resolveScalar(node)
	if got != "something" {
		t.Errorf("expected 'something', got %v", got)
	}
}

func TestResolveScalar_UntaggedOrBang(t *testing.T) {
	// tag == "" routes to resolveUntaggedScalar
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "", Value: "42"}
	got := resolveScalar(node)
	if got != 42 {
		t.Errorf("expected 42, got %v (%T)", got, got)
	}

	// tag == "!" routes to resolveUntaggedScalar
	node2 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!", Value: "hello"}
	got2 := resolveScalar(node2)
	if got2 != "hello" {
		t.Errorf("expected 'hello', got %v", got2)
	}
}

func TestResolveUntaggedScalar_AllBranches(t *testing.T) {
	tests := []struct {
		name  string
		value string
		check func(t *testing.T, got any)
	}{
		{"empty is nil", "", func(t *testing.T, got any) {
			if got != nil {
				t.Errorf("expected nil, got %v", got)
			}
		}},
		{"tilde is nil", "~", func(t *testing.T, got any) {
			if got != nil {
				t.Errorf("expected nil, got %v", got)
			}
		}},
		{"null is nil", "null", func(t *testing.T, got any) {
			if got != nil {
				t.Errorf("expected nil, got %v", got)
			}
		}},
		{"NULL is nil", "NULL", func(t *testing.T, got any) {
			if got != nil {
				t.Errorf("expected nil, got %v", got)
			}
		}},
		{"true", "true", func(t *testing.T, got any) {
			if got != true {
				t.Errorf("expected true, got %v", got)
			}
		}},
		{"True", "True", func(t *testing.T, got any) {
			if got != true {
				t.Errorf("expected true, got %v", got)
			}
		}},
		{"yes", "yes", func(t *testing.T, got any) {
			if got != true {
				t.Errorf("expected true, got %v", got)
			}
		}},
		{"on", "on", func(t *testing.T, got any) {
			if got != true {
				t.Errorf("expected true, got %v", got)
			}
		}},
		{"false", "false", func(t *testing.T, got any) {
			if got != false {
				t.Errorf("expected false, got %v", got)
			}
		}},
		{"no", "no", func(t *testing.T, got any) {
			if got != false {
				t.Errorf("expected false, got %v", got)
			}
		}},
		{"off", "off", func(t *testing.T, got any) {
			if got != false {
				t.Errorf("expected false, got %v", got)
			}
		}},
		{"integer", "42", func(t *testing.T, got any) {
			if got != 42 {
				t.Errorf("expected 42, got %v (%T)", got, got)
			}
		}},
		{"negative int", "-7", func(t *testing.T, got any) {
			if got != -7 {
				t.Errorf("expected -7, got %v", got)
			}
		}},
		{"hex int", "0x1A", func(t *testing.T, got any) {
			if got != 26 {
				t.Errorf("expected 26, got %v", got)
			}
		}},
		{"octal 0o", "0o777", func(t *testing.T, got any) {
			if got != 511 {
				t.Errorf("expected 511, got %v", got)
			}
		}},
		{"octal 0O", "0O10", func(t *testing.T, got any) {
			if got != 8 {
				t.Errorf("expected 8, got %v", got)
			}
		}},
		{"invalid octal", "0oZZZ", func(t *testing.T, got any) {
			if got != "0oZZZ" {
				t.Errorf("expected string fallback, got %v", got)
			}
		}},
		{"float", "3.14", func(t *testing.T, got any) {
			if got != 3.14 {
				t.Errorf("expected 3.14, got %v", got)
			}
		}},
		{"float scientific", "1.5e2", func(t *testing.T, got any) {
			if got != 150.0 {
				t.Errorf("expected 150.0, got %v", got)
			}
		}},
		{".inf", ".inf", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsInf(f, 1) {
				t.Errorf("expected +Inf, got %v", got)
			}
		}},
		{"+.inf", "+.inf", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsInf(f, 1) {
				t.Errorf("expected +Inf, got %v", got)
			}
		}},
		{"-.inf", "-.inf", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsInf(f, -1) {
				t.Errorf("expected -Inf, got %v", got)
			}
		}},
		{".nan", ".nan", func(t *testing.T, got any) {
			f, ok := got.(float64)
			if !ok || !math.IsNaN(f) {
				t.Errorf("expected NaN, got %v", got)
			}
		}},
		{"timestamp", "2020-01-15", func(t *testing.T, got any) {
			if _, ok := got.(time.Time); !ok {
				t.Errorf("expected time.Time, got %T: %v", got, got)
			}
		}},
		{"plain string", "hello world", func(t *testing.T, got any) {
			if got != "hello world" {
				t.Errorf("expected 'hello world', got %v", got)
			}
		}},
		{"string that starts with 0 but not octal", "0abc", func(t *testing.T, got any) {
			if got != "0abc" {
				t.Errorf("expected '0abc', got %v", got)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "", Value: tt.value}
			got := resolveUntaggedScalar(node, tt.value)
			tt.check(t, got)
		})
	}
}

func TestLooksLikeTimestamp(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"2020-01-01", true},
		{"2020-01-01T00:00:00Z", true},
		{"1999-12-31", true},
		{"short", false},
		{"abcd-ef-gh", false},
		{"12345", false},
		{"hello world", false},
		{"2020x01-01", false},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := looksLikeTimestamp(tt.value); got != tt.want {
				t.Errorf("looksLikeTimestamp(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
