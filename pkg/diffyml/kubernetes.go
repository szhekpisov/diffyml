// kubernetes.go - Kubernetes resource detection and matching.
//
// Detects K8s resources by checking for apiVersion, kind, and metadata fields.
// Matches resources across documents using apiVersion + kind + metadata.name (or generateName).
// Key functions: IsKubernetesResource(), K8sResourceIdentifier().
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
func IsKubernetesResource(doc any) bool {
	// Get map values from either OrderedMap or regular map
	getVal := func(doc any, key string) (any, bool) {
		switch m := doc.(type) {
		case *OrderedMap:
			val, ok := m.Values[key]
			return val, ok
		case map[string]any:
			val, ok := m[key]
			return val, ok
		default:
			return nil, false
		}
	}

	// Check for apiVersion
	apiVersion, ok := getVal(doc, "apiVersion")
	if !ok {
		return false
	}
	if _, isStr := apiVersion.(string); !isStr {
		return false
	}

	// Check for kind
	kind, ok := getVal(doc, "kind")
	if !ok {
		return false
	}
	if _, isStr := kind.(string); !isStr {
		return false
	}

	// Check for metadata
	metadata, ok := getVal(doc, "metadata")
	if !ok {
		return false
	}

	// Check for name or generateName in metadata
	metaName, hasName := getVal(metadata, "name")
	metaGenName, hasGenName := getVal(metadata, "generateName")
	if (!hasName || metaName == nil) && (!hasGenName || metaGenName == nil) {
		return false
	}

	return true
}

// k8sGetVal extracts a value by key from an OrderedMap or map[string]any.
func k8sGetVal(m any, key string) any {
	switch v := m.(type) {
	case *OrderedMap:
		return v.Values[key]
	case map[string]any:
		return v[key]
	default:
		return nil
	}
}

// K8sResourceIdentifier returns a unique identifier for a Kubernetes resource.
// When ignoreApiVersion is false: "apiVersion:kind:namespace/name" or "apiVersion:kind:name".
// When ignoreApiVersion is true: "kind:namespace/name" or "kind:name".
func K8sResourceIdentifier(doc any, ignoreApiVersion bool) string {
	if !IsKubernetesResource(doc) {
		return ""
	}

	apiVersion, _ := k8sGetVal(doc, "apiVersion").(string) // safe: IsKubernetesResource() pre-validates these fields
	kind, _ := k8sGetVal(doc, "kind").(string)             // safe: IsKubernetesResource() pre-validates these fields
	metadata := k8sGetVal(doc, "metadata")
	nameVal := k8sGetVal(metadata, "name")
	if nameVal == nil {
		nameVal = k8sGetVal(metadata, "generateName")
	}
	name := fmt.Sprintf("%v", nameVal)

	if ignoreApiVersion {
		if namespace := k8sGetVal(metadata, "namespace"); namespace != nil {
			return fmt.Sprintf("%s:%v/%s", kind, namespace, name)
		}
		return fmt.Sprintf("%s:%s", kind, name)
	}

	if namespace := k8sGetVal(metadata, "namespace"); namespace != nil {
		return fmt.Sprintf("%s:%s:%v/%s", apiVersion, kind, namespace, name)
	}
	return fmt.Sprintf("%s:%s:%s", apiVersion, kind, name)
}

// IdentifierWithAdditional gets an identifier value from a map,
// checking default fields (name, id) and any additional specified fields.
func IdentifierWithAdditional(m map[string]any, additionalIdentifiers []string) any {
	// Check additional identifiers first (they take priority)
	for _, field := range additionalIdentifiers {
		if val, ok := m[field]; ok {
			return val
		}
	}

	// Fall back to default fields
	if name, ok := m["name"]; ok {
		return name
	}
	if id, ok := m["id"]; ok {
		return id
	}

	return nil
}

// CanMatchByIdentifierWithAdditional checks if list items can be matched by identifier,
// including additional identifier fields.
func CanMatchByIdentifierWithAdditional(list []any, additionalIdentifiers []string) bool {
	if len(list) == 0 {
		return false
	}

	hasIdentifier := false
	for _, item := range list {
		// Check for OrderedMap
		if om, ok := item.(*OrderedMap); ok {
			id := getIdentifierFromOrderedMap(om, additionalIdentifiers)
			if isComparableIdentifier(id) {
				hasIdentifier = true
			}
			continue
		}

		// Check for regular map
		m, ok := item.(map[string]any)
		if !ok {
			// Not a map, can't match by identifier
			return false
		}
		id := IdentifierWithAdditional(m, additionalIdentifiers)
		if isComparableIdentifier(id) {
			hasIdentifier = true
		}
	}
	return hasIdentifier
}

// getIdentifierFromOrderedMap extracts identifier from OrderedMap
func getIdentifierFromOrderedMap(om *OrderedMap, additionalIdentifiers []string) any {
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
func isComparableIdentifier(id any) bool {
	if id == nil {
		return false
	}
	return reflect.TypeOf(id).Comparable()
}

// matchK8sDocuments matches Kubernetes documents from two slices by their identifiers.
// Returns a map from 'from' index to 'to' index, and lists of unmatched indices.
func matchK8sDocuments(from, to []any, opts *Options) (matched map[int]int, unmatchedFrom, unmatchedTo []int) {
	matched = make(map[int]int)
	ignoreApiVersion := opts != nil && opts.IgnoreApiVersion

	// Build index of 'to' documents by K8s identifier
	toIndex := make(map[string]int)
	toMatched := make([]bool, len(to))

	for i, doc := range to {
		if id := K8sResourceIdentifier(doc, ignoreApiVersion); id != "" {
			if _, exists := toIndex[id]; !exists {
				toIndex[id] = i
			}
		}
	}

	// Match 'from' documents to 'to' documents
	for i, doc := range from {
		id := K8sResourceIdentifier(doc, ignoreApiVersion)
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
func compareK8sDocs(from, to []any, opts *Options) []Difference {
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
			fromOrder := make([]any, len(pairs))
			for i, p := range pairs {
				fromOrder[i] = K8sResourceIdentifier(from[p.fromIdx], ignoreApiVersion)
			}
			// Re-sort by toIdx for to-order
			slices.SortFunc(pairs, func(a, b idxPair) int { return cmp.Compare(a.toIdx, b.toIdx) })
			toOrder := make([]any, len(pairs))
			for i, p := range pairs {
				toOrder[i] = K8sResourceIdentifier(from[p.fromIdx], ignoreApiVersion)
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

		nodeDiffs := compareNodes(pathPrefix, fromDoc, toDoc, opts)
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

		nodeDiffs := compareNodes(pathPrefix, fromDoc, toDoc, opts)
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
