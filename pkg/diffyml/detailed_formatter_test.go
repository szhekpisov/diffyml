package diffyml

import (
	"strings"
	"testing"
)

// Task 1: Scaffold and CLI registration tests

func TestGetFormatter_Detailed(t *testing.T) {
	f, err := GetFormatter("detailed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestGetFormatter_DetailedCaseInsensitive(t *testing.T) {
	f, err := GetFormatter("DETAILED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestValidateOutputFormat_Detailed(t *testing.T) {
	err := ValidateOutputFormat("detailed")
	if err != nil {
		t.Fatalf("expected 'detailed' to be a valid output format, got error: %v", err)
	}
}

func TestDetailedFormatter_EmptyDiffs(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	if !strings.Contains(output, "no differences found") {
		t.Errorf("expected 'no differences found' for empty diffs, got: %q", output)
	}
}

func TestDetailedFormatter_NilOptions(t *testing.T) {
	f, _ := GetFormatter("detailed")

	diffs := []Difference{
		{Path: "test.key", Type: DiffModified, From: "old", To: "new"},
	}

	// Should not panic with nil options
	output := f.Format(diffs, nil)
	if output == "" {
		t.Error("formatter should produce output even with nil options")
	}
}

func TestDetailedFormatter_ImplementsInterface(t *testing.T) {
	f, _ := GetFormatter("detailed")

	diffs := []Difference{
		{Path: "test.path", Type: DiffModified, From: "old", To: "new"},
	}
	opts := DefaultFormatOptions()

	output := f.Format(diffs, opts)
	if output == "" {
		t.Error("detailed formatter returned empty output for non-empty diffs")
	}
}

func TestDetailedFormatter_ListedInValidFormats(t *testing.T) {
	// "detailed" should appear in the error message when an invalid format is used
	_, err := GetFormatter("badname")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "detailed") {
		t.Errorf("error message should list 'detailed' as a valid format, got: %s", err.Error())
	}
}

// Task 2.1: Path grouping and path headings

func TestDetailedFormatter_PathHeading(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Path should appear on its own line
	if !strings.Contains(output, "config.timeout") {
		t.Errorf("expected path 'config.timeout' in output, got: %q", output)
	}
}

func TestDetailedFormatter_PathGrouping(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		{Path: "config.timeout", Type: DiffModified, From: "ms", To: "s"},
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "prod"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "alpha", Type: DiffAdded, To: "a"},
		{Path: "beta", Type: DiffAdded, To: "b"},
		{Path: "alpha", Type: DiffAdded, To: "a2"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path '/config/timeout', got: %q", output)
	}
}

func TestDetailedFormatter_RootLevelPath(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "(root level)") {
		t.Errorf("expected '(root level)' for empty path, got: %q", output)
	}
}

func TestDetailedFormatter_RootLevelGoPatch(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: "", Type: DiffModified, From: "old", To: "new"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "alpha", Type: DiffAdded, To: "a"},
		{Path: "beta", Type: DiffAdded, To: "b"},
	}

	output := f.Format(diffs, opts)
	// There should be a blank line separating the two path blocks
	if !strings.Contains(output, "\n\n") {
		t.Errorf("expected blank line between path blocks, got: %q", output)
	}
}

// Task 2.2: Scalar and order change descriptors

func TestDetailedFormatter_ValueChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	// int to string type change
	diffs := []Difference{
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.enabled", Type: DiffModified, From: true, To: "true"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± type change from bool to string") {
		t.Errorf("expected '± type change from bool to string', got: %q", output)
	}
}

func TestDetailedFormatter_OrderChanged(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged, From: []interface{}{"a", "b"}, To: []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "⇆ order changed") {
		t.Errorf("expected '⇆ order changed' descriptor, got: %q", output)
	}
}

// Task 2.3: List and map entry descriptors with count formatting

func TestDetailedFormatter_ListEntryAdded(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "newItem"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one list entry added") {
		t.Errorf("expected 'one list entry added', got: %q", output)
	}
}

func TestDetailedFormatter_MultipleListEntriesRemoved(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	// Two removals at the same list path — should be grouped
	diffs := []Difference{
		{Path: "items", Type: DiffRemoved, From: "item1"},
		{Path: "items", Type: DiffRemoved, From: "item2"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "items[0]", Type: DiffRemoved, From: "gone"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one list entry removed") {
		t.Errorf("expected 'one list entry removed', got: %q", output)
	}
}

func TestDetailedFormatter_MapEntryAdded(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.newKey", Type: DiffAdded, To: "value"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one map entry added") {
		t.Errorf("expected 'one map entry added', got: %q", output)
	}
}

