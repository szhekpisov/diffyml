package diffyml

import (
	"testing"
)

func TestMaskDifferences_NoOptions_ReturnsUnchanged(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"data", "password"}, Type: DiffModified, From: "old", To: "new", DocumentKind: "Secret"},
	}
	got, err := MaskDifferences(diffs, MaskOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].From != "old" || got[0].To != "new" {
		t.Errorf("expected values unchanged, got from=%v to=%v", got[0].From, got[0].To)
	}
}

func TestMaskDifferences_AutoMaskSecretData(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"data", "password"}, Type: DiffModified, From: "aGVsbG8=", To: "d29ybGQ=", DocumentKind: "Secret"},
		{Path: DiffPath{"metadata", "name"}, Type: DiffModified, From: "old", To: "new", DocumentKind: "Secret"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true})
	if got[0].From != "***" || got[0].To != "***" {
		t.Errorf("data.password should be masked, got from=%v to=%v", got[0].From, got[0].To)
	}
	if got[1].From != "old" || got[1].To != "new" {
		t.Errorf("metadata.name should NOT be masked, got from=%v to=%v", got[1].From, got[1].To)
	}
}

func TestMaskDifferences_AutoMaskStringData(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"stringData", "token"}, Type: DiffModified, From: "abc", To: "xyz", DocumentKind: "Secret"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true})
	if got[0].From != "***" || got[0].To != "***" {
		t.Errorf("stringData should be masked, got from=%v to=%v", got[0].From, got[0].To)
	}
}

func TestMaskDifferences_DoesNotMaskNonSecretKinds(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"data", "config"}, Type: DiffModified, From: "old", To: "new", DocumentKind: "ConfigMap"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true})
	if got[0].From != "old" {
		t.Errorf("ConfigMap data should NOT be auto-masked, got %v", got[0].From)
	}
}

func TestMaskDifferences_HandlesDocIndexPrefix(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"[1]", "data", "key"}, Type: DiffModified, From: "a", To: "b", DocumentKind: "Secret"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true})
	if got[0].From != "***" {
		t.Errorf("[1].data.key should be masked, got %v", got[0].From)
	}
}

func TestMaskDifferences_CustomMaskPath(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"data", "api_key"}, Type: DiffModified, From: "old", To: "new", DocumentKind: "ConfigMap"},
		{Path: DiffPath{"data", "log_level"}, Type: DiffModified, From: "info", To: "debug", DocumentKind: "ConfigMap"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskPaths: []string{"data.api_key"}})
	if got[0].From != "***" {
		t.Errorf("data.api_key should be masked, got %v", got[0].From)
	}
	if got[1].From != "info" {
		t.Errorf("data.log_level should NOT be masked, got %v", got[1].From)
	}
}

func TestMaskDifferences_CustomMaskPath_PrefixMatch(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"secrets", "db", "password"}, Type: DiffModified, From: "a", To: "b"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskPaths: []string{"secrets"}})
	if got[0].From != "***" {
		t.Errorf("secrets.db.password should be masked via prefix, got %v", got[0].From)
	}
}

func TestMaskDifferences_RegexPath(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "DB_PASSWORD"}, Type: DiffModified, From: "a", To: "b"},
		{Path: DiffPath{"config", "log_level"}, Type: DiffModified, From: "info", To: "debug"},
	}
	got, err := MaskDifferences(diffs, MaskOptions{MaskPathRegexp: []string{`(?i)password`}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].From != "***" {
		t.Errorf("DB_PASSWORD should be masked by regex, got %v", got[0].From)
	}
	if got[1].From != "info" {
		t.Errorf("log_level should not match password regex, got %v", got[1].From)
	}
}

func TestMaskDifferences_InvalidRegex_ReturnsError(t *testing.T) {
	_, err := MaskDifferences(nil, MaskOptions{MaskPathRegexp: []string{"["}})
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestMaskDifferences_CustomPlaceholder(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"data", "x"}, Type: DiffModified, From: "a", To: "b", DocumentKind: "Secret"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true, Placeholder: "<redacted>"})
	if got[0].From != "<redacted>" {
		t.Errorf("expected custom placeholder, got %v", got[0].From)
	}
}

