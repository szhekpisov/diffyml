// yamlutil.go - Shared YAML serialization utilities.
package diffyml

import (
	iparse "github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
)

func marshalStructuredYAML(val interface{}) (string, bool) {
	return iparse.MarshalStructuredYAML(val)
}
