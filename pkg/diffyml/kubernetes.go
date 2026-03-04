// kubernetes.go - Kubernetes resource detection and matching.
//
// Detects K8s resources by checking for apiVersion, kind, and metadata fields.
// Matches resources across documents using apiVersion + kind + metadata.name (or generateName).
// Key functions: IsKubernetesResource(), GetK8sIdentifier().
package diffyml

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
)

// k8sDocumentPath is the diff path used for document-level changes (e.g. order).
const k8sDocumentPath = "(document)"

// IsKubernetesResource checks if a document has the structure of a Kubernetes resource.
// A Kubernetes resource must have apiVersion, kind, and metadata fields,
// where metadata is a map containing at least a name field.
func IsKubernetesResource(doc interface{}) bool {
	_, ok := asK8sResource(doc)
	return ok
}

// k8sResource holds the pre-validated fields of a Kubernetes resource document.
type k8sResource struct {
	om         *OrderedMap
	apiVersion string
	kind       string
	metaOM     *OrderedMap
}

// asK8sResource validates that doc is a K8s resource and returns its key fields.
func asK8sResource(doc interface{}) (k8sResource, bool) {
	om := toOrderedMap(doc)
	if om == nil {
		return k8sResource{}, false
	}

	apiVersion, ok := om.Values["apiVersion"]
	if !ok {
		return k8sResource{}, false
	}
	apiStr, isStr := apiVersion.(string)
	if !isStr {
		return k8sResource{}, false
	}

	kind, ok := om.Values["kind"]
	if !ok {
		return k8sResource{}, false
	}
	kindStr, isStr := kind.(string)
	if !isStr {
		return k8sResource{}, false
	}

	metadata, ok := om.Values["metadata"]
	if !ok {
		return k8sResource{}, false
	}
	metaOM := toOrderedMap(metadata)
	if metaOM == nil {
		return k8sResource{}, false
	}

	metaName, hasName := metaOM.Values["name"]
	metaGenName, hasGenName := metaOM.Values["generateName"]
	if (!hasName || metaName == nil) && (!hasGenName || metaGenName == nil) {
		return k8sResource{}, false
	}

	return k8sResource{om: om, apiVersion: apiStr, kind: kindStr, metaOM: metaOM}, true
}

// GetK8sResourceIdentifier returns a unique identifier for a Kubernetes resource.
// When ignoreApiVersion is false: "apiVersion:kind:namespace/name" or "apiVersion:kind:name".
// When ignoreApiVersion is true: "kind:namespace/name" or "kind:name".
func GetK8sResourceIdentifier(doc interface{}, ignoreApiVersion bool) string {
	res, ok := asK8sResource(doc)
	if !ok {
		return ""
	}

	nameVal, _ := res.metaOM.Values["name"]
	if nameVal == nil {
		nameVal = res.metaOM.Values["generateName"]
	}
	name := fmt.Sprintf("%v", nameVal)

	if ignoreApiVersion {
		if namespace := res.metaOM.Values["namespace"]; namespace != nil {
			return fmt.Sprintf("%s:%v/%s", res.kind, namespace, name)
		}
		return fmt.Sprintf("%s:%s", res.kind, name)
	}

	if namespace := res.metaOM.Values["namespace"]; namespace != nil {
		return fmt.Sprintf("%s:%s:%v/%s", res.apiVersion, res.kind, namespace, name)
	}
	return fmt.Sprintf("%s:%s:%s", res.apiVersion, res.kind, name)
}

// GetIdentifierWithAdditional gets an identifier value from a map,
// checking default fields (name, id) and any additional specified fields.
func GetIdentifierWithAdditional(m map[string]interface{}, additionalIdentifiers []string) interface{} {
	return getIdentifierFromOrderedMap(toOrderedMap(m), additionalIdentifiers)
}

// CanMatchByIdentifierWithAdditional checks if list items can be matched by identifier,
// including additional identifier fields.
func CanMatchByIdentifierWithAdditional(list []interface{}, additionalIdentifiers []string) bool {
	if len(list) == 0 {
		return false
	}

	hasIdentifier := false
	for _, item := range list {
		om := toOrderedMap(item)
		if om == nil {
			// Not a map type, can't match by identifier
			return false
		}
		id := getIdentifierFromOrderedMap(om, additionalIdentifiers)
		if isComparableIdentifier(id) {
			hasIdentifier = true
		}
	}
	return hasIdentifier
}

// getIdentifierFromOrderedMap extracts identifier from OrderedMap
func getIdentifierFromOrderedMap(om *OrderedMap, additionalIdentifiers []string) interface{} {
	// Check additional identifiers first (they take priority)
	for _, field := range additionalIdentifiers {
		if val, ok := om.Values[field]; ok {
			return val
		}
	}

	// Fall back to default fields
	if name, ok := om.Values["name"]; ok {
		return name
	}
	if id, ok := om.Values["id"]; ok {
		return id
	}

	return nil
}

// isComparableIdentifier returns true when a value can be safely used as a Go map key.
func isComparableIdentifier(id interface{}) bool {
	if id == nil {
		return false
	}
	return reflect.TypeOf(id).Comparable()
}

