package diffyml_test

import (
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
