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

	docs := []any{om}
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

	docs := []any{om}
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
	// diffyml.go:176 — specifically for list items ([]any branch)
	// If index-- instead of index++, the list prefix "items" gets index 0,
	// then index becomes -1, making subsequent paths get negative indices.
	// This causes the list prefix to sort AFTER its own items, breaking order.
	//
	// Test: "items" prefix should have a LOWER index than its first child "items.0"
	list := []any{"item0", "item1"}

	om := NewOrderedMap()
	om.Keys = []string{"items"}
	om.Values["items"] = list

	docs := []any{om}
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
		{Path: DiffPath{"beta", "key"}, Type: DiffModified, From: "a", To: "b"},
		{Path: DiffPath{"alpha", "key"}, Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	if diffs[0].Path.String() != "alpha.key" {
		t.Errorf("expected alpha.key first, got %s", diffs[0].Path.String())
	}
	if diffs[1].Path.String() != "beta.key" {
		t.Errorf("expected beta.key second, got %s", diffs[1].Path.String())
	}
}

func TestSortDiffsWithOrder_SameRootAlphaFallback(t *testing.T) {
	// diffyml.go:307 — `rootI < rootJ` alphabetical fallback when both roots
	// lack pathOrder entries. Mutation `<` → `<=` would break tie-breaking.
	pathOrder := map[string]int{}

	diffs := []Difference{
		{Path: DiffPath{"zebra", "key"}, Type: DiffModified, From: "a", To: "b"},
		{Path: DiffPath{"apple", "key"}, Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	if diffs[0].Path.String() != "apple.key" {
		t.Errorf("expected apple.key first (alphabetical fallback), got %s", diffs[0].Path.String())
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
		{Path: DiffPath{"root", "apple", "child", "deep"}, Type: DiffModified, From: "a", To: "b"},
		{Path: DiffPath{"root", "zebra", "child", "deep"}, Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	// root.zebra has lower order (1) than root.apple (2), so zebra's child should come first.
	// If mutation breaks parent traversal, alphabetical order puts apple first (wrong).
	if diffs[0].Path.String() != "root.zebra.child.deep" {
		t.Errorf("expected root.zebra.child.deep first (parent order), got %s", diffs[0].Path.String())
	}
	if diffs[1].Path.String() != "root.apple.child.deep" {
		t.Errorf("expected root.apple.child.deep second, got %s", diffs[1].Path.String())
	}
}

func TestSortDiffsWithOrder_DepthDifference(t *testing.T) {
	// diffyml.go:350 — `depthI != depthJ` → `==` skips depth sorting
	// diffyml.go:351 — `depthI < depthJ` → `<=`
	pathOrder := map[string]int{
		"root": 0,
	}

	diffs := []Difference{
		{Path: DiffPath{"root", "a", "b", "c"}, Type: DiffModified, From: "a", To: "b"}, // depth 3
		{Path: DiffPath{"root", "x"}, Type: DiffModified, From: "c", To: "d"},           // depth 1
	}

	sortDiffsWithOrder(diffs, pathOrder)

	// Shallower path (depth 1) should come first
	if diffs[0].Path.String() != "root.x" {
		t.Errorf("expected root.x first (shallower depth), got %s", diffs[0].Path.String())
	}
	if diffs[1].Path.String() != "root.a.b.c" {
		t.Errorf("expected root.a.b.c second (deeper), got %s", diffs[1].Path.String())
	}
}

func TestSortDiffsWithOrder_AlphabeticalFallback(t *testing.T) {
	// diffyml.go:355 — `pathI < pathJ` → `<=` final alphabetical fallback
	pathOrder := map[string]int{
		"root": 0,
	}

	// Same root, same depth, different alphabetical paths
	diffs := []Difference{
		{Path: DiffPath{"root", "zebra"}, Type: DiffModified, From: "a", To: "b"},
		{Path: DiffPath{"root", "apple"}, Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	if diffs[0].Path.String() != "root.apple" {
		t.Errorf("expected root.apple first (alphabetical), got %s", diffs[0].Path.String())
	}
	if diffs[1].Path.String() != "root.zebra" {
		t.Errorf("expected root.zebra second (alphabetical), got %s", diffs[1].Path.String())
	}
}

func TestExtractPathOrder_OrderedMapNestedIndexIncrement(t *testing.T) {
	// Kills INCREMENT_DECREMENT at diffyml.go:141 (index++ → index--)
	// The existing TestExtractPathOrder_IndexIncrement uses flat OrderedMap with
	// scalar values — prefix is always "" so the code at line 138-141 is skipped.
	// We need a NESTED OrderedMap so that recursion enters the OrderedMap branch
	// with a non-empty prefix, hitting the index++ at line 141.
	child1 := NewOrderedMap()
	child1.Keys = []string{"x"}
	child1.Values["x"] = "val"

	child2 := NewOrderedMap()
	child2.Keys = []string{"y"}
	child2.Values["y"] = "val"

	child3 := NewOrderedMap()
	child3.Keys = []string{"z"}
	child3.Values["z"] = "val"

	root := NewOrderedMap()
	root.Keys = []string{"first", "second", "third"}
	root.Values["first"] = child1
	root.Values["second"] = child2
	root.Values["third"] = child3

	docs := []any{root}
	pathOrder := extractPathOrder(docs, nil, &Options{})

	// Each nested OrderedMap prefix must get a strictly increasing index.
	// With index-- mutation, they'd all get 0 or decreasing indices.
	firstIdx := pathOrder["first"]
	secondIdx := pathOrder["second"]
	thirdIdx := pathOrder["third"]

	if firstIdx >= secondIdx {
		t.Errorf("first (%d) should be less than second (%d)", firstIdx, secondIdx)
	}
	if secondIdx >= thirdIdx {
		t.Errorf("second (%d) should be less than third (%d)", secondIdx, thirdIdx)
	}
}

func TestIsListEntryDiff_EmptyPath(t *testing.T) {
	// Kills CONDITIONALS_BOUNDARY at diffyml.go:219 (len(path) > 0 → >= 0)
	// With >= 0, empty path would proceed to path[len(path)-1] = path[-1] → panic.
	diff := Difference{Path: nil, Type: DiffModified, From: "a", To: "b"}
	result := isListEntryDiff(diff)
	if result {
		t.Error("empty path should not be detected as list entry")
	}
}

func TestIsListEntryDiff_ToNilUsesFrom(t *testing.T) {
	// Kills CONDITIONALS_NEGATION at diffyml.go:242 (diff.To != nil → == nil)
	// When To is nil, the code should use From for the map-identifier heuristic.
	// With the negation, it picks nil To instead → returns false.
	om := NewOrderedMap()
	om.Keys = []string{"name", "value"}
	om.Values["name"] = "my-item"
	om.Values["value"] = "data"

	diff := Difference{Path: DiffPath{"items"}, Type: DiffRemoved, From: om, To: nil}
	if !isListEntryDiff(diff) {
		t.Error("expected true when To is nil but From has identifier field 'name'")
	}
}

func TestIsListEntryDiff_ToHasIdentifier(t *testing.T) {
	// Counterpart: when To is non-nil, it should be used. Mutation picks nil From → false.
	om := NewOrderedMap()
	om.Keys = []string{"name", "value"}
	om.Values["name"] = "my-item"
	om.Values["value"] = "data"

	diff := Difference{Path: DiffPath{"items"}, Type: DiffAdded, From: nil, To: om}
	if !isListEntryDiff(diff) {
		t.Error("expected true when From is nil but To has identifier field 'name'")
	}
}

func TestIsListEntryDiff_SingleCharPath(t *testing.T) {
	// Test numeric last segment detection
	// DiffPath{"0"} has HasNumericLast() = true
	diff := Difference{Path: DiffPath{"0"}, Type: DiffAdded, To: "value"}
	if !isListEntryDiff(diff) {
		t.Error("path with numeric last segment should be detected as list entry")
	}

	// DiffPath{"x", ""} — last segment is empty, not numeric
	diff2 := Difference{Path: DiffPath{"x", ""}, Type: DiffAdded, To: "value"}
	if isListEntryDiff(diff2) {
		t.Error("path with empty last segment should NOT be detected as list entry")
	}

	// DiffPath{"items", "0"} — last segment is "0" (numeric)
	diff3 := Difference{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: "value"}
	if !isListEntryDiff(diff3) {
		t.Error("path with numeric index segment should be detected as list entry")
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
	if diffs[0].Path.String() != "beta.key" {
		t.Errorf("expected beta.key first (document order), got %s", diffs[0].Path.String())
	}
	if diffs[1].Path.String() != "alpha.key" {
		t.Errorf("expected alpha.key second (document order), got %s", diffs[1].Path.String())
	}
}

func TestSortDiffsWithOrder_OneHasOrderOtherNot(t *testing.T) {
	// Exercises the branch where within the same root, one path has a
	// pathOrder entry and the other doesn't. The one with order should
	// come first regardless of alphabetical ordering.
	pathOrder := map[string]int{
		"root":       0,
		"root.zebra": 1,
		// "root.apple" intentionally absent from pathOrder
	}

	diffs := []Difference{
		{Path: DiffPath{"root", "apple"}, Type: DiffModified, From: "a", To: "b"},
		{Path: DiffPath{"root", "zebra"}, Type: DiffModified, From: "c", To: "d"},
	}

	sortDiffsWithOrder(diffs, pathOrder)

	// root.zebra has pathOrder entry → should come first
	if diffs[0].Path.String() != "root.zebra" {
		t.Errorf("expected root.zebra first (has order), got %s", diffs[0].Path.String())
	}
	if diffs[1].Path.String() != "root.apple" {
		t.Errorf("expected root.apple second (no order), got %s", diffs[1].Path.String())
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
		if diffs[i].Path.String() != exp {
			t.Errorf("diff[%d].Path = %q, want %q", i, diffs[i].Path.String(), exp)
		}
	}
}
