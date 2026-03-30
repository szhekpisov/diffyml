// comparator.go - Core YAML comparison logic.
//
// Compares YAML documents at the AST level to detect semantic differences.
// Handles maps, lists (ordered and unordered), scalars, and multi-document files.
// Integrates with kubernetes.go for K8s resource matching.
package diffyml

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// compareDocs compares two slices of YAML documents and returns differences.
func compareDocs(from, to []any, opts *Options) []Difference {
	// Check if Kubernetes detection is enabled and documents are K8s resources
	if opts != nil && opts.DetectKubernetes && hasK8sDocuments(from, to) {
		return compareK8sDocs(from, to, opts)
	}

	var diffs []Difference

	// For simplicity, compare document by document
	// If document counts differ, that's handled by comparing indices
	maxLen := max(len(from), len(to))

	for i := range maxLen {
		var fromDoc, toDoc any
		if i < len(from) {
			fromDoc = from[i]
		}
		if i < len(to) {
			toDoc = to[i]
		}

		// Build path prefix for multi-document files
		var pathPrefix DiffPath
		if maxLen > 1 {
			pathPrefix = DiffPath{"[" + strconv.Itoa(i) + "]"}
		}

		nodeDiffs := compareNodes(pathPrefix, fromDoc, toDoc, opts)
		// Set DocumentIndex for all differences in this document
		for j := range nodeDiffs {
			nodeDiffs[j].DocumentIndex = i
		}
		diffs = append(diffs, nodeDiffs...)
	}

	return diffs
}

// hasK8sDocuments checks if any documents in either slice are Kubernetes resources.
func hasK8sDocuments(from, to []any) bool {
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

// compareNodeNils handles nil cases for compareNodes.
// Returns diffs and true if an early return is appropriate.
func compareNodeNils(path DiffPath, from, to any, opts *Options) ([]Difference, bool) {
	if from == nil && to == nil {
		return nil, true
	}
	if from == nil {
		return []Difference{{
			Path: path,
			Type: DiffAdded,
			From: nil,
			To:   to,
		}}, true
	}
	if to == nil {
		if opts != nil && opts.IgnoreValueChanges {
			return nil, true
		}
		return []Difference{{
			Path: path,
			Type: DiffModified,
			From: from,
			To:   nil,
		}}, true
	}
	return nil, false
}

// compareNodes recursively compares two YAML nodes.
func compareNodes(path DiffPath, from, to any, opts *Options) []Difference {
	if diffs, done := compareNodeNils(path, from, to, opts); done {
		return diffs
	}

	// Compare based on type — handle type mismatches inline without reflect
	switch fromVal := from.(type) {
	case *OrderedMap:
		if toVal, ok := to.(*OrderedMap); ok {
			return compareOrderedMaps(path, fromVal, toVal, opts)
		}
	case map[string]any:
		if toVal, ok := to.(map[string]any); ok {
			return compareMaps(path, fromVal, toVal, opts)
		}
	case []any:
		if toVal, ok := to.([]any); ok {
			return compareLists(path, fromVal, toVal, opts)
		}
	default:
		// Both are scalars — check if same concrete type
		if sameScalarType(from, to) {
			if !equalValues(from, to, opts) {
				if opts != nil && opts.IgnoreValueChanges {
					return nil
				}
				return []Difference{{
					Path: path,
					Type: DiffModified,
					From: from,
					To:   to,
				}}
			}
			return nil
		}
	}

	// Type mismatch
	if opts != nil && opts.IgnoreValueChanges {
		return nil
	}
	return []Difference{{
		Path: path,
		Type: DiffModified,
		From: from,
		To:   to,
	}}
}

// sameScalarType returns true if both values have the same concrete type
// without using reflect. Covers all types produced by YAML parsing.
func sameScalarType(a, b any) bool {
	switch a.(type) {
	case string:
		_, ok := b.(string)
		return ok
	case int:
		_, ok := b.(int)
		return ok
	case float64:
		_, ok := b.(float64)
		return ok
	case bool:
		_, ok := b.(bool)
		return ok
	default:
		// Fallback for rare types (int64, uint64, time.Time, etc.
		// produced by the yaml.v3 decoder for values outside common ranges).
		return fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b)
	}
}

