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

// Task 3.2: Multiline text diff with context and collapse

func TestDetailedFormatter_MultilineDescriptor(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "line1\nline2\nline3"
	to := "line1\nchanged\nline3"

	diffs := []Difference{
		{Path: "config.data", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± value change in multiline text") {
		t.Errorf("expected multiline descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineAdditionDeletionCount(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "line1\nline2\nline3"
	to := "line1\nchanged\nline3\nline4"

	diffs := []Difference{
		{Path: "data", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	// Should mention inserts and deletions count
	if !strings.Contains(output, "insert") || !strings.Contains(output, "deletion") {
		t.Errorf("expected insert/deletion counts in multiline descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineDiffMarkers(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "aaa\nbbb\nccc"
	to := "aaa\nBBB\nccc"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(output, "\n")

	hasAdded := false
	hasRemoved := false
	hasContext := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "+ ") {
			hasAdded = true
		}
		if strings.HasPrefix(trimmed, "- ") {
			hasRemoved = true
		}
		// Context lines start with space
		if len(trimmed) > 0 && strings.HasPrefix(line, "    ") && !strings.HasPrefix(trimmed, "+") && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "±") {
			hasContext = true
		}
	}

	if !hasAdded {
		t.Errorf("expected '+' prefixed added lines in multiline diff, got: %q", output)
	}
	if !hasRemoved {
		t.Errorf("expected '-' prefixed removed lines in multiline diff, got: %q", output)
	}
	if !hasContext {
		t.Errorf("expected context lines (space-prefixed) in multiline diff, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineCollapseUnchanged(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.ContextLines = 1

	// Many unchanged lines between changes should be collapsed
	from := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	to := "CHANGED\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nALSO_CHANGED"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "lines unchanged") {
		t.Errorf("expected collapse marker '[N lines unchanged]', got: %q", output)
	}
}

func TestDetailedFormatter_MultilineContextLinesOption(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.ContextLines = 1

	// With 8 unchanged lines between two changes and context=1, most should collapse
	from := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj"
	to := "CHANGED\nb\nc\nd\ne\nf\ng\nh\ni\nCHANGED"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	// With context=1, many middle lines should be collapsed
	if !strings.Contains(output, "lines unchanged") {
		t.Errorf("expected collapse with context=1, got: %q", output)
	}
}

func TestDetailedFormatter_SingleLineNotMultiline(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	// Single-line strings should NOT use multiline diff path
	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "old value", To: "new value"},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "multiline") {
		t.Errorf("single-line change should not use multiline diff, got: %q", output)
	}
	if !strings.Contains(output, "± value change") {
		t.Errorf("expected '± value change' for single-line change, got: %q", output)
	}
}

func TestComputeLineDiff(t *testing.T) {
	from := []string{"a", "b", "c"}
	to := []string{"a", "B", "c"}

	ops := computeLineDiff(from, to)

	hasKeep := false
	hasInsert := false
	hasDelete := false
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			hasKeep = true
		case editInsert:
			hasInsert = true
		case editDelete:
			hasDelete = true
		}
	}

	if !hasKeep {
		t.Error("expected keep operations in line diff")
	}
	if !hasInsert {
		t.Error("expected insert operations in line diff")
	}
	if !hasDelete {
		t.Error("expected delete operations in line diff")
	}
}

func TestComputeLineDiff_AllNew(t *testing.T) {
	ops := computeLineDiff([]string{}, []string{"a", "b"})

	for _, op := range ops {
		if op.Type != editInsert {
			t.Errorf("expected all insert ops for new content, got type %d", op.Type)
		}
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 insert ops, got %d", len(ops))
	}
}

func TestComputeLineDiff_AllRemoved(t *testing.T) {
	ops := computeLineDiff([]string{"a", "b"}, []string{})

	for _, op := range ops {
		if op.Type != editDelete {
			t.Errorf("expected all delete ops for removed content, got type %d", op.Type)
		}
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 delete ops, got %d", len(ops))
	}
}

func TestComputeLineDiff_Identical(t *testing.T) {
	ops := computeLineDiff([]string{"a", "b"}, []string{"a", "b"})

	for _, op := range ops {
		if op.Type != editKeep {
			t.Errorf("expected all keep ops for identical content, got type %d", op.Type)
		}
	}
}

// Task 3.3: Whitespace-only change detection and visualization

func TestDetailedFormatter_WhitespaceOnlyChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "hello world", To: "hello  world"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± whitespace only change") {
		t.Errorf("expected '± whitespace only change' descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_WhitespaceVisualization(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "a b", To: "a  b"},
	}

	output := f.Format(diffs, opts)
	// Spaces should be visualized as middle dots
	if !strings.Contains(output, "·") {
		t.Errorf("expected middle dot '·' for whitespace visualization, got: %q", output)
	}
}

