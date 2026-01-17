// parser.go - YAML parsing wrapper.
//
// Wraps gopkg.in/yaml.v3 to parse YAML content into Go interface{} values.
// Handles multi-document YAML files (--- separators).
// Key types: ParseError (with line/column info), DocumentParser (streaming).
package diffyml

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// DocumentParser allows incremental parsing of multi-document YAML.
// Documents are parsed one at a time to reduce peak memory usage.
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
// The returned document can be nil (for empty YAML documents).
// After Next returns io.EOF, subsequent calls will also return io.EOF.
func (p *DocumentParser) Next() (interface{}, error) {
	if p.done {
		return nil, io.EOF
	}

	var doc interface{}
	err := p.decoder.Decode(&doc)
	if err == io.EOF {
		p.done = true
		// If we haven't parsed any documents, return one nil document
		if p.docNum == 0 {
			p.docNum++
			return nil, nil
		}
		return nil, io.EOF
	}
	if err != nil {
		return nil, wrapParseError(err)
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

// --- yaml.Node based parsing for reduced memory usage ---

// parseNodes parses YAML content into a slice of yaml.Node trees.
// This uses less memory than parsing to interface{} because it avoids
// creating Go maps and slices for each YAML structure.
func parseNodes(content []byte) ([]*yaml.Node, error) {
	var nodes []*yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(content))

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, wrapParseError(err)
		}
		nodes = append(nodes, &node)
	}

	return nodes, nil
}

// NodeDocumentParser allows incremental parsing of multi-document YAML into yaml.Node trees.
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
		return nil, wrapParseError(err)
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

// ParseError represents a YAML parsing error with location information.
type ParseError struct {
	Line    int    // Line number where error occurred (1-based)
	Column  int    // Column number where error occurred (1-based)
	Message string // Error message
	Err     error  // Underlying error
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("yaml: line %d: %s", e.Line, e.Message)
	}
	return fmt.Sprintf("yaml: %s", e.Message)
}

// Unwrap returns the underlying error.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// wrapParseError wraps a yaml parsing error with line information if available.
func wrapParseError(err error) error {
	if err == nil {
		return nil
	}

	// yaml.v3 includes line info in the error message
	// Try to extract it if possible
	if typeErr, ok := err.(*yaml.TypeError); ok {
		return &ParseError{
			Message: typeErr.Error(),
			Err:     err,
		}
	}

	// Return original error if we can't wrap it nicely
	return err
}

// parse parses YAML content into a slice of documents.
// Each document is represented as interface{} which can be:
// - *OrderedMap for mappings (preserves field order)
// - []interface{} for sequences
// - scalar values (string, int, float64, bool, nil)
func parse(content []byte) ([]interface{}, error) {
	// Use ParseWithOrder to preserve field order
	docs, err := ParseWithOrder(content)
	if err != nil {
		return nil, err
	}

	// If no documents were parsed, return an empty document
	if len(docs) == 0 {
		docs = append(docs, nil)
	}

	return docs, nil
}
