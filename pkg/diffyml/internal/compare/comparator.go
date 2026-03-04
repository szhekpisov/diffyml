// comparator.go - Core YAML comparison logic.
//
// Compares YAML documents at the AST level to detect semantic differences.
// Handles maps, lists (ordered and unordered), scalars, and multi-document files.
// Integrates with kubernetes.go for K8s resource matching.
package compare

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// NodeComparerFn is a callback for recursively comparing two YAML nodes.
// It is injected into CompareK8sDocs to break the comparator ↔ kubernetes dependency.
type NodeComparerFn func(path string, from, to interface{}, opts *types.Options) []types.Difference

// CompareDocs compares two slices of YAML documents and returns differences.
func CompareDocs(from, to []interface{}, opts *types.Options) []types.Difference {
	// Check if Kubernetes detection is enabled and documents are K8s resources
	if opts != nil && opts.DetectKubernetes && HasK8sDocuments(from, to) {
		return CompareK8sDocs(from, to, opts, CompareNodes)
	}

	var diffs []types.Difference

	// For simplicity, compare document by document
	// If document counts differ, that's handled by comparing indices
	maxLen := max(len(from), len(to))

	for i := range maxLen {
		var fromDoc, toDoc interface{}
		if i < len(from) {
			fromDoc = from[i]
		}
		if i < len(to) {
			toDoc = to[i]
		}

		// Build path prefix for multi-document files
		pathPrefix := ""
		if maxLen > 1 {
			pathPrefix = fmt.Sprintf("[%d]", i)
		}

		nodeDiffs := CompareNodes(pathPrefix, fromDoc, toDoc, opts)
		// Set DocumentIndex for all differences in this document
		for j := range nodeDiffs {
			nodeDiffs[j].DocumentIndex = i
		}
		diffs = append(diffs, nodeDiffs...)
	}

	return diffs
}

// HasK8sDocuments checks if any documents in either slice are Kubernetes resources.
func HasK8sDocuments(from, to []interface{}) bool {
	for _, doc := range from {
		if IsKubernetesResource(doc) {
			return true
		}
	}
	for _, doc := range to {
		if IsKubernetesResource(doc) {
			return true
		}
	}
	return false
}

// CompareNodes recursively compares two YAML nodes.
func CompareNodes(path string, from, to interface{}, opts *types.Options) []types.Difference {
	// Auto-convert map[string]interface{} to *types.OrderedMap so there is only one code path.
	if om := types.ToOrderedMap(from); om != nil {
		from = om
	}
	if om := types.ToOrderedMap(to); om != nil {
		to = om
	}

	var diffs []types.Difference

	// Handle nil cases
	if from == nil && to == nil {
		return diffs
	}

	if from == nil {
		// Value was added
		return []types.Difference{{
			Path: types.CleanPath(path),
			Type: types.DiffAdded,
			From: nil,
			To:   to,
		}}
	}

	if to == nil {
		// Value changed to null - treat as modification (key still exists)
		if opts != nil && opts.IgnoreValueChanges {
			return diffs
		}
		return []types.Difference{{
			Path: types.CleanPath(path),
			Type: types.DiffModified,
			From: from,
			To:   nil,
		}}
	}

	// Get types
	fromType := reflect.TypeOf(from)
	toType := reflect.TypeOf(to)

	// Type mismatch - treat as modification
	if fromType != toType {
		if opts != nil && opts.IgnoreValueChanges {
			return diffs
		}
		return []types.Difference{{
			Path: types.CleanPath(path),
			Type: types.DiffModified,
			From: from,
			To:   to,
		}}
	}

	// Compare based on type
	switch fromVal := from.(type) {
	case *types.OrderedMap:
		// Type-equality guard above ensures to is also *types.OrderedMap
		toOM := to.(*types.OrderedMap)
		diffs = append(diffs, CompareOrderedMaps(path, fromVal, toOM, opts)...)

	case []interface{}:
		toVal := to.([]interface{})
		diffs = append(diffs, CompareLists(path, fromVal, toVal, opts)...)

	default:
		// Scalar comparison
		if !EqualValues(from, to, opts) {
			if opts != nil && opts.IgnoreValueChanges {
				return diffs
			}
			diffs = append(diffs, types.Difference{
				Path: types.CleanPath(path),
				Type: types.DiffModified,
				From: from,
				To:   to,
			})
		}
	}

	return diffs
}

