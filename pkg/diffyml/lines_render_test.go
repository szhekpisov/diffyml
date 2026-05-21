package diffyml

import (
	"strings"
	"testing"
)

func TestLinePrefix(t *testing.T) {
	on := &FormatOptions{LineNumbers: true}
	off := &FormatOptions{LineNumbers: false}
	if got := linePrefix(on, 12); got != "12: " {
		t.Errorf("enabled+known: got %q, want %q", got, "12: ")
	}
	if got := linePrefix(on, 0); got != "" {
		t.Errorf("enabled+unknown: got %q, want empty", got)
	}
	if got := linePrefix(off, 12); got != "" {
		t.Errorf("disabled: got %q, want empty", got)
	}
}

func TestAdvanceLine(t *testing.T) {
	if got := advanceLine(5); got != 6 {
		t.Errorf("advanceLine(5)=%d, want 6", got)
	}
	if got := advanceLine(0); got != 0 {
		t.Errorf("advanceLine(0)=%d, want 0 (unknown stays unknown)", got)
	}
}

func TestInsertLineNumber(t *testing.T) {
	tests := []struct {
		name string
		line string
		num  int
		want string
	}{
		{"plain", "    key: value", 8, "    8: key: value"},
		{"list marker", "    - item", 3, "    - 3: item"},
		{"ansi escape", "\x1b[32m    key: value", 8, "\x1b[32m    8: key: value"},
		{"no indent", "key: value", 1, "1: key: value"},
		{"all spaces", "    ", 5, "    5: "}, // space-skip loop must stop at len, not overrun
	}
	for _, tc := range tests {
		if got := insertLineNumber(tc.line, tc.num); got != tc.want {
			t.Errorf("%s: insertLineNumber(%q,%d)=%q, want %q", tc.name, tc.line, tc.num, got, tc.want)
		}
	}
}

func TestWriteEntryWithLineNumber(t *testing.T) {
	f := &DetailedFormatter{}
	on := &FormatOptions{LineNumbers: true}
	off := &FormatOptions{LineNumbers: false}

	// Disabled: block written unchanged.
	var sb strings.Builder
	f.writeEntryWithLineNumber(&sb, "    a: 1\n", 5, off)
	if sb.String() != "    a: 1\n" {
		t.Errorf("disabled: got %q", sb.String())
	}

	// Unknown line: unchanged.
	sb.Reset()
	f.writeEntryWithLineNumber(&sb, "    a: 1\n", 0, on)
	if sb.String() != "    a: 1\n" {
		t.Errorf("unknown line: got %q", sb.String())
	}

	// Multi-line block: only first line gets the number.
	sb.Reset()
	f.writeEntryWithLineNumber(&sb, "    a:\n      b: 1\n", 5, on)
	if sb.String() != "    5: a:\n      b: 1\n" {
		t.Errorf("multiline: got %q", sb.String())
	}

	// No trailing newline: still prefixed (defensive branch).
	sb.Reset()
	f.writeEntryWithLineNumber(&sb, "    a: 1", 5, on)
	if sb.String() != "    5: a: 1" {
		t.Errorf("no-newline: got %q", sb.String())
	}
}

// TestDetailedFormatter_LineNumbersEndToEnd renders through the public Format path
// with line numbers enabled, exercising the value-change, multiline, type-change,
// and add/remove-entry renderers together.
func TestDetailedFormatter_LineNumbersEndToEnd(t *testing.T) {
	from := `scalar: old
multi: |
  keep
  drop
typed: 5
gone: bye
list:
- a
`
	to := `scalar: new
multi: |
  keep
  add
typed: hello
list:
- a
- b
`
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	f := &DetailedFormatter{}
	out := f.Format(diffs, &FormatOptions{LineNumbers: true, ContextLines: 4})

	for _, want := range []string{
		"- 1: old", // scalar value change, from line
		"+ 1: new", // scalar value change, to line
		"3: keep",  // multiline content numbering starts at indicator+1
		"6: gone",  // removed entry anchored at its source line
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q\n--- output ---\n%s", want, out)
		}
	}
}

// TestResolveAddRemoveLine_ListItems covers the identifier and positional branches.
func TestResolveAddRemoveLine_ListItems(t *testing.T) {
	// Identifier-matched list: a named item is added/removed at the list level.
	from := `svc:
- name: a
  port: 1
`
	to := `svc:
- name: a
  port: 1
- name: b
  port: 2
`
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	added := findDiff(t, diffs, "svc")
	if added.Type != DiffAdded {
		t.Fatalf("expected DiffAdded at svc, got %d", added.Type)
	}
	if added.LineTo != 4 { // "- name: b" is on line 4 in 'to'
		t.Errorf("identifier add: LineTo=%d, want 4", added.LineTo)
	}

	// Positional list of scalars: appended item, path already complete.
	from2 := "nums:\n- 1\n"
	to2 := "nums:\n- 1\n- 2\n"
	diffs2, err := Compare([]byte(from2), []byte(to2), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	d := findDiff(t, diffs2, "nums.1")
	if d.Type != DiffAdded || d.LineTo != 3 { // "- 2" is line 3 in 'to'
		t.Errorf("positional add: type=%d LineTo=%d, want added/3", d.Type, d.LineTo)
	}
}

// TestBuildLineMap_MergeKeyAndEmptyDoc covers the "<<" merge-key and empty-document
// branches of the node walker.
func TestBuildLineMap_MergeKeyAndEmptyDoc(t *testing.T) {
	from := `base: &b
  k: 1
merged:
  <<: *b
  extra: 1
`
	to := `base: &b
  k: 1
merged:
  <<: *b
  extra: 2
`
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	d := findDiff(t, diffs, "merged.extra")
	if d.LineFrom != 5 || d.LineTo != 5 {
		t.Errorf("merged.extra: got %d/%d, want 5/5", d.LineFrom, d.LineTo)
	}

	// Empty documents must not panic and yield no line info.
	if _, err := Compare([]byte(""), []byte(""), &Options{CaptureLineNumbers: true}); err != nil {
		t.Fatalf("empty docs: %v", err)
	}
}

// TestCaptureLineNumbers_ParseError ensures parse errors propagate when capturing.
func TestCaptureLineNumbers_ParseError(t *testing.T) {
	if _, err := Compare([]byte("a: [1, 2"), []byte("a: 1"), &Options{CaptureLineNumbers: true}); err == nil {
		t.Error("expected parse error for malformed 'from' input")
	}
	if _, err := Compare([]byte("a: 1"), []byte("b: ]["), &Options{CaptureLineNumbers: true}); err == nil {
		t.Error("expected parse error for malformed 'to' input")
	}
}
