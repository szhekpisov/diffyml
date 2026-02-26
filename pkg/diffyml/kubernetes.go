// kubernetes.go - Kubernetes resource detection and matching.
//
// Detects K8s resources by checking for apiVersion, kind, and metadata fields.
// Matches resources across documents using apiVersion + kind + metadata.name (or generateName).
// Key functions: IsKubernetesResource(), GetK8sIdentifier().
package diffyml

import (
	"fmt"
	"reflect"
)

// IsKubernetesResource checks if a document has the structure of a Kubernetes resource.
// A Kubernetes resource must have apiVersion, kind, and metadata fields,
// where metadata is a map containing at least a name field.
func IsKubernetesResource(doc interface{}) bool {
	// Get map values from either OrderedMap or regular map
	getVal := func(doc interface{}, key string) (interface{}, bool) {
		switch m := doc.(type) {
		case *OrderedMap:
			val, ok := m.Values[key]
			return val, ok
		case map[string]interface{}:
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

// GetK8sResourceIdentifier returns a unique identifier for a Kubernetes resource.
// Format: "apiVersion:kind:namespace/name" or "apiVersion:kind:name" if no namespace.
func GetK8sResourceIdentifier(doc interface{}) string {
	if !IsKubernetesResource(doc) {
		return ""
	}

	// Helper to get value from either OrderedMap or regular map
	getVal := func(doc interface{}, key string) interface{} {
		switch m := doc.(type) {
		case *OrderedMap:
			return m.Values[key]
		case map[string]interface{}:
			return m[key]
		default:
			return nil
		}
	}

	apiVersion := getVal(doc, "apiVersion").(string)
	kind := getVal(doc, "kind").(string)
	metadata := getVal(doc, "metadata")
	nameVal := getVal(metadata, "name")
	if nameVal == nil {
		nameVal = getVal(metadata, "generateName")
	}
	name := fmt.Sprintf("%v", nameVal)

	if namespace := getVal(metadata, "namespace"); namespace != nil {
		return fmt.Sprintf("%s:%s:%v/%s", apiVersion, kind, namespace, name)
	}
	return fmt.Sprintf("%s:%s:%s", apiVersion, kind, name)
}

// GetIdentifierWithAdditional gets an identifier value from a map,
// checking default fields (name, id) and any additional specified fields.
func GetIdentifierWithAdditional(m map[string]interface{}, additionalIdentifiers []string) interface{} {
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
func CanMatchByIdentifierWithAdditional(list []interface{}, additionalIdentifiers []string) bool {
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
		m, ok := item.(map[string]interface{})
		if !ok {
			// Not a map, can't match by identifier
			return false
		}
		id := GetIdentifierWithAdditional(m, additionalIdentifiers)
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
func matchK8sDocuments(from, to []interface{}) (matched map[int]int, unmatchedFrom, unmatchedTo []int) {
	matched = make(map[int]int)

	// Build index of 'to' documents by K8s identifier
	toIndex := make(map[string]int)
	toMatched := make([]bool, len(to))

	for i, doc := range to {
		if id := GetK8sResourceIdentifier(doc); id != "" {
			toIndex[id] = i
		}
	}

	// Match 'from' documents to 'to' documents
	for i, doc := range from {
		id := GetK8sResourceIdentifier(doc)
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
func compareK8sDocs(from, to []interface{}, opts *Options) []Difference {
	var diffs []Difference

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(from, to)

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

	// Report removed documents (in 'from' but not matched in 'to')
	for _, fromIdx := range unmatchedFrom {
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

	// Report added documents (in 'to' but not matched from 'from')
	for _, toIdx := range unmatchedTo {
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
