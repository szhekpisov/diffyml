// comparator.go - Core YAML comparison logic.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
)

type nodeComparerFn = compare.NodeComparerFn

func compareDocs(from, to []interface{}, opts *Options) []Difference {
	return compare.CompareDocs(from, to, opts)
}

func hasK8sDocuments(from, to []interface{}) bool {
	return compare.HasK8sDocuments(from, to)
}

func compareNodes(path string, from, to interface{}, opts *Options) []Difference {
	return compare.CompareNodes(path, from, to, opts)
}

func compareOrderedMaps(path string, from, to *OrderedMap, opts *Options) []Difference {
	return compare.CompareOrderedMaps(path, from, to, opts)
}

func compareLists(path string, from, to []interface{}, opts *Options) []Difference {
	return compare.CompareLists(path, from, to, opts)
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

func canMatchByIdentifier(list []interface{}, opts *Options) bool {
	return compare.CanMatchByIdentifier(list, opts)
}

func getIdentifier(val interface{}, opts *Options) interface{} {
	return compare.GetIdentifier(val, opts)
}

func compareListsByIdentifier(path string, from, to []interface{}, opts *Options) []Difference {
	return compare.CompareListsByIdentifier(path, from, to, opts)
}

func equalValues(from, to interface{}, opts *Options) bool {
	return compare.EqualValues(from, to, opts)
}

func deepEqual(from, to interface{}, opts *Options) bool {
	return compare.DeepEqual(from, to, opts)
}