// compareOrderedMaps compares two OrderedMap nodes preserving source document order
func compareOrderedMaps(path DiffPath, from, to *OrderedMap, opts *Options) []Difference {
	var diffs []Difference

	// First iterate over 'from' keys in their original order
	for _, key := range from.Keys {
		fromVal := from.Values[key]
		toVal, toOk := to.Values[key]

		if !toOk {
			// Key was removed — report at parent path with key-value wrapped in OrderedMap
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffRemoved,
				From: &OrderedMap{Keys: []string{key}, Values: map[string]any{key: fromVal}},
				To:   nil,
			})
		} else {
			// Key exists in both - recurse
			childPath := path.Append(key)
			diffs = append(diffs, compareNodes(childPath, fromVal, toVal, opts)...)
		}
	}

	// Then iterate over 'to' keys to find additions (in their original order)
	// Use from.Values for lookup instead of a separate tracking map
	for _, key := range to.Keys {
		if _, inFrom := from.Values[key]; !inFrom {
			// Key was added — report at parent path with key-value wrapped in OrderedMap
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffAdded,
				From: nil,
				To:   &OrderedMap{Keys: []string{key}, Values: map[string]any{key: to.Values[key]}},
			})
		}
	}

	return diffs
}

// compareMaps compares two map nodes.
func compareMaps(path DiffPath, from, to map[string]any, opts *Options) []Difference {
	var diffs []Difference

	// Get all keys from both maps and sort for deterministic output
	allKeys := make(map[string]bool)
	for k := range from {
		allKeys[k] = true
	}
	for k := range to {
		allKeys[k] = true
	}

	// Sort keys for deterministic output
	sortedKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// Iterate over sorted keys
	for _, key := range sortedKeys {
		fromVal, fromOk := from[key]
		toVal, toOk := to[key]

		switch {
		case !fromOk:
			// Key was added — report at parent path
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffAdded,
				From: nil,
				To:   &OrderedMap{Keys: []string{key}, Values: map[string]any{key: toVal}},
			})
		case !toOk:
			// Key was removed — report at parent path
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffRemoved,
				From: &OrderedMap{Keys: []string{key}, Values: map[string]any{key: fromVal}},
				To:   nil,
			})
		default:
			// Key exists in both - recurse
			childPath := path.Append(key)
			diffs = append(diffs, compareNodes(childPath, fromVal, toVal, opts)...)
		}
	}

	return diffs
}

// compareLists compares two list nodes.
func compareLists(path DiffPath, from, to []any, opts *Options) []Difference {
	// Try to match items by identifier (for lists of maps with name/id fields)
	if canMatchByIdentifier(from, opts) && canMatchByIdentifier(to, opts) {
		return compareListsByIdentifier(path, from, to, opts)
	}

	// If explicitly ignoring order, compare as sets
	if opts != nil && opts.IgnoreOrderChanges {
		return compareListsUnordered(path, from, to, opts)
	}

	// For lists without identifiers:
	// - If items are structurally heterogeneous (different keys), use unordered
	// - Otherwise, use positional to get nested diffs
	if areListItemsHeterogeneous(from, to) {
		return compareListsUnordered(path, from, to, opts)
	}

	return compareListsPositional(path, from, to, opts)
}