// matchK8sDocuments matches Kubernetes documents from two slices by their identifiers.
// Returns a map from 'from' index to 'to' index, and lists of unmatched indices.
func matchK8sDocuments(from, to []interface{}, opts *Options) (matched map[int]int, unmatchedFrom, unmatchedTo []int) {
	matched = make(map[int]int)
	ignoreApiVersion := opts != nil && opts.IgnoreApiVersion

	// Build index of 'to' documents by K8s identifier
	toIndex := make(map[string]int)
	toMatched := make([]bool, len(to))

	for i, doc := range to {
		if id := GetK8sResourceIdentifier(doc, ignoreApiVersion); id != "" {
			if _, exists := toIndex[id]; !exists {
				toIndex[id] = i
			}
		}
	}

	// Match 'from' documents to 'to' documents
	for i, doc := range from {
		id := GetK8sResourceIdentifier(doc, ignoreApiVersion)
		if id != "" {
			if toIdx, ok := toIndex[id]; ok {
				matched[i] = toIdx
				toMatched[toIdx] = true
				continue
			}
		}
		unmatchedFrom = append(unmatchedFrom, i)
	}

	// Find unmatched 'to' documents
	for i := range to {
		if !toMatched[i] {
			unmatchedTo = append(unmatchedTo, i)
		}
	}

	return matched, unmatchedFrom, unmatchedTo
}

// compareK8sDocs compares Kubernetes documents matching by resource identifier.
// The compareFn callback is used for recursive node comparison, breaking the
// circular dependency between kubernetes.go and comparator.go.
func compareK8sDocs(from, to []interface{}, opts *Options, compareFn nodeComparerFn) []Difference {
	var diffs []Difference

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(from, to, opts)
	ignoreApiVersion := opts != nil && opts.IgnoreApiVersion

	// Detect document order changes
	if !opts.IgnoreOrderChanges && len(matched) >= 2 {
		// Build sorted (fromIdx, toIdx) pairs in one pass
		type idxPair struct{ fromIdx, toIdx int }
		pairs := make([]idxPair, 0, len(matched))
		for fromIdx, toIdx := range matched {
			pairs = append(pairs, idxPair{fromIdx, toIdx})
		}
		slices.SortFunc(pairs, func(a, b idxPair) int { return cmp.Compare(a.fromIdx, b.fromIdx) })

		// Check if toIdx values are monotonically increasing
		orderChanged := !slices.IsSortedFunc(pairs, func(a, b idxPair) int { return cmp.Compare(a.toIdx, b.toIdx) })

		if orderChanged {
			fromOrder := make([]interface{}, len(pairs))
			for i, p := range pairs {
				fromOrder[i] = GetK8sResourceIdentifier(from[p.fromIdx], ignoreApiVersion)
			}
			// Re-sort by toIdx for to-order
			slices.SortFunc(pairs, func(a, b idxPair) int { return cmp.Compare(a.toIdx, b.toIdx) })
			toOrder := make([]interface{}, len(pairs))
			for i, p := range pairs {
				toOrder[i] = GetK8sResourceIdentifier(from[p.fromIdx], ignoreApiVersion)
			}

			diffs = append(diffs, Difference{
				Path: k8sDocumentPath,
				Type: DiffOrderChanged,
				From: fromOrder,
				To:   toOrder,
			})
		}
	}

	// Compare matched documents
	for fromIdx, toIdx := range matched {
		fromDoc := from[fromIdx]
		toDoc := to[toIdx]

		// Build path prefix using 'from' index for consistency
		pathPrefix := ""
		if len(from) > 1 || len(to) > 1 {
			pathPrefix = fmt.Sprintf("[%d]", fromIdx)
		}

		nodeDiffs := compareFn(pathPrefix, fromDoc, toDoc, opts)
		// Set DocumentIndex for all differences in this document
		for i := range nodeDiffs {
			nodeDiffs[i].DocumentIndex = fromIdx
		}
		diffs = append(diffs, nodeDiffs...)
	}

	// Detect renames among unmatched documents
	renameMatched, remainingFrom, remainingTo := detectRenames(from, to, unmatchedFrom, unmatchedTo, opts)

	// Compare rename-matched pairs using "to" index for path context
	for fromIdx, toIdx := range renameMatched {
		fromDoc := from[fromIdx]
		toDoc := to[toIdx]

		pathPrefix := ""
		if len(from) > 1 || len(to) > 1 {
			pathPrefix = fmt.Sprintf("[%d]", toIdx)
		}

		nodeDiffs := compareFn(pathPrefix, fromDoc, toDoc, opts)
		for i := range nodeDiffs {
			nodeDiffs[i].DocumentIndex = toIdx
		}
		diffs = append(diffs, nodeDiffs...)
	}

	// Report removed documents (in 'from' but not matched or rename-matched in 'to')
	for _, fromIdx := range remainingFrom {
		if from[fromIdx] == nil {
			continue
		}
		pathPrefix := fmt.Sprintf("[%d]", fromIdx)
		diffs = append(diffs, Difference{
			Path:          cleanPath(pathPrefix),
			Type:          DiffRemoved,
			From:          from[fromIdx],
			To:            nil,
			DocumentIndex: fromIdx,
		})
	}

	// Report added documents (in 'to' but not matched or rename-matched from 'from')
	for _, toIdx := range remainingTo {
		if to[toIdx] == nil {
			continue
		}
		pathPrefix := fmt.Sprintf("[%d]", toIdx)
		diffs = append(diffs, Difference{
			Path:          cleanPath(pathPrefix),
			Type:          DiffAdded,
			From:          nil,
			To:            to[toIdx],
			DocumentIndex: toIdx,
		})
	}

	return diffs
}
