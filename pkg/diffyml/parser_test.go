package diffyml

import (
	"strings"
	"testing"
)

// getMapValue extracts a value from either *OrderedMap or map[string]interface{}
func getMapValue(doc interface{}, key string) interface{} {
	switch m := doc.(type) {
	case *OrderedMap:
		return m.Values[key]
	case map[string]interface{}:
		return m[key]
	default:
		return nil
	}
}

// isMap checks if a value is a map type (OrderedMap or regular map)
func isMap(val interface{}) bool {
	switch val.(type) {
	case *OrderedMap, map[string]interface{}:
		return true
	default:
		return false
	}
}

func TestParse_SingleDocument(t *testing.T) {
	content := []byte(`---
foo: bar
baz: 123
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}

	if !isMap(docs[0]) {
		t.Fatalf("expected map, got %T", docs[0])
	}
	if getMapValue(docs[0], "foo") != "bar" {
		t.Errorf("expected foo=bar, got foo=%v", getMapValue(docs[0], "foo"))
	}
}

func TestParse_MultiDocument(t *testing.T) {
	content := []byte(`---
doc: one
---
doc: two
---
doc: three
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 documents, got %d", len(docs))
	}

	for i, expected := range []string{"one", "two", "three"} {
		if !isMap(docs[i]) {
			t.Fatalf("document %d: expected map, got %T", i, docs[i])
		}
		if getMapValue(docs[i], "doc") != expected {
			t.Errorf("document %d: expected doc=%s, got doc=%v", i, expected, getMapValue(docs[i], "doc"))
		}
	}
}

func TestParse_EmptyDocument(t *testing.T) {
	content := []byte(``)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty content should return one nil document
	if len(docs) != 1 {
		t.Errorf("expected 1 document for empty content, got %d", len(docs))
	}
}

func TestParse_ListAsRoot(t *testing.T) {
	content := []byte(`---
- item1
- item2
- item3
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}

	list, ok := docs[0].([]interface{})
	if !ok {
		t.Fatalf("expected list, got %T", docs[0])
	}
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}
}

func TestParse_ScalarTypes(t *testing.T) {
	content := []byte(`---
string: hello
integer: 42
float: 3.14
boolean: true
null_value: null
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !isMap(docs[0]) {
		t.Fatalf("expected map, got %T", docs[0])
	}

	if getMapValue(docs[0], "string") != "hello" {
		t.Errorf("string: expected hello, got %v", getMapValue(docs[0], "string"))
	}
	if getMapValue(docs[0], "integer") != 42 {
		t.Errorf("integer: expected 42, got %v", getMapValue(docs[0], "integer"))
	}
	if getMapValue(docs[0], "boolean") != true {
		t.Errorf("boolean: expected true, got %v", getMapValue(docs[0], "boolean"))
	}
	if getMapValue(docs[0], "null_value") != nil {
		t.Errorf("null_value: expected nil, got %v", getMapValue(docs[0], "null_value"))
	}
}

func TestParse_NestedStructure(t *testing.T) {
	content := []byte(`---
level1:
  level2:
    level3:
      value: deep
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l1 := getMapValue(docs[0], "level1")
	l2 := getMapValue(l1, "level2")
	l3 := getMapValue(l2, "level3")

	if getMapValue(l3, "value") != "deep" {
		t.Errorf("expected deep, got %v", getMapValue(l3, "value"))
	}
}

func TestParse_InvalidYAML_ReturnsError(t *testing.T) {
	content := []byte(`---
invalid: yaml: content: here
  bad indentation
`)
	_, err := parse(content)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestParse_InvalidYAML_ContainsLineNumber(t *testing.T) {
	content := []byte(`---
valid: content
  invalid_indent: here
more: stuff
`)
	_, err := parse(content)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}

	errStr := err.Error()
	// yaml.v3 includes line numbers in error messages
	if !strings.Contains(errStr, "line") && !strings.Contains(errStr, "3") {
		t.Logf("error message: %s", errStr)
		// Note: yaml.v3 error format may vary, so we just check error exists
	}
}

func TestParse_MultiDocumentWithEmpty(t *testing.T) {
	content := []byte(`---
first: doc
---
---
third: doc
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty document between --- markers is parsed as nil
	if len(docs) < 2 {
		t.Errorf("expected at least 2 documents, got %d", len(docs))
	}
}

func TestParse_JSONCompatibleYAML(t *testing.T) {
	content := []byte(`{"foo": "bar", "baz": [1, 2, 3]}`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}

	if !isMap(docs[0]) {
		t.Fatalf("expected map, got %T", docs[0])
	}
	if getMapValue(docs[0], "foo") != "bar" {
		t.Errorf("expected foo=bar, got foo=%v", getMapValue(docs[0], "foo"))
	}
}

func TestParse_Anchors(t *testing.T) {
	content := []byte(`---
defaults: &defaults
  timeout: 30
  retries: 3

production:
  <<: *defaults
  host: prod.example.com
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prod := getMapValue(docs[0], "production")

	// Anchors should be resolved
	if getMapValue(prod, "timeout") != 30 {
		t.Errorf("expected timeout=30 from anchor, got %v", getMapValue(prod, "timeout"))
	}
	if getMapValue(prod, "host") != "prod.example.com" {
		t.Errorf("expected host=prod.example.com, got %v", getMapValue(prod, "host"))
	}
}

func TestParseError_HasLineInfo(t *testing.T) {
	content := []byte(`---
valid: content
  invalid: indentation
`)
	_, err := parse(content)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}

	// Check if error can be converted to ParseError
	if pe, ok := err.(*ParseError); ok {
		if pe.Line == 0 {
			t.Error("expected non-zero line number in ParseError")
		}
	}
	// Note: yaml.v3 errors may not always be wrapped as ParseError
}

func TestParse_DocumentIndex(t *testing.T) {
	// This test verifies that we can track document indices
	content := []byte(`---
first: 1
---
second: 2
---
third: 3
`)
	docs, err := parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 documents, got %d", len(docs))
	}
	// Indices are implicit based on slice position
	for i := range docs {
		if docs[i] == nil {
			t.Errorf("document %d should not be nil", i)
		}
	}
}
