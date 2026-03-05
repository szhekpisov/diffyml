// kubernetes.go - Kubernetes resource detection and matching.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
)

func IsKubernetesResource(doc interface{}) bool {
	return compare.IsKubernetesResource(doc)
}

func GetK8sResourceIdentifier(doc interface{}, ignoreApiVersion bool) string {
	return compare.GetK8sResourceIdentifier(doc, ignoreApiVersion)
}

func GetIdentifierWithAdditional(v interface{}, additionalIdentifiers []string) interface{} {
	return compare.GetIdentifierWithAdditional(v, additionalIdentifiers)
}

func CanMatchByIdentifierWithAdditional(list []interface{}, additionalIdentifiers []string) bool {
	return compare.CanMatchByIdentifierWithAdditional(list, additionalIdentifiers)
}

func matchK8sDocuments(from, to []interface{}, opts *Options) (matched map[int]int, unmatchedFrom, unmatchedTo []int) {
	return compare.MatchK8sDocuments(from, to, opts)
}

func compareK8sDocs(from, to []interface{}, opts *Options, compareFn nodeComparerFn) []Difference {
	return compare.CompareK8sDocs(from, to, opts, compareFn)
}
