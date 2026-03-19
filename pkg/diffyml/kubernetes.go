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
var k8sDocumentPath = DiffPath{"(document)"}

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

// k8sResourceFields holds the extracted fields from a Kubernetes resource document.
type k8sResourceFields struct {
	apiVersion string
	kind       string
	name       string
	namespace  string // empty if cluster-scoped
}

// k8sExtractFields extracts common fields from a K8s resource document.
// Returns false if the document is not a valid K8s resource.
func k8sExtractFields(doc any) (k8sResourceFields, bool) {
	if !IsKubernetesResource(doc) {
		return k8sResourceFields{}, false
	}
	apiVersion, _ := k8sGetVal(doc, "apiVersion").(string)
	kind, _ := k8sGetVal(doc, "kind").(string)
	metadata := k8sGetVal(doc, "metadata")
	nameVal := k8sGetVal(metadata, "name")
	if nameVal == nil {
		nameVal = k8sGetVal(metadata, "generateName")
	}
	var ns string
	if nsVal := k8sGetVal(metadata, "namespace"); nsVal != nil {
		ns = fmt.Sprintf("%v", nsVal)
	}
	return k8sResourceFields{
		apiVersion: apiVersion,
		kind:       kind,
		name:       fmt.Sprintf("%v", nameVal),
		namespace:  ns,
	}, true
}

// K8sResourceIdentifier returns a unique identifier for a Kubernetes resource.
// When ignoreApiVersion is false: "apiVersion:kind:namespace/name" or "apiVersion:kind:name".
// When ignoreApiVersion is true: "kind:namespace/name" or "kind:name".
func K8sResourceIdentifier(doc any, ignoreApiVersion bool) string {
	f, ok := k8sExtractFields(doc)
	if !ok {
		return ""
	}
	if ignoreApiVersion {
		if f.namespace != "" {
			return fmt.Sprintf("%s:%s/%s", f.kind, f.namespace, f.name)
		}
		return fmt.Sprintf("%s:%s", f.kind, f.name)
	}
	if f.namespace != "" {
		return fmt.Sprintf("%s:%s:%s/%s", f.apiVersion, f.kind, f.namespace, f.name)
	}
	return fmt.Sprintf("%s:%s:%s", f.apiVersion, f.kind, f.name)
}

// K8sResourceDisplayName returns a slash-separated display name for a Kubernetes resource.
// Format: "apiVersion/kind/name" or "apiVersion/kind/namespace/name".
// Returns empty string if the document is not a valid K8s resource.
func K8sResourceDisplayName(doc any) string {
	f, ok := k8sExtractFields(doc)
	if !ok {
		return ""
	}
	if f.namespace != "" {
		return fmt.Sprintf("%s/%s/%s/%s", f.apiVersion, f.kind, f.namespace, f.name)
	}
	return fmt.Sprintf("%s/%s/%s", f.apiVersion, f.kind, f.name)
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

// detectK8sOrderChanges detects document order changes among matched K8s documents.
func detectK8sOrderChanges(matched map[int]int, from []any, ignoreApiVersion bool) *Difference {
	if len(matched) < 2 {
		return nil
	}

	type idxPair struct{ fromIdx, toIdx int }
	pairs := make([]idxPair, 0, len(matched))
	for fromIdx, toIdx := range matched {
		pairs = append(pairs, idxPair{fromIdx, toIdx})
	}
	slices.SortFunc(pairs, func(a, b idxPair) int { return cmp.Compare(a.fromIdx, b.fromIdx) })

	orderChanged := !slices.IsSortedFunc(pairs, func(a, b idxPair) int { return cmp.Compare(a.toIdx, b.toIdx) })

	if !orderChanged {
		return nil
	}

	fromOrder := make([]any, len(pairs))
	for i, p := range pairs {
		fromOrder[i] = K8sResourceIdentifier(from[p.fromIdx], ignoreApiVersion)
	}
	slices.SortFunc(pairs, func(a, b idxPair) int { return cmp.Compare(a.toIdx, b.toIdx) })
	toOrder := make([]any, len(pairs))
	for i, p := range pairs {
		toOrder[i] = K8sResourceIdentifier(from[p.fromIdx], ignoreApiVersion)
	}

	// DocumentName is intentionally empty: order-change diffs span the entire
	// document set, so no single resource name applies.
	return &Difference{
		Path: k8sDocumentPath,
		Type: DiffOrderChanged,
		From: fromOrder,
		To:   toOrder,
	}
}

// compareMatchedK8sDocs compares matched and rename-matched K8s document pairs.
func compareMatchedK8sDocs(matched map[int]int, from, to []any, opts *Options, useToIdx bool) []Difference {
	var diffs []Difference
	for fromIdx, toIdx := range matched {
		fromDoc := from[fromIdx]
		toDoc := to[toIdx]

		docIdx := fromIdx
		if useToIdx {
			docIdx = toIdx
		}

		var pathPrefix DiffPath
		if len(from) > 1 || len(to) > 1 {
			pathPrefix = DiffPath{fmt.Sprintf("[%d]", docIdx)}
		}

		nodeDiffs := compareNodes(pathPrefix, fromDoc, toDoc, opts)
		docName := K8sResourceDisplayName(toDoc)
		for i := range nodeDiffs {
			nodeDiffs[i].DocumentIndex = docIdx
			nodeDiffs[i].DocumentName = docName
		}
		diffs = append(diffs, nodeDiffs...)
	}
	return diffs
}

// compareK8sDocs compares Kubernetes documents matching by resource identifier.
func compareK8sDocs(from, to []any, opts *Options) []Difference {
	var diffs []Difference

	matched, unmatchedFrom, unmatchedTo := matchK8sDocuments(from, to, opts)
	ignoreApiVersion := opts != nil && opts.IgnoreApiVersion

	// Detect document order changes
	if !opts.IgnoreOrderChanges {
		if orderDiff := detectK8sOrderChanges(matched, from, ignoreApiVersion); orderDiff != nil {
			diffs = append(diffs, *orderDiff)
		}
	}

	// Compare matched documents
	diffs = append(diffs, compareMatchedK8sDocs(matched, from, to, opts, false)...)

	// Detect renames among unmatched documents
	renameMatched, remainingFrom, remainingTo := detectRenames(from, to, unmatchedFrom, unmatchedTo, opts)

	// Compare rename-matched pairs using "to" index for path context
	diffs = append(diffs, compareMatchedK8sDocs(renameMatched, from, to, opts, true)...)

	// Report removed documents
	for _, fromIdx := range remainingFrom {
		if from[fromIdx] == nil {
			continue
		}
		pathPrefix := DiffPath{fmt.Sprintf("[%d]", fromIdx)}
		diffs = append(diffs, Difference{
			Path:          pathPrefix,
			Type:          DiffRemoved,
			From:          from[fromIdx],
			To:            nil,
			DocumentIndex: fromIdx,
			DocumentName:  K8sResourceDisplayName(from[fromIdx]),
		})
	}

	// Report added documents
	for _, toIdx := range remainingTo {
		if to[toIdx] == nil {
			continue
		}
		pathPrefix := DiffPath{fmt.Sprintf("[%d]", toIdx)}
		diffs = append(diffs, Difference{
			Path:          pathPrefix,
			Type:          DiffAdded,
			From:          nil,
			To:            to[toIdx],
			DocumentIndex: toIdx,
			DocumentName:  K8sResourceDisplayName(to[toIdx]),
		})
	}

	return diffs
}
