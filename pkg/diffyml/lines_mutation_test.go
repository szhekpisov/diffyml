package diffyml

import (
	"strings"
	"testing"
)

// TestOptions_shouldCaptureLines pins the capture gate: enabled only when the flag
// is set AND no chroot option is active.
func TestOptions_shouldCaptureLines(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want bool
	}{
		{"enabled, no chroot", Options{CaptureLineNumbers: true}, true},
		{"flag off", Options{CaptureLineNumbers: false}, false},
		{"chroot set", Options{CaptureLineNumbers: true, Chroot: "x"}, false},
		{"chroot-from set", Options{CaptureLineNumbers: true, ChrootFrom: "x"}, false},
		{"chroot-to set", Options{CaptureLineNumbers: true, ChrootTo: "x"}, false},
	}
	for _, tc := range tests {
		if got := tc.opts.shouldCaptureLines(); got != tc.want {
			t.Errorf("%s: shouldCaptureLines()=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLinePrefix_NilOpts(t *testing.T) {
	if got := linePrefix(nil, 5); got != "" {
		t.Errorf("linePrefix(nil,5)=%q, want empty (must not deref nil opts)", got)
	}
}

// TestAppendPathSegment pins the segment formatting so it matches DiffPath.String().
func TestAppendPathSegment(t *testing.T) {
	tests := []struct {
		base, seg, want string
	}{
		{"", "a", "a"},
		{"a", "b", "a.b"},
		{"a", "helm.sh/chart", "a[helm.sh/chart]"}, // dotted key bracket-quoted
		{"", "[0]", "[0]"},                         // doc index, no leading dot
		{"a", "[x]", "a[x]"},                       // bracket-prefixed segment: no dot inserted
	}
	for _, tc := range tests {
		if got := string(appendPathSegment([]byte(tc.base), tc.seg)); got != tc.want {
			t.Errorf("appendPathSegment(%q,%q)=%q, want %q", tc.base, tc.seg, got, tc.want)
		}
	}
}

// TestBuildLineMap_NoRootNoMergeKey asserts the walker never registers the empty
// document-root path nor a YAML merge ("<<") key.
func TestBuildLineMap_NoRootNoMergeKey(t *testing.T) {
	src := `base: &b
  k: 1
merged:
  <<: *b
  extra: 1
`
	_, nodes, err := parseWithNodes([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	m := buildLineMap(nodes, 1, &Options{})
	if _, ok := m[""]; ok {
		t.Error("line map must not contain the empty root path")
	}
	for k := range m {
		if strings.Contains(k, "<<") {
			t.Errorf("line map must not register merge key, found %q", k)
		}
	}
	// Sanity: a real key is present.
	if m["merged.extra"] == 0 {
		t.Error("expected merged.extra to be registered")
	}
}

func TestParseWithNodes_Empty(t *testing.T) {
	docs, nodes, err := parseWithNodes([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0] != nil {
		t.Errorf("empty content: docs=%v, want [nil]", docs)
	}
	if len(nodes) != 1 || nodes[0] != nil {
		t.Errorf("empty content: nodes len=%d, want 1 nil node", len(nodes))
	}
}

// TestMultilineDiff_PerLineNumbers pins per-line numbering across multiple deletes
// and inserts, exercising both the from- and to-cursor advances.
func TestMultilineDiff_PerLineNumbers(t *testing.T) {
	from := "t: |\n  a\n  b\n  c\n  z\n"
	to := "t: |\n  a\n  X\n  Y\n  z\n"
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	out := (&DetailedFormatter{}).Format(diffs, &FormatOptions{LineNumbers: true, ContextLines: 4})
	for _, want := range []string{"- 3: b", "- 4: c", "+ 3: X", "+ 4: Y"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in multiline output\n%s", want, out)
		}
	}
}

// TestMultilineDiff_CollapseAdvancesCursors pins cursor advancement across a
// collapsed unchanged block so the post-collapse line numbers stay correct.
func TestMultilineDiff_CollapseAdvancesCursors(t *testing.T) {
	from := "t: |\n  k1\n  k2\n  k3\n  k4\n  k5\n  k6\n  OLD\n"
	to := "t: |\n  k1\n  k2\n  k3\n  k4\n  k5\n  k6\n  NEW\n"
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	out := (&DetailedFormatter{}).Format(diffs, &FormatOptions{LineNumbers: true, ContextLines: 1})
	for _, want := range []string{"unchanged]", "7: k6", "- 8: OLD", "+ 8: NEW"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in collapsed multiline output\n%s", want, out)
		}
	}
}

// TestTypeChange_StructuredLineNumber pins the first-line-only numbering of a
// structured type-change value.
func TestTypeChange_StructuredLineNumber(t *testing.T) {
	from := "key: 5\n"
	to := "key:\n  a: 1\n  b: 2\n"
	diffs, err := Compare([]byte(from), []byte(to), &Options{CaptureLineNumbers: true})
	if err != nil {
		t.Fatal(err)
	}
	out := (&DetailedFormatter{}).Format(diffs, &FormatOptions{LineNumbers: true})
	for _, want := range []string{"- 1: 5", "+ 1: a: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in type-change output\n%s", want, out)
		}
	}
	// Second structured line must NOT be numbered.
	if strings.Contains(out, "1: b: 2") || strings.Contains(out, "2: b: 2") {
		t.Errorf("second structured line must not be numbered\n%s", out)
	}
	if !strings.Contains(out, "+ b: 2") {
		t.Errorf("expected unnumbered '+ b: 2'\n%s", out)
	}
}
