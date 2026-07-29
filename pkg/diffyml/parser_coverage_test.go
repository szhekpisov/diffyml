package diffyml

import (
	"errors"
	"io"
	"testing"

	"go.yaml.in/yaml/v3"
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

func TestParseError_Error(t *testing.T) {
	t.Run("with line", func(t *testing.T) {
		pe := &ParseError{Line: 5, Column: 3, Message: "bad indent"}
		got := pe.Error()
		expected := "yaml: line 5: column 3: bad indent"
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
		if pe.Message != "unmarshal errors:\n  test error" {
			t.Errorf("unexpected message: %q", pe.Message)
		}
	})

	t.Run("other error", func(t *testing.T) {
		orig := errors.New("some other error")
		got := wrapParseError(orig)
		var pe *ParseError
		if !errors.As(got, &pe) {
			t.Fatalf("expected *ParseError, got %T", got)
		}
		if !errors.Is(got, orig) {
			t.Errorf("expected wrapped original error, got %v", got)
		}
		if pe.Line != 0 || pe.Column != 0 || pe.Message != orig.Error() {
			t.Errorf("unexpected ParseError fields: %#v", pe)
		}
	})

	t.Run("line location", func(t *testing.T) {
		orig := errors.New("yaml: line 7: bad indentation")
		got := wrapParseError(orig)
		var pe *ParseError
		if !errors.As(got, &pe) {
			t.Fatalf("expected *ParseError, got %T", got)
		}
		if pe.Line != 7 || pe.Column != 0 || pe.Message != "bad indentation" {
			t.Errorf("unexpected ParseError fields: %#v", pe)
		}
		if got.Error() != orig.Error() {
			t.Errorf("expected error text %q, got %q", orig, got)
		}
	})

	t.Run("line and column location", func(t *testing.T) {
		orig := errors.New("yaml: line 7: column 4: bad indentation")
		got := wrapParseError(orig)
		var pe *ParseError
		if !errors.As(got, &pe) {
			t.Fatalf("expected *ParseError, got %T", got)
		}
		if pe.Line != 7 || pe.Column != 4 || pe.Message != "bad indentation" {
			t.Errorf("unexpected ParseError fields: %#v", pe)
		}
		if got.Error() != orig.Error() {
			t.Errorf("expected error text %q, got %q", orig, got)
		}
	})

	t.Run("malformed locations", func(t *testing.T) {
		tests := []struct {
			name       string
			message    string
			wantLine   int
			wantColumn int
			wantDetail string
		}{
			{
				name:       "line without separator",
				message:    "yaml: line nope",
				wantDetail: "line nope",
			},
			{
				name:       "non-numeric line",
				message:    "yaml: line nope: bad indentation",
				wantDetail: "line nope: bad indentation",
			},
			{
				name:       "zero line",
				message:    "yaml: line 0: bad indentation",
				wantDetail: "line 0: bad indentation",
			},
			{
				name:       "column without separator",
				message:    "yaml: line 7: column nope",
				wantLine:   7,
				wantDetail: "column nope",
			},
			{
				name:       "non-numeric column",
				message:    "yaml: line 7: column nope: bad indentation",
				wantLine:   7,
				wantDetail: "column nope: bad indentation",
			},
			{
				name:       "zero column",
				message:    "yaml: line 7: column 0: bad indentation",
				wantLine:   7,
				wantDetail: "column 0: bad indentation",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				line, column, detail := parseErrorLocation(tt.message)
				if line != tt.wantLine || column != tt.wantColumn || detail != tt.wantDetail {
					t.Errorf(
						"parseErrorLocation(%q) = (%d, %d, %q), want (%d, %d, %q)",
						tt.message, line, column, detail, tt.wantLine, tt.wantColumn, tt.wantDetail,
					)
				}
			})
		}
	})
}