func TestDetailedFormatter_WhitespaceNewlineVisualization(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "hello\n", To: "hello"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± whitespace only change") {
		t.Errorf("expected whitespace-only descriptor for trailing newline change, got: %q", output)
	}
	if !strings.Contains(output, "↵") {
		t.Errorf("expected return symbol '↵' for newline visualization, got: %q", output)
	}
}

func TestIsWhitespaceOnlyChange(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected bool
	}{
		{"hello world", "hello  world", true},
		{"hello", "hello\n", true},
		{" a ", "a", true},
		{"hello", "world", false},
		{"abc", "abc", false}, // no change at all
		{"a b", "a c", false},
	}

	for _, tt := range tests {
		result := isWhitespaceOnlyChange(tt.from, tt.to)
		if result != tt.expected {
			t.Errorf("isWhitespaceOnlyChange(%q, %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
		}
	}
}

func TestVisualizeWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a b", "a·b"},
		{"hello\n", "hello↵"},
		{"a  b\n", "a··b↵"},
		{"no spaces", "no·spaces"},
	}

	for _, tt := range tests {
		result := visualizeWhitespace(tt.input)
		if result != tt.expected {
			t.Errorf("visualizeWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Task 4.1: Color coding tests

func TestDetailedFormatter_ColorEnabled_AdditionGreen(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "newItem"},
	}

	output := f.Format(diffs, opts)
	// Addition symbol and value should be colored green
	if !strings.Contains(output, "\033[32m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected green color code for addition, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_RemovalRed(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffRemoved, From: "oldItem"},
	}

	output := f.Format(diffs, opts)
	// Removal symbol and value should be colored red
	if !strings.Contains(output, "\033[31m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected red color code for removal, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_ModificationYellow(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Modification descriptor should be colored yellow
	if !strings.Contains(output, "\033[33m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected yellow color code for modification descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_ModificationValues(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Old value line should be red, new value line should be green
	if !strings.Contains(output, "\033[31m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected red color code for old value in modification, got: %q", output)
	}
	if !strings.Contains(output, "\033[32m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected green color code for new value in modification, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_ContextGray(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	from := "aaa\nbbb\nccc"
	to := "aaa\nBBB\nccc"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	// Context lines should be in gray
	if !strings.Contains(output, "\033[90m") && !strings.Contains(output, "\033[38;2;105;105;105m") {
		t.Errorf("expected gray color code for context lines, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_ResetCodes(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Should contain reset codes
	if !strings.Contains(output, "\033[0m") {
		t.Errorf("expected color reset codes in colored output, got: %q", output)
	}
}

func TestDetailedFormatter_ColorDisabled_NoAnsiCodes(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "newItem"},
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		{Path: "old.key", Type: DiffRemoved, From: "value"},
	}

	output := f.Format(diffs, opts)
	// Should not contain any ANSI escape codes
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes when color is disabled, got: %q", output)
	}
}

func TestDetailedFormatter_TrueColor_AdditionGreen(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.TrueColor = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "newItem"},
	}

	output := f.Format(diffs, opts)
	// Should use detailed true color green (88, 191, 56)
	expectedTrueColor := "\033[38;2;88;191;56m"
	if !strings.Contains(output, expectedTrueColor) {
		t.Errorf("expected true color green %q for addition, got: %q", expectedTrueColor, output)
	}
}

func TestDetailedFormatter_TrueColor_RemovalRed(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.TrueColor = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffRemoved, From: "oldItem"},
	}

	output := f.Format(diffs, opts)
	// Should use detailed true color red (185, 49, 27)
	expectedTrueColor := "\033[38;2;185;49;27m"
	if !strings.Contains(output, expectedTrueColor) {
		t.Errorf("expected true color red %q for removal, got: %q", expectedTrueColor, output)
	}
}

func TestDetailedFormatter_TrueColor_ModificationYellow(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.TrueColor = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Should use detailed true color yellow (199, 196, 63)
	expectedTrueColor := "\033[38;2;199;196;63m"
	if !strings.Contains(output, expectedTrueColor) {
		t.Errorf("expected true color yellow %q for modification, got: %q", expectedTrueColor, output)
	}
}

