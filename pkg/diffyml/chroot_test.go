package diffyml

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// nodeFromYAML parses a YAML string into a DocumentNode (post merge-resolve)
// for chroot navigation tests. Tests that previously constructed synthetic
// map[string]any / []any structures use this helper plus nodeToInterface on
// the chrooted result to keep their assertions written against the any view.
func nodeFromYAML(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader([]byte(src))).Decode(&n); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	resolveMergeKeys(&n)
	return &n
}

func TestNavigateToPath_SimplePath(t *testing.T) {
	doc := nodeFromYAML(t, `
level1:
  level2:
    value: found
`)
	result, err := navigateToPath(doc, "level1.level2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := nodeToInterface(result).(*OrderedMap)
	if !ok {
		t.Fatalf("expected map, got %T", nodeToInterface(result))
	}
	if m.Values["value"] != "found" {
		t.Errorf("expected value=found, got value=%v", m.Values["value"])
	}
}

func TestNavigateToPath_SingleLevel(t *testing.T) {
	doc := nodeFromYAML(t, "data: hello\n")
	result, err := navigateToPath(doc, "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := nodeToInterface(result); got != "hello" {
		t.Errorf("expected hello, got %v", got)
	}
}

func TestNavigateToPath_EmptyPath(t *testing.T) {
	doc := nodeFromYAML(t, "key: value\n")
	result, err := navigateToPath(doc, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty path returns the original document (DocumentNode unwrap is
	// transparent — nodeToInterface materializes the root mapping).
	m, ok := nodeToInterface(result).(*OrderedMap)
	if !ok {
		t.Fatalf("expected map, got %T", nodeToInterface(result))
	}
	if m.Values["key"] != "value" {
		t.Errorf("expected key=value in result")
	}
}

func TestNavigateToPath_PathNotFound(t *testing.T) {
	doc := nodeFromYAML(t, "existing: value\n")
	_, err := navigateToPath(doc, "nonexistent.path")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestNavigateToPath_PathThroughScalar(t *testing.T) {
	doc := nodeFromYAML(t, "scalar: value\n")
	_, err := navigateToPath(doc, "scalar.deeper")
	if err == nil {
		t.Error("expected error when navigating through scalar")
	}
}

func TestNavigateToPath_ListIndex(t *testing.T) {
	doc := nodeFromYAML(t, `
items:
  - name: first
  - name: second
  - name: third
`)
	result, err := navigateToPath(doc, "items[1]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := nodeToInterface(result).(*OrderedMap)
	if !ok {
		t.Fatalf("expected map, got %T", nodeToInterface(result))
	}
	if m.Values["name"] != "second" {
		t.Errorf("expected name=second, got name=%v", m.Values["name"])
	}
}

func TestNavigateToPath_ListIndexOutOfBounds(t *testing.T) {
	doc := nodeFromYAML(t, "items:\n  - a\n  - b\n")
	_, err := navigateToPath(doc, "items[5]")
	if err == nil {
		t.Error("expected error for out of bounds index")
	}
}

func TestNavigateToPath_InvalidListIndex(t *testing.T) {
	doc := nodeFromYAML(t, "items:\n  - a\n  - b\n")
	_, err := navigateToPath(doc, "items[foo]")
	if err == nil {
		t.Fatal("expected error for invalid list index")
	}
}

func TestNavigateToPath_InvalidPathSyntax(t *testing.T) {
	doc := nodeFromYAML(t, "items:\n  - a\n  - b\n")
	_, err := navigateToPath(doc, "items[0")
	if err == nil {
		t.Fatal("expected error for invalid path syntax")
	}
}

func TestNavigateToPath_NestedListAccess(t *testing.T) {
	doc := nodeFromYAML(t, `
data:
  - nested:
      - x
      - y
      - z
`)
	result, err := navigateToPath(doc, "data[0].nested[2]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := nodeToInterface(result); got != "z" {
		t.Errorf("expected z, got %v", got)
	}
}

func TestApplyChroot_ToList(t *testing.T) {
	doc := nodeFromYAML(t, `
items:
  - name: one
  - name: two
`)
	// When listToDocuments is false, return the list as a single document slot.
	result, err := applyChroot(doc, "items", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 document, got %d", len(result))
	}

	list, ok := nodeToInterface(result[0]).([]any)
	if !ok {
		t.Fatalf("expected list, got %T", nodeToInterface(result[0]))
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items in list, got %d", len(list))
	}
}

func TestApplyChroot_ListToDocuments(t *testing.T) {
	doc := nodeFromYAML(t, `
items:
  - name: one
  - name: two
  - name: three
`)
	// When listToDocuments is true, each list item becomes a document.
	result, err := applyChroot(doc, "items", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 documents, got %d", len(result))
	}

	for i, expected := range []string{"one", "two", "three"} {
		m, ok := nodeToInterface(result[i]).(*OrderedMap)
		if !ok {
			t.Fatalf("document %d: expected map, got %T", i, nodeToInterface(result[i]))
		}
		if m.Values["name"] != expected {
			t.Errorf("document %d: expected name=%s, got name=%v", i, expected, m.Values["name"])
		}
	}
}

func TestApplyChroot_NonListWithListToDocuments(t *testing.T) {
	doc := nodeFromYAML(t, `
data:
  key: value
`)
	// Path points to non-list but listToDocuments is true: return as single doc.
	result, err := applyChroot(doc, "data", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 document, got %d", len(result))
	}
}

func TestApplyChroot_EmptyPath(t *testing.T) {
	doc := nodeFromYAML(t, "key: value\n")
	result, err := applyChroot(doc, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 document, got %d", len(result))
	}
}

func TestApplyChroot_PathNotFound(t *testing.T) {
	doc := nodeFromYAML(t, "existing: value\n")
	_, err := applyChroot(doc, "nonexistent", false)
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

// --- Mutation testing: chroot.go ---

func TestNavigateToPath_IndexAtExactLength(t *testing.T) {
	// `seg.index >= len(list)` → `> len(list)`: if mutated, accessing
	// index == len(list) would slice-panic instead of returning an error.
	doc := nodeFromYAML(t, "items:\n  - a\n  - b\n  - c\n")
	_, err := navigateToPath(doc, "items[3]")
	if err == nil {
		t.Error("expected error for index == len(list), but got nil")
	}
}

func TestSplitPath_ConsecutiveDots(t *testing.T) {
	parts, err := splitPath("a..b")
	if err != nil {
		t.Fatalf("splitPath(\"a..b\") failed: %v", err)
	}
	for i, part := range parts {
		if part == "" {
			t.Errorf("splitPath(\"a..b\") produced empty segment at index %d", i)
		}
	}
	if len(parts) != 2 {
		t.Errorf("splitPath(\"a..b\") produced %d parts, want 2: %v", len(parts), parts)
	}
}

func TestSplitPath_TrailingDot(t *testing.T) {
	parts, err := splitPath("a.b.")
	if err != nil {
		t.Fatalf("splitPath(\"a.b.\") failed: %v", err)
	}
	for i, part := range parts {
		if part == "" {
			t.Errorf("splitPath(\"a.b.\") produced empty segment at index %d", i)
		}
	}
	if len(parts) != 2 {
		t.Errorf("splitPath(\"a.b.\") produced %d parts, want 2: %v", len(parts), parts)
	}
}

func TestNavigateToPath_BareIndex(t *testing.T) {
	// Bare "[0]" path without a key prefix on a top-level sequence document.
	doc := nodeFromYAML(t, "- first\n- second\n- third\n")
	result, err := navigateToPath(doc, "[0]")
	if err != nil {
		t.Fatalf("navigateToPath(list, \"[0]\") failed: %v", err)
	}
	if got := nodeToInterface(result); got != "first" {
		t.Errorf("navigateToPath(list, \"[0]\") = %v, want \"first\"", got)
	}
}

func TestParsePath_LeadingDot(t *testing.T) {
	segments, err := parsePath(".items")
	if err != nil {
		t.Fatalf("parsePath(\".items\") failed: %v", err)
	}
	if len(segments) != 1 {
		t.Errorf("parsePath(\".items\") returned %d segments, want 1", len(segments))
	}
	if len(segments) > 0 && segments[0].key != "items" {
		t.Errorf("parsePath(\".items\")[0].key = %q, want \"items\"", segments[0].key)
	}
}

func TestSplitPath_SimpleKey(t *testing.T) {
	parts, err := splitPath("key")
	if err != nil {
		t.Fatalf("splitPath(\"key\") failed: %v", err)
	}
	if len(parts) != 1 {
		t.Errorf("splitPath(\"key\") returned %d parts, want 1", len(parts))
	}
	if len(parts) > 0 && parts[0] != "key" {
		t.Errorf("splitPath(\"key\")[0] = %q, want \"key\"", parts[0])
	}
}

func TestNavigateToPath_IndexOnNonList_ErrorMessage(t *testing.T) {
	doc := nodeFromYAML(t, "data:\n  key: value\n")
	_, err := navigateToPath(doc, "data[0]")
	if err == nil {
		t.Fatal("expected error for index access into non-list")
	}
	if !strings.Contains(err.Error(), "expected list") {
		t.Errorf("expected error mentioning \"expected list\", got: %v", err)
	}
}

func TestNavigateToPath_NegativeIndex(t *testing.T) {
	doc := nodeFromYAML(t, "items:\n  - a\n  - b\n  - c\n")
	_, err := navigateToPath(doc, "items[-1]")
	if err == nil {
		t.Fatal("expected error for negative list index, got nil")
	}
}

func TestNavigateToPath_KeyThroughScalar_ErrorMessage(t *testing.T) {
	doc := nodeFromYAML(t, "scalar: 42\n")
	_, err := navigateToPath(doc, "scalar.field")
	if err == nil {
		t.Fatal("expected error for key access through scalar")
	}
	if !strings.Contains(err.Error(), "expected map") {
		t.Errorf("expected error mentioning \"expected map\", got: %v", err)
	}
}

func TestParsePath_ExtraAfterBracket(t *testing.T) {
	if _, err := parsePath("key[1]extra"); err == nil {
		t.Fatal("expected error for \"key[1]extra\", got nil")
	}
}

func TestSplitPath_NestedOpenBracket(t *testing.T) {
	if _, err := splitPath("a[[]"); err == nil {
		t.Fatal("expected error for nested '[' in path, got nil")
	}
}

func TestSplitPath_UnterminatedBracket(t *testing.T) {
	if _, err := splitPath("a[1"); err == nil {
		t.Fatal("expected error for unterminated '[', got nil")
	}
}

func TestApplyChroot_EmptyPath_ListDoc_WrapsNotFlattens(t *testing.T) {
	listDoc := nodeFromYAML(t, "- a\n- b\n- c\n")
	result, err := applyChroot(listDoc, "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 doc (list wrapped, not flattened), got %d", len(result))
	}
	inner, ok := nodeToInterface(result[0]).([]any)
	if !ok {
		t.Fatalf("expected list inside result[0], got %T", nodeToInterface(result[0]))
	}
	if len(inner) != 3 {
		t.Errorf("expected inner list of len 3, got %d", len(inner))
	}
}

func TestCompareWithChroot_BothFiles(t *testing.T) {
	from := []byte(`---
root:
  data:
    name: from
    value: 100
`)
	to := []byte(`---
root:
  data:
    name: to
    value: 200
`)
	opts := &Options{
		Chroot: "root.data",
	}

	diffs, err := Compare(from, to, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find differences in name and value
	if len(diffs) < 2 {
		t.Errorf("expected at least 2 diffs, got %d", len(diffs))
	}

	// Paths should be relative to chroot
	hasNameDiff := false
	hasValueDiff := false
	for _, d := range diffs {
		if d.Path.String() == "name" {
			hasNameDiff = true
		}
		if d.Path.String() == "value" {
			hasValueDiff = true
		}
	}
	if !hasNameDiff {
		t.Error("expected diff for 'name' field")
	}
	if !hasValueDiff {
		t.Error("expected diff for 'value' field")
	}
}

func TestCompareWithChroot_SeparateFromTo(t *testing.T) {
	from := []byte(`---
section_a:
  value: from_a
section_b:
  value: from_b
`)
	to := []byte(`---
section_a:
  value: to_a
section_b:
  value: to_b
`)
	opts := &Options{
		ChrootFrom: "section_a",
		ChrootTo:   "section_b",
	}

	diffs, err := Compare(from, to, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should compare section_a.value (from_a) with section_b.value (to_b)
	if len(diffs) != 1 {
		t.Errorf("expected 1 diff, got %d", len(diffs))
	}

	if len(diffs) > 0 && diffs[0].From != "from_a" {
		t.Errorf("expected From=from_a, got From=%v", diffs[0].From)
	}
	if len(diffs) > 0 && diffs[0].To != "to_b" {
		t.Errorf("expected To=to_b, got To=%v", diffs[0].To)
	}
}