// CompareOrderedMaps compares two OrderedMap nodes preserving source document order.
func CompareOrderedMaps(path string, from, to *types.OrderedMap, opts *types.Options) []types.Difference {
	var diffs []types.Difference

	// Track which keys from 'to' have been seen
	toSeen := make(map[string]bool)

	// First iterate over 'from' keys in their original order
	for _, key := range from.Keys {
		toSeen[key] = true
		childPath := types.JoinPath(path, key)
		fromVal := from.Values[key]
		toVal, toOk := to.Values[key]

		if !toOk {
			// Key was removed
			diffs = append(diffs, types.Difference{
				Path: types.CleanPath(childPath),
				Type: types.DiffRemoved,
				From: fromVal,
				To:   nil,
			})
		} else {
			// Key exists in both - recurse
			diffs = append(diffs, CompareNodes(childPath, fromVal, toVal, opts)...)
		}
	}

	// Then iterate over 'to' keys to find additions (in their original order)
	for _, key := range to.Keys {
		if !toSeen[key] {
			childPath := types.JoinPath(path, key)
			diffs = append(diffs, types.Difference{
				Path: types.CleanPath(childPath),
				Type: types.DiffAdded,
				From: nil,
				To:   to.Values[key],
			})
		}
	}

	return diffs
}

// CompareLists compares two list nodes.
func CompareLists(path string, from, to []interface{}, opts *types.Options) []types.Difference {
	// Try to match items by identifier (for lists of maps with name/id fields)
	if CanMatchByIdentifier(from, opts) && CanMatchByIdentifier(to, opts) {
		return CompareListsByIdentifier(path, from, to, opts)
	}

	// If explicitly ignoring order, compare as sets
	if opts != nil && opts.IgnoreOrderChanges {
		return CompareListsUnordered(path, from, to, opts)
	}

	// For lists without identifiers:
	// - If items are structurally heterogeneous (different keys), use unordered
	// - Otherwise, use positional to get nested diffs
	if AreListItemsHeterogeneous(from, to) {
		return CompareListsUnordered(path, from, to, opts)
	}

	return CompareListsPositional(path, from, to, opts)
}

// AreListItemsHeterogeneous checks if list items have different structural keys.
// This helps decide whether to use unordered (for heterogeneous) or positional (for homogeneous) comparison.
func AreListItemsHeterogeneous(from, to []interface{}) bool {
	// Get keys from all items in both lists
	allKeys := make(map[string]bool)

	extractKeys := func(list []interface{}) {
		for _, item := range list {
			if v, ok := item.(*types.OrderedMap); ok {
				for _, key := range v.Keys {
					allKeys[key] = true
				}
			}
		}
	}

	extractKeys(from)
	extractKeys(to)

	// If we don't have map items, not heterogeneous
	if len(allKeys) == 0 {
		return false
	}

	// Check if each map item uses a distinct set of keys (only one key per item)
	// This is a heuristic: items with single, different keys are likely heterogeneous
	// (e.g., {namespaceSelector: ...} vs {ipBlock: ...})
	checkSingleDistinctKeys := func(list []interface{}) bool {
		for _, item := range list {
			v, ok := item.(*types.OrderedMap)
			if !ok {
				return false
			}
			if len(v.Keys) != 1 {
				return false
			}
		}
		return true
	}

	// If items have single keys and there are multiple different keys, it's heterogeneous
	if checkSingleDistinctKeys(from) && checkSingleDistinctKeys(to) && len(allKeys) > 1 {
		return true
	}

	return false
}

// areListsMaps checks if all items in a list are maps
// CompareListsPositional compares lists by position.
func CompareListsPositional(path string, from, to []interface{}, opts *types.Options) []types.Difference {
	var diffs []types.Difference

	minLen := min(len(from), len(to))

	// Compare elements present in both lists
	for i := range minLen {
		childPath := fmt.Sprintf("%s.%d", path, i)
		diffs = append(diffs, CompareNodes(childPath, from[i], to[i], opts)...)
	}
	// Items added (present in 'to' but not 'from')
	for i := minLen; i < len(to); i++ {
		childPath := fmt.Sprintf("%s.%d", path, i)
		diffs = append(diffs, types.Difference{
			Path: types.CleanPath(childPath),
			Type: types.DiffAdded,
			From: nil,
			To:   to[i],
		})
	}
	// Items removed (present in 'from' but not 'to')
	for i := minLen; i < len(from); i++ {
		childPath := fmt.Sprintf("%s.%d", path, i)
		diffs = append(diffs, types.Difference{
			Path: types.CleanPath(childPath),
			Type: types.DiffRemoved,
			From: from[i],
			To:   nil,
		})
	}

	return diffs
}

