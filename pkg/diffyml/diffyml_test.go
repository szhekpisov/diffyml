package diffyml

import (
	"testing"
)

// --- Mutation testing: diffyml.go sorting/ordering ---

func TestExtractPathOrder_EmptyPrefix(t *testing.T) {
	// diffyml.go:136 — if `prefix != ""` is mutated to `prefix == ""`,
	// then empty prefix gets registered instead of non-empty prefixes.
	// We verify that keys at the top level get correct ordering indices.
	om := NewOrderedMap()
	om.Keys = []string{"beta", "alpha"}
	om.Values["beta"] = "val1"
	om.Values["alpha"] = "val2"

	docs := []interface{}{om}
	opts := &Options{}
	pathOrder := extractPathOrder(docs, nil, opts)

	// "beta" should come before "alpha" because it appears first in the map
	betaIdx, hasBeta := pathOrder["beta"]
	alphaIdx, hasAlpha := pathOrder["alpha"]
	if !hasBeta || !hasAlpha {
		t.Fatalf("pathOrder missing keys: beta=%v alpha=%v", hasBeta, hasAlpha)
	}
	if betaIdx >= alphaIdx {
		t.Errorf("beta (idx=%d) should have lower index than alpha (idx=%d)", betaIdx, alphaIdx)
	}

	// Empty prefix should NOT be registered
	if _, hasEmpty := pathOrder[""]; hasEmpty {
		t.Error("empty prefix should not be registered in pathOrder")
	}
}

func TestExtractPathOrder_IndexIncrement(t *testing.T) {
	// diffyml.go:176 — if `index++` is mutated to `index--`, successive paths
	// would get decreasing indices instead of increasing ones.
	om := NewOrderedMap()
	om.Keys = []string{"first", "second", "third"}
	om.Values["first"] = "a"
	om.Values["second"] = "b"
	om.Values["third"] = "c"

	docs := []interface{}{om}
	opts := &Options{}
	pathOrder := extractPathOrder(docs, nil, opts)

	firstIdx := pathOrder["first"]
	secondIdx := pathOrder["second"]
	thirdIdx := pathOrder["third"]

	// Each successive key must have a strictly larger index
	if firstIdx >= secondIdx {
		t.Errorf("first (idx=%d) should be less than second (idx=%d)", firstIdx, secondIdx)
	}
	if secondIdx >= thirdIdx {
		t.Errorf("second (idx=%d) should be less than third (idx=%d)", secondIdx, thirdIdx)
	}
}

func TestExtractPathOrder_ListIndexIncrement(t *testing.T) {
	// diffyml.go:176 — specifically for list items ([]interface{} branch)
	// If index-- instead of index++, the list prefix "items" gets index 0,
	// then index becomes -1, making subsequent paths get negative indices.
	// This causes the list prefix to sort AFTER its own items, breaking order.
	//
	// Test: "items" prefix should have a LOWER index than its first child "items.0"
	list := []interface{}{"item0", "item1"}

	om := NewOrderedMap()
	om.Keys = []string{"items"}
	om.Values["items"] = list

	docs := []interface{}{om}
	opts := &Options{}
	pathOrder := extractPathOrder(docs, nil, opts)

	itemsIdx, hasItems := pathOrder["items"]
	idx0, has0 := pathOrder["items.0"]
	if !hasItems || !has0 {
		t.Fatalf("missing path entries: items=%v items.0=%v", hasItems, has0)
	}
	// "items" is registered first (at index N), then index increments,
	// then "items.0" gets N+1. With mutation index--, "items.0" gets N-1 < N.
	if itemsIdx >= idx0 {
		t.Errorf("items prefix (idx=%d) should have lower index than items.0 (idx=%d)", itemsIdx, idx0)
	}
}