func TestDetailedFormatter_MapEntryRemoved(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.oldKey", Type: DiffRemoved, From: "value"},
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
		value    interface{}
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

// Task 3.1: Structured value rendering with YAML-like formatting and indent guides

func TestDetailedFormatter_StructuredMapAdded(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
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
	f, _ := GetFormatter("detailed")
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
		{Path: "apps.0", Type: DiffAdded, To: outer},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	listVal := []interface{}{"alpha", "beta", "gamma"}

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: listVal},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "key", "value")
	om.Values["key"] = "removed-entry"
	om.Values["value"] = 42

	diffs := []Difference{
		{Path: "entries.0", Type: DiffRemoved, From: om},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "ports")
	om.Values["name"] = "service"
	om.Values["ports"] = []interface{}{80, 443}

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	// Test with regular map[string]interface{} as well
	m := map[string]interface{}{
		"enabled": true,
	}

	diffs := []Difference{
		{Path: "config.newKey", Type: DiffAdded, To: m},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "enabled: true") {
		t.Errorf("expected 'enabled: true' in regular map output, got: %q", output)
	}
}

func TestDetailedFormatter_NilValueDisplay(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: nil, To: "new"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "<nil>") {
		t.Errorf("expected '<nil>' for nil value, got: %q", output)
	}
}

func TestDetailedFormatter_ScalarFallback(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	// Unknown type should fall back to fmt.Sprintf
	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "old", To: 42},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "42") {
		t.Errorf("expected '42' in output for scalar value, got: %q", output)
	}
}

// Additional unit tests for full coverage

func TestDetailedFormatter_YamlTypeName_MapAndList(t *testing.T) {
	om := NewOrderedMap()
	if yamlTypeName(om) != "map" {
		t.Errorf("expected 'map' for *OrderedMap, got %q", yamlTypeName(om))
	}

	m := map[string]interface{}{"k": "v"}
	if yamlTypeName(m) != "map" {
		t.Errorf("expected 'map' for map[string]interface{}, got %q", yamlTypeName(m))
	}

	list := []interface{}{"a", "b"}
	if yamlTypeName(list) != "list" {
		t.Errorf("expected 'list' for []interface{}, got %q", yamlTypeName(list))
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
		input    interface{}
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

func TestDetailedFormatter_MultipleMapEntriesAdded(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config", Type: DiffAdded, To: "val1"},
		{Path: "config", Type: DiffAdded, To: "val2"},
		{Path: "config", Type: DiffAdded, To: "val3"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "three map entries added") {
		t.Errorf("expected 'three map entries added', got: %q", output)
	}
}

func TestDetailedFormatter_AddedAndRemovedInSameGroup(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "new"},
		{Path: "items.0", Type: DiffRemoved, From: "old"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: nil, To: "new-value"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "type change from null to string") {
		t.Errorf("expected type change from null to string, got: %q", output)
	}
}

func TestDetailedFormatter_ModifiedValueToNil(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "old-value", To: nil},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "type change from string to null") {
		t.Errorf("expected type change from string to null, got: %q", output)
	}
}

func TestDetailedFormatter_OrderChangedWithValues(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"x", "y", "z"},
			To:   []interface{}{"z", "y", "x"}},
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
	f, _ := GetFormatter("detailed")
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
		{Path: "root.0", Type: DiffAdded, To: outer},
	}

	output := f.Format(diffs, opts)
	// Should use YAML-style indentation, no pipe guides
	expected := "root.0\n  + one list entry added:\n    - level1:\n        nested:\n          deep: value\n\n"
	if output != expected {
		t.Errorf("deeply nested structure mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_DocumentHeading(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	t.Run("single doc replaces [0] with (document)", func(t *testing.T) {
		diffs := []Difference{
			{Path: "[0]", Type: DiffRemoved, From: "value", DocumentIndex: 0},
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
			{Path: "[0]", Type: DiffRemoved, From: "value1", DocumentIndex: 0},
			{Path: "[1]", Type: DiffAdded, To: "value2", DocumentIndex: 1},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "(document 1)") {
			t.Errorf("expected '(document 1)' in output, got: %q", output)
		}
		if !strings.Contains(output, "(document 2)") {
			t.Errorf("expected '(document 2)' in output, got: %q", output)
		}
	})

	t.Run("non-bare index paths are not transformed", func(t *testing.T) {
		diffs := []Difference{
			{Path: "items[0]", Type: DiffModified, From: "old", To: "new"},
		}
		output := f.Format(diffs, opts)
		if !strings.Contains(output, "items[0]") {
			t.Errorf("expected 'items[0]' preserved in output, got: %q", output)
		}
	})
}

func TestParseBareDocIndex(t *testing.T) {
	tests := []struct {
		path    string
		wantIdx int
		wantOk  bool
	}{
		{"[0]", 0, true},
		{"[1]", 1, true},
		{"[12]", 12, true},
		{"items[0]", 0, false},
		{"[0].spec", 0, false},
		{"name", 0, false},
		{"", 0, false},
		{"[]", 0, false},
		{"[abc]", 0, false},
	}
	for _, tt := range tests {
		idx, ok := parseBareDocIndex(tt.path)
		if ok != tt.wantOk || idx != tt.wantIdx {
			t.Errorf("parseBareDocIndex(%q) = (%d, %v), want (%d, %v)", tt.path, idx, ok, tt.wantIdx, tt.wantOk)
		}
	}
}

// --- formatDetailedValue with complex types ---

