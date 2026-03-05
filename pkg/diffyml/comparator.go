// comparator.go - Core YAML comparison logic.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
)

type nodeComparerFn = compare.NodeComparerFn

func compareNodes(path string, from, to interface{}, opts *Options) []Difference {
	return compare.CompareNodes(path, from, to, opts)
}

func compareOrderedMaps(path string, from, to *OrderedMap, opts *Options) []Difference {
	return compare.CompareOrderedMaps(path, from, to, opts)
}

func areListItemsHeterogeneous(from, to []interface{}) bool {
	return compare.AreListItemsHeterogeneous(from, to)
}

func compareListsPositional(path string, from, to []interface{}, opts *Options) []Difference {
	return compare.CompareListsPositional(path, from, to, opts)
}

func compareListsUnordered(path string, from, to []interface{}, opts *Options) []Difference {
	return compare.CompareListsUnordered(path, from, to, opts)
}

func compareListsByIdentifier(path string, from, to []interface{}, opts *Options) []Difference {
	return compare.CompareListsByIdentifier(path, from, to, opts)
}

func deepEqual(from, to interface{}, opts *Options) bool {
	return compare.DeepEqual(from, to, opts)
}
