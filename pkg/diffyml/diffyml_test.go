package diffyml

import (
	"testing"
)

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
