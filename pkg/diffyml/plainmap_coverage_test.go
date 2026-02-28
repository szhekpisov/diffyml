package diffyml

import (
	"strings"
	"testing"
)

// Tests for code paths that handle map[string]interface{} values.
// The parser always produces *OrderedMap, but these branches exist as
// defensive handling for direct callers. We test them to kill gremlins mutants.

// --- compareNodes with map[string]interface{} ---

func TestCompareNodes_PlainMaps_Equal(t *testing.T) {
	from := map[string]interface{}{"a": "1", "b": "2"}
	to := map[string]interface{}{"a": "1", "b": "2"}

	diffs := compareNodes("root", from, to, nil)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs for equal plain maps, got %d: %v", len(diffs), diffs)
	}
}

func TestCompareNodes_PlainMaps_Modified(t *testing.T) {
	from := map[string]interface{}{"a": "1", "b": "2"}
	to := map[string]interface{}{"a": "1", "b": "changed"}

	diffs := compareNodes("root", from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Type != DiffModified {
		t.Errorf("expected DiffModified, got %v", diffs[0].Type)
	}
	if diffs[0].Path != "root.b" {
		t.Errorf("expected path root.b, got %s", diffs[0].Path)
	}
}

func TestCompareNodes_PlainMaps_Added(t *testing.T) {
	from := map[string]interface{}{"a": "1"}
	to := map[string]interface{}{"a": "1", "b": "2"}

	diffs := compareNodes("root", from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Type != DiffAdded {
		t.Errorf("expected DiffAdded, got %v", diffs[0].Type)
	}
}

func TestCompareNodes_PlainMaps_Removed(t *testing.T) {
	from := map[string]interface{}{"a": "1", "b": "2"}
	to := map[string]interface{}{"a": "1"}

	diffs := compareNodes("root", from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Type != DiffRemoved {
		t.Errorf("expected DiffRemoved, got %v", diffs[0].Type)
	}
}

func TestCompareNodes_PlainMaps_Nested(t *testing.T) {
	from := map[string]interface{}{
		"parent": map[string]interface{}{"child": "old"},
	}
	to := map[string]interface{}{
		"parent": map[string]interface{}{"child": "new"},
	}

	diffs := compareNodes("", from, to, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != "parent.child" {
		t.Errorf("expected path parent.child, got %s", diffs[0].Path)
	}
}

// --- deepEqual with map[string]interface{} ---

func TestDeepEqual_PlainMaps_Equal(t *testing.T) {
	a := map[string]interface{}{"x": "1", "y": "2"}
	b := map[string]interface{}{"x": "1", "y": "2"}

	if !deepEqual(a, b, nil) {
		t.Error("expected equal plain maps to be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentValues(t *testing.T) {
	a := map[string]interface{}{"x": "1"}
	b := map[string]interface{}{"x": "2"}

	if deepEqual(a, b, nil) {
		t.Error("expected different plain maps to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentKeys(t *testing.T) {
	a := map[string]interface{}{"x": "1"}
	b := map[string]interface{}{"y": "1"}

	if deepEqual(a, b, nil) {
		t.Error("expected maps with different keys to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentLengths(t *testing.T) {
	a := map[string]interface{}{"x": "1"}
	b := map[string]interface{}{"x": "1", "y": "2"}

	if deepEqual(a, b, nil) {
		t.Error("expected maps with different lengths to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_Nested(t *testing.T) {
	a := map[string]interface{}{"m": map[string]interface{}{"k": "v"}}
	b := map[string]interface{}{"m": map[string]interface{}{"k": "v"}}

	if !deepEqual(a, b, nil) {
		t.Error("expected nested equal plain maps to be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_NestedDifferent(t *testing.T) {
	a := map[string]interface{}{"m": map[string]interface{}{"k": "v1"}}
	b := map[string]interface{}{"m": map[string]interface{}{"k": "v2"}}

	if deepEqual(a, b, nil) {
		t.Error("expected nested different plain maps to not be deepEqual")
	}
}

// --- compareListsPositional: added/removed items ---

func TestCompareListsPositional_ItemsAdded(t *testing.T) {
	from := []interface{}{"a"}
	to := []interface{}{"a", "b", "c"}

	diffs := compareListsPositional("list", from, to, nil)

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
	from := []interface{}{"a", "b", "c"}
	to := []interface{}{"a"}

	diffs := compareListsPositional("list", from, to, nil)

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
	from := []interface{}{"a", "b"}
	to := []interface{}{"x", "y", "z"}

	diffs := compareListsPositional("list", from, to, nil)

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

// --- detailed_formatter with map[string]interface{} ---

func TestDetailedFormatter_PlainMapValue(t *testing.T) {
	diffs := []Difference{
		{
			Path: "items.0",
			Type: DiffAdded,
			From: nil,
			To:   map[string]interface{}{"name": "test", "value": "123"},
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
	// Exercises renderFirstKeyValueYAML with map[string]interface{} val (line 291-295)
	diffs := []Difference{
		{
			Path: "items.0",
			Type: DiffAdded,
			From: nil,
			To: map[string]interface{}{
				"spec": map[string]interface{}{"replicas": 3, "image": "nginx"},
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
			Path: "items.0",
			Type: DiffRemoved,
			From: map[string]interface{}{"key": "val"},
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
			Path: "items.0",
			Type: DiffAdded,
			From: nil,
			To: map[string]interface{}{
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

// --- buildFilePairsFromMap: one-sided file pairs ---

func TestBuildFilePairsFromMap_OnlyFrom(t *testing.T) {
	m := map[string][2][]byte{
		"deleted.yaml": {[]byte("content"), nil},
	}
	pairs := buildFilePairsFromMap(m)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Type != FilePairOnlyFrom {
		t.Errorf("expected FilePairOnlyFrom, got %v", pairs[0].Type)
	}
}

func TestBuildFilePairsFromMap_OnlyTo(t *testing.T) {
	m := map[string][2][]byte{
		"added.yaml": {nil, []byte("content")},
	}
	pairs := buildFilePairsFromMap(m)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Type != FilePairOnlyTo {
		t.Errorf("expected FilePairOnlyTo, got %v", pairs[0].Type)
	}
}

func TestBuildFilePairsFromMap_Mixed(t *testing.T) {
	m := map[string][2][]byte{
		"both.yaml":    {[]byte("a"), []byte("b")},
		"added.yaml":   {nil, []byte("new")},
		"deleted.yaml": {[]byte("old"), nil},
	}
	pairs := buildFilePairsFromMap(m)
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	// Sorted alphabetically: added, both, deleted
	types := map[string]FilePairType{}
	for _, p := range pairs {
		types[p.Name] = p.Type
	}

	if types["added.yaml"] != FilePairOnlyTo {
		t.Errorf("added.yaml: expected FilePairOnlyTo, got %v", types["added.yaml"])
	}
	if types["both.yaml"] != FilePairBothExist {
		t.Errorf("both.yaml: expected FilePairBothExist, got %v", types["both.yaml"])
	}
	if types["deleted.yaml"] != FilePairOnlyFrom {
		t.Errorf("deleted.yaml: expected FilePairOnlyFrom, got %v", types["deleted.yaml"])
	}
}
