// comparator.go - Core YAML comparison logic.
//
// Compares YAML documents at the AST level to detect semantic differences.
// Handles maps, lists (ordered and unordered), scalars, and multi-document files.
// Integrates with kubernetes.go for K8s resource matching.
package diffyml

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// compareDocs compares two slices of YAML documents and returns differences.
func compareDocs(from, to []interface{}, opts *Options) []Difference {
	// Check if Kubernetes detection is enabled and documents are K8s resources
	if opts != nil && opts.DetectKubernetes && hasK8sDocuments(from, to) {
		return compareK8sDocs(from, to, opts)
	}

	var diffs []Difference

	// For simplicity, compare document by document
	// If document counts differ, that's handled by comparing indices
	maxLen := len(from)
	if len(to) > maxLen {
		maxLen = len(to)
	}

	for i := 0; i < maxLen; i++ {
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
func hasK8sDocuments(from, to []interface{}) bool {
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

// compareNodes recursively compares two YAML nodes.
func compareNodes(path string, from, to interface{}, opts *Options) []Difference {
	var diffs []Difference

	// Handle nil cases
	if from == nil && to == nil {
		return diffs
	}

	if from == nil {
		// Value was added
		return []Difference{{
			Path: cleanPath(path),
			Type: DiffAdded,
			From: nil,
			To:   to,
		}}
	}

	if to == nil {
		// Value changed to null - treat as modification (key still exists)
		if opts != nil && opts.IgnoreValueChanges {
			return diffs
		}
		return []Difference{{
			Path: cleanPath(path),
			Type: DiffModified,
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
		return []Difference{{
			Path: cleanPath(path),
			Type: DiffModified,
			From: from,
			To:   to,
		}}
	}

	// Compare based on type
	switch fromVal := from.(type) {
	case *OrderedMap:
		// Type-equality guard above ensures to is also *OrderedMap
		toOrderedMap := to.(*OrderedMap)
		diffs = append(diffs, compareOrderedMaps(path, fromVal, toOrderedMap, opts)...)

	case map[string]interface{}:
		// Type-equality guard above ensures to is also map[string]interface{}
		toMap := to.(map[string]interface{})
		diffs = append(diffs, compareMaps(path, fromVal, toMap, opts)...)

	case []interface{}:
		toVal := to.([]interface{})
		diffs = append(diffs, compareLists(path, fromVal, toVal, opts)...)

	default:
		// Scalar comparison
		if !equalValues(from, to, opts) {
			if opts != nil && opts.IgnoreValueChanges {
				return diffs
			}
			diffs = append(diffs, Difference{
				Path: cleanPath(path),
				Type: DiffModified,
				From: from,
				To:   to,
			})
		}
	}

	return diffs
}

// compareOrderedMaps compares two OrderedMap nodes preserving source document order
func compareOrderedMaps(path string, from, to *OrderedMap, opts *Options) []Difference {
	var diffs []Difference

	// Track which keys from 'to' have been seen
	toSeen := make(map[string]bool)

	// First iterate over 'from' keys in their original order
	for _, key := range from.Keys {
		toSeen[key] = true
		childPath := joinPath(path, key)
		fromVal := from.Values[key]
		toVal, toOk := to.Values[key]

		if !toOk {
			// Key was removed
			diffs = append(diffs, Difference{
				Path: cleanPath(childPath),
				Type: DiffRemoved,
				From: fromVal,
				To:   nil,
			})
		} else {
			// Key exists in both - recurse
			diffs = append(diffs, compareNodes(childPath, fromVal, toVal, opts)...)
		}
	}

	// Then iterate over 'to' keys to find additions (in their original order)
	for _, key := range to.Keys {
		if !toSeen[key] {
			childPath := joinPath(path, key)
			diffs = append(diffs, Difference{
				Path: cleanPath(childPath),
				Type: DiffAdded,
				From: nil,
				To:   to.Values[key],
			})
		}
	}

	return diffs
}

// compareMaps compares two map nodes.
func compareMaps(path string, from, to map[string]interface{}, opts *Options) []Difference {
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
		childPath := joinPath(path, key)
		fromVal, fromOk := from[key]
		toVal, toOk := to[key]

		switch {
		case !fromOk:
			// Key was added
			diffs = append(diffs, Difference{
				Path: cleanPath(childPath),
				Type: DiffAdded,
				From: nil,
				To:   toVal,
			})
		case !toOk:
			// Key was removed
			diffs = append(diffs, Difference{
				Path: cleanPath(childPath),
				Type: DiffRemoved,
				From: fromVal,
				To:   nil,
			})
		default:
			// Key exists in both - recurse
			diffs = append(diffs, compareNodes(childPath, fromVal, toVal, opts)...)
		}
	}

	return diffs
}

// compareLists compares two list nodes.
func compareLists(path string, from, to []interface{}, opts *Options) []Difference {
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
func areListItemsHeterogeneous(from, to []interface{}) bool {
	// Get keys from all items in both lists
	allKeys := make(map[string]bool)

	extractKeys := func(list []interface{}) {
		for _, item := range list {
			switch v := item.(type) {
			case *OrderedMap:
				for _, key := range v.Keys {
					allKeys[key] = true
				}
			case map[string]interface{}:
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
	checkSingleDistinctKeys := func(list []interface{}) bool {
		for _, item := range list {
			switch v := item.(type) {
			case *OrderedMap:
				if len(v.Keys) != 1 {
					return false
				}
			case map[string]interface{}:
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
func compareListsPositional(path string, from, to []interface{}, opts *Options) []Difference {
	var diffs []Difference

	maxLen := len(from)
	if len(to) > maxLen {
		maxLen = len(to)
	}

	for i := 0; i < maxLen; i++ {
		childPath := fmt.Sprintf("%s.%d", path, i)
		var fromVal, toVal interface{}

		if i < len(from) {
			fromVal = from[i]
		}
		if i < len(to) {
			toVal = to[i]
		}

		//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
		if i >= len(from) {
			// Item was added
			diffs = append(diffs, Difference{
				Path: cleanPath(childPath),
				Type: DiffAdded,
				From: nil,
				To:   toVal,
			})
		} else if i >= len(to) {
			// Item was removed
			diffs = append(diffs, Difference{
				Path: cleanPath(childPath),
				Type: DiffRemoved,
				From: fromVal,
				To:   nil,
			})
		} else {
			// Both exist - recurse
			diffs = append(diffs, compareNodes(childPath, fromVal, toVal, opts)...)
		}
	}

	return diffs
}

// compareListsUnordered compares lists ignoring order.
func compareListsUnordered(path string, from, to []interface{}, opts *Options) []Difference {
	var diffs []Difference

	// Track which items have been matched
	toMatched := make([]bool, len(to))

	for i, fromItem := range from {
		found := false
		for j, toItem := range to {
			if toMatched[j] {
				continue
			}
			if deepEqual(fromItem, toItem, opts) {
				toMatched[j] = true
				found = true
				break
			}
		}
		if !found {
			// Item was removed
			diffs = append(diffs, Difference{
				Path: fmt.Sprintf("%s.%d", cleanPath(path), i),
				Type: DiffRemoved,
				From: fromItem,
				To:   nil,
			})
		}
	}

	// Check for added items
	for j, toItem := range to {
		if !toMatched[j] {
			diffs = append(diffs, Difference{
				Path: fmt.Sprintf("%s.%d", cleanPath(path), j),
				Type: DiffAdded,
				From: nil,
				To:   toItem,
			})
		}
	}

	return diffs
}

// canMatchByIdentifier checks if list items can be matched by identifier.
// Returns true only if all items are maps and at least one has a "name" or "id" field.
func canMatchByIdentifier(list []interface{}, opts *Options) bool {
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	return CanMatchByIdentifierWithAdditional(list, additional)
}

// getIdentifier gets the identifier value from a map or OrderedMap.
func getIdentifier(val interface{}, opts *Options) interface{} {
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	if om, ok := val.(*OrderedMap); ok {
		return getIdentifierFromOrderedMap(om, additional)
	}
	if m, ok := val.(map[string]interface{}); ok {
		return GetIdentifierWithAdditional(m, additional)
	}
	return nil
}

// compareListsByIdentifier compares lists matching by identifier field.
func compareListsByIdentifier(path string, from, to []interface{}, opts *Options) []Difference {
	var diffs []Difference

	// Build index of from items by identifier, preserving order
	fromIndex := make(map[interface{}]int)
	fromIDs := make([]interface{}, 0)
	fromNoID := make([]int, 0)
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
	toIndex := make(map[interface{}]int)
	toNoID := make([]int, 0)
	for i, item := range to {
		id := getIdentifier(item, opts)
		if isComparableIdentifier(id) {
			toIndex[id] = i
			continue
		}
		toNoID = append(toNoID, i)
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
			diffs = append(diffs, compareNodes(childPath, fromItem, toItem, opts)...)
		} else {
			// Item was removed - report at the list level (dyff-style)
			diffs = append(diffs, Difference{
				Path: cleanPath(path),
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
					Path: cleanPath(path),
					Type: DiffAdded,
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
			if deepEqual(fromItem, to[toIdx], opts) {
				toNoIDMatched[j] = true
				found = true
				break
			}
		}
		if !found {
			diffs = append(diffs, Difference{
				Path: fmt.Sprintf("%s.%d", cleanPath(path), fromIdx),
				Type: DiffRemoved,
				From: fromItem,
				To:   nil,
			})
		}
	}
	for j, toIdx := range toNoID {
		if toNoIDMatched[j] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("%s.%d", cleanPath(path), toIdx),
			Type: DiffAdded,
			From: nil,
			To:   to[toIdx],
		})
	}

	return diffs
}

// equalValues compares two scalar values for equality.
func equalValues(from, to interface{}, opts *Options) bool {
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

// deepEqual compares two values deeply with options.
func deepEqual(from, to interface{}, opts *Options) bool {
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
	case *OrderedMap:
		// Type-equality guard above ensures to is also *OrderedMap
		toOrderedMap := to.(*OrderedMap)
		if len(fromVal.Values) != len(toOrderedMap.Values) {
			return false
		}
		for k, fv := range fromVal.Values {
			tv, ok := toOrderedMap.Values[k]
			if !ok || !deepEqual(fv, tv, opts) {
				return false
			}
		}
		return true

	case map[string]interface{}:
		// Type-equality guard above ensures to is also map[string]interface{}
		toMap := to.(map[string]interface{})
		if len(fromVal) != len(toMap) {
			return false
		}
		for k, fv := range fromVal {
			tv, ok := toMap[k]
			if !ok || !deepEqual(fv, tv, opts) {
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
			if !deepEqual(fromVal[i], toVal[i], opts) {
				return false
			}
		}
		return true

	default:
		return equalValues(from, to, opts)
	}
}

// joinPath joins path segments with a dot.
func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// cleanPath removes leading dots from path.
func cleanPath(path string) string {
	return strings.TrimPrefix(path, ".")
}