func TestSortDiffsWithOrder_RootOrdering(t *testing.T) {
	// diffyml.go:305 — `orderI < orderJ` → `<=`
	// Two roots with different order should be correctly sorted.
	pathOrder := map[string]int{
		"alpha": 0,
		"beta":  1,
	}

	diffs := []Difference{
		{Path: "beta.key", Type: DiffModified, From: "a", To: "b"},
		{Path: "alpha.key", Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	if diffs[0].Path != "alpha.key" {
		t.Errorf("expected alpha.key first, got %s", diffs[0].Path)
	}
	if diffs[1].Path != "beta.key" {
		t.Errorf("expected beta.key second, got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_SameRootAlphaFallback(t *testing.T) {
	// diffyml.go:307 — `rootI < rootJ` alphabetical fallback when both roots
	// lack pathOrder entries. Mutation `<` → `<=` would break tie-breaking.
	pathOrder := map[string]int{}

	diffs := []Difference{
		{Path: "zebra.key", Type: DiffModified, From: "a", To: "b"},
		{Path: "apple.key", Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	if diffs[0].Path != "apple.key" {
		t.Errorf("expected apple.key first (alphabetical fallback), got %s", diffs[0].Path)
	}
}

func TestSortDiffsWithOrder_ParentOrderTieBreak(t *testing.T) {
	// diffyml.go:333 — `lastDot == -1` → `!=` breaks parent path traversal
	// diffyml.go:343 — `parentOrderI != parentOrderJ` → `==` skips parent ordering
	//
	// Key: parent order must DISAGREE with alphabetical order to detect mutation.
	// "root.zebra" (order=1) vs "root.apple" (order=2) — parent order says zebra first,
	// but alphabetical says apple first. Only the original uses parent order.
	pathOrder := map[string]int{
		"root":       0,
		"root.zebra": 1,
		"root.apple": 2,
	}

	diffs := []Difference{
		{Path: "root.apple.child.deep", Type: DiffModified, From: "a", To: "b"},
		{Path: "root.zebra.child.deep", Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	// root.zebra has lower order (1) than root.apple (2), so zebra's child should come first.
	// If mutation breaks parent traversal, alphabetical order puts apple first (wrong).
	if diffs[0].Path != "root.zebra.child.deep" {
		t.Errorf("expected root.zebra.child.deep first (parent order), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "root.apple.child.deep" {
		t.Errorf("expected root.apple.child.deep second, got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_DepthDifference(t *testing.T) {
	// diffyml.go:350 — `depthI != depthJ` → `==` skips depth sorting
	// diffyml.go:351 — `depthI < depthJ` → `<=`
	pathOrder := map[string]int{
		"root": 0,
	}

	diffs := []Difference{
		{Path: "root.a.b.c", Type: DiffModified, From: "a", To: "b"}, // depth 3
		{Path: "root.x", Type: DiffModified, From: "c", To: "d"},     // depth 1
	}

	sortDiffsWithOrder(diffs, pathOrder)

	// Shallower path (depth 1) should come first
	if diffs[0].Path != "root.x" {
		t.Errorf("expected root.x first (shallower depth), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "root.a.b.c" {
		t.Errorf("expected root.a.b.c second (deeper), got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_AlphabeticalFallback(t *testing.T) {
	// diffyml.go:355 — `pathI < pathJ` → `<=` final alphabetical fallback
	pathOrder := map[string]int{
		"root": 0,
	}

	// Same root, same depth, different alphabetical paths
	diffs := []Difference{
		{Path: "root.zebra", Type: DiffModified, From: "a", To: "b"},
		{Path: "root.apple", Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	if diffs[0].Path != "root.apple" {
		t.Errorf("expected root.apple first (alphabetical), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "root.zebra" {
		t.Errorf("expected root.zebra second (alphabetical), got %s", diffs[1].Path)
	}
}

func TestIsListEntryDiff_SingleCharPath(t *testing.T) {
	// diffyml.go:222 — `len(path) > 1` ensures we don't check single-char paths
	// diffyml.go:224 — `lastDot >= 0` boundary at position 0
	// diffyml.go:224 — `lastDot < len(path)-1` boundary when dot is last char

	// Path ".0" — dot at position 0, suffix is "0" (a digit)
	diff := Difference{Path: ".0", Type: DiffAdded, To: "value"}
	if !isListEntryDiff(diff) {
		t.Error("path '.0' should be detected as list entry")
	}

	// Path "x." — dot is last char, lastDot == len(path)-1
	diff2 := Difference{Path: "x.", Type: DiffAdded, To: "value"}
	if isListEntryDiff(diff2) {
		t.Error("path 'x.' should NOT be detected as list entry (dot at end)")
	}

	// Path ending with ']'
	diff3 := Difference{Path: "items[0]", Type: DiffAdded, To: "value"}
	if !isListEntryDiff(diff3) {
		t.Error("path 'items[0]' should be detected as list entry")
	}
}

func TestSortDiffsWithOrder_ViaCompare(t *testing.T) {
	// Integration test: ensures the full pipeline uses correct ordering.
	// Exercises: extractPathOrder with prefix != "" (line 136),
	// index++ (line 176), and sortDiffsWithOrder.
	from := []byte(`---
beta:
  key: old_b
alpha:
  key: old_a
`)
	to := []byte(`---
beta:
  key: new_b
alpha:
  key: new_a
`)

	diffs, err := Compare(from, to, nil)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(diffs) < 2 {
		t.Fatalf("expected at least 2 diffs, got %d", len(diffs))
	}

	// "beta" appears first in the source YAML, so beta.key should come before alpha.key
	if diffs[0].Path != "beta.key" {
		t.Errorf("expected beta.key first (document order), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "alpha.key" {
		t.Errorf("expected alpha.key second (document order), got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_MultipleRoots(t *testing.T) {
	// Exercises multiple roots with document-order sorting.
	// Kills mutant at line 305 (orderI < orderJ).
	from := []byte(`---
charlie:
  v: 1
alpha:
  v: 2
bravo:
  v: 3
`)
	to := []byte(`---
charlie:
  v: 10
alpha:
  v: 20
bravo:
  v: 30
`)

	diffs, err := Compare(from, to, nil)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(diffs) != 3 {
		t.Fatalf("expected 3 diffs, got %d", len(diffs))
	}

	// Should preserve YAML document order: charlie, alpha, bravo
	expected := []string{"charlie.v", "alpha.v", "bravo.v"}
	for i, exp := range expected {
		if diffs[i].Path != exp {
			t.Errorf("diff[%d].Path = %q, want %q", i, diffs[i].Path, exp)
		}
	}
}