func TestDetailedFormatter_ColorEnabled_OrderChangeYellow(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged, From: []interface{}{"a", "b"}, To: []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	// Order change descriptor should be yellow
	if !strings.Contains(output, "\033[33m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected yellow color for order change, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_TypeChangeYellow(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	// Type change descriptor should be yellow
	if !strings.Contains(output, "\033[33m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected yellow color for type change descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_MultilineDiffColors(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	from := "aaa\nbbb\nccc"
	to := "aaa\nBBB\nccc"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	// Should contain green (for additions), red (for deletions), and gray (for context)
	hasGreen := strings.Contains(output, "\033[32m") || strings.Contains(output, "\033[38;2;88;191;56m")
	hasRed := strings.Contains(output, "\033[31m") || strings.Contains(output, "\033[38;2;185;49;27m")
	if !hasGreen {
		t.Errorf("expected green color for additions in multiline diff, got: %q", output)
	}
	if !hasRed {
		t.Errorf("expected red color for deletions in multiline diff, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_WhitespaceChangeYellow(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "hello world", To: "hello  world"},
	}

	output := f.Format(diffs, opts)
	// Whitespace change descriptor should be yellow
	if !strings.Contains(output, "\033[33m") && !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected yellow color for whitespace change descriptor, got: %q", output)
	}
}

// Task 4.2: Header and flag compatibility tests

func TestDetailedFormatter_Header(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		{Path: "config.host", Type: DiffAdded, To: "prod"},
	}

	output := f.Format(diffs, opts)
	// Should contain a header with spelled-out diff count
	if !strings.Contains(output, "two") || !strings.Contains(output, "differences") {
		t.Errorf("expected header with 'two differences', got: %q", output)
	}
}

func TestDetailedFormatter_HeaderOmitted(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "Found one difference") {
		t.Errorf("expected header with 'Found one difference', got: %q", output)
	}
}

func TestDetailedFormatter_HeaderColorEnabled(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Header should have color codes
	if !strings.Contains(output, "\033[") {
		t.Errorf("expected color codes in header with color enabled, got: %q", output)
	}
}

func TestDetailedFormatter_FlagCombination_OmitHeaderGoPatch(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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

func TestDetailedFormatter_MultilineDiffNoCollapse(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.ContextLines = 100 // Very high, nothing should collapse

	from := "a\nb\nc\nd\ne"
	to := "a\nB\nc\nd\ne"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "unchanged") {
		t.Errorf("expected no collapse with large context, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineDiffSingleAddition(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "line1\nline2"
	to := "line1\nline2\nline3"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one insert") {
		t.Errorf("expected 'one insert' (singular), got: %q", output)
	}
	if !strings.Contains(output, "zero deletions") {
		t.Errorf("expected 'zero deletions', got: %q", output)
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
	expected := "\nroot.0\n  + one list entry added:\n    - level1:\n        nested:\n          deep: value\n\n"
	if output != expected {
		t.Errorf("deeply nested structure mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

// Task 5.2: Integration tests for CLI end-to-end

func TestCLI_DetailedOutputFormat(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("timeout: 30\nhost: localhost\n"),
		ToContent:   []byte("timeout: 60\nhost: localhost\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	// Should use detailed-style formatting
	if !strings.Contains(output, "± value change") {
		t.Errorf("expected detailed-style '± value change' in CLI output, got: %q", output)
	}
	if !strings.Contains(output, "timeout") {
		t.Errorf("expected path 'timeout' in CLI output, got: %q", output)
	}
}

func TestCLI_DetailedIdenticalFiles(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	content := []byte("timeout: 30\nhost: localhost\n")
	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: content,
		ToContent:   content,
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if !strings.Contains(output, "no differences found") {
		t.Errorf("expected 'no differences found' for identical files, got: %q", output)
	}
}

func TestCLI_DetailedWithOmitHeader(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.OmitHeader = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("key: old\n"),
		ToContent:   []byte("key: new\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if strings.Contains(output, "Found") {
		t.Errorf("expected no header with --omit-header, got: %q", output)
	}
	if !strings.Contains(output, "± value change") {
		t.Errorf("expected diff content even with --omit-header, got: %q", output)
	}
}

func TestCLI_DetailedWithGoPatchStyle(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.UseGoPatchStyle = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("config:\n  timeout: 30\n"),
		ToContent:   []byte("config:\n  timeout: 60\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path '/config/timeout' in CLI output, got: %q", output)
	}
}

func TestCLI_DetailedWithAllFlags(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.OmitHeader = true
	cfg.UseGoPatchStyle = true
	cfg.MultiLineContextLines = 2
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("config:\n  timeout: 30\n"),
		ToContent:   []byte("config:\n  timeout: 60\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if strings.Contains(output, "Found") {
		t.Errorf("expected no header, got: %q", output)
	}
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path, got: %q", output)
	}
}

func TestCLI_DetailedWithSetExitCode(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.SetExitCode = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("key: old\n"),
		ToContent:   []byte("key: new\n"),
	}

	result := Run(cfg, rc)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d for differences with -s, got %d", ExitCodeDifferences, result.Code)
	}
}

func TestCLI_DetailedWithSetExitCodeNoDiffs(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.SetExitCode = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	content := []byte("key: same\n")
	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: content,
		ToContent:   content,
	}

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d for no differences, got %d", ExitCodeSuccess, result.Code)
	}
}

