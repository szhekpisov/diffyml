package diffyml

import (
	"strings"
	"testing"
)

// Task 5.3: Baseline rendering snapshot tests

func TestDetailedFormatter_Snapshot_ScalarModification(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)
	expected := "config.timeout\n  ± value change\n    - 30\n    + 60\n\n"
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
	expected := "config.port\n  ± type change from int to string\n    - 8080\n    + 8080\n\n"
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
	expected := "items.0\n  + one list entry added:\n    - newItem\n\n"
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
	expected := "config.oldKey\n  - one map entry removed:\n    oldKey: value\n\n"
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
	expected := "items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
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
	expected := "key\n  ± whitespace only change\n    - a·b\n    + a··b\n\n"
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
	expected := "(root level)\n  ± value change\n    - old\n    + new\n\n"
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
	expected := "/\n  ± value change\n    - old\n    + new\n\n"
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
	expected := "services.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n"
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
	expected := "alpha\n  ± value change\n    - a1\n    + a2\n\nbeta\n  ± value change\n    - b1\n    + b2\n\n"
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
	expected := "config.timeout\n  ± value change\n    - 30\n    + 60\n\n" +
		"config.verbose\n  + one map entry added:\n    verbose: true\n\n" +
		"services.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n" +
		"config.port\n  ± type change from int to string\n    - 8080\n    + 8080\n\n" +
		"items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("full comparison snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
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
	expected := "items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n"
	if output != expected {
		t.Errorf("order change snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
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
	expected := "services.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n"
	if output != expected {
		t.Errorf("list entry snapshot mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}
