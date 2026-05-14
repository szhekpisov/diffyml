package diffyml

import (
	"strings"
	"testing"
)

// diffAt returns the first difference whose path matches the given string.
func diffAt(t *testing.T, diffs []Difference, path string) Difference {
	t.Helper()
	for _, d := range diffs {
		if d.Path.String() == path {
			return d
		}
	}
	t.Fatalf("no difference found at path %q; got %d diffs", path, len(diffs))
	return Difference{}
}

// --- parsing: OrderedMap line capture ---

func TestParseWithOrder_CapturesLineNumbers(t *testing.T) {
	docs, err := ParseWithOrder([]byte("name: app\nspec:\n  replicas: 2\n  image: nginx\n"))
	if err != nil {
		t.Fatalf("ParseWithOrder: %v", err)
	}
	om, ok := docs[0].(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap, got %T", docs[0])
	}
	if om.Line != 1 {
		t.Errorf("root Line = %d, want 1", om.Line)
	}
	if got := om.lineFor("name"); got != 1 {
		t.Errorf("lineFor(name) = %d, want 1", got)
	}
	if got := om.lineFor("spec"); got != 2 {
		t.Errorf("lineFor(spec) = %d, want 2", got)
	}
	spec := om.Values["spec"].(*OrderedMap)
	if spec.Line != 3 {
		t.Errorf("spec.Line = %d, want 3", spec.Line)
	}
	if got := spec.lineFor("replicas"); got != 3 {
		t.Errorf("lineFor(replicas) = %d, want 3", got)
	}
	if got := spec.lineFor("image"); got != 4 {
		t.Errorf("lineFor(image) = %d, want 4", got)
	}
}

func TestParseWithOrder_MergeKeyLineNumbers(t *testing.T) {
	src := "base: &b\n  shared: 1\nchild:\n  <<: *b\n  own: 2\n"
	docs, err := ParseWithOrder([]byte(src))
	if err != nil {
		t.Fatalf("ParseWithOrder: %v", err)
	}
	root := docs[0].(*OrderedMap)
	child := root.Values["child"].(*OrderedMap)
	// Merged key inherits the anchor definition's line.
	if got := child.lineFor("shared"); got != 2 {
		t.Errorf("merged key lineFor(shared) = %d, want 2", got)
	}
	// Own key keeps its own line.
	if got := child.lineFor("own"); got != 5 {
		t.Errorf("lineFor(own) = %d, want 5", got)
	}
}

func TestOrderedMap_lineFor_Unknown(t *testing.T) {
	if got := (*OrderedMap)(nil).lineFor("x"); got != 0 {
		t.Errorf("nil OrderedMap lineFor = %d, want 0", got)
	}
	om := &OrderedMap{Keys: []string{"x"}, Values: map[string]any{"x": 1}}
	if got := om.lineFor("x"); got != 0 {
		t.Errorf("synthesized OrderedMap lineFor = %d, want 0", got)
	}
	if got := om.lineFor("missing"); got != 0 {
		t.Errorf("missing key lineFor = %d, want 0", got)
	}
}

func TestNodeLine(t *testing.T) {
	if got := nodeLine(&OrderedMap{Line: 7}); got != 7 {
		t.Errorf("nodeLine(OrderedMap{Line:7}) = %d, want 7", got)
	}
	if got := nodeLine("scalar"); got != 0 {
		t.Errorf("nodeLine(scalar) = %d, want 0", got)
	}
	if got := nodeLine([]any{1, 2}); got != 0 {
		t.Errorf("nodeLine(slice) = %d, want 0", got)
	}
	if got := nodeLine(nil); got != 0 {
		t.Errorf("nodeLine(nil) = %d, want 0", got)
	}
}

// --- comparator: FromLine / ToLine propagation ---

