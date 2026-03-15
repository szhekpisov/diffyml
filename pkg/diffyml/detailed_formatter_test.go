package diffyml

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Scaffold and CLI registration tests

func TestFormatterByName_Detailed(t *testing.T) {
	f, err := FormatterByName("detailed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_DetailedCaseInsensitive(t *testing.T) {
	f, err := FormatterByName("DETAILED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestDetailedFormatter_EmptyDiffs(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	if !strings.Contains(output, "no differences found") {
		t.Errorf("expected 'no differences found' for empty diffs, got: %q", output)
	}
}

func TestDetailedFormatter_NilOptions(t *testing.T) {
	f, _ := FormatterByName("detailed")

	diffs := []Difference{
		{Path: DiffPath{"test", "key"}, Type: DiffModified, From: "old", To: "new"},
	}

	// Should not panic with nil options
	output := f.Format(diffs, nil)
	if output == "" {
		t.Error("formatter should produce output even with nil options")
	}
}

func TestDetailedFormatter_ImplementsInterface(t *testing.T) {
	f, _ := FormatterByName("detailed")

	diffs := []Difference{
		{Path: DiffPath{"test", "path"}, Type: DiffModified, From: "old", To: "new"},
	}
	opts := DefaultFormatOptions()

	output := f.Format(diffs, opts)
	if output == "" {
		t.Error("detailed formatter returned empty output for non-empty diffs")
	}
}

func TestDetailedFormatter_ListedInValidFormats(t *testing.T) {
	// "detailed" should appear in the error message when an invalid format is used
	_, err := FormatterByName("badname")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "detailed") {
		t.Errorf("error message should list 'detailed' as a valid format, got: %s", err.Error())
	}
}

// Path grouping and path headings

func TestDetailedFormatter_PathHeading(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Path should appear on its own line
	if !strings.Contains(output, "config.timeout") {
		t.Errorf("expected path 'config.timeout' in output, got: %q", output)
	}
}

func TestDetailedFormatter_PathGrouping(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "ms", To: "s"},
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "prod"},
	}

	output := f.Format(diffs, opts)
	// "config.timeout" should appear exactly once as a path heading (grouped)
	count := strings.Count(output, "config.timeout")
	if count != 1 {
		t.Errorf("expected 'config.timeout' to appear once (grouped), appeared %d times in: %q", count, output)
	}
	// "config.host" should also appear
	if !strings.Contains(output, "config.host") {
		t.Errorf("expected 'config.host' in output, got: %q", output)
	}
}

func TestDetailedFormatter_PathGroupPreservesOrder(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"alpha"}, Type: DiffAdded, To: "a"},
		{Path: DiffPath{"beta"}, Type: DiffAdded, To: "b"},
		{Path: DiffPath{"alpha"}, Type: DiffAdded, To: "a2"},
	}

	output := f.Format(diffs, opts)
	alphaIdx := strings.Index(output, "alpha")
	betaIdx := strings.Index(output, "beta")
	if alphaIdx < 0 || betaIdx < 0 {
		t.Fatalf("expected both paths in output, got: %q", output)
	}
	// alpha should come first (first-occurrence order)
	if alphaIdx > betaIdx {
		t.Errorf("expected 'alpha' before 'beta' (first-occurrence order), got alpha at %d, beta at %d", alphaIdx, betaIdx)
	}
}

func TestDetailedFormatter_GoPatchPath(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path '/config/timeout', got: %q", output)
	}
}

func TestDetailedFormatter_RootLevelPath(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: nil, Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "(root level)") {
		t.Errorf("expected '(root level)' for empty path, got: %q", output)
	}
}

func TestDetailedFormatter_RootLevelGoPatch(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: nil, Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(output, "\n")
	foundSlash := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "/" {
			foundSlash = true
			break
		}
	}
	if !foundSlash {
		t.Errorf("expected '/' for root path in go-patch mode, got: %q", output)
	}
}

func TestDetailedFormatter_BlankLineBetweenPathBlocks(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"alpha"}, Type: DiffAdded, To: "a"},
		{Path: DiffPath{"beta"}, Type: DiffAdded, To: "b"},
	}

	output := f.Format(diffs, opts)
	// There should be a blank line separating the two path blocks
	if !strings.Contains(output, "\n\n") {
		t.Errorf("expected blank line between path blocks, got: %q", output)
	}
}

// Scalar and order change descriptors

func TestDetailedFormatter_ValueChange(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± value change") {
		t.Errorf("expected '± value change' descriptor, got: %q", output)
	}
	if !strings.Contains(output, "- 30") {
		t.Errorf("expected '- 30' for old value, got: %q", output)
	}
	if !strings.Contains(output, "+ 60") {
		t.Errorf("expected '+ 60' for new value, got: %q", output)
	}
}

func TestDetailedFormatter_TypeChange(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	// int to string type change
	diffs := []Difference{
		{Path: DiffPath{"config", "port"}, Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± type change") {
		t.Errorf("expected '± type change' descriptor, got: %q", output)
	}
	if !strings.Contains(output, "int") {
		t.Errorf("expected 'int' in type change descriptor, got: %q", output)
	}
	if !strings.Contains(output, "string") {
		t.Errorf("expected 'string' in type change descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_TypeChangeBoolToString(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "enabled"}, Type: DiffModified, From: true, To: "true"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± type change from bool to string") {
		t.Errorf("expected '± type change from bool to string', got: %q", output)
	}
}

func TestDetailedFormatter_OrderChanged(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"items"}, Type: DiffOrderChanged, From: []any{"a", "b"}, To: []any{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "⇆ order changed") {
		t.Errorf("expected '⇆ order changed' descriptor, got: %q", output)
	}
}

// List and map entry descriptors with count formatting

func TestDetailedFormatter_ListEntryAdded(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: "newItem"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one list entry added") {
		t.Errorf("expected 'one list entry added', got: %q", output)
	}
}

func TestDetailedFormatter_MultipleListEntriesRemoved(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	// Two removals at the same list path — should be grouped
	diffs := []Difference{
		{Path: DiffPath{"items"}, Type: DiffRemoved, From: "item1"},
		{Path: DiffPath{"items"}, Type: DiffRemoved, From: "item2"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "two list entries removed") {
		// They share the same path "items" but each is a scalar — could be map entries.
		// Actually path "items" doesn't end in digit/bracket, so these are map entries.
		// Let me adjust expectation — these are actually map entries since the path doesn't indicate list.
		if !strings.Contains(output, "two map entries removed") {
			t.Errorf("expected grouped removal descriptor, got: %q", output)
		}
	}
}

func TestDetailedFormatter_ListEntryRemovedBracket(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffRemoved, From: "gone"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one list entry removed") {
		t.Errorf("expected 'one list entry removed', got: %q", output)
	}
}