// areListItemsHeterogeneous checks if list items have different structural keys.
// This helps decide whether to use unordered (for heterogeneous) or positional (for homogeneous) comparison.
func areListItemsHeterogeneous(from, to []any) bool {
	// Get keys from all items in both lists
	allKeys := make(map[string]bool)

	extractKeys := func(list []any) {
		for _, item := range list {
			switch v := item.(type) {
			case *OrderedMap:
				for _, key := range v.Keys {
					allKeys[key] = true
				}
			case map[string]any:
				for key := range v {
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
	checkSingleDistinctKeys := func(list []any) bool {
		for _, item := range list {
			switch v := item.(type) {
			case *OrderedMap:
				if len(v.Keys) != 1 {
					return false
				}
			case map[string]any:
				if len(v) != 1 {
					return false
				}
			default:
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
// compareListsPositional compares lists by position.
func compareListsPositional(path DiffPath, from, to []any, opts *Options) []Difference {
	var diffs []Difference

	minLen := min(len(from), len(to))

	// Compare elements present in both lists
	for i := range minLen {
		childPath := path.Append(strconv.Itoa(i))
		diffs = append(diffs, compareNodes(childPath, from[i], to[i], opts)...)
	}
	// Items added (present in 'to' but not 'from')
	for i := minLen; i < len(to); i++ {
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(i)),
			Type: DiffAdded,
			From: nil,
			To:   to[i],
		})
	}
	// Items removed (present in 'from' but not 'to')
	for i := minLen; i < len(from); i++ {
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(i)),
			Type: DiffRemoved,
			From: from[i],
			To:   nil,
		})
	}

	return diffs
}

// compareListsUnordered compares lists ignoring order.
// Exact matches (via deepEqual) are removed first regardless of position,
// then remaining items are compared positionally via compareNodes to produce
// precise nested diffs instead of coarse remove+add.
func compareListsUnordered(path DiffPath, from, to []any, opts *Options) []Difference {
	fromMatched := make([]bool, len(from))
	toMatched := make([]bool, len(to))

	for i, fromItem := range from {
		for j, toItem := range to {
			if toMatched[j] {
				continue
			}
			if deepEqual(fromItem, toItem, opts) {
				fromMatched[i] = true
				toMatched[j] = true
				break
			}
		}
	}

	// Walk both lists with cursors, skipping matched items.
	// Unmatched items are paired positionally (from-index used for diff paths).
	var diffs []Difference
	fi, tj := 0, 0
	for fi < len(from) && tj < len(to) {
		if fromMatched[fi] {
			fi++
			continue
		}
		if toMatched[tj] {
			tj++
			continue
		}
		diffs = append(diffs, compareNodes(path.Append(strconv.Itoa(fi)), from[fi], to[tj], opts)...)
		fi++
		tj++
	}
	for ; fi < len(from); fi++ {
		if fromMatched[fi] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(fi)),
			Type: DiffRemoved,
			From: from[fi],
		})
	}
	for ; tj < len(to); tj++ {
		if toMatched[tj] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(tj)),
			Type: DiffAdded,
			To:   to[tj],
		})
	}

	return diffs
}

// canMatchByIdentifier checks if list items can be matched by identifier.
// Returns true only if all items are maps and at least one has a "name" or "id" field.
func canMatchByIdentifier(list []any, opts *Options) bool {
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	return CanMatchByIdentifierWithAdditional(list, additional)
}

// getIdentifier gets the identifier value from a map or OrderedMap.
func getIdentifier(val any, opts *Options) any {
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	if om, ok := val.(*OrderedMap); ok {
		return getIdentifierFromOrderedMap(om, additional)
	}
	if m, ok := val.(map[string]any); ok {
		return IdentifierWithAdditional(m, additional)
	}
	return nil
}

