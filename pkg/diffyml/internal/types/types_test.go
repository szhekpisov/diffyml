package types

import (
	"fmt"
	"testing"
)

func TestSortDiffsWithOrder_RootOrdering(t *testing.T) {
	pathOrder := map[string]int{
		"alpha": 0,
		"beta":  1,
	}
	diffs := []Difference{
		{Path: "beta.key", Type: DiffModified, From: "a", To: "b"},
		{Path: "alpha.key", Type: DiffModified, From: "c", To: "d"},
	}
	SortDiffsWithOrder(diffs, pathOrder)
	if diffs[0].Path != "alpha.key" {
		t.Errorf("expected alpha.key first, got %s", diffs[0].Path)
	}
	if diffs[1].Path != "beta.key" {
		t.Errorf("expected beta.key second, got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_SameRootAlphaFallback(t *testing.T) {
	pathOrder := map[string]int{}
	diffs := []Difference{
		{Path: "zebra.key", Type: DiffModified, From: "a", To: "b"},
		{Path: "apple.key", Type: DiffModified, From: "c", To: "d"},
	}
	SortDiffsWithOrder(diffs, pathOrder)
	if diffs[0].Path != "apple.key" {
		t.Errorf("expected apple.key first (alphabetical fallback), got %s", diffs[0].Path)
	}
}

func TestSortDiffsWithOrder_ParentOrderTieBreak(t *testing.T) {
	pathOrder := map[string]int{
		"root":       0,
		"root.zebra": 1,
		"root.apple": 2,
	}
	diffs := []Difference{
		{Path: "root.apple.child.deep", Type: DiffModified, From: "a", To: "b"},
		{Path: "root.zebra.child.deep", Type: DiffModified, From: "c", To: "d"},
	}
	SortDiffsWithOrder(diffs, pathOrder)
	if diffs[0].Path != "root.zebra.child.deep" {
		t.Errorf("expected root.zebra.child.deep first (parent order), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "root.apple.child.deep" {
		t.Errorf("expected root.apple.child.deep second, got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_DepthDifference(t *testing.T) {
	pathOrder := map[string]int{
		"root": 0,
	}
	diffs := []Difference{
		{Path: "root.a.b.c", Type: DiffModified, From: "a", To: "b"},
		{Path: "root.x", Type: DiffModified, From: "c", To: "d"},
	}
	SortDiffsWithOrder(diffs, pathOrder)
	if diffs[0].Path != "root.x" {
		t.Errorf("expected root.x first (shallower depth), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "root.a.b.c" {
		t.Errorf("expected root.a.b.c second (deeper), got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_AlphabeticalFallback(t *testing.T) {
	pathOrder := map[string]int{
		"root": 0,
	}
	diffs := []Difference{
		{Path: "root.zebra", Type: DiffModified, From: "a", To: "b"},
		{Path: "root.apple", Type: DiffModified, From: "c", To: "d"},
	}
	SortDiffsWithOrder(diffs, pathOrder)
	if diffs[0].Path != "root.apple" {
		t.Errorf("expected root.apple first (alphabetical), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "root.zebra" {
		t.Errorf("expected root.zebra second (alphabetical), got %s", diffs[1].Path)
	}
}

func TestSortDiffsWithOrder_OneHasOrderOtherNot(t *testing.T) {
	pathOrder := map[string]int{
		"root":       0,
		"root.zebra": 1,
	}
	diffs := []Difference{
		{Path: "root.apple", Type: DiffModified, From: "a", To: "b"},
		{Path: "root.zebra", Type: DiffModified, From: "c", To: "d"},
	}
	SortDiffsWithOrder(diffs, pathOrder)
	if diffs[0].Path != "root.zebra" {
		t.Errorf("expected root.zebra first (has order), got %s", diffs[0].Path)
	}
	if diffs[1].Path != "root.apple" {
		t.Errorf("expected root.apple second (no order), got %s", diffs[1].Path)
	}
}

func TestIsListEntryDiff_SingleCharPath(t *testing.T) {
	diff := Difference{Path: ".0", Type: DiffAdded, To: "value"}
	if !IsListEntryDiff(diff) {
		t.Error("path '.0' should be detected as list entry")
	}
	diff2 := Difference{Path: "x.", Type: DiffAdded, To: "value"}
	if IsListEntryDiff(diff2) {
		t.Error("path 'x.' should NOT be detected as list entry (dot at end)")
	}
	diff3 := Difference{Path: "items[0]", Type: DiffAdded, To: "value"}
	if !IsListEntryDiff(diff3) {
		t.Error("path 'items[0]' should be detected as list entry")
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, "zero"},
		{1, "one"},
		{2, "two"},
		{3, "three"},
		{4, "four"},
		{5, "five"},
		{6, "six"},
		{7, "seven"},
		{8, "eight"},
		{9, "nine"},
		{10, "ten"},
		{11, "eleven"},
		{12, "twelve"},
		{13, "13"},
		{100, "100"},
	}
	for _, tt := range tests {
		result := FormatCount(tt.n)
		if result != tt.expected {
			t.Errorf("FormatCount(%d) = %q, want %q", tt.n, result, tt.expected)
		}
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		n        int
		singular string
		plural   string
		expected string
	}{
		{1, "entry", "entries", "entry"},
		{2, "entry", "entries", "entries"},
		{0, "entry", "entries", "entries"},
	}
	for _, tt := range tests {
		result := Pluralize(tt.n, tt.singular, tt.plural)
		if result != tt.expected {
			t.Errorf("Pluralize(%d, %q, %q) = %q, want %q", tt.n, tt.singular, tt.plural, result, tt.expected)
		}
	}
}

func BenchmarkSortDiffsWithOrder(b *testing.B) {
	makeDiffs := func(n int) ([]Difference, map[string]int) {
		diffs := make([]Difference, n)
		pathOrder := make(map[string]int)
		for i := 0; i < n; i++ {
			path := fmt.Sprintf("root.section-%03d.key-%03d", i%10, i)
			diffs[i] = Difference{
				Path: path,
				Type: DiffModified,
				From: "old",
				To:   "new",
			}
			pathOrder[path] = i
		}
		return diffs, pathOrder
	}
	sizes := []int{100, 1000}
	for _, n := range sizes {
		diffs, pathOrder := makeDiffs(n)
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			buf := make([]Difference, len(diffs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				copy(buf, diffs)
				SortDiffsWithOrder(buf, pathOrder)
			}
		})
	}
}