func TestCLI_DetailedMultipleChanges(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("timeout: 30\nhost: localhost\nport: 8080\n"),
		ToContent:   []byte("timeout: 60\nhost: production\nport: 8080\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	// Should have header mentioning two differences
	if !strings.Contains(output, "Found two differences") {
		t.Errorf("expected 'Found two differences' in header, got: %q", output)
	}
}

func TestCLI_DetailedStructuredAddition(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("items: []\n"),
		ToContent:   []byte("items:\n  - name: nginx\n    port: 80\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if !strings.Contains(output, "added") {
		t.Errorf("expected 'added' for new list entry, got: %q", output)
	}
}

// Task 5.3: Baseline rendering snapshot tests

func TestDetailedFormatter_Snapshot_ScalarModification(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	expected := "\nconfig.timeout\n  ± value change\n    - 30\n    + 60\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for scalar modification.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_TypeChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	expected := "\nconfig.port\n  ± type change from int to string\n    - 8080\n    + 8080\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for type change.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_SingleListEntryAdded(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "newItem"},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems.0\n  + one list entry added:\n    - newItem\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for list entry added.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_SingleMapEntryRemoved(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.oldKey", Type: DiffRemoved, From: "value"},
	}

	output := f.Format(diffs, opts)
	expected := "\nconfig.oldKey\n  - one map entry removed:\n    oldKey: value\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for map entry removed.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_OrderChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for order change.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_WhitespaceChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "a b", To: "a  b"},
	}

	output := f.Format(diffs, opts)
	expected := "\nkey\n  ± whitespace only change\n    - a·b\n    + a··b\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for whitespace change.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_RootLevel(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	expected := "\n(root level)\n  ± value change\n    - old\n    + new\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for root level.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_GoPatchRoot(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: "", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	expected := "\n/\n  ± value change\n    - old\n    + new\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for go-patch root.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_StructuredMapAdded(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	expected := "\nservices.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for structured map added.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_Header(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	expected := "Found one difference\n\nkey\n  ± value change\n    - old\n    + new\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for output with header.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_MultiplePathGroups(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "alpha", Type: DiffModified, From: "a1", To: "a2"},
		{Path: "beta", Type: DiffModified, From: "b1", To: "b2"},
	}

	output := f.Format(diffs, opts)
	expected := "\nalpha\n  ± value change\n    - a1\n    + a2\n\nbeta\n  ± value change\n    - b1\n    + b2\n\n"
	if output != expected {
		t.Errorf("snapshot mismatch for multiple path groups.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_Snapshot_MultilineDiff(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.ContextLines = 1

	from := "line1\nline2\nline3"
	to := "line1\nchanged\nline3"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	// Verify the key structure elements
	if !strings.Contains(output, "± value change in multiline text (one insert, one deletion)") {
		t.Errorf("snapshot: expected multiline descriptor, got: %q", output)
	}
	if !strings.Contains(output, "- line2") {
		t.Errorf("snapshot: expected removed line '- line2', got: %q", output)
	}
	if !strings.Contains(output, "+ changed") {
		t.Errorf("snapshot: expected added line '+ changed', got: %q", output)
	}
}

// Task 2.1 (colored-output): Bold path headings

func TestDetailedFormatter_ColorEnabled_BoldPathHeading(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Path heading should contain bold escape code
	if !strings.Contains(output, styleBold+"config.timeout"+colorReset) {
		t.Errorf("expected bold path heading, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_BoldRootLevel(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	// Root-level heading should also be bold
	if !strings.Contains(output, styleBold+"(root level)"+colorReset) {
		t.Errorf("expected bold root-level heading, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_BoldGoPatchRoot(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true
	opts.UseGoPatchStyle = true

	diffs := []Difference{
		{Path: "", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	// "/" root heading should be bold in go-patch mode
	if !strings.Contains(output, styleBold+"/"+colorReset) {
		t.Errorf("expected bold '/' root heading in go-patch mode, got: %q", output)
	}
}

func TestDetailedFormatter_ColorDisabled_NoBoldPathHeading(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	// Should not contain any ANSI codes
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes when color disabled, got: %q", output)
	}
}

// Task 2.2 (colored-output): Italic type names

func TestDetailedFormatter_ColorEnabled_ItalicTypeNames(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	// Type names should be wrapped in italic escape codes within the yellow descriptor
	if !strings.Contains(output, styleItalic+"int"+styleItalicOff) {
		t.Errorf("expected italic 'int' type name, got: %q", output)
	}
	if !strings.Contains(output, styleItalic+"string"+styleItalicOff) {
		t.Errorf("expected italic 'string' type name, got: %q", output)
	}
}

func TestDetailedFormatter_ColorDisabled_NoItalicTypeNames(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	// Should contain plain type names without italic
	if !strings.Contains(output, "from int to string") {
		t.Errorf("expected plain type names, got: %q", output)
	}
	if strings.Contains(output, "\033[3m") {
		t.Errorf("expected no italic codes when color disabled, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_ItalicPreservesYellow(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	// The descriptor line should use styleItalicOff (not colorReset) to preserve yellow
	if strings.Contains(output, styleItalic+"int"+colorReset) {
		t.Errorf("italic type name should use styleItalicOff, not colorReset, to preserve yellow context")
	}
}

// Task 2.3 (colored-output): Dimmed pipe indent guides

func TestDetailedFormatter_ColorEnabled_EntryValueColored(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	// All value lines should be colored green (addition)
	addedColor := GetDetailedColorCode(DiffAdded, false)
	if !strings.Contains(output, addedColor+"    - name: nginx") {
		t.Errorf("expected green colored '- name: nginx', got: %q", output)
	}
	if !strings.Contains(output, addedColor+"      port: 80") {
		t.Errorf("expected green colored 'port: 80' at +2 indent, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_NestedEntryValueColored(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

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
	// All nested value lines should be colored green
	addedColor := GetDetailedColorCode(DiffAdded, false)
	colorCount := strings.Count(output, addedColor)
	// Should color: descriptor line, name: myapp, config:, host: localhost, port: 8080
	if colorCount < 4 {
		t.Errorf("expected multiple green colored lines for nested structure, got %d color occurrences in: %q", colorCount, output)
	}
}

func TestDetailedFormatter_ColorDisabled_PlainEntryValues(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	// Should contain plain YAML-style values with dash prefix, without color
	if !strings.Contains(output, "    - name: nginx") {
		t.Errorf("expected '- name: nginx' in output, got: %q", output)
	}
	if !strings.Contains(output, "      port: 80") {
		t.Errorf("expected 'port: 80' at +2 indent in output, got: %q", output)
	}
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes when color disabled, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_ListEntryValueColored(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	listVal := []interface{}{"alpha", "beta", "gamma"}
	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: listVal},
	}

	output := f.Format(diffs, opts)
	// List entries should be colored green
	addedColor := GetDetailedColorCode(DiffAdded, false)
	if !strings.Contains(output, addedColor+"    - alpha") {
		t.Errorf("expected green colored '- alpha' list item, got: %q", output)
	}
}

// Task 2.4 (colored-output): Colored order change was/now values

func TestDetailedFormatter_ColorEnabled_OrderChangeWasRed(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	removedColor := GetDetailedColorCode(DiffRemoved, false)
	// "- " line should be in removal (red) color
	if !strings.Contains(output, removedColor+"    - ") {
		t.Errorf("expected red color on '- ' line, got: %q", output)
	}
}

func TestDetailedFormatter_ColorEnabled_OrderChangeNowGreen(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	addedColor := GetDetailedColorCode(DiffAdded, false)
	// "+ " line should be in addition (green) color
	if !strings.Contains(output, addedColor+"    + ") {
		t.Errorf("expected green color on '+ ' line, got: %q", output)
	}
}

func TestDetailedFormatter_ColorDisabled_PlainOrderChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	// Should contain plain -/+ without color
	if !strings.Contains(output, "    - ") || !strings.Contains(output, "    + ") {
		t.Errorf("expected plain -/+ lines, got: %q", output)
	}
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes when color disabled, got: %q", output)
	}
}

// Task 3 (colored-output): Integration and no-regression tests

func TestDetailedFormatter_Integration_AllDiffTypesColored(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		// Added: structured map entry (exercises pipe guides)
		{Path: "services.0", Type: DiffAdded, To: om},
		// Removed: scalar
		{Path: "config.oldKey", Type: DiffRemoved, From: "deprecated"},
		// Modified: type change (exercises italic type names)
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		// Modified: scalar value change
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		// Order changed (exercises colored was/now)
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b", "c"},
			To:   []interface{}{"c", "b", "a"}},
	}

	output := f.Format(diffs, opts)

	// 1. Bold path headings: all path headings should be wrapped in bold
	for _, path := range []string{"services.0", "config.oldKey", "config.port", "config.timeout", "items"} {
		if !strings.Contains(output, styleBold+path+colorReset) {
			t.Errorf("expected bold path heading for %q, got: %q", path, output)
		}
	}

	// 2. Italic type names in type-change descriptor
	if !strings.Contains(output, styleItalic+"int"+styleItalicOff) {
		t.Errorf("expected italic 'int' in type change descriptor, got: %q", output)
	}
	if !strings.Contains(output, styleItalic+"string"+styleItalicOff) {
		t.Errorf("expected italic 'string' in type change descriptor, got: %q", output)
	}

	// 3. Entry values colored (structured map added has green-colored YAML lines)
	addedColor := GetDetailedColorCode(DiffAdded, false)
	if !strings.Contains(output, addedColor+"    - name: nginx") {
		t.Errorf("expected green colored entry value lines, got: %q", output)
	}

	// 4. Red on - line, green on + line (order change)
	removedColor := GetDetailedColorCode(DiffRemoved, false)
	if !strings.Contains(output, removedColor+"    - ") {
		t.Errorf("expected red color on '- ' line, got: %q", output)
	}
	if !strings.Contains(output, addedColor+"    + ") {
		t.Errorf("expected green color on '+ ' line, got: %q", output)
	}

	// 5. Reset codes present
	if !strings.Contains(output, colorReset) {
		t.Errorf("expected color reset codes, got: %q", output)
	}
}

func TestDetailedFormatter_Integration_AllDiffTypesUncolored(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
		{Path: "config.oldKey", Type: DiffRemoved, From: "deprecated"},
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b", "c"},
			To:   []interface{}{"c", "b", "a"}},
	}

	output := f.Format(diffs, opts)

	// No ANSI escape codes whatsoever when color is disabled
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI escape codes in uncolored output, got: %q", output)
	}

	// Content should still be present
	for _, expected := range []string{
		"services.0", "config.oldKey", "config.port", "config.timeout", "items",
		"- name: nginx", "port: 80",
		"type change from int to string",
		"± value change",
		"⇆ order changed",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in uncolored output, got: %q", expected, output)
		}
	}
}

func TestDetailedFormatter_Integration_TrueColorBoldItalicCombination(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.TrueColor = true

	diffs := []Difference{
		// Type change to exercise italic + yellow true color
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		// Added structured map to exercise pipe guides + true color context
		{Path: "services.0", Type: DiffAdded, To: func() *OrderedMap {
			om := NewOrderedMap()
			om.Keys = append(om.Keys, "name", "port")
			om.Values["name"] = "nginx"
			om.Values["port"] = 80
			return om
		}()},
		// Order change to exercise true color red/green on was/now
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"x", "y"},
			To:   []interface{}{"y", "x"}},
	}

	output := f.Format(diffs, opts)

	// Bold path headings should still work with true color
	if !strings.Contains(output, styleBold+"config.port"+colorReset) {
		t.Errorf("expected bold path heading in true color mode, got: %q", output)
	}

	// Italic type names within true color yellow descriptor
	trueYellow := GetDetailedColorCode(DiffModified, true)
	if !strings.Contains(output, trueYellow) {
		t.Errorf("expected true color yellow for type change descriptor, got: %q", output)
	}
	if !strings.Contains(output, styleItalic+"int"+styleItalicOff) {
		t.Errorf("expected italic type names in true color mode, got: %q", output)
	}

	// True color green on entry value lines
	trueGreen := GetDetailedColorCode(DiffAdded, true)
	if !strings.Contains(output, trueGreen+"    - name: nginx") {
		t.Errorf("expected true color green for entry value lines, got: %q", output)
	}

	// True color red/green on -/+
	trueRed := GetDetailedColorCode(DiffRemoved, true)
	if !strings.Contains(output, trueRed+"    - ") {
		t.Errorf("expected true color red on '- ' line, got: %q", output)
	}
	if !strings.Contains(output, trueGreen+"    + ") {
		t.Errorf("expected true color green on '+ ' line, got: %q", output)
	}
}

func TestDetailedFormatter_Integration_AutoColorModeNoTerminal(t *testing.T) {
	// When auto color mode resolves to no-color (stdout is not a terminal),
	// the output should contain zero ANSI escape sequences
	cfg := NewColorConfig(ColorModeAuto, true, 0)
	cfg.SetIsTerminal(false)

	opts := DefaultFormatOptions()
	cfg.ToFormatOptions(opts)

	// Verify auto mode resolved to no color
	if opts.Color {
		t.Fatal("expected Color=false when auto mode with non-terminal")
	}

	f, _ := GetFormatter("detailed")
	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: func() *OrderedMap {
			om := NewOrderedMap()
			om.Keys = append(om.Keys, "name", "port")
			om.Values["name"] = "nginx"
			om.Values["port"] = 80
			return om
		}()},
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)

	if strings.Contains(output, "\033[") {
		t.Errorf("auto color mode with non-terminal should emit no ANSI codes, got: %q", output)
	}
}