// CompareListsUnordered compares lists ignoring order.
func CompareListsUnordered(path string, from, to []interface{}, opts *types.Options) []types.Difference {
	var diffs []types.Difference

	// Track which items have been matched
	toMatched := make([]bool, len(to))

	for i, fromItem := range from {
		found := false
		for j, toItem := range to {
			if toMatched[j] {
				continue
			}
			if DeepEqual(fromItem, toItem, opts) {
				toMatched[j] = true
				found = true
				break
			}
		}
		if !found {
			// Item was removed
			diffs = append(diffs, types.Difference{
				Path: fmt.Sprintf("%s.%d", types.CleanPath(path), i),
				Type: types.DiffRemoved,
				From: fromItem,
				To:   nil,
			})
		}
	}

	// Check for added items
	for j, toItem := range to {
		if !toMatched[j] {
			diffs = append(diffs, types.Difference{
				Path: fmt.Sprintf("%s.%d", types.CleanPath(path), j),
				Type: types.DiffAdded,
				From: nil,
				To:   toItem,
			})
		}
	}

	return diffs
}

// CanMatchByIdentifier checks if list items can be matched by identifier.
// Returns true only if all items are maps and at least one has a "name" or "id" field.
func CanMatchByIdentifier(list []interface{}, opts *types.Options) bool {
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	return CanMatchByIdentifierWithAdditional(list, additional)
}

// GetIdentifier gets the identifier value from an OrderedMap.
func GetIdentifier(val interface{}, opts *types.Options) interface{} {
	om, ok := val.(*types.OrderedMap)
	if !ok {
		return nil
	}
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	return GetIdentifierFromOrderedMap(om, additional)
}