func TestMaskDifferences_OrderChangedDiffs_NotMasked(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"(document)"}, Type: DiffOrderChanged, From: []any{"a", "b"}, To: []any{"b", "a"}, DocumentKind: "Secret"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true})
	fromList, ok := got[0].From.([]any)
	if !ok || len(fromList) != 2 || fromList[0] != "a" {
		t.Errorf("order-change identifiers should not be masked, got %v", got[0].From)
	}
}

func TestMaskDifferences_PreservesDiffCount(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"data", "a"}, Type: DiffModified, From: "1", To: "2", DocumentKind: "Secret"},
		{Path: DiffPath{"data", "b"}, Type: DiffModified, From: "3", To: "4", DocumentKind: "Secret"},
		{Path: DiffPath{"metadata", "labels", "app"}, Type: DiffAdded, From: nil, To: "x", DocumentKind: "Secret"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true})
	if len(got) != 3 {
		t.Errorf("masking must not change diff count, got %d", len(got))
	}
}

func TestMaskDifferences_WholeSecretAdded_MasksDataSubtree(t *testing.T) {
	secretDoc := &OrderedMap{
		Keys: []string{"apiVersion", "kind", "metadata", "data"},
		Values: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata":   &OrderedMap{Keys: []string{"name"}, Values: map[string]any{"name": "foo"}},
			"data":       &OrderedMap{Keys: []string{"password"}, Values: map[string]any{"password": "aGVsbG8="}},
		},
	}
	diffs := []Difference{
		{Path: DiffPath{"[0]"}, Type: DiffAdded, From: nil, To: secretDoc, DocumentKind: "Secret"},
	}
	got, _ := MaskDifferences(diffs, MaskOptions{MaskSecrets: true})
	root, ok := got[0].To.(*OrderedMap)
	if !ok {
		t.Fatalf("expected OrderedMap, got %T", got[0].To)
	}
	if root.Values["apiVersion"] != "v1" {
		t.Errorf("apiVersion should be preserved, got %v", root.Values["apiVersion"])
	}
	if root.Values["kind"] != "Secret" {
		t.Errorf("kind should be preserved, got %v", root.Values["kind"])
	}
	dataMap, ok := root.Values["data"].(*OrderedMap)
	if !ok {
		t.Fatalf("expected data to be OrderedMap, got %T", root.Values["data"])
	}
	if dataMap.Values["password"] != "***" {
		t.Errorf("data.password should be masked, got %v", dataMap.Values["password"])
	}
}

func TestMaskValueRecursive_Scalar(t *testing.T) {
	if got := maskValueRecursive("hello", "X"); got != "X" {
		t.Errorf("expected 'X', got %v", got)
	}
	if got := maskValueRecursive(42, "X"); got != "X" {
		t.Errorf("expected 'X' for int, got %v", got)
	}
	if got := maskValueRecursive(nil, "X"); got != nil {
		t.Errorf("nil should pass through, got %v", got)
	}
}

func TestMaskValueRecursive_OrderedMap_PreservesKeys(t *testing.T) {
	in := &OrderedMap{
		Keys:   []string{"a", "b"},
		Values: map[string]any{"a": "1", "b": "2"},
	}
	out, ok := maskValueRecursive(in, "X").(*OrderedMap)
	if !ok {
		t.Fatalf("expected OrderedMap")
	}
	if len(out.Keys) != 2 || out.Keys[0] != "a" || out.Keys[1] != "b" {
		t.Errorf("key order not preserved: %v", out.Keys)
	}
	if out.Values["a"] != "X" || out.Values["b"] != "X" {
		t.Errorf("values not masked: %v", out.Values)
	}
}