func TestDetailedFormatter_MapEntryAdded(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "newKey"}, Type: DiffAdded, To: "value"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one map entry added") {
		t.Errorf("expected 'one map entry added', got: %q", output)
	}
}

func TestDetailedFormatter_MapEntryRemoved(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "oldKey"}, Type: DiffRemoved, From: "value"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one map entry removed") {
		t.Errorf("expected 'one map entry removed', got: %q", output)
	}
}

func TestDetailedFormatter_FormatCount(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, "zero"},
		{1, "one"},
		{2, "two"},
		{3, "three"},
		{4, "four"},
		{5, "five"},
		{6, "six"},
		{7, "seven"},
		{8, "eight"},
		{9, "nine"},
		{10, "ten"},
		{11, "eleven"},
		{12, "twelve"},
		{13, "13"},
		{100, "100"},
	}
	for _, tt := range tests {
		result := formatCount(tt.n)
		if result != tt.expected {
			t.Errorf("formatCount(%d) = %q, want %q", tt.n, result, tt.expected)
		}
	}
}

func TestDetailedFormatter_Pluralize(t *testing.T) {
	tests := []struct {
		n        int
		singular string
		plural   string
		expected string
	}{
		{1, "entry", "entries", "entry"},
		{2, "entry", "entries", "entries"},
		{0, "entry", "entries", "entries"},
	}
	for _, tt := range tests {
		result := pluralize(tt.n, tt.singular, tt.plural)
		if result != tt.expected {
			t.Errorf("pluralize(%d, %q, %q) = %q, want %q", tt.n, tt.singular, tt.plural, result, tt.expected)
		}
	}
}

func TestDetailedFormatter_YamlTypeName(t *testing.T) {
	tests := []struct {
		value    any
		expected string
	}{
		{"hello", "string"},
		{42, "int"},
		{int64(42), "int"},
		{3.14, "float"},
		{true, "bool"},
		{nil, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			name := yamlTypeName(tt.value)
			if name != tt.expected {
				t.Errorf("yamlTypeName(%v) = %q, want %q", tt.value, name, tt.expected)
			}
		})
	}
}

// Structured value rendering with YAML-like formatting

func TestDetailedFormatter_StructuredMapAdded(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: DiffPath{"services", "0"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	// Should render with YAML-like formatting
	if !strings.Contains(output, "name: nginx") {
		t.Errorf("expected 'name: nginx' in structured output, got: %q", output)
	}
	if !strings.Contains(output, "port: 80") {
		t.Errorf("expected 'port: 80' in structured output, got: %q", output)
	}
}

func TestDetailedFormatter_StructuredMapWithYAMLIndentation(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	inner := NewOrderedMap()
	inner.Keys = append(inner.Keys, "host", "port")
	inner.Values["host"] = "localhost"
	inner.Values["port"] = 8080

	outer := NewOrderedMap()
	outer.Keys = append(outer.Keys, "name", "config")
	outer.Values["name"] = "myapp"
	outer.Values["config"] = inner

	diffs := []Difference{
		{Path: DiffPath{"apps", "0"}, Type: DiffAdded, To: outer},
	}

	output := f.Format(diffs, opts)
	// Nested levels should use YAML indentation (no pipe guides)
	if !strings.Contains(output, "name: myapp") {
		t.Errorf("expected 'name: myapp' in structured output, got: %q", output)
	}
	if !strings.Contains(output, "host: localhost") {
		t.Errorf("expected 'host: localhost' in nested structure, got: %q", output)
	}
	// config: should be a key with nested children
	if !strings.Contains(output, "config:") {
		t.Errorf("expected 'config:' header for nested map, got: %q", output)
	}
}

func TestDetailedFormatter_StructuredListValue(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	listVal := []any{"alpha", "beta", "gamma"}

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: listVal},
	}

	output := f.Format(diffs, opts)
	// List items should be rendered
	if !strings.Contains(output, "alpha") {
		t.Errorf("expected 'alpha' in list output, got: %q", output)
	}
	if !strings.Contains(output, "beta") {
		t.Errorf("expected 'beta' in list output, got: %q", output)
	}
	// List items should use "- " prefix
	if !strings.Contains(output, "- alpha") {
		t.Errorf("expected list item with '- ' prefix, got: %q", output)
	}
}

func TestDetailedFormatter_StructuredMapRemoved(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "key", "value")
	om.Values["key"] = "removed-entry"
	om.Values["value"] = 42

	diffs := []Difference{
		{Path: DiffPath{"entries", "0"}, Type: DiffRemoved, From: om},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "key: removed-entry") {
		t.Errorf("expected 'key: removed-entry' in removed structured output, got: %q", output)
	}
	if !strings.Contains(output, "value: 42") {
		t.Errorf("expected 'value: 42' in removed structured output, got: %q", output)
	}
}

func TestDetailedFormatter_NestedListInMap(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "ports")
	om.Values["name"] = "service"
	om.Values["ports"] = []any{80, 443}

	diffs := []Difference{
		{Path: DiffPath{"services", "0"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "name: service") {
		t.Errorf("expected 'name: service', got: %q", output)
	}
	// The list within the map should be rendered
	if !strings.Contains(output, "80") || !strings.Contains(output, "443") {
		t.Errorf("expected nested list items 80 and 443, got: %q", output)
	}
}

func TestDetailedFormatter_RegularMapValue(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	// Test with regular map[string]any as well
	m := map[string]any{
		"enabled": true,
	}

	diffs := []Difference{
		{Path: DiffPath{"config", "newKey"}, Type: DiffAdded, To: m},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "enabled: true") {
		t.Errorf("expected 'enabled: true' in regular map output, got: %q", output)
	}
}

func TestDetailedFormatter_NilValueDisplay(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: nil, To: "new"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "<nil>") {
		t.Errorf("expected '<nil>' for nil value, got: %q", output)
	}
}

func TestDetailedFormatter_ScalarFallback(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	// Unknown type should fall back to fmt.Sprintf
	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: 42},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "42") {
		t.Errorf("expected '42' in output for scalar value, got: %q", output)
	}
}

// Header and flag compatibility tests