// CompareListsByIdentifier compares lists matching by identifier field.
func CompareListsByIdentifier(path string, from, to []interface{}, opts *types.Options) []types.Difference {
	var diffs []types.Difference

	// Build index of from items by identifier, preserving order
	fromIndex := make(map[interface{}]int)
	fromIDs := make([]interface{}, 0)
	fromNoID := make([]int, 0)
	for i, item := range from {
		id := GetIdentifier(item, opts)
		if IsComparableIdentifier(id) {
			fromIndex[id] = i
			fromIDs = append(fromIDs, id)
			continue
		}
		fromNoID = append(fromNoID, i)
	}

	// Build index of to items by identifier.
	toIndex := make(map[interface{}]int)
	toNoID := make([]int, 0)
	toIDCount := 0
	for i, item := range to {
		id := GetIdentifier(item, opts)
		if IsComparableIdentifier(id) {
			toIDCount++
			toIndex[id] = i
			continue
		}
		toNoID = append(toNoID, i)
	}

	// Detect order changes among matched identifiers.
	// Only when identifiers are unique in both lists (duplicates make order comparison meaningless).
	hasUniqueIDs := len(fromIDs) == len(fromIndex) && len(toIndex) == toIDCount
	if hasUniqueIDs && (opts == nil || !opts.IgnoreOrderChanges) {
		// Collect identifiers that exist in both, in from-order
		var commonFromOrder []interface{}
		for _, id := range fromIDs {
			if _, ok := toIndex[id]; ok {
				commonFromOrder = append(commonFromOrder, id)
			}
		}

		if len(commonFromOrder) >= 2 {
			// Build to-order for the common identifiers
			// Collect (toIdx, id) pairs for common IDs, then sort by toIdx
			type idxID struct {
				idx int
				id  interface{}
			}
			var toSorted []idxID
			for _, id := range commonFromOrder {
				toSorted = append(toSorted, idxID{toIndex[id], id})
			}
			slices.SortFunc(toSorted, func(a, b idxID) int {
				return cmp.Compare(a.idx, b.idx)
			})

			// Check if from-order and to-order differ
			orderChanged := false
			for i, id := range commonFromOrder {
				if id != toSorted[i].id {
					orderChanged = true
					break
				}
			}

			if orderChanged {
				toOrder := make([]interface{}, len(toSorted))
				for i, s := range toSorted {
					toOrder[i] = s.id
				}
				diffs = append(diffs, types.Difference{
					Path: types.CleanPath(path),
					Type: types.DiffOrderChanged,
					From: commonFromOrder,
					To:   toOrder,
				})
			}
		}
	}

	// Compare items with matching identifiers (in order they appear in 'from')
	for _, id := range fromIDs {
		fromIdx := fromIndex[id]
		fromItem := from[fromIdx]
		if toIdx, ok := toIndex[id]; ok {
			toItem := to[toIdx]
			// Items match by identifier - compare their contents
			// Use identifier value in path instead of index (dyff-style)
			idStr := fmt.Sprintf("%v", id)
			childPath := fmt.Sprintf("%s.%s", path, idStr)
			diffs = append(diffs, CompareNodes(childPath, fromItem, toItem, opts)...)
		} else {
			// Item was removed - report at the list level (dyff-style)
			diffs = append(diffs, types.Difference{
				Path: types.CleanPath(path),
				Type: types.DiffRemoved,
				From: fromItem,
				To:   nil,
			})
		}
	}

	// Check for added items (in to but not in from) - in order they appear in 'to'
	for _, toItem := range to {
		id := GetIdentifier(toItem, opts)
		if IsComparableIdentifier(id) {
			if _, inFrom := fromIndex[id]; !inFrom {
				// Item was added - report at the list level (dyff-style)
				diffs = append(diffs, types.Difference{
					Path: types.CleanPath(path),
					Type: types.DiffAdded,
					From: nil,
					To:   toItem,
				})
			}
		}
	}

	// Fallback for entries that do not have usable identifiers:
	// compare them as unordered values so removals/additions are still reported.
	toNoIDMatched := make([]bool, len(toNoID))
	for _, fromIdx := range fromNoID {
		fromItem := from[fromIdx]
		found := false
		for j, toIdx := range toNoID {
			if toNoIDMatched[j] {
				continue
			}
			if DeepEqual(fromItem, to[toIdx], opts) {
				toNoIDMatched[j] = true
				found = true
				break
			}
		}
		if !found {
			diffs = append(diffs, types.Difference{
				Path: fmt.Sprintf("%s.%d", types.CleanPath(path), fromIdx),
				Type: types.DiffRemoved,
				From: fromItem,
				To:   nil,
			})
		}
	}
	for j, toIdx := range toNoID {
		if toNoIDMatched[j] {
			continue
		}
		diffs = append(diffs, types.Difference{
			Path: fmt.Sprintf("%s.%d", types.CleanPath(path), toIdx),
			Type: types.DiffAdded,
			From: nil,
			To:   to[toIdx],
		})
	}

	return diffs
}

// EqualValues compares two scalar values for equality.
func EqualValues(from, to interface{}, opts *types.Options) bool {
	// Handle whitespace comparison
	if opts != nil && opts.IgnoreWhitespaceChanges {
		if fromStr, ok := from.(string); ok {
			if toStr, ok := to.(string); ok {
				return strings.TrimSpace(fromStr) == strings.TrimSpace(toStr)
			}
		}
	}

	return reflect.DeepEqual(from, to)
}

// DeepEqual compares two values deeply with options.
func DeepEqual(from, to interface{}, opts *types.Options) bool {
	// Auto-convert map[string]interface{} to *types.OrderedMap.
	if om := types.ToOrderedMap(from); om != nil {
		from = om
	}
	if om := types.ToOrderedMap(to); om != nil {
		to = om
	}

	// Handle nil
	if from == nil && to == nil {
		return true
	}
	if from == nil || to == nil {
		return false
	}

	// Check types
	if reflect.TypeOf(from) != reflect.TypeOf(to) {
		return false
	}

	switch fromVal := from.(type) {
	case *types.OrderedMap:
		// Type-equality guard above ensures to is also *types.OrderedMap
		toOM := to.(*types.OrderedMap)
		if len(fromVal.Values) != len(toOM.Values) {
			return false
		}
		for k, fv := range fromVal.Values {
			tv, ok := toOM.Values[k]
			if !ok || !DeepEqual(fv, tv, opts) {
				return false
			}
		}
		return true

	case []interface{}:
		toVal := to.([]interface{})
		if len(fromVal) != len(toVal) {
			return false
		}
		for i := range fromVal {
			if !DeepEqual(fromVal[i], toVal[i], opts) {
				return false
			}
		}
		return true

	default:
		return EqualValues(from, to, opts)
	}
}
