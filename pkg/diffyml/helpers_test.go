package diffyml_test

import (
	"testing"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

// yml converts a YAML string to bytes for testing.
func yml(s string) []byte {
	return []byte(s)
}

// compare is a helper function that wraps diffyml.Compare for testing.
func compare(from, to []byte, opts *diffyml.Options) ([]diffyml.Difference, error) {
	return diffyml.Compare(from, to, opts)
}

// hasModification checks if there's a modification diff with the given from/to values.
func hasModification(diffs []diffyml.Difference, from, to interface{}) bool {
	for _, d := range diffs {
		if d.Type == diffyml.DiffModified {
			if d.From == from && d.To == to {
				return true
			}
			// Also check string representation for different types
			if fromStr, ok := d.From.(string); ok {
				if toStr, ok := d.To.(string); ok {
					if fromStr == from && toStr == to {
						return true
					}
				}
			}
		}
	}
	return false
}

// hasDiffType checks if there's a diff of the given type.
func hasDiffType(diffs []diffyml.Difference, diffType diffyml.DiffType) bool {
	for _, d := range diffs {
		if d.Type == diffType {
			return true
		}
	}
	return false
}

// noDiffs returns a check function that asserts no differences were found.
func noDiffs() func(*testing.T, []diffyml.Difference) {
	return func(t *testing.T, diffs []diffyml.Difference) {
		t.Helper()
		if len(diffs) != 0 {
			t.Errorf("expected 0 diffs, got %d", len(diffs))
		}
	}
}

// singleDiff returns a check function that asserts exactly 1 diff with the given path and type.
// Pass "" for path to skip the path check.
func singleDiff(path string, dt diffyml.DiffType) func(*testing.T, []diffyml.Difference) {
	return func(t *testing.T, diffs []diffyml.Difference) {
		t.Helper()
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}
		if path != "" && diffs[0].Path != path {
			t.Errorf("expected path %q, got %q", path, diffs[0].Path)
		}
		if diffs[0].Type != dt {
			t.Errorf("expected %v, got %v", dt, diffs[0].Type)
		}
	}
}

// diffCount returns a check function that asserts the exact number of differences.
func diffCount(n int) func(*testing.T, []diffyml.Difference) {
	return func(t *testing.T, diffs []diffyml.Difference) {
		t.Helper()
		if len(diffs) != n {
			t.Fatalf("expected %d diffs, got %d", n, len(diffs))
		}
	}
}

// hasTypes returns a check function that asserts at least 1 diff exists and
// all given types are present.
func hasTypes(types ...diffyml.DiffType) func(*testing.T, []diffyml.Difference) {
	return func(t *testing.T, diffs []diffyml.Difference) {
		t.Helper()
		if len(diffs) == 0 {
			t.Fatal("expected at least 1 diff, got 0")
		}
		for _, dt := range types {
			if !hasDiffType(diffs, dt) {
				t.Errorf("expected diff type %v not found", dt)
			}
		}
	}
}

// mod returns a check function that asserts at least 1 modification from→to exists.
func mod(from, to interface{}) func(*testing.T, []diffyml.Difference) {
	return func(t *testing.T, diffs []diffyml.Difference) {
		t.Helper()
		if len(diffs) == 0 {
			t.Fatal("expected at least 1 diff, got 0")
		}
		if !hasModification(diffs, from, to) {
			t.Errorf("expected modification from %v to %v", from, to)
		}
	}
}