// detectListOrderChanges detects order changes among matched list identifiers.
func detectListOrderChanges(path DiffPath, fromIDs []any, fromIndex, toIndex map[any]int, toIDCount int) *Difference {
	hasUniqueIDs := len(fromIDs) == len(fromIndex) && len(toIndex) == toIDCount

	if !hasUniqueIDs {
		return nil
	}

	// Collect identifiers that exist in both, in from-order
	var commonFromOrder []any
	for _, id := range fromIDs {
		if _, ok := toIndex[id]; ok {
			commonFromOrder = append(commonFromOrder, id)
		}
	}

	if len(commonFromOrder) < 2 {
		return nil
	}

	// Build to-order for the common identifiers
	type idxID struct {
		idx int
		id  any
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

	if !orderChanged {
		return nil
	}

	toOrder := make([]any, len(toSorted))
	for i, s := range toSorted {
		toOrder[i] = s.id
	}
	return &Difference{
		Path: path,
		Type: DiffOrderChanged,
		From: commonFromOrder,
		To:   toOrder,
	}
}

// compareUnidentifiedItems compares list items that lack usable identifiers.
// Same strategy as compareListsUnordered: exact-match first, then positional remainder.
func compareUnidentifiedItems(path DiffPath, from, to []any, fromNoID, toNoID []int, opts *Options) []Difference {
	fromNoIDMatched := make([]bool, len(fromNoID))
	toNoIDMatched := make([]bool, len(toNoID))
	for fi, fromIdx := range fromNoID {
		for tj, toIdx := range toNoID {
			if toNoIDMatched[tj] {
				continue
			}
			if deepEqual(from[fromIdx], to[toIdx], opts) {
				fromNoIDMatched[fi] = true
				toNoIDMatched[tj] = true
				break
			}
		}
	}

	var diffs []Difference
	fi, tj := 0, 0
	for fi < len(fromNoID) && tj < len(toNoID) {
		if fromNoIDMatched[fi] {
			fi++
			continue
		}
		if toNoIDMatched[tj] {
			tj++
			continue
		}
		diffs = append(diffs, compareNodes(path.Append(strconv.Itoa(fromNoID[fi])), from[fromNoID[fi]], to[toNoID[tj]], opts)...)
		fi++
		tj++
	}
	for ; fi < len(fromNoID); fi++ {
		if fromNoIDMatched[fi] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(fromNoID[fi])),
			Type: DiffRemoved,
			From: from[fromNoID[fi]],
		})
	}
	for ; tj < len(toNoID); tj++ {
		if toNoIDMatched[tj] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(toNoID[tj])),
			Type: DiffAdded,
			To:   to[toNoID[tj]],
		})
	}
	return diffs
}

// compareListsByIdentifier compares lists matching by identifier field.
func compareListsByIdentifier(path DiffPath, from, to []any, opts *Options) []Difference {
	var diffs []Difference

	// Build index of from items by identifier, preserving order
	fromIndex := make(map[any]int, len(from))
	fromIDs := make([]any, 0, len(from))
	var fromNoID []int
	for i, item := range from {
		id := getIdentifier(item, opts)
		if isComparableIdentifier(id) {
			fromIndex[id] = i
			fromIDs = append(fromIDs, id)
			continue
		}
		fromNoID = append(fromNoID, i)
	}

	// Build index of to items by identifier.
	toIndex := make(map[any]int, len(to))
	var toNoID []int
	toIDCount := 0
	for i, item := range to {
		id := getIdentifier(item, opts)
		if isComparableIdentifier(id) {
			toIDCount++
			toIndex[id] = i
			continue
		}
		toNoID = append(toNoID, i)
	}

	// Detect order changes among matched identifiers.
	if opts == nil || !opts.IgnoreOrderChanges {
		if orderDiff := detectListOrderChanges(path, fromIDs, fromIndex, toIndex, toIDCount); orderDiff != nil {
			diffs = append(diffs, *orderDiff)
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
			idStr := sprintIdentifier(id)
			childPath := path.Append(idStr)
			diffs = append(diffs, compareNodes(childPath, fromItem, toItem, opts)...)
		} else {
			// Item was removed - report at the list level (dyff-style)
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffRemoved,
				From: fromItem,
				To:   nil,
			})
		}
	}

	// Check for added items (in to but not in from) - in order they appear in 'to'
	for _, toItem := range to {
		id := getIdentifier(toItem, opts)
		if isComparableIdentifier(id) {
			if _, inFrom := fromIndex[id]; !inFrom {
				// Item was added - report at the list level (dyff-style)
				diffs = append(diffs, Difference{
					Path: path,
					Type: DiffAdded,
					From: nil,
					To:   toItem,
				})
			}
		}
	}

	// Fallback for entries that do not have usable identifiers
	diffs = append(diffs, compareUnidentifiedItems(path, from, to, fromNoID, toNoID, opts)...)

	return diffs
}

