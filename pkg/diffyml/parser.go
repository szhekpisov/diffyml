// parser.go - YAML parsing wrapper.
//
// Wraps go.yaml.in/yaml/v3 to parse YAML content into Go any values.
// Handles multi-document YAML files (--- separators).
// Key types: ParseError (with line/column info), DocumentParser (streaming).
package diffyml

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
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
func (p *DocumentParser) Next() (any, error) {
	// gomutants:disable-next-line BRANCH_IF reason="defensive guard; yaml.Decoder.Decode returns io.EOF idempotently after exhaustion, so the same final return path is reached either way"
	if p.done {
		return nil, io.EOF
	}

	var doc any
	err := p.decoder.Decode(&doc)
	if errors.Is(err, io.EOF) {
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
	// yaml.v3 includes line info in the error message
	// Try to extract it if possible
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		return &ParseError{
			Message: typeErr.Error(),
			Err:     err,
		}
	}

	// Return original error if we can't wrap it nicely
	return err
}

// parse parses YAML content into a slice of documents.
// Each document is represented as any which can be:
// - *OrderedMap for mappings (preserves field order)
// - []any for sequences
// - scalar values (string, int, float64, bool, nil)
func parse(content []byte) ([]any, error) {
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

// parseWithNodes parses YAML content into documents and also returns the raw
// yaml.Node trees (which carry source line/column info) for line-number capture.
// When no documents are parsed, a single nil document is returned with a nil node
// (no source position), mirroring parse().
func parseWithNodes(content []byte) ([]any, []*yaml.Node, error) {
	docs, nodes, err := parseWithOrderAndNodes(content)
	if err != nil {
		return nil, nil, err
	}

	if len(docs) == 0 {
		docs = append(docs, nil)
		nodes = append(nodes, nil)
	}

	return docs, nodes, nil
}
