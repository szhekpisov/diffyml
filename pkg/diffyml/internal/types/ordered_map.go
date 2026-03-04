package types

import "sort"

// OrderedMap is a map that preserves insertion order of keys.
type OrderedMap struct {
	Keys   []string
	Values map[string]YAMLValue
}

// NewOrderedMap creates an empty OrderedMap.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Keys:   nil,
		Values: make(map[string]YAMLValue),
	}
}

// ToOrderedMap converts a map[string]interface{} to an *OrderedMap with
// alphabetically sorted keys. If the value is already an *OrderedMap it is
// returned as-is. Returns nil for any other type.
func ToOrderedMap(v interface{}) *OrderedMap {
	switch m := v.(type) {
	case *OrderedMap:
		return m
	case map[string]interface{}:
		om := NewOrderedMap()
		om.Keys = SortedMapKeys(m)
		for k, val := range m {
			om.Values[k] = val
		}
		return om
	default:
		return nil
	}
}

// SortedMapKeys returns the keys of a map[string]interface{} in sorted order.
func SortedMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
