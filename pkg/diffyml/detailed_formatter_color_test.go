package diffyml

import (
	"strings"
	"testing"
)

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
