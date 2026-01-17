package diffyml

import (
	"testing"
)

func TestNavigateToPath_SimplePath(t *testing.T) {
	doc := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"value": "found",
			},
		},
	}

	result, err := navigateToPath(doc, "level1.level2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["value"] != "found" {
		t.Errorf("expected value=found, got value=%v", m["value"])
	}
}

func TestNavigateToPath_SingleLevel(t *testing.T) {
	doc := map[string]interface{}{
		"data": "hello",
	}

	result, err := navigateToPath(doc, "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "hello" {
		t.Errorf("expected hello, got %v", result)
	}
}

func TestNavigateToPath_EmptyPath(t *testing.T) {
	doc := map[string]interface{}{
		"key": "value",
	}

	result, err := navigateToPath(doc, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty path returns original document
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value in result")
	}
}

func TestNavigateToPath_PathNotFound(t *testing.T) {
	doc := map[string]interface{}{
		"existing": "value",
	}

	_, err := navigateToPath(doc, "nonexistent.path")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestNavigateToPath_PathThroughScalar(t *testing.T) {
	doc := map[string]interface{}{
		"scalar": "value",
	}

	_, err := navigateToPath(doc, "scalar.deeper")
	if err == nil {
		t.Error("expected error when navigating through scalar")
	}
}

func TestNavigateToPath_ListIndex(t *testing.T) {
	doc := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "first"},
			map[string]interface{}{"name": "second"},
			map[string]interface{}{"name": "third"},
		},
	}

	result, err := navigateToPath(doc, "items[1]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["name"] != "second" {
		t.Errorf("expected name=second, got name=%v", m["name"])
	}
}

func TestNavigateToPath_ListIndexOutOfBounds(t *testing.T) {
	doc := map[string]interface{}{
		"items": []interface{}{"a", "b"},
	}

	_, err := navigateToPath(doc, "items[5]")
	if err == nil {
		t.Error("expected error for out of bounds index")
	}
}

func TestNavigateToPath_InvalidListIndex(t *testing.T) {
	doc := map[string]interface{}{
		"items": []interface{}{"a", "b"},
	}

	_, err := navigateToPath(doc, "items[foo]")
	if err == nil {
		t.Fatal("expected error for invalid list index")
	}
}

func TestNavigateToPath_InvalidPathSyntax(t *testing.T) {
	doc := map[string]interface{}{
		"items": []interface{}{"a", "b"},
	}

	_, err := navigateToPath(doc, "items[0")
	if err == nil {
		t.Fatal("expected error for invalid path syntax")
	}
}

func TestNavigateToPath_NestedListAccess(t *testing.T) {
	doc := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"nested": []interface{}{"x", "y", "z"},
			},
		},
	}

	result, err := navigateToPath(doc, "data[0].nested[2]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "z" {
		t.Errorf("expected z, got %v", result)
	}
}

func TestApplyChroot_ToList(t *testing.T) {
	doc := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "one"},
			map[string]interface{}{"name": "two"},
		},
	}

	// When listToDocuments is false, return the list as single doc
	result, err := applyChroot(doc, "items", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 document, got %d", len(result))
	}

	list, ok := result[0].([]interface{})
	if !ok {
		t.Fatalf("expected list, got %T", result[0])
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items in list, got %d", len(list))
	}
}

func TestApplyChroot_ListToDocuments(t *testing.T) {
	doc := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "one"},
			map[string]interface{}{"name": "two"},
			map[string]interface{}{"name": "three"},
		},
	}

	// When listToDocuments is true, each list item becomes a document
	result, err := applyChroot(doc, "items", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 documents, got %d", len(result))
	}

	for i, expected := range []string{"one", "two", "three"} {
		m, ok := result[i].(map[string]interface{})
		if !ok {
			t.Fatalf("document %d: expected map, got %T", i, result[i])
		}
		if m["name"] != expected {
			t.Errorf("document %d: expected name=%s, got name=%v", i, expected, m["name"])
		}
	}
}

func TestApplyChroot_NonListWithListToDocuments(t *testing.T) {
	doc := map[string]interface{}{
		"data": map[string]interface{}{
			"key": "value",
		},
	}

	// When path points to non-list but listToDocuments is true, return as single doc
	result, err := applyChroot(doc, "data", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 document, got %d", len(result))
	}
}

func TestApplyChroot_EmptyPath(t *testing.T) {
	doc := map[string]interface{}{
		"key": "value",
	}

	result, err := applyChroot(doc, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 document, got %d", len(result))
	}
}

func TestApplyChroot_PathNotFound(t *testing.T) {
	doc := map[string]interface{}{
		"existing": "value",
	}

	_, err := applyChroot(doc, "nonexistent", false)
	if err == nil {
		t.Error("expected error for non-existent path")
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
		if d.Path == "name" {
			hasNameDiff = true
		}
		if d.Path == "value" {
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