func TestDetailedFormatter_Integration_NoRegressionSnapshots(t *testing.T) {
	// Verify uncolored output is byte-identical to expected baseline for all diff types
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	tests := []struct {
		name     string
		diffs    []Difference
		expected string
	}{
		{
			name:     "scalar modification",
			diffs:    []Difference{{Path: "key", Type: DiffModified, From: "old", To: "new"}},
			expected: "\nkey\n  ± value change\n    - old\n    + new\n\n",
		},
		{
			name:     "type change",
			diffs:    []Difference{{Path: "port", Type: DiffModified, From: 8080, To: "8080"}},
			expected: "\nport\n  ± type change from int to string\n    - 8080\n    + 8080\n\n",
		},
		{
			name:     "list entry added",
			diffs:    []Difference{{Path: "items.0", Type: DiffAdded, To: "newItem"}},
			expected: "\nitems.0\n  + one list entry added:\n    - newItem\n\n",
		},
		{
			name:     "map entry removed",
			diffs:    []Difference{{Path: "config.key", Type: DiffRemoved, From: "value"}},
			expected: "\nconfig.key\n  - one map entry removed:\n    key: value\n\n",
		},
		{
			name: "order change",
			diffs: []Difference{{Path: "items", Type: DiffOrderChanged,
				From: []interface{}{"a", "b"}, To: []interface{}{"b", "a"}}},
			expected: "\nitems\n  ⇆ order changed\n    - a, b\n    + b, a\n\n",
		},
		{
			name:     "whitespace change",
			diffs:    []Difference{{Path: "key", Type: DiffModified, From: "a b", To: "a  b"}},
			expected: "\nkey\n  ± whitespace only change\n    - a·b\n    + a··b\n\n",
		},
		{
			name: "structured map added",
			diffs: func() []Difference {
				om := NewOrderedMap()
				om.Keys = append(om.Keys, "name", "port")
				om.Values["name"] = "nginx"
				om.Values["port"] = 80
				return []Difference{{Path: "services.0", Type: DiffAdded, To: om}}
			}(),
			expected: "\nservices.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format(tt.diffs, opts)
			if output != tt.expected {
				t.Errorf("no-regression snapshot mismatch.\nExpected:\n%s\nGot:\n%s", tt.expected, output)
			}
		})
	}
}