func TestCompare_ModifiedScalarLineNumbers(t *testing.T) {
	from := []byte("a: 1\nb:\n  c: old\n")
	to := []byte("a: 1\nb:\n  c: new\n  d: extra\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	mod := diffAt(t, diffs, "b.c")
	if mod.FromLine != 3 || mod.ToLine != 3 {
		t.Errorf("b.c lines = (%d,%d), want (3,3)", mod.FromLine, mod.ToLine)
	}
	added := diffAt(t, diffs, "b")
	if added.Type != DiffAdded {
		t.Fatalf("expected b to be DiffAdded, got %v", added.Type)
	}
	if added.ToLine != 4 || added.FromLine != 0 {
		t.Errorf("added b.d lines = (%d,%d), want (0,4)", added.FromLine, added.ToLine)
	}
}

func TestCompare_ModifiedScalarShiftedLines(t *testing.T) {
	// The same key sits on different lines in each file.
	from := []byte("x: 1\nval: old\n")
	to := []byte("x: 1\ny: 2\nz: 3\nval: new\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	mod := diffAt(t, diffs, "val")
	if mod.FromLine != 2 || mod.ToLine != 4 {
		t.Errorf("val lines = (%d,%d), want (2,4)", mod.FromLine, mod.ToLine)
	}
}

func TestCompare_RemovedKeyLineNumber(t *testing.T) {
	from := []byte("a: 1\nstale: gone\n")
	to := []byte("a: 1\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Type != DiffRemoved {
		t.Fatalf("expected DiffRemoved, got %v", diffs[0].Type)
	}
	if diffs[0].FromLine != 2 || diffs[0].ToLine != 0 {
		t.Errorf("removed lines = (%d,%d), want (2,0)", diffs[0].FromLine, diffs[0].ToLine)
	}
}

func TestCompare_TypeMismatchLineNumbers(t *testing.T) {
	from := []byte("k: 1\nval: 42\n")
	to := []byte("k: 1\nval: text\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	mod := diffAt(t, diffs, "val")
	if mod.FromLine != 2 || mod.ToLine != 2 {
		t.Errorf("type-mismatch lines = (%d,%d), want (2,2)", mod.FromLine, mod.ToLine)
	}
}

func TestCompare_NilToModifiedLineNumber(t *testing.T) {
	from := []byte("a: 1\nval: something\n")
	to := []byte("a: 1\nval:\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	mod := diffAt(t, diffs, "val")
	if mod.Type != DiffModified {
		t.Fatalf("expected DiffModified, got %v", mod.Type)
	}
	if mod.FromLine != 2 {
		t.Errorf("nil-to modified FromLine = %d, want 2", mod.FromLine)
	}
}

func TestCompare_PositionalListItemLineNumbers(t *testing.T) {
	from := []byte("items:\n  - a: 1\n  - a: 2\n")
	to := []byte("items:\n  - a: 1\n  - a: 2\n  - a: 3\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	added := diffAt(t, diffs, "items.2")
	if added.Type != DiffAdded {
		t.Fatalf("expected DiffAdded at items.2, got %v", added.Type)
	}
	if added.ToLine != 4 {
		t.Errorf("added list item ToLine = %d, want 4", added.ToLine)
	}
}

func TestCompare_IdentifierListItemLineNumbers(t *testing.T) {
	from := []byte("list:\n  - name: x\n    v: 1\n")
	to := []byte("list:\n  - name: x\n    v: 1\n  - name: y\n    v: 2\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	added := diffAt(t, diffs, "list")
	if added.Type != DiffAdded {
		t.Fatalf("expected DiffAdded at list, got %v", added.Type)
	}
	if added.ToLine != 4 {
		t.Errorf("added identifier list item ToLine = %d, want 4", added.ToLine)
	}
}

func TestCompare_IdentifierListNestedModifiedLineNumbers(t *testing.T) {
	from := []byte("list:\n  - name: x\n    v: 1\n")
	to := []byte("list:\n  - name: x\n    v: 2\n")
	diffs, err := Compare(from, to, &Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	mod := diffAt(t, diffs, "list.x.v")
	if mod.FromLine != 3 || mod.ToLine != 3 {
		t.Errorf("list.x.v lines = (%d,%d), want (3,3)", mod.FromLine, mod.ToLine)
	}
}

func TestCompare_LineNumbersSurviveChroot(t *testing.T) {
	from := []byte("top:\n  spec:\n    replicas: 2\n")
	to := []byte("top:\n  spec:\n    replicas: 3\n")
	diffs, err := Compare(from, to, &Options{Chroot: "top"})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	mod := diffAt(t, diffs, "spec.replicas")
	if mod.FromLine != 3 || mod.ToLine != 3 {
		t.Errorf("chrooted lines = (%d,%d), want (3,3)", mod.FromLine, mod.ToLine)
	}
}

// --- formatter helpers ---

func TestLineAnnotation(t *testing.T) {
	cases := []struct {
		name string
		diff Difference
		want string
	}{
		{"both equal", Difference{FromLine: 3, ToLine: 3}, " (L3)"},
		{"both differ", Difference{FromLine: 3, ToLine: 5}, " (L3 → L5)"},
		{"to only", Difference{ToLine: 7}, " (L7)"},
		{"from only", Difference{FromLine: 9}, " (L9)"},
		{"neither", Difference{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lineAnnotation(tc.diff); got != tc.want {
				t.Errorf("lineAnnotation = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDiffLine(t *testing.T) {
	if got := diffLine(Difference{FromLine: 2, ToLine: 5}); got != 5 {
		t.Errorf("diffLine prefers ToLine: got %d, want 5", got)
	}
	if got := diffLine(Difference{FromLine: 2}); got != 2 {
		t.Errorf("diffLine falls back to FromLine: got %d, want 2", got)
	}
	if got := diffLine(Difference{}); got != 0 {
		t.Errorf("diffLine with no lines: got %d, want 0", got)
	}
}

// --- formatter rendering ---

func TestCompactFormatter_LineNumbers(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: 2, FromLine: 4, ToLine: 4},
	}
	f := &CompactFormatter{}
	on := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(on, "(L4)") {
		t.Errorf("expected line annotation in compact output, got:\n%s", on)
	}
	off := f.Format(diffs, &FormatOptions{ShowLineNumbers: false})
	if strings.Contains(off, "(L4)") {
		t.Errorf("line annotation should be absent when disabled, got:\n%s", off)
	}
}

func TestCompactFormatter_LineNumbers_Colored(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: 2, FromLine: 4, ToLine: 4},
	}
	opts := &FormatOptions{ShowLineNumbers: true, Color: true}
	out := (&CompactFormatter{}).Format(diffs, opts)
	p := resolvedPalette(opts)
	want := colorStart(opts, p.ColorCode(ColorRoleContext, opts.TrueColor)) + " (L4)" + colorEnd(opts)
	if !strings.Contains(out, want) {
		t.Errorf("expected color-wrapped line annotation, got:\n%q", out)
	}
}

func TestDetailedFormatter_LineNumbers(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: "x", To: "y", FromLine: 4, ToLine: 6},
	}
	f := &DetailedFormatter{}
	on := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(on, "(L4 → L6)") {
		t.Errorf("expected line range in detailed output, got:\n%s", on)
	}
	off := f.Format(diffs, &FormatOptions{ShowLineNumbers: false})
	if strings.Contains(off, "L4") {
		t.Errorf("line annotation should be absent when disabled, got:\n%s", off)
	}
}

func TestDetailedFormatter_LineNumbers_TypeChange(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: "str", FromLine: 2, ToLine: 2},
	}
	f := &DetailedFormatter{}
	plain := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(plain, "type change") || !strings.Contains(plain, "(L2)") {
		t.Errorf("expected type change with line in plain output, got:\n%s", plain)
	}
	colored := f.Format(diffs, &FormatOptions{ShowLineNumbers: true, Color: true})
	if !strings.Contains(colored, "(L2)") {
		t.Errorf("expected line annotation in colored type change output, got:\n%s", colored)
	}
}

func TestDetailedFormatter_LineNumbers_Multiline(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"cfg"}, Type: DiffModified, From: "one\ntwo\n", To: "one\nTWO\n", FromLine: 5, ToLine: 5},
	}
	f := &DetailedFormatter{}
	out := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(out, "multiline text") || !strings.Contains(out, "(L5)") {
		t.Errorf("expected multiline descriptor with line, got:\n%s", out)
	}
}

func TestDetailedFormatter_LineNumbers_Whitespace(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"w"}, Type: DiffModified, From: "a b", To: "a  b", FromLine: 8, ToLine: 8},
	}
	f := &DetailedFormatter{}
	out := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(out, "whitespace only change (L8)") {
		t.Errorf("expected whitespace descriptor with line, got:\n%s", out)
	}
}

