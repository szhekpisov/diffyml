package diffyml

import (
	"errors"
	"io"
	"testing"

	"gopkg.in/yaml.v3"
)

// drainDocumentParser reads all documents from a DocumentParser and returns them.
// If wantErr is true, returns (nil, true) when an error is encountered.
func drainDocumentParser(t *testing.T, p *DocumentParser, wantErr bool) (docs []any, gotErr bool) {
	t.Helper()
	for {
		doc, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if wantErr {
				return nil, true
			}
			t.Fatalf("unexpected error: %v", err)
		}
		docs = append(docs, doc)
	}
	return docs, false
}

// drainNodeDocumentParser reads all nodes from a NodeDocumentParser and returns them.
func drainNodeDocumentParser(t *testing.T, p *NodeDocumentParser, wantErr bool) (nodes []*yaml.Node, gotErr bool) {
	t.Helper()
	for {
		node, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if wantErr {
				return nil, true
			}
			t.Fatalf("unexpected error: %v", err)
		}
		nodes = append(nodes, node)
	}
	return nodes, false
}

func TestDocumentParser(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantDocs  int
		wantFirst any // expected first doc value (nil means nil doc)
		wantErr   bool
	}{
		{
			name:     "single document",
			input:    "foo: bar\n",
			wantDocs: 1,
		},
		{
			name:     "empty content",
			input:    "",
			wantDocs: 1, // returns one nil document
		},
		{
			name:     "multi document",
			input:    "a: 1\n---\nb: 2\n",
			wantDocs: 2,
		},
		{
			name:    "invalid yaml",
			input:   ":\n  :\n    - :\n  bad:\n    indent\n  wrong:\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewDocumentParser([]byte(tt.input))

			if p.Done() {
				t.Fatal("parser should not be done before any calls")
			}
			if p.DocumentCount() != 0 {
				t.Fatalf("expected initial doc count 0, got %d", p.DocumentCount())
			}

			docs, gotErr := drainDocumentParser(t, p, tt.wantErr)
			if gotErr {
				return
			}

			if tt.wantErr {
				t.Fatal("expected error but got none")
			}

			if len(docs) != tt.wantDocs {
				t.Errorf("expected %d docs, got %d", tt.wantDocs, len(docs))
			}

			if !p.Done() {
				t.Error("parser should be done after EOF")
			}
			if p.DocumentCount() != tt.wantDocs {
				t.Errorf("expected document count %d, got %d", tt.wantDocs, p.DocumentCount())
			}
		})
	}

	// Sub-test: done stays EOF
	t.Run("done stays EOF", func(t *testing.T) {
		p := NewDocumentParser([]byte("x: 1\n"))
		drainDocumentParser(t, p, false)
		// Subsequent call should still be EOF
		_, err := p.Next()
		if !errors.Is(err, io.EOF) {
			t.Errorf("expected io.EOF on repeated call, got %v", err)
		}
	})
}

func TestNodeDocumentParser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDocs int
		wantErr  bool
	}{
		{
			name:     "single document",
			input:    "foo: bar\n",
			wantDocs: 1,
		},
		{
			name:     "empty content",
			input:    "",
			wantDocs: 0, // NodeDocumentParser returns EOF immediately for empty
		},
		{
			name:     "multi document",
			input:    "a: 1\n---\nb: 2\n",
			wantDocs: 2,
		},
		{
			name:    "invalid yaml",
			input:   ":\n  :\n    - :\n  bad:\n    indent\n  wrong:\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewNodeDocumentParser([]byte(tt.input))

			if p.Done() {
				t.Fatal("parser should not be done before any calls")
			}
			if p.DocumentCount() != 0 {
				t.Fatalf("expected initial doc count 0, got %d", p.DocumentCount())
			}

			nodes, gotErr := drainNodeDocumentParser(t, p, tt.wantErr)
			if gotErr {
				return
			}

			if tt.wantErr {
				t.Fatal("expected error but got none")
			}

			if len(nodes) != tt.wantDocs {
				t.Errorf("expected %d nodes, got %d", tt.wantDocs, len(nodes))
			}

			if !p.Done() {
				t.Error("parser should be done after EOF")
			}
			if p.DocumentCount() != tt.wantDocs {
				t.Errorf("expected document count %d, got %d", tt.wantDocs, p.DocumentCount())
			}
		})
	}

	t.Run("done stays EOF", func(t *testing.T) {
		p := NewNodeDocumentParser([]byte("x: 1\n"))
		drainNodeDocumentParser(t, p, false)
		_, err := p.Next()
		if !errors.Is(err, io.EOF) {
			t.Errorf("expected io.EOF on repeated call, got %v", err)
		}
	})
}

func TestParseNodes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "valid multi-doc",
			input:     "a: 1\n---\nb: 2\n",
			wantCount: 2,
		},
		{
			name:      "empty",
			input:     "",
			wantCount: 0,
		},
		{
			name:      "single",
			input:     "key: value\n",
			wantCount: 1,
		},
		{
			name:    "invalid YAML",
			input:   ":\n  :\n    - :\n  bad:\n    indent\n  wrong:\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parseNodes([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(nodes) != tt.wantCount {
				t.Errorf("expected %d nodes, got %d", tt.wantCount, len(nodes))
			}
		})
	}
}

func TestParseError_Error(t *testing.T) {
	t.Run("with line", func(t *testing.T) {
		pe := &ParseError{Line: 5, Column: 3, Message: "bad indent"}
		got := pe.Error()
		expected := "yaml: line 5: bad indent"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("without line", func(t *testing.T) {
		pe := &ParseError{Line: 0, Message: "generic error"}
		got := pe.Error()
		expected := "yaml: generic error"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestParseError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	pe := &ParseError{Message: "wrapper", Err: inner}
	if !errors.Is(pe, inner) {
		t.Error("expected errors.Is to find inner error through Unwrap")
	}
}

func TestWrapParseError(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		if got := wrapParseError(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("yaml TypeError", func(t *testing.T) {
		typeErr := &yaml.TypeError{Errors: []string{"test error"}}
		got := wrapParseError(typeErr)
		var pe *ParseError
		if !errors.As(got, &pe) {
			t.Fatalf("expected *ParseError, got %T", got)
		}
		if !errors.Is(pe.Err, typeErr) {
			t.Error("expected Err to wrap original TypeError")
		}
	})

	t.Run("other error", func(t *testing.T) {
		orig := errors.New("some other error")
		got := wrapParseError(orig)
		if !errors.Is(got, orig) {
			t.Errorf("expected original error returned unchanged, got %v", got)
		}
	})
}
