package diffyml

import (
	"strings"
	"testing"
)

// --- SerializeValue tests ---

func TestSerializeValue_Nil(t *testing.T) {
	got := SerializeValue(nil)
	if got != "<none>" {
		t.Errorf("SerializeValue(nil) = %q, want %q", got, "<none>")
	}
}

func TestSerializeValue_String(t *testing.T) {
	got := SerializeValue("hello")
	if got != "hello" {
		t.Errorf("SerializeValue(\"hello\") = %q, want %q", got, "hello")
	}
}

func TestSerializeValue_Int(t *testing.T) {
	got := SerializeValue(42)
	if got != "42" {
		t.Errorf("SerializeValue(42) = %q, want %q", got, "42")
	}
}

func TestSerializeValue_Bool(t *testing.T) {
	got := SerializeValue(true)
	if got != "true" {
		t.Errorf("SerializeValue(true) = %q, want %q", got, "true")
	}
}

func TestSerializeValue_Float(t *testing.T) {
	got := SerializeValue(3.14)
	if got != "3.14" {
		t.Errorf("SerializeValue(3.14) = %q, want %q", got, "3.14")
	}
}

func TestSerializeValue_OrderedMap(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = []string{"name", "port"}
	om.Values["name"] = "http"
	om.Values["port"] = 80

	got := SerializeValue(om)
	if !strings.Contains(got, "name: http") || !strings.Contains(got, "port: 80") {
		t.Errorf("SerializeValue(OrderedMap) = %q, want to contain name and port", got)
	}
}

func TestSerializeValue_Map(t *testing.T) {
	m := map[string]any{"key": "value"}
	got := SerializeValue(m)
	if !strings.Contains(got, "key: value") {
		t.Errorf("SerializeValue(map) = %q, want to contain 'key: value'", got)
	}
}

func TestSerializeValue_Slice(t *testing.T) {
	s := []any{"a", "b", "c"}
	got := SerializeValue(s)
	if !strings.Contains(got, "- a") || !strings.Contains(got, "- b") {
		t.Errorf("SerializeValue(slice) = %q, want to contain list items", got)
	}
}

// --- FormatSummaryOutput tests ---

func TestFormatSummaryOutput_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	got := FormatSummaryOutput("Test summary text.", opts)

	if !strings.Contains(got, "AI Summary:") {
		t.Errorf("FormatSummaryOutput missing header, got: %s", got)
	}
	if !strings.Contains(got, "Test summary text.") {
		t.Errorf("FormatSummaryOutput missing body, got: %s", got)
	}
	if !strings.HasPrefix(got, "\n") {
		t.Errorf("FormatSummaryOutput should start with blank line, got: %s", got)
	}
}

func TestFormatSummaryOutput_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	got := FormatSummaryOutput("Test summary.", opts)

	if !strings.Contains(got, colorCyan) {
		t.Errorf("FormatSummaryOutput with color should use cyan, got: %s", got)
	}
	if !strings.Contains(got, styleBold) {
		t.Errorf("FormatSummaryOutput with color should use bold, got: %s", got)
	}
	if !strings.Contains(got, colorReset) {
		t.Errorf("FormatSummaryOutput with color should reset, got: %s", got)
	}
}

func TestFormatSummaryOutput_NilOpts(t *testing.T) {
	got := FormatSummaryOutput("nil opts test", nil)
	if !strings.Contains(got, "AI Summary:") {
		t.Errorf("expected 'AI Summary:' header, got: %s", got)
	}
	if !strings.Contains(got, "nil opts test") {
		t.Errorf("expected summary text, got: %s", got)
	}
}