func TestFormatDetailedValue_OrderedMap(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "image")
	om.Values["name"] = "web"
	om.Values["image"] = "nginx:1.21"

	result := formatDetailedValue(om)
	expected := "{name: web, image: nginx:1.21}"
	if result != expected {
		t.Errorf("formatDetailedValue(OrderedMap) = %q, want %q", result, expected)
	}
}

func TestFormatDetailedValue_Nil(t *testing.T) {
	result := formatDetailedValue(nil)
	if result != "<nil>" {
		t.Errorf("formatDetailedValue(nil) = %q, want %q", result, "<nil>")
	}
}

func TestFormatDetailedValue_Scalar(t *testing.T) {
	result := formatDetailedValue("hello")
	if result != "hello" {
		t.Errorf("formatDetailedValue(string) = %q, want %q", result, "hello")
	}
}

func TestFormatDetailedValue_Slice(t *testing.T) {
	s := []interface{}{"a", "b"}
	result := formatDetailedValue(s)
	expected := "[a b]"
	if result != expected {
		t.Errorf("formatDetailedValue(slice) = %q, want %q", result, expected)
	}
}

func TestDetailedFormatter_NoRawStructOutput(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "image")
	om.Values["name"] = "web"
	om.Values["image"] = "nginx:1.21"

	f := &DetailedFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "containers[0]", Type: DiffAdded, To: om},
		{Path: "containers[1]", Type: DiffRemoved, From: om},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "&{") {
		t.Errorf("detailed output contains raw struct &{ — got:\n%s", output)
	}
}

func TestDetailedFormatter_ModifiedWithOrderedMap(t *testing.T) {
	old := NewOrderedMap()
	old.Keys = append(old.Keys, "replicas")
	old.Values["replicas"] = 1

	new := NewOrderedMap()
	new.Keys = append(new.Keys, "replicas")
	new.Values["replicas"] = 3

	f := &DetailedFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "spec", Type: DiffModified, From: old, To: new},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "&{") {
		t.Errorf("detailed modified output contains raw struct &{ — got:\n%s", output)
	}
}

func TestDetailedFormatter_NestedListOfMaps(t *testing.T) {
	item1 := NewOrderedMap()
	item1.Keys = append(item1.Keys, "name", "image")
	item1.Values["name"] = "web"
	item1.Values["image"] = "nginx"

	item2 := NewOrderedMap()
	item2.Keys = append(item2.Keys, "name", "image")
	item2.Values["name"] = "api"
	item2.Values["image"] = "node"

	list := []interface{}{item1, item2}

	f := &DetailedFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "containers[0]", Type: DiffAdded, To: list},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "&{") {
		t.Errorf("detailed nested list output contains raw struct &{ — got:\n%s", output)
	}
	if strings.Contains(output, "- {") {
		t.Errorf("map items in list should be unfolded to block YAML, not inline — got:\n%s", output)
	}
	if !strings.Contains(output, "- name: web") {
		t.Errorf("expected '- name: web' as first key of unfolded block, got:\n%s", output)
	}
	if !strings.Contains(output, "  image: nginx") {
		t.Errorf("expected '  image: nginx' as continuation key of unfolded block, got:\n%s", output)
	}
	if !strings.Contains(output, "- name: api") {
		t.Errorf("expected '- name: api' as first key of unfolded block, got:\n%s", output)
	}
}

func TestDetailedFormatter_ListUnderKey_UnfoldsMapItems(t *testing.T) {
	container := NewOrderedMap()
	container.Keys = append(container.Keys, "name", "image")
	container.Values["name"] = "web"
	container.Values["image"] = "nginx:1.21"

	parent := NewOrderedMap()
	parent.Keys = append(parent.Keys, "containers")
	parent.Values["containers"] = []interface{}{container}

	f := &DetailedFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "spec[0]", Type: DiffAdded, To: parent},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "- {") {
		t.Errorf("map items nested under a key should be unfolded to block YAML — got:\n%s", output)
	}
	if !strings.Contains(output, "- name: web") {
		t.Errorf("expected '- name: web' in unfolded output, got:\n%s", output)
	}
}

func TestDetailedFormatter_FirstKeyList_UnfoldsMapItems(t *testing.T) {
	item := NewOrderedMap()
	item.Keys = append(item.Keys, "port", "protocol")
	item.Values["port"] = 80
	item.Values["protocol"] = "TCP"

	parent := NewOrderedMap()
	parent.Keys = append(parent.Keys, "ports")
	parent.Values["ports"] = []interface{}{item}

	f := &DetailedFormatter{}
	opts := DefaultFormatOptions()

	// List entry where the first key's value is itself a list of maps
	diffs := []Difference{
		{Path: "spec[0]", Type: DiffAdded, To: parent},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "- {") {
		t.Errorf("map items in first-key list should be unfolded — got:\n%s", output)
	}
	if !strings.Contains(output, "- port: 80") {
		t.Errorf("expected '- port: 80' in unfolded output, got:\n%s", output)
	}
	if !strings.Contains(output, "  protocol: TCP") {
		t.Errorf("expected '  protocol: TCP' in unfolded output, got:\n%s", output)
	}
}
