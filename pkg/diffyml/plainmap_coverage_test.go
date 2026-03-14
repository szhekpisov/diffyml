package diffyml

import (
	"strings"
	"testing"
)

// Tests for code paths that handle map[string]any values.
// The parser always produces *OrderedMap, but these branches exist as
// defensive handling for direct callers. We test them to kill gremlins mutants.

// --- compareNodes with map[string]any ---

func TestCompareNodes_PlainMaps_Equal(t *testing.T) {
	from := map[string]any{"a": "1", "b": "2"}
	to := map[string]any{"a": "1", "b": "2"}

	diffs := compareNodes(DiffPath{"root"}, from, to, nil)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs for equal plain maps, got %d: %v", len(diffs), diffs)
	}
}

func TestCompareNodes_PlainMaps_Modified(t *testing.T) {
	from := map[string]any{"a": "1", "b": "2"}
	to := map[string]any{"a": "1", "b": "changed"}

	diffs := compareNodes(DiffPath{"root"}, from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Type != DiffModified {
		t.Errorf("expected DiffModified, got %v", diffs[0].Type)
	}
	if diffs[0].Path.String() != "root.b" {
		t.Errorf("expected path root.b, got %s", diffs[0].Path.String())
	}
}

func TestCompareNodes_PlainMaps_Added(t *testing.T) {
	from := map[string]any{"a": "1"}
	to := map[string]any{"a": "1", "b": "2"}

	diffs := compareNodes(DiffPath{"root"}, from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Type != DiffAdded {
		t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
	}
}

func TestCompareNodes_PlainMaps_Removed(t *testing.T) {
	from := map[string]any{"a": "1", "b": "2"}
	to := map[string]any{"a": "1"}

	diffs := compareNodes(DiffPath{"root"}, from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Type != DiffRemoved {
		t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
	}
}

func TestCompareNodes_PlainMaps_Nested(t *testing.T) {
	from := map[string]any{
		"parent": map[string]any{"child": "old"},
	}
	to := map[string]any{
		"parent": map[string]any{"child": "new"},
	}

	diffs := compareNodes(nil, from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path.String() != "parent.child" {
		t.Errorf("expected path parent.child, got %s", diffs[0].Path.String())
	}
}

// --- deepEqual with map[string]any ---

func TestDeepEqual_PlainMaps_Equal(t *testing.T) {
	a := map[string]any{"x": "1", "y": "2"}
	b := map[string]any{"x": "1", "y": "2"}

	if !deepEqual(a, b, nil) {
		t.Error("expected equal plain maps to be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentValues(t *testing.T) {
	a := map[string]any{"x": "1"}
	b := map[string]any{"x": "2"}

	if deepEqual(a, b, nil) {
		t.Error("expected different plain maps to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentKeys(t *testing.T) {
	a := map[string]any{"x": "1"}
	b := map[string]any{"y": "1"}

	if deepEqual(a, b, nil) {
		t.Error("expected maps with different keys to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentLengths(t *testing.T) {
	a := map[string]any{"x": "1"}
	b := map[string]any{"x": "1", "y": "2"}

	if deepEqual(a, b, nil) {
		t.Error("expected maps with different lengths to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_Nested(t *testing.T) {
	a := map[string]any{"m": map[string]any{"k": "v"}}
	b := map[string]any{"m": map[string]any{"k": "v"}}

	if !deepEqual(a, b, nil) {
		t.Error("expected nested equal plain maps to be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_NestedDifferent(t *testing.T) {
	a := map[string]any{"m": map[string]any{"k": "v1"}}
	b := map[string]any{"m": map[string]any{"k": "v2"}}

	if deepEqual(a, b, nil) {
		t.Error("expected nested different plain maps to not be deepEqual")
	}
}

// --- compareListsPositional: added/removed items ---

func TestCompareListsPositional_ItemsAdded(t *testing.T) {
	from := []any{"a"}
	to := []any{"a", "b", "c"}

	diffs := compareListsPositional(DiffPath{"list"}, from, to, nil)

	added := 0
	for _, d := range diffs {
		if d.Type == DiffAdded {
			added++
		}
	}
	if added != 2 {
		t.Errorf("expected 2 added diffs, got %d", added)
	}
}

func TestCompareListsPositional_ItemsRemoved(t *testing.T) {
	from := []any{"a", "b", "c"}
	to := []any{"a"}

	diffs := compareListsPositional(DiffPath{"list"}, from, to, nil)

	removed := 0
	for _, d := range diffs {
		if d.Type == DiffRemoved {
			removed++
		}
	}
	if removed != 2 {
		t.Errorf("expected 2 removed diffs, got %d", removed)
	}
}

func TestCompareListsPositional_BothAddedAndRemoved(t *testing.T) {
	from := []any{"a", "b"}
	to := []any{"x", "y", "z"}

	diffs := compareListsPositional(DiffPath{"list"}, from, to, nil)

	var modified, added int
	for _, d := range diffs {
		switch d.Type {
		case DiffModified:
			modified++
		case DiffAdded:
			added++
		}
	}
	if modified != 2 {
		t.Errorf("expected 2 modified, got %d", modified)
	}
	if added != 1 {
		t.Errorf("expected 1 added, got %d", added)
	}
}

// --- detailed_formatter with map[string]any ---

func TestDetailedFormatter_PlainMapValue(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"items", "0"},
			Type: DiffAdded,
			From: nil,
			To:   map[string]any{"name": "test", "value": "123"},
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	if !strings.Contains(result, "name") || !strings.Contains(result, "test") {
		t.Errorf("expected plain map to be rendered with key-value pairs, got:\n%s", result)
	}
}

func TestDetailedFormatter_PlainMapNested(t *testing.T) {
	// Exercises renderFirstKeyValueYAML with map[string]any val (line 291-295)
	diffs := []Difference{
		{
			Path: DiffPath{"items", "0"},
			Type: DiffAdded,
			From: nil,
			To: map[string]any{
				"spec": map[string]any{"replicas": 3, "image": "nginx"},
				"meta": "simple",
			},
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	if !strings.Contains(result, "spec") || !strings.Contains(result, "replicas") {
		t.Errorf("expected nested plain map to render nested keys, got:\n%s", result)
	}
}

func TestDetailedFormatter_PlainMapRemoved(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"items", "0"},
			Type: DiffRemoved,
			From: map[string]any{"key": "val"},
			To:   nil,
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	if !strings.Contains(result, "key") {
		t.Errorf("expected plain map to be rendered, got:\n%s", result)
	}
}

func TestDetailedFormatter_PlainMapWithMultilineValue(t *testing.T) {
	// Exercises renderFirstKeyValueYAML default case with multiline string (line 302-303)
	diffs := []Difference{
		{
			Path: DiffPath{"items", "0"},
			Type: DiffAdded,
			From: nil,
			To: map[string]any{
				"config": "line1\nline2\nline3",
			},
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	if !strings.Contains(result, "config") {
		t.Errorf("expected multiline value in plain map to render, got:\n%s", result)
	}
}