// equalValues compares two scalar values for equality.
func equalValues(from, to any, opts *Options) bool {
	if opts != nil {
		if fromStr, ok := from.(string); ok {
			if toStr, ok := to.(string); ok {
				if fromStr == toStr {
					return true
				}

				if opts.FormatStrings && couldBeJSON(fromStr) && couldBeJSON(toStr) {
					if equal, matched := jsonCanonicalEqual(fromStr, toStr); matched {
						return equal
					}
				}

				if opts.IgnoreWhitespaceChanges {
					return strings.TrimSpace(fromStr) == strings.TrimSpace(toStr)
				}

				return false
			}
		}
	}

	return from == to
}

// couldBeJSON returns true if s starts with '{' or '[', indicating it might be
// a JSON object or array. Used as a cheap pre-check to avoid expensive
// json.Unmarshal calls on strings that are clearly not JSON.
func couldBeJSON(s string) bool {
	return len(s) >= 2 && (s[0] == '{' || s[0] == '[')
}

// jsonCanonicalEqual attempts to parse both strings as JSON and compares
// their canonical forms. Returns (equal, true) if both parsed as JSON,
// or (false, false) if either is not valid JSON.
func jsonCanonicalEqual(a, b string) (bool, bool) {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false, false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false, false
	}
	// json.Marshal cannot fail on values produced by json.Unmarshal.
	ca, _ := json.Marshal(va)
	cb, _ := json.Marshal(vb)
	return bytes.Equal(ca, cb), true
}

// deepEqualOrderedMaps checks deep equality between two OrderedMaps.
func deepEqualOrderedMaps(from, to *OrderedMap, opts *Options) bool {
	if len(from.Values) != len(to.Values) {
		return false
	}
	for k, fv := range from.Values {
		tv, ok := to.Values[k]
		if !ok || !deepEqual(fv, tv, opts) {
			return false
		}
	}
	return true
}

// deepEqualMaps checks deep equality between two maps.
func deepEqualMaps(from, to map[string]any, opts *Options) bool {
	if len(from) != len(to) {
		return false
	}
	for k, fv := range from {
		tv, ok := to[k]
		if !ok || !deepEqual(fv, tv, opts) {
			return false
		}
	}
	return true
}

// deepEqualSlices checks deep equality between two slices.
func deepEqualSlices(from, to []any, opts *Options) bool {
	if len(from) != len(to) {
		return false
	}
	for i := range from {
		if !deepEqual(from[i], to[i], opts) {
			return false
		}
	}
	return true
}

// deepEqual compares two values deeply with options.
func deepEqual(from, to any, opts *Options) bool {
	if from == nil && to == nil {
		return true
	}
	if from == nil || to == nil {
		return false
	}

	switch fromVal := from.(type) {
	case *OrderedMap:
		if toVal, ok := to.(*OrderedMap); ok {
			return deepEqualOrderedMaps(fromVal, toVal, opts)
		}
		return false
	case map[string]any:
		if toVal, ok := to.(map[string]any); ok {
			return deepEqualMaps(fromVal, toVal, opts)
		}
		return false
	case []any:
		if toVal, ok := to.([]any); ok {
			return deepEqualSlices(fromVal, toVal, opts)
		}
		return false
	default:
		if !sameScalarType(from, to) {
			return false
		}
		return equalValues(from, to, opts)
	}
}

// sprintIdentifier converts an identifier value to a string.
// Fast-paths common types (string, int) to avoid fmt.Sprint overhead.
func sprintIdentifier(id any) string {
	switch v := id.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprint(id)
	}
}

// countToIDs counts the total number of items with comparable identifiers in a list.
// Used to detect duplicate identifiers (if count != len(toIndex), there are duplicates).
