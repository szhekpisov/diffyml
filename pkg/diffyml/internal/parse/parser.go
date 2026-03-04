package parse

import (
	"bytes"
	"io"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
	"gopkg.in/yaml.v3"
)

// DocumentParser allows incremental parsing of multi-document YAML.
type DocumentParser struct {
	decoder *yaml.Decoder
	docNum  int
	done    bool
}

// NewDocumentParser creates a parser that reads documents incrementally.
func NewDocumentParser(content []byte) *DocumentParser {
	return &DocumentParser{
		decoder: yaml.NewDecoder(bytes.NewReader(content)),
		docNum:  0,
		done:    false,
	}
}

// Next returns the next document, or io.EOF when no more documents.
func (p *DocumentParser) Next() (interface{}, error) {
	if p.done {
		return nil, io.EOF
	}

	var doc interface{}
	err := p.decoder.Decode(&doc)
	if err == io.EOF {
		p.done = true
		if p.docNum == 0 {
			p.docNum++
			return nil, nil
		}
		return nil, io.EOF
	}
	if err != nil {
		return nil, WrapParseError(err)
	}

	p.docNum++
	return doc, nil
}

// DocumentCount returns the number of documents parsed so far.
func (p *DocumentParser) DocumentCount() int {
	return p.docNum
}

// Done returns true if all documents have been parsed.
func (p *DocumentParser) Done() bool {
	return p.done
}

// NodeDocumentParser allows incremental parsing into yaml.Node trees.
type NodeDocumentParser struct {
	decoder *yaml.Decoder
	docNum  int
	done    bool
}

// NewNodeDocumentParser creates a parser that reads documents as yaml.Node trees.
func NewNodeDocumentParser(content []byte) *NodeDocumentParser {
	return &NodeDocumentParser{
		decoder: yaml.NewDecoder(bytes.NewReader(content)),
		docNum:  0,
		done:    false,
	}
}

// Next returns the next document as a yaml.Node, or io.EOF when no more documents.
func (p *NodeDocumentParser) Next() (*yaml.Node, error) {
	if p.done {
		return nil, io.EOF
	}

	var node yaml.Node
	err := p.decoder.Decode(&node)
	if err == io.EOF {
		p.done = true
		return nil, io.EOF
	}
	if err != nil {
		return nil, WrapParseError(err)
	}

	p.docNum++
	return &node, nil
}

// DocumentCount returns the number of documents parsed so far.
func (p *NodeDocumentParser) DocumentCount() int {
	return p.docNum
}

// Done returns true if all documents have been parsed.
func (p *NodeDocumentParser) Done() bool {
	return p.done
}

// ParseNodes parses YAML content into a slice of yaml.Node trees.
func ParseNodes(content []byte) ([]*yaml.Node, error) {
	var nodes []*yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(content))

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, WrapParseError(err)
		}
		nodes = append(nodes, &node)
	}

	return nodes, nil
}

// WrapParseError wraps a yaml parsing error with line information if available.
func WrapParseError(err error) error {
	if err == nil {
		return nil
	}

	if typeErr, ok := err.(*yaml.TypeError); ok {
		return &types.ParseError{
			Message: typeErr.Error(),
			Err:     err,
		}
	}

	return err
}

// Parse parses YAML content into a slice of documents.
func Parse(content []byte) ([]interface{}, error) {
	docs, err := ParseWithOrder(content)
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		docs = append(docs, nil)
	}

	return docs, nil
}

// ParseWithOrder parses YAML content into documents using OrderedMap for mappings.
func ParseWithOrder(content []byte) ([]interface{}, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var docs []interface{}

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, WrapParseError(err)
		}
		docs = append(docs, NodeToInterface(&node))
	}

	return docs, nil
}

// NodeToInterface converts a yaml.Node tree into Go values,
// using *OrderedMap for mapping nodes to preserve key order.
func NodeToInterface(node *yaml.Node) interface{} {
	return nodeToInterfaceWithCycleDetection(node, make(map[*yaml.Node]bool))
}

func nodeToInterfaceWithCycleDetection(node *yaml.Node, seen map[*yaml.Node]bool) interface{} {
	if node == nil {
		return nil
	}

	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil
		}
		return nodeToInterfaceWithCycleDetection(node.Content[0], seen)
	}

	switch node.Kind {
	case yaml.MappingNode:
		om := types.NewOrderedMap()
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if key == "<<" {
				merged := nodeToInterfaceWithCycleDetection(node.Content[i+1], seen)
				if mergedMap, ok := merged.(*types.OrderedMap); ok {
					for _, mk := range mergedMap.Keys {
						if _, exists := om.Values[mk]; !exists {
							om.Keys = append(om.Keys, mk)
							om.Values[mk] = mergedMap.Values[mk]
						}
					}
				}
				continue
			}
			val := nodeToInterfaceWithCycleDetection(node.Content[i+1], seen)
			om.Keys = append(om.Keys, key)
			om.Values[key] = val
		}
		return om

	case yaml.SequenceNode:
		list := make([]interface{}, 0, len(node.Content))
		for _, child := range node.Content {
			list = append(list, nodeToInterfaceWithCycleDetection(child, seen))
		}
		return list

	case yaml.ScalarNode:
		return ResolveScalar(node)

	case yaml.AliasNode:
		if seen[node.Alias] {
			return nil
		}
		seen[node.Alias] = true
		result := nodeToInterfaceWithCycleDetection(node.Alias, seen)
		delete(seen, node.Alias)
		return result

	default:
		return nil
	}
}

// ResolveScalar converts a scalar yaml.Node into the appropriate Go type.
func ResolveScalar(node *yaml.Node) interface{} {
	var val interface{}
	if err := node.Decode(&val); err != nil {
		return node.Value
	}
	return val
}