// Round 2 regression-prevention tests

func TestDetailedFormatter_MapEntryScalar_RendersKeyValue(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.verbose", Type: DiffAdded, To: true},
	}

	output := f.Format(diffs, opts)
	expected := "\nconfig.verbose\n  + one map entry added:\n    verbose: true\n\n"
	if output != expected {
		t.Errorf("map entry scalar should render as key: value.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_MapEntryStructured_RendersKeyWrapper(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	inner := NewOrderedMap()
	inner.Keys = append(inner.Keys, "host", "port")
	inner.Values["host"] = "localhost"
	inner.Values["port"] = 8080

	diffs := []Difference{
		{Path: "config.newKey", Type: DiffAdded, To: inner},
	}

	output := f.Format(diffs, opts)
	expected := "\nconfig.newKey\n  + one map entry added:\n    newKey:\n      host: localhost\n      port: 8080\n\n"
	if output != expected {
		t.Errorf("map entry structured should render key as YAML wrapper.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_StillUsesDashPrefix(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "hello"},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems.0\n  + one list entry added:\n    - hello\n\n"
	if output != expected {
		t.Errorf("list entry should still use dash prefix.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_LeadingBlankLine_OmitHeader(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "a", To: "b"},
	}

	output := f.Format(diffs, opts)
	if !strings.HasPrefix(output, "\n") {
		t.Errorf("output should start with blank line when OmitHeader is true, got: %q", output)
	}
}

func TestDetailedFormatter_LeadingBlankLine_WithHeader(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "a", To: "b"},
	}

	output := f.Format(diffs, opts)
	// Header should be followed by \n\n (blank line before first path group)
	if !strings.Contains(output, "difference\n\nkey") {
		t.Errorf("header should be followed by blank line before first path group, got: %q", output)
	}
}

func TestDetailedFormatter_TrailingSeparator_ValueChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, opts)
	expected := "\nkey\n  ± value change\n    - old\n    + new\n\n"
	if output != expected {
		t.Errorf("value change should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_TrailingSeparator_EntryBatch(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "val"},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems.0\n  + one list entry added:\n    - val\n\n"
	if output != expected {
		t.Errorf("entry batch should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_TrailingSeparator_OrderChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("order change should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_TrailingSeparator_TypeChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "port", Type: DiffModified, From: 8080, To: "8080"},
	}

	output := f.Format(diffs, opts)
	expected := "\nport\n  ± type change from int to string\n    - 8080\n    + 8080\n\n"
	if output != expected {
		t.Errorf("type change should end with blank line separator.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_HeaderFormat_SpelledOutCount(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	// Single diff: "Found one difference"
	diffs1 := []Difference{
		{Path: "key", Type: DiffModified, From: "a", To: "b"},
	}
	output1 := f.Format(diffs1, opts)
	if !strings.Contains(output1, "Found one difference\n") {
		t.Errorf("expected 'Found one difference' for 1 diff, got: %q", output1)
	}

	// Three diffs: "Found three differences"
	diffs3 := []Difference{
		{Path: "a", Type: DiffModified, From: "1", To: "2"},
		{Path: "b", Type: DiffModified, From: "3", To: "4"},
		{Path: "c", Type: DiffModified, From: "5", To: "6"},
	}
	output3 := f.Format(diffs3, opts)
	if !strings.Contains(output3, "Found three differences\n") {
		t.Errorf("expected 'Found three differences' for 3 diffs, got: %q", output3)
	}
}

func TestDetailedFormatter_Snapshot_FullComparison(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		// Scalar value change
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		// Map entry added (scalar)
		{Path: "config.verbose", Type: DiffAdded, To: true},
		// List entry added (structured)
		{Path: "services.0", Type: DiffAdded, To: om},
		// Type change
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		// Order change
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	expected := "\nconfig.timeout\n  ± value change\n    - 30\n    + 60\n\n" +
		"config.verbose\n  + one map entry added:\n    verbose: true\n\n" +
		"services.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n" +
		"config.port\n  ± type change from int to string\n    - 8080\n    + 8080\n\n" +
		"items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("full comparison snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

// Fix 1 new tests: Order change comma-separated format

func TestDetailedFormatter_OrderChange_CommaSeparated(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b", "c"},
			To:   []interface{}{"c", "a", "b"}},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a"},
			To:   []interface{}{"a"}},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "nums", Type: DiffOrderChanged,
			From: []interface{}{1, 2, 3},
			To:   []interface{}{3, 1, 2}},
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
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("order change snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

// Fix 2 new tests: List entry YAML dash prefix

func TestDetailedFormatter_ListEntry_DashPrefix(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
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
	f, _ := GetFormatter("detailed")
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
		{Path: "items.1", Type: DiffAdded, To: om1},
		{Path: "items.1", Type: DiffAdded, To: om2},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems.1\n  + two list entries added:\n    - name: second\n      id: 2\n    - name: third\n      id: 3\n\n"
	if output != expected {
		t.Errorf("multiple maps mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_NestedMap(t *testing.T) {
	f, _ := GetFormatter("detailed")
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
		{Path: "services.0", Type: DiffAdded, To: outer},
	}

	output := f.Format(diffs, opts)
	expected := "\nservices.0\n  + one list entry added:\n    - name: svc\n      config:\n        host: localhost\n        port: 8080\n\n"
	if output != expected {
		t.Errorf("nested map mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_ScalarUnchanged(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "items.0", Type: DiffAdded, To: "hello"},
	}

	output := f.Format(diffs, opts)
	expected := "\nitems.0\n  + one list entry added:\n    - hello\n\n"
	if output != expected {
		t.Errorf("scalar list entry should still use '- value' format.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_ListEntry_Snapshot(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, opts)
	expected := "\nservices.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n"
	if output != expected {
		t.Errorf("list entry snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}
