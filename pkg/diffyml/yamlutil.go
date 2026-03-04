// yamlutil.go - Shared YAML serialization utilities.
package diffyml

import (
	iparse "github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
	"gopkg.in/yaml.v3"
)

func marshalStructuredYAML(val interface{}) (string, bool) {
	return iparse.MarshalStructuredYAML(val)
}

func orderedMapToGeneric(om *OrderedMap) *yaml.Node {
	return iparse.OrderedMapToGeneric(om)
}

func valueToYAMLNode(val interface{}) *yaml.Node {
	return iparse.ValueToYAMLNode(val)
}
