// ordered_map.go - OrderedMap preserves YAML field order during parsing.
package diffyml

import (
	iparse "github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
	"gopkg.in/yaml.v3"
)

// OrderedMap is a map that preserves insertion order of keys.
type OrderedMap = types.OrderedMap

// NewOrderedMap creates an empty OrderedMap.
func NewOrderedMap() *OrderedMap {
	return types.NewOrderedMap()
}

// ParseWithOrder parses YAML content into documents using OrderedMap for mappings.
func ParseWithOrder(content []byte) ([]interface{}, error) {
	return iparse.ParseWithOrder(content)
}

// nodeToInterface converts a yaml.Node tree into Go values.
func nodeToInterface(node *yaml.Node) interface{} {
	return iparse.NodeToInterface(node)
}

// toOrderedMap converts to *OrderedMap.
func toOrderedMap(v interface{}) *OrderedMap {
	return types.ToOrderedMap(v)
}

// resolveScalar converts a scalar yaml.Node into the appropriate Go type.
func resolveScalar(node *yaml.Node) interface{} {
	return iparse.ResolveScalar(node)
}
