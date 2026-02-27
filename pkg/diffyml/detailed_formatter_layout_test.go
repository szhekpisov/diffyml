package diffyml

import (
	"strings"
	"testing"
)

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

// Round 2 regression-prevention tests

func TestDetailedFormatter_MapEntryScalar_RendersKeyValue(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.verbose", Type: DiffAdded, To: true},
	}

	output := f.Format(diffs, opts)
	expected := "config.verbose\n  + one map entry added:\n    verbose: true\n\n"
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
	expected := "config.newKey\n  + one map entry added:\n    newKey:\n      host: localhost\n      port: 8080\n\n"
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
	expected := "items.0\n  + one list entry added:\n    - hello\n\n"
	if output != expected {
		t.Errorf("list entry should still use dash prefix.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func TestDetailedFormatter_NoLeadingBlankLine_OmitHeader(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "a", To: "b"},
	}

	output := f.Format(diffs, opts)
	if strings.HasPrefix(output, "\n") {
		t.Errorf("output should NOT start with blank line when OmitHeader is true, got: %q", output)
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
	expected := "key\n  ± value change\n    - old\n    + new\n\n"
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
	expected := "items.0\n  + one list entry added:\n    - val\n\n"
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
	expected := "items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
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
	expected := "port\n  ± type change from int to string\n    - 8080\n    + 8080\n\n"
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
	expected := "items.1\n  + two list entries added:\n    - name: second\n      id: 2\n    - name: third\n      id: 3\n\n"
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
	expected := "services.0\n  + one list entry added:\n    - name: svc\n      config:\n        host: localhost\n        port: 8080\n\n"
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
	expected := "items.0\n  + one list entry added:\n    - hello\n\n"
	if output != expected {
		t.Errorf("scalar list entry should still use '- value' format.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}