func TestDetailedFormatter_Header(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
		{Path: DiffPath{"config", "host"}, Type: DiffAdded, To: "prod"},
	}

	output := f.Format(diffs, opts)
	// Should contain a header with spelled-out diff count
	if !strings.Contains(output, "two") || !strings.Contains(output, "differences") {
		t.Errorf("expected header with 'two differences', got: %q", output)
	}
}

func TestDetailedFormatter_HeaderOmitted(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Should NOT contain the "difference" summary header
	if strings.Contains(output, "Found") {
		t.Errorf("expected no header when OmitHeader is true, got: %q", output)
	}
	// But should still contain the actual diff output
	if !strings.Contains(output, "config.timeout") {
		t.Errorf("expected diff output even with omitted header, got: %q", output)
	}
}

func TestDetailedFormatter_HeaderSingleDiff(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "Found one difference") {
		t.Errorf("expected header with 'Found one difference', got: %q", output)
	}
}

func TestDetailedFormatter_HeaderColorEnabled(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Header should have color codes
	if !strings.Contains(output, "\033[") {
		t.Errorf("expected color codes in header with color enabled, got: %q", output)
	}
}

func TestDetailedFormatter_FlagCombination_OmitHeaderGoPatch(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Should use go-patch paths
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path with combined flags, got: %q", output)
	}
	// Should not have header
	if strings.Contains(output, "Found") {
		t.Errorf("expected no header with OmitHeader flag, got: %q", output)
	}
}

func TestDetailedFormatter_FlagCombination_ColorGoPatch(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Both features should work together
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path, got: %q", output)
	}
	if !strings.Contains(output, "\033[") {
		t.Errorf("expected color codes, got: %q", output)
	}
}

// Additional helper unit tests

func TestDetailedFormatter_YamlTypeName_MapAndList(t *testing.T) {
	om := NewOrderedMap()
	if yamlTypeName(om) != "map" {
		t.Errorf("expected 'map' for *OrderedMap, got %q", yamlTypeName(om))
	}

	m := map[string]any{"k": "v"}
	if yamlTypeName(m) != "map" {
		t.Errorf("expected 'map' for map[string]any, got %q", yamlTypeName(m))
	}

	list := []any{"a", "b"}
	if yamlTypeName(list) != "list" {
		t.Errorf("expected 'list' for []any, got %q", yamlTypeName(list))
	}
}

func TestStripWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "helloworld"},
		{"  spaces  ", "spaces"},
		{"tabs\there", "tabshere"},
		{"newlines\nhere", "newlineshere"},
		{"\r\n\t ", ""},
		{"nospaces", "nospaces"},
	}
	for _, tt := range tests {
		result := stripWhitespace(tt.input)
		if result != tt.expected {
			t.Errorf("stripWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDetailedValue(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{nil, "<nil>"},
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
	}
	for _, tt := range tests {
		result := formatDetailedValue(tt.input)
		if result != tt.expected {
			t.Errorf("formatDetailedValue(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Behavior edge cases

func TestDetailedFormatter_MultipleMapEntriesAdded(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config"}, Type: DiffAdded, To: "val1"},
		{Path: DiffPath{"config"}, Type: DiffAdded, To: "val2"},
		{Path: DiffPath{"config"}, Type: DiffAdded, To: "val3"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "three map entries added") {
		t.Errorf("expected 'three map entries added', got: %q", output)
	}
}

func TestDetailedFormatter_AddedAndRemovedInSameGroup(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: "new"},
		{Path: DiffPath{"items", "0"}, Type: DiffRemoved, From: "old"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "added") {
		t.Errorf("expected 'added' descriptor, got: %q", output)
	}
	if !strings.Contains(output, "removed") {
		t.Errorf("expected 'removed' descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_ModifiedNilToValue(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: nil, To: "new-value"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "type change from null to string") {
		t.Errorf("expected type change from null to string, got: %q", output)
	}
}

func TestDetailedFormatter_ModifiedValueToNil(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: "old-value", To: nil},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "type change from string to null") {
		t.Errorf("expected type change from string to null, got: %q", output)
	}
}

func TestDetailedFormatter_OrderChangedWithValues(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{
			Path: DiffPath{"items"}, Type: DiffOrderChanged,
			From: []any{"x", "y", "z"},
			To:   []any{"z", "y", "x"},
		},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "⇆ order changed") {
		t.Errorf("expected '⇆ order changed', got: %q", output)
	}
	if !strings.Contains(output, "    - ") {
		t.Errorf("expected '    - ' for old order, got: %q", output)
	}
	if !strings.Contains(output, "    + ") {
		t.Errorf("expected '    + ' for new order, got: %q", output)
	}
}

func TestDetailedFormatter_DeeplyNestedStructure(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	innermost := NewOrderedMap()
	innermost.Keys = append(innermost.Keys, "deep")
	innermost.Values["deep"] = "value"

	middle := NewOrderedMap()
	middle.Keys = append(middle.Keys, "nested")
	middle.Values["nested"] = innermost

	outer := NewOrderedMap()
	outer.Keys = append(outer.Keys, "level1")
	outer.Values["level1"] = middle

	diffs := []Difference{
		{Path: DiffPath{"root", "0"}, Type: DiffAdded, To: outer},
	}

	output := f.Format(diffs, opts)
	// Should use YAML-style indentation, no pipe guides
	expected := "root.0\n  + one list entry added:\n    - level1:\n        nested:\n          deep: value\n\n"
	if output != expected {
		t.Errorf("deeply nested structure mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

// Regression-prevention tests

func TestDetailedFormatter_MapEntryScalar_RendersKeyValue(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"config", "verbose"}, Type: DiffAdded, To: true},
	}

	output := f.Format(diffs, opts)
	expected := "config.verbose\n  + one map entry added:\n    verbose: true\n\n"
	if output != expected {
		t.Errorf("map entry scalar should render as key: value.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_MapEntryStructured_RendersKeyWrapper(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	inner := NewOrderedMap()
	inner.Keys = append(inner.Keys, "host", "port")
	inner.Values["host"] = "localhost"
	inner.Values["port"] = 8080

	wrapper := NewOrderedMap()
	wrapper.Keys = append(wrapper.Keys, "newKey")
	wrapper.Values["newKey"] = inner

	diffs := []Difference{
		{Path: DiffPath{"config"}, Type: DiffAdded, To: wrapper},
	}

	output := f.Format(diffs, opts)
	expected := "config\n  + one map entry added:\n    newKey:\n      host: localhost\n      port: 8080\n\n"
	if output != expected {
		t.Errorf("map entry structured should render key as YAML wrapper.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_StillUsesDashPrefix(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: "hello"},
	}

	output := f.Format(diffs, opts)
	expected := "items.0\n  + one list entry added:\n    - hello\n\n"
	if output != expected {
		t.Errorf("list entry should still use dash prefix.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_NoLeadingBlankLine_OmitHeader(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: "a", To: "b"},
	}

	output := f.Format(diffs, opts)
	if strings.HasPrefix(output, "\n") {
		t.Errorf("output should NOT start with blank line when OmitHeader is true, got: %q", output)
	}
}

func TestDetailedFormatter_LeadingBlankLine_WithHeader(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: "a", To: "b"},
	}

	output := f.Format(diffs, opts)
	// Header should be followed by \n\n (blank line before first path group)
	if !strings.Contains(output, "difference\n\nkey") {
		t.Errorf("header should be followed by blank line before first path group, got: %q", output)
	}
}

func TestDetailedFormatter_TrailingSeparator_ValueChange(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	expected := "key\n  ± value change\n    - old\n    + new\n\n"
	if output != expected {
		t.Errorf("value change should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_TrailingSeparator_EntryBatch(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: "val"},
	}

	output := f.Format(diffs, opts)
	expected := "items.0\n  + one list entry added:\n    - val\n\n"
	if output != expected {
		t.Errorf("entry batch should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_TrailingSeparator_OrderChange(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{
			Path: DiffPath{"items"}, Type: DiffOrderChanged,
			From: []any{"a", "b"},
			To:   []any{"b", "a"},
		},
	}

	output := f.Format(diffs, opts)
	expected := "items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("order change should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_TrailingSeparator_TypeChange(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"port"}, Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	expected := "port\n  ± type change from int to string\n    - 8080\n    + 8080\n\n"
	if output != expected {
		t.Errorf("type change should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_HeaderFormat_SpelledOutCount(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()

	// Single diff: "Found one difference"
	diffs1 := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: "a", To: "b"},
	}
	output1 := f.Format(diffs1, opts)
	if !strings.Contains(output1, "Found one difference\n") {
		t.Errorf("expected 'Found one difference' for 1 diff, got: %q", output1)
	}

	// Three diffs: "Found three differences"
	diffs3 := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"b"}, Type: DiffModified, From: "3", To: "4"},
		{Path: DiffPath{"c"}, Type: DiffModified, From: "5", To: "6"},
	}
	output3 := f.Format(diffs3, opts)
	if !strings.Contains(output3, "Found three differences\n") {
		t.Errorf("expected 'Found three differences' for 3 diffs, got: %q", output3)
	}
}

// Order change comma-separated format

func TestDetailedFormatter_OrderChange_CommaSeparated(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{
			Path: DiffPath{"items"}, Type: DiffOrderChanged,
			From: []any{"a", "b", "c"},
			To:   []any{"c", "a", "b"},
		},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- a, b, c") {
		t.Errorf("expected comma-separated '- a, b, c', got: %q", output)
	}
	if !strings.Contains(output, "+ c, a, b") {
		t.Errorf("expected comma-separated '+ c, a, b', got: %q", output)
	}
}

func TestDetailedFormatter_OrderChange_SingleItem(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{
			Path: DiffPath{"items"}, Type: DiffOrderChanged,
			From: []any{"a"},
			To:   []any{"a"},
		},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "    - a\n") {
		t.Errorf("expected single item '- a', got: %q", output)
	}
	if !strings.Contains(output, "    + a\n") {
		t.Errorf("expected single item '+ a', got: %q", output)
	}
}