func TestDetailedFormatter_LineNumbers_EntryBatch(t *testing.T) {
	single := []Difference{
		{Path: DiffPath{"m"}, Type: DiffRemoved, From: &OrderedMap{Keys: []string{"k"}, Values: map[string]any{"k": "v"}}, FromLine: 3},
	}
	f := &DetailedFormatter{}
	out := f.Format(single, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(out, "removed (L3):") {
		t.Errorf("expected single-entry batch header with line, got:\n%s", out)
	}

	// Disabling line numbers drops the annotation even for single entries.
	outOff := f.Format(single, &FormatOptions{ShowLineNumbers: false})
	if strings.Contains(outOff, "(L3)") {
		t.Errorf("batch header should have no line annotation when disabled, got:\n%s", outOff)
	}

	// Multi-entry batches omit the annotation (heterogeneous lines).
	multi := []Difference{
		{Path: DiffPath{"m"}, Type: DiffRemoved, From: &OrderedMap{Keys: []string{"k1"}, Values: map[string]any{"k1": "v"}}, FromLine: 3},
		{Path: DiffPath{"m"}, Type: DiffRemoved, From: &OrderedMap{Keys: []string{"k2"}, Values: map[string]any{"k2": "v"}}, FromLine: 7},
	}
	outMulti := f.Format(multi, &FormatOptions{ShowLineNumbers: true})
	if strings.Contains(outMulti, "(L3)") || strings.Contains(outMulti, "(L7)") {
		t.Errorf("multi-entry batch should not show a single line, got:\n%s", outMulti)
	}
}

func TestDetailedFormatter_LineNumbers_OrderChanged(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"l"}, Type: DiffOrderChanged, From: []any{"a", "b"}, To: []any{"b", "a"}, FromLine: 2, ToLine: 2},
	}
	f := &DetailedFormatter{}
	out := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(out, "order changed (L2)") {
		t.Errorf("expected order changed descriptor with line, got:\n%s", out)
	}
}

