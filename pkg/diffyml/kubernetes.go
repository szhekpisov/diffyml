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

	"go.yaml.in/yaml/v3"
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

	// apiVersion and kind must be present as string values. A single type-
	// assertion check covers both "missing key" (getVal returns nil) and
	// "wrong type" — the redundant existence check was dead defensive code.
	apiVersion, _ := getVal(doc, "apiVersion")
	if _, ok := apiVersion.(string); !ok {
		return false
	}
	kind, _ := getVal(doc, "kind")
	if _, ok := kind.(string); !ok {
		return false
	}

	// metadata may be any value — getVal(nil/non-map, …) returns (nil, false)
	// for name and generateName, so the guard below also catches missing
	// metadata.
	metadata, _ := getVal(doc, "metadata")
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
	return f.displayName()
}

// K8sResourceKind returns the "kind" field of a Kubernetes resource document.
// Returns empty string if the document is not a valid K8s resource (the
// zero-value k8sResourceFields has an empty kind, so no explicit guard).
func K8sResourceKind(doc any) string {
	f, _ := k8sExtractFields(doc)
	return f.kind
}

// displayName renders k8sResourceFields in the same format as K8sResourceDisplayName.
func (f k8sResourceFields) displayName() string {
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

// k8sDocs is the parallel ([]any) view of a []*yaml.Node slice, used by the
// K8s match/rename path so the per-doc nodeToInterface materialization happens
// at most once even when matchK8sDocuments and detectRenames both run.
func materializeK8sDocs(nodes []*yaml.Node) []any {
	docs := make([]any, len(nodes))
	for i, n := range nodes {
		docs[i] = nodeToInterface(n)
	}
	return docs
}

// matchK8sDocuments matches Kubernetes documents from two slices by their
// identifiers. Operates on parsed node trees; the parallel materialized any
// view is built once internally for K8sResourceIdentifier lookups.
func matchK8sDocuments(from, to []*yaml.Node, opts *Options) (matched map[int]int, unmatchedFrom, unmatchedTo []int) {
	fromDocs := materializeK8sDocs(from)
	toDocs := materializeK8sDocs(to)
	return matchK8sDocsValues(fromDocs, toDocs, opts)
}

// matchK8sDocsValues is the inner implementation used by both
// matchK8sDocuments and compareK8sDocs (which already has the cached any view).
func matchK8sDocsValues(fromDocs, toDocs []any, opts *Options) (matched map[int]int, unmatchedFrom, unmatchedTo []int) {
	matched = make(map[int]int)
	ignoreApiVersion := opts != nil && opts.IgnoreApiVersion

	toIndex := make(map[string]int)
	toMatched := make([]bool, len(toDocs))
	for i, doc := range toDocs {
		if id := K8sResourceIdentifier(doc, ignoreApiVersion); id != "" {
			if _, exists := toIndex[id]; !exists {
				toIndex[id] = i
			}
		}
	}

	for i, doc := range fromDocs {
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

	for i := range toDocs {
		if !toMatched[i] {
			unmatchedTo = append(unmatchedTo, i)
		}
	}

	return matched, unmatchedFrom, unmatchedTo
}

// detectK8sOrderChanges detects document order changes among matched K8s
// documents. Operates on the already-materialized fromDocs any view.
func detectK8sOrderChanges(matched map[int]int, from []any, ignoreApiVersion bool) *Difference {
	// No early-return for len(matched) < 2: the loop builds an empty/singleton
	// pairs slice, IsSortedFunc is trivially true, orderChanged stays false,
	// and the function returns nil naturally.
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

// compareMatchedK8sDocs compares matched (and rename-matched) K8s document
// pairs. Takes both the node slices (for descent into the comparator) and the
// cached any view (for K8s metadata extraction).
func compareMatchedK8sDocs(matched map[int]int, fromNodes, toNodes []*yaml.Node, fromDocs, toDocs []any, opts *Options, useToIdx bool) []Difference {
	var diffs []Difference
	for fromIdx, toIdx := range matched {
		fromN := fromNodes[fromIdx]
		toN := toNodes[toIdx]
		toDoc := toDocs[toIdx]

		docIdx := fromIdx
		if useToIdx {
			docIdx = toIdx
		}

		var pathPrefix DiffPath
		if len(fromNodes) > 1 || len(toNodes) > 1 {
			pathPrefix = DiffPath{fmt.Sprintf("[%d]", docIdx)}
		}

		nodeDiffs := compareNodes(pathPrefix, fromN, toN, opts)
		var docName, docKind string
		if f, ok := k8sExtractFields(toDoc); ok {
			docName = f.displayName()
			docKind = f.kind
		}
		for i := range nodeDiffs {
			nodeDiffs[i].DocumentIndex = docIdx
			nodeDiffs[i].DocumentName = docName
			nodeDiffs[i].DocumentKind = docKind
		}
		diffs = append(diffs, nodeDiffs...)
	}
	return diffs
}

// compareK8sDocs compares Kubernetes document node trees by matching them on
// their resource identifier (apiVersion + kind + namespace/name). The
// materialized any view is built once and shared across matching, order
// detection, rename detection, and add/remove emission.
func compareK8sDocs(fromNodes, toNodes []*yaml.Node, opts *Options) []Difference {
	var diffs []Difference
	fromDocs := materializeK8sDocs(fromNodes)
	toDocs := materializeK8sDocs(toNodes)

	matched, unmatchedFrom, unmatchedTo := matchK8sDocsValues(fromDocs, toDocs, opts)
	// opts is guaranteed non-nil by the caller (compareDocs gates on opts != nil),
	// and we dereference opts.IgnoreOrderChanges immediately below.
	ignoreApiVersion := opts.IgnoreApiVersion

	if !opts.IgnoreOrderChanges {
		if orderDiff := detectK8sOrderChanges(matched, fromDocs, ignoreApiVersion); orderDiff != nil {
			diffs = append(diffs, *orderDiff)
		}
	}

	diffs = append(diffs, compareMatchedK8sDocs(matched, fromNodes, toNodes, fromDocs, toDocs, opts, false)...)

	// Rename detection consumes the any view (similarity scoring serializes
	// each unmatched doc back to YAML bytes).
	renameMatched, remainingFrom, remainingTo := detectRenames(fromDocs, toDocs, unmatchedFrom, unmatchedTo, opts)

	diffs = append(diffs, compareMatchedK8sDocs(renameMatched, fromNodes, toNodes, fromDocs, toDocs, opts, true)...)

	for _, fromIdx := range remainingFrom {
		if fromDocs[fromIdx] == nil {
			continue
		}
		pathPrefix := DiffPath{fmt.Sprintf("[%d]", fromIdx)}
		var docName, docKind string
		if f, ok := k8sExtractFields(fromDocs[fromIdx]); ok {
			docName = f.displayName()
			docKind = f.kind
		}
		diffs = append(diffs, Difference{
			Path:          pathPrefix,
			Type:          DiffRemoved,
			From:          fromDocs[fromIdx],
			To:            nil,
			DocumentIndex: fromIdx,
			DocumentName:  docName,
			DocumentKind:  docKind,
		})
	}

	for _, toIdx := range remainingTo {
		if toDocs[toIdx] == nil {
			continue
		}
		pathPrefix := DiffPath{fmt.Sprintf("[%d]", toIdx)}
		var docName, docKind string
		if f, ok := k8sExtractFields(toDocs[toIdx]); ok {
			docName = f.displayName()
			docKind = f.kind
		}
		diffs = append(diffs, Difference{
			Path:          pathPrefix,
			Type:          DiffAdded,
			From:          nil,
			To:            toDocs[toIdx],
			DocumentIndex: toIdx,
			DocumentName:  docName,
			DocumentKind:  docKind,
		})
	}

	return diffs
}