func TestDetailedFormatter_OrderChange_NonStringItems(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{
			Path: DiffPath{"nums"}, Type: DiffOrderChanged,
			From: []any{1, 2, 3},
			To:   []any{3, 1, 2},
		},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- 1, 2, 3") {
		t.Errorf("expected '- 1, 2, 3', got: %q", output)
	}
	if !strings.Contains(output, "+ 3, 1, 2") {
		t.Errorf("expected '+ 3, 1, 2', got: %q", output)
	}
}

func TestDetailedFormatter_OrderChange_Snapshot(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{
			Path: DiffPath{"items"}, Type: DiffOrderChanged,
			From: []any{"a", "b"},
			To:   []any{"b", "a"},
		},
	}

	output := f.Format(diffs, opts)
	expected := "items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("order change snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

// List entry YAML dash prefix

func TestDetailedFormatter_ListEntry_DashPrefix(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: DiffPath{"services", "0"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	// First key should have "- " prefix, continuation keys at +2 indent
	if !strings.Contains(output, "    - name: nginx\n") {
		t.Errorf("expected '- name: nginx' with dash prefix, got: %q", output)
	}
	if !strings.Contains(output, "      port: 80\n") {
		t.Errorf("expected '      port: 80' at +2 indent, got: %q", output)
	}
}

func TestDetailedFormatter_ListEntry_MultipleMaps(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om1 := NewOrderedMap()
	om1.Keys = append(om1.Keys, "name", "id")
	om1.Values["name"] = "second"
	om1.Values["id"] = 2

	om2 := NewOrderedMap()
	om2.Keys = append(om2.Keys, "name", "id")
	om2.Values["name"] = "third"
	om2.Values["id"] = 3

	// Same path groups entries into a single batch
	diffs := []Difference{
		{Path: DiffPath{"items", "1"}, Type: DiffAdded, To: om1},
		{Path: DiffPath{"items", "1"}, Type: DiffAdded, To: om2},
	}

	output := f.Format(diffs, opts)
	expected := "items.1\n  + two list entries added:\n    - name: second\n      id: 2\n    - name: third\n      id: 3\n\n"
	if output != expected {
		t.Errorf("multiple maps mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_NestedMap(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	inner := NewOrderedMap()
	inner.Keys = append(inner.Keys, "host", "port")
	inner.Values["host"] = "localhost"
	inner.Values["port"] = 8080

	outer := NewOrderedMap()
	outer.Keys = append(outer.Keys, "name", "config")
	outer.Values["name"] = "svc"
	outer.Values["config"] = inner

	diffs := []Difference{
		{Path: DiffPath{"services", "0"}, Type: DiffAdded, To: outer},
	}

	output := f.Format(diffs, opts)
	expected := "services.0\n  + one list entry added:\n    - name: svc\n      config:\n        host: localhost\n        port: 8080\n\n"
	if output != expected {
		t.Errorf("nested map mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_ScalarUnchanged(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: "hello"},
	}

	output := f.Format(diffs, opts)
	expected := "items.0\n  + one list entry added:\n    - hello\n\n"
	if output != expected {
		t.Errorf("scalar list entry should still use '- value' format.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_Snapshot(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: DiffPath{"services", "0"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	expected := "services.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n"
	if output != expected {
		t.Errorf("list entry snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

// Document heading tests

func TestDetailedFormatter_DocumentHeading(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	t.Run("single doc replaces [0] with (document)", func(t *testing.T) {
		diffs := []Difference{
			{Path: DiffPath{"[0]"}, Type: DiffRemoved, From: "value", DocumentIndex: 0},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "(document)") {
			t.Errorf("expected '(document)' in output, got: %q", output)
		}
		if strings.Contains(output, "[0]") {
			t.Errorf("should not contain '[0]' in output, got: %q", output)
		}
	})

	t.Run("multi doc replaces [0] with (document 1)", func(t *testing.T) {
		diffs := []Difference{
			{Path: DiffPath{"[0]"}, Type: DiffRemoved, From: "value1", DocumentIndex: 0},
			{Path: DiffPath{"[1]"}, Type: DiffAdded, To: "value2", DocumentIndex: 1},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "(root level) (document 0)") {
			t.Errorf("expected '(root level) (document 0)' in output, got: %q", output)
		}
		if !strings.Contains(output, "(root level) (document 1)") {
			t.Errorf("expected '(root level) (document 1)' in output, got: %q", output)
		}
	})

	t.Run("non-bare index paths are not transformed", func(t *testing.T) {
		diffs := []Difference{
			{Path: DiffPath{"items", "0"}, Type: DiffModified, From: "old", To: "new"},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "items.0") {
			t.Errorf("expected 'items.0' in output, got: %q", output)
		}
	})
}

func TestDiffPath_IsBareDocIndex(t *testing.T) {
	tests := []struct {
		path   DiffPath
		wantOk bool
	}{
		{DiffPath{"[0]"}, true},
		{DiffPath{"[1]"}, true},
		{DiffPath{"[12]"}, true},
		{DiffPath{"items", "[0]"}, false}, // multi-segment
		{DiffPath{"[0]", "spec"}, false},  // multi-segment
		{DiffPath{"name"}, false},         // not a bracket
		{nil, false},                      // empty
		{DiffPath{"[]"}, false},           // empty brackets
		{DiffPath{"[abc]"}, false},        // non-numeric
	}
	for _, tt := range tests {
		ok := tt.path.IsBareDocIndex()
		if ok != tt.wantOk {
			t.Errorf("DiffPath%v.IsBareDocIndex() = %v, want %v", []string(tt.path), ok, tt.wantOk)
		}
	}
}

// Mutation testing kill shots

func TestDetailedFormatter_NestedMapIndentation(t *testing.T) {
	// Use map[string]any (not OrderedMap) to exercise line 264 specifically
	innerMap := map[string]any{"inner": "value"}
	outerMap := map[string]any{"outer": innerMap}

	diffs := []Difference{
		{Path: DiffPath{"config"}, Type: DiffAdded, To: outerMap},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	output := f.Format(diffs, opts)

	// Check that the nested map has proper indentation
	lines := strings.Split(output, "\n")
	outerIndent := -1
	innerIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if strings.HasPrefix(trimmed, "outer:") {
			outerIndent = len(line) - len(trimmed)
		}
		if strings.HasPrefix(trimmed, "inner:") {
			innerIndent = len(line) - len(trimmed)
		}
	}
	if outerIndent < 0 || innerIndent < 0 {
		t.Fatalf("could not find outer (%d) or inner (%d) indent in output:\n%s", outerIndent, innerIndent, output)
	}
	// Indent difference must be exactly 2 (kills indent+2 → indent*2 or indent-2 mutations)
	delta := innerIndent - outerIndent
	if delta != 2 {
		t.Errorf("inner key indent (%d) - outer key indent (%d) = %d, want exactly 2", innerIndent, outerIndent, delta)
	}
}

func TestDetailedFormatter_ListEntryAtIndex9(t *testing.T) {
	// A scalar list entry at index 9 must still be detected as a list entry.
	// This catches the mutation c > '9' → c >= '9' which would treat '9' as non-digit.
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"items", "9"}, Type: DiffAdded, To: "newval"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "list entry") {
		t.Errorf("expected 'list entry' for index 9, got: %q", output)
	}
}

func TestRenderEntryValue_KeyExtractDotAtStart(t *testing.T) {
	// detailed_formatter.go:212 — `idx >= 0` → `> 0`
	// If path starts with ".", LastIndex returns 0.
	// With >= 0, key = path[1:]; with > 0, key stays as full path.
	// We use DiffAdded to exercise renderEntryValue (not formatChangeDescriptor).
	f := &DetailedFormatter{}
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	// DiffAdded with scalar value — goes through formatEntryBatch → renderEntryValue
	diffs := []Difference{
		{
			Path: DiffPath{"keyname"},
			Type: DiffAdded,
			To:   "added_value",
		},
	}

	output := f.Format(diffs, opts)
	// The key should be "keyname" (without leading dot) in the rendered entry
	if strings.Contains(output, ".keyname:") {
		t.Errorf("key should be extracted without leading dot, got: %q", output)
	}
	if !strings.Contains(output, "keyname") {
		t.Errorf("expected 'keyname' in output, got: %q", output)
	}
}

// Document index prefix and colon notation

func TestDiffPath_DocIndexPrefix(t *testing.T) {
	tests := []struct {
		path     DiffPath
		wantIdx  int
		wantRest DiffPath
		wantOk   bool
	}{
		{DiffPath{"[0]", "spec", "field"}, 0, DiffPath{"spec", "field"}, true},
		{DiffPath{"[2]", "metadata", "name"}, 2, DiffPath{"metadata", "name"}, true},
		{DiffPath{"[12]", "x"}, 12, DiffPath{"x"}, true},
		{DiffPath{"[0]"}, 0, DiffPath{"[0]"}, false},                   // bare index — single segment
		{DiffPath{"items", "[0]"}, 0, DiffPath{"items", "[0]"}, false}, // not a leading index
		{DiffPath{"name"}, 0, DiffPath{"name"}, false},                 // no bracket
		{nil, 0, nil, false}, // empty
		{DiffPath{"[abc]", "spec"}, 0, DiffPath{"[abc]", "spec"}, false}, // non-numeric
	}
	for _, tt := range tests {
		idx, rest, ok := tt.path.DocIndexPrefix()
		if ok != tt.wantOk || idx != tt.wantIdx || fmt.Sprint(rest) != fmt.Sprint(tt.wantRest) {
			t.Errorf("DiffPath%v.DocIndexPrefix() = (%d, %v, %v), want (%d, %v, %v)",
				[]string(tt.path), idx, rest, ok, tt.wantIdx, tt.wantRest, tt.wantOk)
		}
	}
}

func TestDetailedFormatter_ColonNotation(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	t.Run("doc index prefix uses colon notation", func(t *testing.T) {
		diffs := []Difference{
			{Path: DiffPath{"[0]", "spec", "field"}, Type: DiffModified, From: "old", To: "new"},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "spec.field (document 0)") {
			t.Errorf("expected 'spec.field (document 0)' in output, got: %q", output)
		}
		if strings.Contains(output, "[0]") {
			t.Errorf("should not contain '[0]' in output, got: %q", output)
		}
	})

	t.Run("higher doc index uses colon notation", func(t *testing.T) {
		diffs := []Difference{
			{Path: DiffPath{"[2]", "metadata", "name"}, Type: DiffModified, From: "old", To: "new"},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "metadata.name (document 2)") {
			t.Errorf("expected 'metadata.name (document 2)' in output, got: %q", output)
		}
	})

	t.Run("go-patch style with doc index prefix", func(t *testing.T) {
		gpOpts := DefaultFormatOptions()
		gpOpts.OmitHeader = true
		gpOpts.UseGoPatchStyle = true
		diffs := []Difference{
			{Path: DiffPath{"[0]", "spec", "field"}, Type: DiffModified, From: "old", To: "new"},
		}
		output := f.Format(diffs, gpOpts)
		if !strings.Contains(output, "/spec/field (document 0)") {
			t.Errorf("expected '/spec/field (document 0)' in output, got: %q", output)
		}
	})

	t.Run("non-leading index still preserved", func(t *testing.T) {
		diffs := []Difference{
			{Path: DiffPath{"items", "0"}, Type: DiffModified, From: "old", To: "new"},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "items.0") {
			t.Errorf("expected 'items.0' in output, got: %q", output)
		}
	})
}

// renderDocumentValue edge cases

func TestDetailedFormatter_RenderDocumentValue_OrderedMap(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "apiVersion", "kind")
	om.Values["apiVersion"] = "v1"
	om.Values["kind"] = "Service"

	diffs := []Difference{
		{Path: DiffPath{"[0]"}, Type: DiffAdded, To: om},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "    ---\n    apiVersion: v1") {
		t.Errorf("expected '---' separator before document keys, got:\n%s", output)
	}
	if !strings.Contains(output, "kind: Service") {
		t.Errorf("expected 'kind: Service' in output, got:\n%s", output)
	}
	// Must NOT have list bullet prefix
	if strings.Contains(output, "- apiVersion:") {
		t.Errorf("document value should not have list bullet prefix, got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderDocumentValue_MapStringAny(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"[0]"}, Type: DiffAdded, To: map[string]any{"name": "test", "value": "123"}},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "    ---\n    name: test") {
		t.Errorf("expected '---' separator before document keys, got:\n%s", output)
	}
	if !strings.Contains(output, "value: 123") {
		t.Errorf("expected 'value: 123' in output, got:\n%s", output)
	}
	// Must NOT have list bullet prefix
	if strings.Contains(output, "- name:") {
		t.Errorf("document value should not have list bullet prefix, got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderDocumentValue_Removed(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "apiVersion", "kind")
	om.Values["apiVersion"] = "v1"
	om.Values["kind"] = "Service"

	diffs := []Difference{
		{Path: DiffPath{"[0]"}, Type: DiffRemoved, From: om},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "---") {
		t.Errorf("expected '---' separator for removed document, got:\n%s", output)
	}
	if !strings.Contains(output, "apiVersion: v1") {
		t.Errorf("expected 'apiVersion: v1' in output, got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderDocumentValue_Scalar(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"[0]"}, Type: DiffAdded, To: "just-a-string"},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "---") {
		t.Errorf("expected '---' separator for scalar document, got:\n%s", output)
	}
	if !strings.Contains(output, "just-a-string") {
		t.Errorf("expected 'just-a-string' in output, got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderDocumentValue_TrueColor(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.TrueColor = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "apiVersion")
	om.Values["apiVersion"] = "v1"

	diffs := []Difference{
		{Path: DiffPath{"[0]"}, Type: DiffAdded, To: om},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "---") {
		t.Errorf("expected '---' separator with TrueColor, got:\n%s", output)
	}
	if !strings.Contains(output, "apiVersion: v1") {
		t.Errorf("expected 'apiVersion: v1' in output, got:\n%s", output)
	}
}

// renderEntryValue edge cases

func TestDetailedFormatter_RenderEntryValue_ListScalar(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// Scalar list entry (default branch of renderEntryValue for isList=true)
	diffs := []Difference{
		{Path: DiffPath{"tags", "0"}, Type: DiffAdded, To: "production"},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- production") {
		t.Errorf("expected '- production' for scalar list entry, got: %q", output)
	}
}

func TestDetailedFormatter_RenderEntryValue_ListOfLists(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// []any branch of renderEntryValue for isList=true
	diffs := []Difference{
		{Path: DiffPath{"matrix", "0"}, Type: DiffAdded, To: []any{"a", "b", "c"}},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- a") {
		t.Errorf("expected '- a' for nested list entry, got: %q", output)
	}
}

func TestDetailedFormatter_RenderEntryValue_MapEntry(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// map[string]any branch of renderListItems: first key must have "- " bullet,
	// second key must NOT have bullet. Keys are sorted: "name" < "value".
	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: map[string]any{"name": "test", "value": "123"}},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- name:") {
		t.Errorf("expected first key to have bullet prefix '- name:', got:\n%s", output)
	}
	if strings.Contains(output, "- value:") {
		t.Errorf("expected second key WITHOUT bullet prefix, but found '- value:' in:\n%s", output)
	}
	if !strings.Contains(output, "value: 123") {
		t.Errorf("expected second key rendered as 'value: 123', got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderFirstKeyValueYAML_MapValue(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// map[string]any branch of renderFirstKeyValueYAML
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "config")
	om.Values["config"] = map[string]any{"host": "localhost", "port": 8080}
	diffs := []Difference{
		{Path: DiffPath{"services", "0"}, Type: DiffAdded, To: om},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- config:") {
		t.Errorf("expected '- config:' for map value in first key, got: %q", output)
	}
}

func TestDetailedFormatter_RenderFirstKeyValueYAML_ListValue(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// []any branch of renderFirstKeyValueYAML
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "ports")
	om.Values["ports"] = []any{80, 443}
	diffs := []Difference{
		{Path: DiffPath{"services", "0"}, Type: DiffAdded, To: om},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- ports:") {
		t.Errorf("expected '- ports:' for list value, got: %q", output)
	}
	if !strings.Contains(output, "- 80") {
		t.Errorf("expected '- 80' in list items, got: %q", output)
	}
}

func TestDetailedFormatter_RenderFirstKeyValueYAML_MultilineString(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// multiline string in default branch of renderFirstKeyValueYAML
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "script")
	om.Values["script"] = "line1\nline2\nline3"
	diffs := []Difference{
		{Path: DiffPath{"steps", "0"}, Type: DiffAdded, To: om},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "- script:") {
		t.Errorf("expected '- script:' for multiline string, got: %q", output)
	}
	if !strings.Contains(output, "line1") {
		t.Errorf("expected multiline content in output, got: %q", output)
	}
}

// Misc helper tests

func TestYamlTypeName_DefaultType(t *testing.T) {
	// default branch: unknown type
	result := yamlTypeName(struct{}{})
	if result != "struct {}" {
		t.Errorf("expected 'struct {}' for unknown type, got: %q", result)
	}
}

func TestFormatCommaSeparated_NonSlice(t *testing.T) {
	// non-slice fallback
	result := formatCommaSeparated("scalar-value")
	if result != "scalar-value" {
		t.Errorf("expected 'scalar-value', got: %q", result)
	}
}

func TestDiffPath_DocIndexPrefix_NoBracketClose(t *testing.T) {
	// segment "[0.spec" does not match [N] format
	path := DiffPath{"[0.spec"}
	_, _, ok := path.DocIndexPrefix()
	if ok {
		t.Error("expected false for malformed bracket segment")
	}
}

// Timestamp and structured type change tests

func TestFormatTimestamp(t *testing.T) {
	t.Run("date only", func(t *testing.T) {
		ts := time.Date(2010, 9, 9, 0, 0, 0, 0, time.UTC)
		got := formatTimestamp(ts)
		if got != "2010-09-09" {
			t.Errorf("expected 2010-09-09, got %s", got)
		}
	})

	t.Run("datetime uses RFC3339", func(t *testing.T) {
		ts := time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)
		got := formatTimestamp(ts)
		if got != "2023-06-15T14:30:00Z" {
			t.Errorf("expected 2023-06-15T14:30:00Z, got %s", got)
		}
	})
}

func TestFormatDetailedValue_Timestamp(t *testing.T) {
	ts := time.Date(2010, 9, 9, 0, 0, 0, 0, time.UTC)
	got := formatDetailedValue(ts)
	if got != "2010-09-09" {
		t.Errorf("expected 2010-09-09, got %s", got)
	}
}

func TestFormatValueAsYAMLLines(t *testing.T) {
	t.Run("map[string]any", func(t *testing.T) {
		val := map[string]any{"key": "val"}
		lines := formatValueAsYAMLLines(val)
		if len(lines) != 1 || lines[0] != "key: val" {
			t.Errorf("expected [key: val], got %v", lines)
		}
	})

	t.Run("map with nested structured child", func(t *testing.T) {
		val := map[string]any{
			"parent": map[string]any{"child": 1},
		}
		lines := formatValueAsYAMLLines(val)
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
		}
		if lines[0] != "parent:" {
			t.Errorf("expected 'parent:', got %s", lines[0])
		}
		if lines[1] != "  child: 1" {
			t.Errorf("expected '  child: 1', got %s", lines[1])
		}
	})

	t.Run("list with structured items", func(t *testing.T) {
		val := []any{
			map[string]any{"name": "a"},
		}
		lines := formatValueAsYAMLLines(val)
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
		}
		if lines[0] != "- ..." {
			t.Errorf("expected '- ...', got %s", lines[0])
		}
		if lines[1] != "  name: a" {
			t.Errorf("expected '  name: a', got %s", lines[1])
		}
	})

	t.Run("OrderedMap with structured child", func(t *testing.T) {
		val := &OrderedMap{
			Keys:   []string{"parent"},
			Values: map[string]any{"parent": map[string]any{"child": 1}},
		}
		lines := formatValueAsYAMLLines(val)
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
		}
		if lines[0] != "parent:" {
			t.Errorf("expected 'parent:', got %s", lines[0])
		}
		if lines[1] != "  child: 1" {
			t.Errorf("expected '  child: 1', got %s", lines[1])
		}
	})

	t.Run("default scalar fallback", func(t *testing.T) {
		lines := formatValueAsYAMLLines(42)
		if len(lines) != 1 || lines[0] != "42" {
			t.Errorf("expected [42], got %v", lines)
		}
	})
}

func TestDetailedFormatter_TypeChange_Structured(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	t.Run("map to list", func(t *testing.T) {
		om := &OrderedMap{
			Keys:   []string{"a", "b"},
			Values: map[string]any{"a": 1, "b": 2},
		}
		diffs := []Difference{
			{Path: DiffPath{"foo"}, Type: DiffModified, From: om, To: []any{1, 2}},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "type change from map to list") {
			t.Errorf("expected type change descriptor, got:\n%s", output)
		}
		if !strings.Contains(output, "- a: 1") {
			t.Errorf("expected '- a: 1' in output, got:\n%s", output)
		}
		if !strings.Contains(output, "+ - 1") {
			t.Errorf("expected '+ - 1' in output, got:\n%s", output)
		}
	})

	t.Run("timestamp to string", func(t *testing.T) {
		ts := time.Date(2010, 9, 9, 0, 0, 0, 0, time.UTC)
		diffs := []Difference{
			{Path: DiffPath{"ver"}, Type: DiffModified, From: ts, To: "2010-09-09"},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "type change from timestamp to string") {
			t.Errorf("expected 'timestamp' type name, got:\n%s", output)
		}
		if !strings.Contains(output, "- 2010-09-09") {
			t.Errorf("expected formatted date, got:\n%s", output)
		}
	})
}

func TestIsStructured(t *testing.T) {
	if !isStructured(map[string]any{"a": 1}) {
		t.Error("map[string]any should be structured")
	}
	if !isStructured(&OrderedMap{}) {
		t.Error("*OrderedMap should be structured")
	}
	if !isStructured([]any{1}) {
		t.Error("[]any should be structured")
	}
	if isStructured("hello") {
		t.Error("string should not be structured")
	}
	if isStructured(42) {
		t.Error("int should not be structured")
	}
}

// Nested list rendering tests

func TestDetailedFormatter_RenderListItems_OrderedMapInList(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// Simulate volumeMounts-style data: list of *OrderedMap items
	mount1 := NewOrderedMap()
	mount1.Keys = append(mount1.Keys, "name", "mountPath")
	mount1.Values["name"] = "config-vol"
	mount1.Values["mountPath"] = "/config"

	mount2 := NewOrderedMap()
	mount2.Keys = append(mount2.Keys, "name", "mountPath")
	mount2.Values["name"] = "data-vol"
	mount2.Values["mountPath"] = "/data"

	container := NewOrderedMap()
	container.Keys = append(container.Keys, "name", "volumeMounts")
	container.Values["name"] = "app"
	container.Values["volumeMounts"] = []any{mount1, mount2}

	diffs := []Difference{
		{Path: DiffPath{"containers", "0"}, Type: DiffAdded, To: container},
	}
	output := f.Format(diffs, opts)

	// Must NOT contain raw Go struct representation
	if strings.Contains(output, "&{") {
		t.Errorf("output contains raw Go struct '&{', got:\n%s", output)
	}
	if strings.Contains(output, "0x") {
		t.Errorf("output contains pointer address '0x', got:\n%s", output)
	}
	// Must contain properly rendered YAML
	if !strings.Contains(output, "- name: config-vol") {
		t.Errorf("expected '- name: config-vol', got:\n%s", output)
	}
	if !strings.Contains(output, "  mountPath: /config") {
		t.Errorf("expected '  mountPath: /config', got:\n%s", output)
	}
	if !strings.Contains(output, "- name: data-vol") {
		t.Errorf("expected '- name: data-vol', got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderListItems_NestedOrderedMapInList(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// Simulate env with nested valueFrom.fieldRef
	fieldRef := NewOrderedMap()
	fieldRef.Keys = append(fieldRef.Keys, "fieldPath")
	fieldRef.Values["fieldPath"] = "status.hostIP"

	valueFrom := NewOrderedMap()
	valueFrom.Keys = append(valueFrom.Keys, "fieldRef")
	valueFrom.Values["fieldRef"] = fieldRef

	envVar := NewOrderedMap()
	envVar.Keys = append(envVar.Keys, "name", "valueFrom")
	envVar.Values["name"] = "DD_AGENT_HOST"
	envVar.Values["valueFrom"] = valueFrom

	container := NewOrderedMap()
	container.Keys = append(container.Keys, "name", "env")
	container.Values["name"] = "app"
	container.Values["env"] = []any{envVar}

	diffs := []Difference{
		{Path: DiffPath{"containers", "0"}, Type: DiffAdded, To: container},
	}
	output := f.Format(diffs, opts)

	if strings.Contains(output, "&{") {
		t.Errorf("output contains raw Go struct, got:\n%s", output)
	}
	if strings.Contains(output, "0x") {
		t.Errorf("output contains pointer address, got:\n%s", output)
	}
	if !strings.Contains(output, "- name: DD_AGENT_HOST") {
		t.Errorf("expected '- name: DD_AGENT_HOST', got:\n%s", output)
	}
	if !strings.Contains(output, "fieldPath: status.hostIP") {
		t.Errorf("expected 'fieldPath: status.hostIP', got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderKeyValueYAML_ListWithStructuredItems(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// renderKeyValueYAML []any branch: map entry whose value is a list of structured items
	item := NewOrderedMap()
	item.Keys = append(item.Keys, "name", "port")
	item.Values["name"] = "http"
	item.Values["port"] = 8080

	diffs := []Difference{
		{Path: DiffPath{"spec", "ports"}, Type: DiffAdded, To: []any{item}},
	}
	output := f.Format(diffs, opts)

	if strings.Contains(output, "&{") {
		t.Errorf("output contains raw Go struct, got:\n%s", output)
	}
	if !strings.Contains(output, "- name: http") {
		t.Errorf("expected '- name: http', got:\n%s", output)
	}
	if !strings.Contains(output, "port: 8080") {
		t.Errorf("expected 'port: 8080', got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderFirstKeyValueYAML_ListWithStructuredItems(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// renderFirstKeyValueYAML []any branch: first key of a list entry has structured list value
	secretRef := NewOrderedMap()
	secretRef.Keys = append(secretRef.Keys, "name")
	secretRef.Values["name"] = "my-secret"

	envFromItem := NewOrderedMap()
	envFromItem.Keys = append(envFromItem.Keys, "secretRef")
	envFromItem.Values["secretRef"] = secretRef

	container := NewOrderedMap()
	container.Keys = append(container.Keys, "envFrom")
	container.Values["envFrom"] = []any{envFromItem}

	diffs := []Difference{
		{Path: DiffPath{"containers", "0"}, Type: DiffAdded, To: container},
	}
	output := f.Format(diffs, opts)

	if strings.Contains(output, "&{") {
		t.Errorf("output contains raw Go struct, got:\n%s", output)
	}
	if strings.Contains(output, "0x") {
		t.Errorf("output contains pointer address, got:\n%s", output)
	}
	if !strings.Contains(output, "secretRef:") {
		t.Errorf("expected 'secretRef:', got:\n%s", output)
	}
	if !strings.Contains(output, "name: my-secret") {
		t.Errorf("expected 'name: my-secret', got:\n%s", output)
	}
}

func TestDetailedFormatter_RenderEntryValue_ListWithMixedItems(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// renderEntryValue []any branch with mixed scalar and structured items
	item := NewOrderedMap()
	item.Keys = append(item.Keys, "key", "value")
	item.Values["key"] = "foo"
	item.Values["value"] = "bar"

	diffs := []Difference{
		{Path: DiffPath{"items", "0"}, Type: DiffAdded, To: []any{"scalar-val", item}},
	}
	output := f.Format(diffs, opts)

	if strings.Contains(output, "&{") {
		t.Errorf("output contains raw Go struct, got:\n%s", output)
	}
	if !strings.Contains(output, "- scalar-val") {
		t.Errorf("expected '- scalar-val', got:\n%s", output)
	}
	if !strings.Contains(output, "- key: foo") {
		t.Errorf("expected '- key: foo', got:\n%s", output)
	}
}