func TestJSONFormatter_LineNumbers(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: 2, FromLine: 3, ToLine: 5},
	}
	f := &JSONFormatter{}
	on := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(on, `"from_line": 3`) || !strings.Contains(on, `"to_line": 5`) {
		t.Errorf("expected line fields in JSON, got:\n%s", on)
	}
	off := f.Format(diffs, &FormatOptions{ShowLineNumbers: false})
	if strings.Contains(off, "from_line") || strings.Contains(off, "to_line") {
		t.Errorf("line fields should be omitted when disabled, got:\n%s", off)
	}
}

func TestGitLabFormatter_LineNumbers(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: 2, FromLine: 9, ToLine: 9},
	}
	f := &GitLabFormatter{}
	on := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(on, `"begin": 9`) {
		t.Errorf("expected real begin line in GitLab output, got:\n%s", on)
	}
	off := f.Format(diffs, &FormatOptions{ShowLineNumbers: false})
	if !strings.Contains(off, `"begin": 1`) {
		t.Errorf("expected default begin line 1 when disabled, got:\n%s", off)
	}
	single := f.FormatSingle(diffs[0], &FormatOptions{ShowLineNumbers: true})
	if !strings.Contains(single, `"begin": 9`) {
		t.Errorf("expected real begin line in GitLab FormatSingle, got:\n%s", single)
	}
}

func TestGitLabBeginLine_FallbackWhenUnknown(t *testing.T) {
	// ShowLineNumbers on, but the diff carries no line info.
	if got := gitLabBeginLine(Difference{}, &FormatOptions{ShowLineNumbers: true}); got != 1 {
		t.Errorf("gitLabBeginLine with unknown line = %d, want 1", got)
	}
	// Nil opts must not panic and falls back to 1.
	if got := gitLabBeginLine(Difference{FromLine: 5}, nil); got != 1 {
		t.Errorf("gitLabBeginLine with nil opts = %d, want 1", got)
	}
	// Disabled: real line is ignored, falls back to 1.
	if got := gitLabBeginLine(Difference{FromLine: 5}, &FormatOptions{ShowLineNumbers: false}); got != 1 {
		t.Errorf("gitLabBeginLine disabled = %d, want 1", got)
	}
}

func TestGitHubFormatter_LineNumbers(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: 2, FromLine: 11, ToLine: 11},
	}
	f := &GitHubFormatter{}
	on := f.Format(diffs, &FormatOptions{ShowLineNumbers: true, FilePath: "deploy.yaml"})
	if !strings.Contains(on, "file=deploy.yaml,line=11,title=") {
		t.Errorf("expected line= param in GitHub output, got:\n%s", on)
	}
	// line= is omitted without a file path.
	noFile := f.Format(diffs, &FormatOptions{ShowLineNumbers: true})
	if strings.Contains(noFile, "line=") {
		t.Errorf("line= should be omitted without file path, got:\n%s", noFile)
	}
	off := f.Format(diffs, &FormatOptions{ShowLineNumbers: false, FilePath: "deploy.yaml"})
	if strings.Contains(off, "line=") {
		t.Errorf("line= should be omitted when disabled, got:\n%s", off)
	}
}

func TestGitHubLine_DisabledOrUnknown(t *testing.T) {
	if got := gitHubLine(Difference{FromLine: 4}, &FormatOptions{ShowLineNumbers: false}); got != 0 {
		t.Errorf("gitHubLine disabled = %d, want 0", got)
	}
	if got := gitHubLine(Difference{FromLine: 4}, &FormatOptions{ShowLineNumbers: true}); got != 4 {
		t.Errorf("gitHubLine enabled = %d, want 4", got)
	}
	if got := gitHubLine(Difference{}, nil); got != 0 {
		t.Errorf("gitHubLine nil opts = %d, want 0", got)
	}
}
