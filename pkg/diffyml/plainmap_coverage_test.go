package diffyml

import (
	"strings"
	"testing"
)

// Tests for code paths that handle map[string]any values.
//
// After the node-pipeline refactor, the live comparator never sees plain maps
// (yaml.Decode → *yaml.Node → nodeToInterface → *OrderedMap). The remaining
// reachable plain-map paths are:
//   - deepEqual (callable with any value type — library-facing utility)
//   - the formatters' rendering of Difference.From/To (still typed any)

// --- deepEqual with map[string]any ---

func TestDeepEqual_PlainMaps_Equal(t *testing.T) {
	a := map[string]any{"x": "1", "y": "2"}
	b := map[string]any{"x": "1", "y": "2"}

	if !deepEqual(a, b, &Options{}) {
		t.Error("expected equal plain maps to be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentValues(t *testing.T) {
	a := map[string]any{"x": "1"}
	b := map[string]any{"x": "2"}

	if deepEqual(a, b, &Options{}) {
		t.Error("expected different plain maps to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentKeys(t *testing.T) {
	a := map[string]any{"x": "1"}
	b := map[string]any{"y": "1"}

	if deepEqual(a, b, &Options{}) {
		t.Error("expected maps with different keys to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_DifferentLengths(t *testing.T) {
	a := map[string]any{"x": "1"}
	b := map[string]any{"x": "1", "y": "2"}

	if deepEqual(a, b, &Options{}) {
		t.Error("expected maps with different lengths to not be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_Nested(t *testing.T) {
	a := map[string]any{"m": map[string]any{"k": "v"}}
	b := map[string]any{"m": map[string]any{"k": "v"}}

	if !deepEqual(a, b, &Options{}) {
		t.Error("expected nested equal plain maps to be deepEqual")
	}
}

func TestDeepEqual_PlainMaps_NestedDifferent(t *testing.T) {
	a := map[string]any{"m": map[string]any{"k": "v1"}}
	b := map[string]any{"m": map[string]any{"k": "v2"}}

	if deepEqual(a, b, &Options{}) {
		t.Error("expected nested different plain maps to not be deepEqual")
	}
}

// --- detailed_formatter with map[string]any From/To ---

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
	// Exercises renderFirstKeyValueYAML with map[string]any val.
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