func TestMaskValueRecursive_NestedStructure(t *testing.T) {
	in := &OrderedMap{
		Keys: []string{"list", "map"},
		Values: map[string]any{
			"list": []any{"a", "b", &OrderedMap{Keys: []string{"x"}, Values: map[string]any{"x": "y"}}},
			"map":  map[string]any{"k": "v"},
		},
	}
	out := maskValueRecursive(in, "X").(*OrderedMap)
	list := out.Values["list"].([]any)
	if list[0] != "X" || list[1] != "X" {
		t.Errorf("list scalars not masked: %v", list)
	}
	nested := list[2].(*OrderedMap)
	if nested.Values["x"] != "X" {
		t.Errorf("nested OrderedMap not masked: %v", nested.Values)
	}
	plainMap := out.Values["map"].(map[string]any)
	if plainMap["k"] != "X" {
		t.Errorf("plain map not masked: %v", plainMap)
	}
}

func TestMaskSecretSubtrees_PlainMap(t *testing.T) {
	in := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"data":       map[string]any{"password": "aGVsbG8="},
	}
	out, ok := maskSecretSubtrees(in, "***").(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", out)
	}
	if out["apiVersion"] != "v1" || out["kind"] != "Secret" {
		t.Errorf("non-data fields should be preserved, got %v", out)
	}
	dataMap, ok := out["data"].(map[string]any)
	if !ok || dataMap["password"] != "***" {
		t.Errorf("data.password should be masked, got %v", out["data"])
	}
}

func TestMaskSecretSubtrees_NonMapValue_Unchanged(t *testing.T) {
	if got := maskSecretSubtrees("scalar", "***"); got != "scalar" {
		t.Errorf("non-map value should pass through, got %v", got)
	}
	if got := maskSecretSubtrees([]any{"a", "b"}, "***"); got == nil {
		t.Errorf("non-map value should pass through, got nil")
	}
}

func TestMaskSecretSubtrees_Nil(t *testing.T) {
	if got := maskSecretSubtrees(nil, "***"); got != nil {
		t.Errorf("nil should pass through, got %v", got)
	}
}

func TestMaskValueRecursive_DoesNotMutateInput(t *testing.T) {
	in := &OrderedMap{Keys: []string{"a"}, Values: map[string]any{"a": "original"}}
	_ = maskValueRecursive(in, "X")
	if in.Values["a"] != "original" {
		t.Errorf("input was mutated: %v", in.Values["a"])
	}
}

func TestPathWithoutDocIndex(t *testing.T) {
	cases := []struct {
		in   DiffPath
		want string
	}{
		{DiffPath{"data", "x"}, "data.x"},
		{DiffPath{"[0]", "data", "x"}, "data.x"},
		{DiffPath{"[0]"}, ""},
		{DiffPath{}, ""},
	}
	for _, c := range cases {
		if got := pathWithoutDocIndex(c.in); got != c.want {
			t.Errorf("pathWithoutDocIndex(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFirstFieldAfterDocIndex(t *testing.T) {
	cases := []struct {
		in     DiffPath
		want   string
		wantOk bool
	}{
		{DiffPath{"data", "x"}, "data", true},
		{DiffPath{"[0]", "data", "x"}, "data", true},
		{DiffPath{"[0]"}, "", false},
		{DiffPath{}, "", false},
	}
	for _, c := range cases {
		got, ok := firstFieldAfterDocIndex(c.in)
		if got != c.want || ok != c.wantOk {
			t.Errorf("firstFieldAfterDocIndex(%v) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.wantOk)
		}
	}
}

func TestIsWholeDocDiff(t *testing.T) {
	cases := []struct {
		in   DiffPath
		want bool
	}{
		{DiffPath{"[0]"}, true},
		{DiffPath{"[0]", "data"}, false},
		{DiffPath{"data"}, false},
		{DiffPath{}, false},
	}
	for _, c := range cases {
		if got := isWholeDocDiff(c.in); got != c.want {
			t.Errorf("isWholeDocDiff(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
