// parser.go - YAML parsing wrapper.
package diffyml

import (
	iparse "github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
	"gopkg.in/yaml.v3"
)

type DocumentParser = iparse.DocumentParser

func NewDocumentParser(content []byte) *DocumentParser {
	return iparse.NewDocumentParser(content)
}

type NodeDocumentParser = iparse.NodeDocumentParser

func NewNodeDocumentParser(content []byte) *NodeDocumentParser {
	return iparse.NewNodeDocumentParser(content)
}

func parseNodes(content []byte) ([]*yaml.Node, error) {
	return iparse.ParseNodes(content)
}

type ParseError = types.ParseError

func wrapParseError(err error) error {
	return iparse.WrapParseError(err)
}

func parse(content []byte) ([]interface{}, error) {
	return iparse.Parse(content)
}
