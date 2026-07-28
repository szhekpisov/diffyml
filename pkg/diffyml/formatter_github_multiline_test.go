package diffyml

import (
	"fmt"
	"strings"
	"testing"
)

// buildMultiline returns a value of n lines where the line at index changeAt
// (0-based) carries the given marker.
func buildMultiline(n, changeAt int, marker string) string {
	lines := make([]string, n)
	for i := range lines {
		if i == changeAt {
			lines[i] = marker
			continue
		}
		lines[i] = fmt.Sprintf("line%d", i)
	}
	return strings.Join(lines, "\n")
}

// githubMessage returns the message portion of a single-annotation output,
// failing the test when the output is not exactly one workflow command.
func githubMessage(t *testing.T, output string) string {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one annotation line, got %d:\n%s", len(lines), output)
	}
	_, msg, found := strings.Cut(lines[0], "::")
	if !found {
		t.Fatalf("output is not a workflow command: %q", output)
	}
	_, msg, found = strings.Cut(msg, "::")
	if !found {
		t.Fatalf("workflow command has no message separator: %q", output)
	}
	return msg
}

func TestEscapeGitHubData(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "no escapes here", "no escapes here"},
		{"percent", "50% off", "50%25 off"},
		{"newline", "a\nb", "a%0Ab"},
		{"carriage return", "a\rb", "a%0Db"},
		{"crlf", "a\r\nb", "a%0D%0Ab"},
		{"empty", "", ""},
		// Percent must be encoded first: a literal "%0A" in the data must not
		// decode back into a newline on GitHub's side.
		{"literal escape sequence", "%0A", "%250A"},
		{"percent then newline", "%\n", "%25%0A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeGitHubData(tt.input); got != tt.want {
				t.Errorf("escapeGitHubData(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGitHubFormatter_EscapesMessage(t *testing.T) {
	f := &GitHubFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"discount"}, Type: DiffModified, From: "10%", To: "20%"},
	}

	output := f.Format(diffs, DefaultFormatOptions())
	msg := githubMessage(t, output)

	if !strings.Contains(msg, "10%25") || !strings.Contains(msg, "20%25") {
		t.Errorf("expected percent signs to be encoded, got: %q", msg)
	}
}

func TestGitHubFormatter_MultilineCollapsed(t *testing.T) {
	from := buildMultiline(140, 106, "minReplicas: 2")
	to := buildMultiline(140, 106, "minReplicas: 3")

	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()
	opts.ContextLines = 2

	output := f.Format([]Difference{
		{Path: DiffPath{"data", "values.yaml"}, Type: DiffModified, From: from, To: to},
	}, opts)

	msg := githubMessage(t, output)

	want := []string{
		"Modified: data[values.yaml] changed in multiline text (one insert, one deletion)",
		"[104 lines unchanged]",
		"  line104", "  line105",
		"- minReplicas: 2",
		"+ minReplicas: 3",
		"  line107", "  line108",
		"[31 lines unchanged]",
	}
	for _, w := range want {
		if !strings.Contains(msg, escapeGitHubData(w)) {
			t.Errorf("expected %q in message, got:\n%s", w, msg)
		}
	}

	// The bug being fixed: unrelated lines must not reach the annotation.
	for _, unwanted := range []string{"line0", "line50", "line139"} {
		if strings.Contains(msg, escapeGitHubData("  "+unwanted)) {
			t.Errorf("expected collapsed line %q to be absent, got:\n%s", unwanted, msg)
		}
	}
}

func TestGithubMultilineDiff_ContextLines(t *testing.T) {
	from := buildMultiline(20, 10, "old value")
	to := buildMultiline(20, 10, "new value")

	tests := []struct {
		name         string
		contextLines int
		wantLeading  string
		wantTrailing string
	}{
		{"zero context", 0, "[10 lines unchanged]", "[9 lines unchanged]"},
		{"one line", 1, "[9 lines unchanged]", "[8 lines unchanged]"},
		{"two lines", 2, "[8 lines unchanged]", "[7 lines unchanged]"},
		{"negative falls back to four", -1, "[6 lines unchanged]", "[5 lines unchanged]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := githubMultilineDiff(from, to, tt.contextLines)
			if !strings.Contains(got, tt.wantLeading) {
				t.Errorf("expected leading marker %q, got:\n%s", tt.wantLeading, got)
			}
			if !strings.Contains(got, tt.wantTrailing) {
				t.Errorf("expected trailing marker %q, got:\n%s", tt.wantTrailing, got)
			}
			if !strings.Contains(got, "- old value") || !strings.Contains(got, "+ new value") {
				t.Errorf("expected changed lines regardless of context, got:\n%s", got)
			}
		})
	}
}

func TestGithubMultilineDiff_SingularMarker(t *testing.T) {
	// One unchanged line beyond the context window on each side.
	from := "a\nb\nOLD\nc\nd"
	to := "a\nb\nNEW\nc\nd"

	got := githubMultilineDiff(from, to, 1)
	if !strings.Contains(got, "[1 line unchanged]") {
		t.Errorf("expected singular line marker, got:\n%s", got)
	}
	if strings.Contains(got, "[1 lines unchanged]") {
		t.Errorf("expected singular wording, got:\n%s", got)
	}
}

func TestGithubMultilineDiff_EditCountWording(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want string
	}{
		{
			name: "single edit each way",
			from: "a\nOLD\nb",
			to:   "a\nNEW\nb",
			want: "multiline text (one insert, one deletion)",
		},
		{
			name: "multiple edits",
			from: "a\nOLD1\nOLD2\nb",
			to:   "a\nNEW1\nNEW2\nb",
			want: "multiline text (two inserts, two deletions)",
		},
		{
			name: "pure insert",
			from: "a\nb",
			to:   "a\nnew\nb",
			want: "multiline text (one insert, zero deletions)",
		},
		{
			name: "pure deletion",
			from: "a\ngone\nb",
			to:   "a\nb",
			want: "multiline text (zero inserts, one deletion)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := githubMultilineDiff(tt.from, tt.to, 4)
			if !strings.Contains(got, tt.want) {
				t.Errorf("expected %q, got:\n%s", tt.want, got)
			}
		})
	}
}

func TestGithubTruncatedValue(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		maxLines int
		want     string
	}{
		{"single line untouched", "just one line", 3, "just one line"},
		{"non-string untouched", 42, 3, "42"},
		{"exactly at cap", "a\nb\nc", 3, "a\nb\nc"},
		{"one over cap", "a\nb\nc\nd", 3, "a\nb\nc\n[1 more line]"},
		{"several over cap", "a\nb\nc\nd\ne\nf", 3, "a\nb\nc\n[3 more lines]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := githubTruncatedValue(tt.val, tt.maxLines); got != tt.want {
				t.Errorf("githubTruncatedValue(%v, %d) = %q, want %q", tt.val, tt.maxLines, got, tt.want)
			}
		})
	}
}

func TestGithubDiffDescription_OneSidedMultiline(t *testing.T) {
	// Only one side needs to be multiline for the line diff to apply: replacing
	// a block scalar with a single line, or vice versa, is still best shown as
	// a diff rather than as "changed from <20 lines> to <1 line>".
	multi := buildMultiline(20, 10, "old value")
	single := "collapsed to one line"

	tests := []struct {
		name string
		from string
		to   string
		want []string
	}{
		{
			name: "from multiline, to single line",
			from: multi,
			to:   single,
			want: []string{"- line0", "- line19", "+ " + single},
		},
		{
			name: "from single line, to multiline",
			from: single,
			to:   multi,
			want: []string{"- " + single, "+ line0", "+ line19"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := Difference{Path: DiffPath{"data", "script"}, Type: DiffModified, From: tt.from, To: tt.to}
			got := githubDiffDescription(diff, DefaultFormatOptions())

			if !strings.Contains(got, "changed in multiline text") {
				t.Fatalf("expected the line-diff path, got:\n%s", got)
			}
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("expected %q in description, got:\n%s", w, got)
				}
			}
			if strings.Contains(got, "changed from ") {
				t.Errorf("expected no from/to wording on the line-diff path, got:\n%s", got)
			}
		})
	}
}

func TestMultilineStrings(t *testing.T) {
	tests := []struct {
		name string
		from any
		to   any
		want bool
	}{
		{"both multiline", "a\nb", "c\nd", true},
		{"only from multiline", "a\nb", "c", true},
		{"only to multiline", "a", "c\nd", true},
		{"neither multiline", "a", "c", false},
		{"from not a string", 42, "c\nd", false},
		{"to not a string", "a\nb", 42, false},
		{"neither a string", 42, 43, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, got := multilineStrings(tt.from, tt.to); got != tt.want {
				t.Errorf("multilineStrings(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestGithubTruncatedValue_StructuredValue(t *testing.T) {
	// A whole added map is never a Go string, but serializes to many lines of
	// YAML. Truncation must apply to the rendered form.
	m := NewOrderedMap()
	for i := range 30 {
		key := fmt.Sprintf("key%02d", i)
		m.Keys = append(m.Keys, key)
		m.Values[key] = fmt.Sprintf("value%d", i)
	}

	got := githubTruncatedValue(m, gitHubMaxValueLines)
	lines := strings.Split(got, "\n")

	if len(lines) != gitHubMaxValueLines+1 {
		t.Fatalf("expected %d lines plus a marker, got %d:\n%s", gitHubMaxValueLines, len(lines), got)
	}
	if want := "[10 more lines]"; lines[len(lines)-1] != want {
		t.Errorf("expected last line %q, got %q", want, lines[len(lines)-1])
	}
	if strings.Contains(got, "key29") {
		t.Errorf("expected lines past the cap to be dropped, got:\n%s", got)
	}
}

func TestGithubDiffDescription_ModifiedStructuredTruncated(t *testing.T) {
	long := buildMultiline(40, -1, "")
	// Not a string pair, so this takes the "changed from X to Y" path; both
	// sides must still be bounded.
	diff := Difference{Path: DiffPath{"data"}, Type: DiffModified, From: long, To: 42}

	got := githubDiffDescription(diff, DefaultFormatOptions())
	if !strings.Contains(got, "changed from ") || !strings.HasSuffix(got, " to 42") {
		t.Errorf("expected the from/to wording, got:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("[%d more lines]", 40-gitHubMaxValueLines)) {
		t.Errorf("expected the from side to be truncated, got:\n%s", got)
	}
}

func TestGitHubFormatter_AddedRemovedMultilineTruncated(t *testing.T) {
	value := buildMultiline(50, -1, "")

	tests := []struct {
		name   string
		diff   Difference
		prefix string
	}{
		{
			name:   "added",
			diff:   Difference{Path: DiffPath{"data", "script"}, Type: DiffAdded, To: value},
			prefix: "Added:",
		},
		{
			name:   "removed",
			diff:   Difference{Path: DiffPath{"data", "script"}, Type: DiffRemoved, From: value},
			prefix: "Removed:",
		},
		{
			name:   "unchanged",
			diff:   Difference{Path: DiffPath{"data", "script"}, Type: DiffUnchanged, To: value},
			prefix: "Unchanged:",
		},
	}

	f := &GitHubFormatter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := githubMessage(t, f.Format([]Difference{tt.diff}, DefaultFormatOptions()))

			if !strings.HasPrefix(msg, tt.prefix) {
				t.Errorf("expected message to start with %q, got: %s", tt.prefix, msg)
			}
			remaining := 50 - gitHubMaxValueLines
			if !strings.Contains(msg, fmt.Sprintf("[%d more lines]", remaining)) {
				t.Errorf("expected truncation marker for %d lines, got: %s", remaining, msg)
			}
			if strings.Contains(msg, "line49") {
				t.Errorf("expected lines past the cap to be dropped, got: %s", msg)
			}
		})
	}
}

func TestGithubDiffDescription_SingleLineMatchesShared(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffAdded, To: "value"},
		{Path: DiffPath{"a"}, Type: DiffRemoved, From: "value"},
		{Path: DiffPath{"a"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"a"}, Type: DiffUnchanged, To: "value"},
		{Path: DiffPath{"a"}, Type: DiffOrderChanged},
		// Multiline but not a string pair: falls through to the shared path.
		{Path: DiffPath{"a"}, Type: DiffModified, From: "old\nvalue", To: 42},
		{Path: DiffPath{"a"}, Type: DiffAdded, To: 42},
	}

	for i, diff := range diffs {
		want := diffDescription(diff)
		if got := githubDiffDescription(diff, DefaultFormatOptions()); got != want {
			t.Errorf("diff %d: got %q, want %q", i, got, want)
		}
	}
}

func TestGithubDiffDescription_NilOptions(t *testing.T) {
	from := buildMultiline(20, 10, "old value")
	to := buildMultiline(20, 10, "new value")

	diff := Difference{Path: DiffPath{"data", "script"}, Type: DiffModified, From: from, To: to}

	// Nil options must behave like the default context of 4.
	got := githubDiffDescription(diff, nil)
	want := githubDiffDescription(diff, DefaultFormatOptions())
	if got != want {
		t.Errorf("nil opts = %q, want %q", got, want)
	}
	if !strings.Contains(got, "[6 lines unchanged]") {
		t.Errorf("expected default context of 4, got:\n%s", got)
	}
}

func TestGithubDiffDescription_DocumentName(t *testing.T) {
	diff := Difference{
		Path:         DiffPath{"data", "script"},
		Type:         DiffModified,
		DocumentName: "v1/ConfigMap/app",
		From:         "a\nOLD\nb",
		To:           "a\nNEW\nb",
	}

	got := githubDiffDescription(diff, DefaultFormatOptions())
	if !strings.Contains(got, "(v1/ConfigMap/app)") {
		t.Errorf("expected document name in description, got: %s", got)
	}
}

func TestGitHubFormatter_FormatSingleMultiline(t *testing.T) {
	f := &GitHubFormatter{}
	diff := Difference{
		Path: DiffPath{"data", "script"},
		Type: DiffModified,
		From: buildMultiline(20, 10, "old value"),
		To:   buildMultiline(20, 10, "new value"),
	}

	msg := githubMessage(t, f.FormatSingle(diff, DefaultFormatOptions()))
	if !strings.Contains(msg, "%0A") {
		t.Errorf("expected encoded line breaks, got: %s", msg)
	}
	if !strings.Contains(msg, escapeGitHubData("- old value")) {
		t.Errorf("expected removed line in message, got: %s", msg)
	}
}

func TestGitHubFormatter_FormatAllMultiline(t *testing.T) {
	f := &GitHubFormatter{}
	groups := []DiffGroup{
		{
			FilePath: "config.yaml",
			Diffs: []Difference{{
				Path: DiffPath{"data", "script"},
				Type: DiffModified,
				From: buildMultiline(30, 15, "old value"),
				To:   buildMultiline(30, 15, "new value"),
			}},
		},
	}

	msg := githubMessage(t, f.FormatAll(groups, DefaultFormatOptions()))
	if !strings.Contains(msg, escapeGitHubData("+ new value")) {
		t.Errorf("expected added line in message, got: %s", msg)
	}
	if !strings.Contains(msg, escapeGitHubData("[11 lines unchanged]")) {
		t.Errorf("expected leading collapse marker, got: %s", msg)
	}
}

func TestGiteaFormatter_MultilineMatchesGitHub(t *testing.T) {
	githubF, _ := FormatterByName("github")
	giteaF, _ := FormatterByName("gitea")

	diffs := []Difference{{
		Path: DiffPath{"data", "script"},
		Type: DiffModified,
		From: buildMultiline(30, 15, "old value"),
		To:   buildMultiline(30, 15, "new value"),
	}}
	opts := DefaultFormatOptions()

	if got, want := giteaF.Format(diffs, opts), githubF.Format(diffs, opts); got != want {
		t.Errorf("gitea multiline output should match github\nGitea:  %s\nGitHub: %s", got, want)
	}
}

func TestCollapseLineDiff_NeverAbsorbsChangedLines(t *testing.T) {
	// A collapsed run is a run of *unchanged* lines. markNearChange always
	// flags an insert or delete at its own index, so callers never hit this,
	// but the run scan must stop at a changed line on its own merits rather
	// than leaning on the mask: swallowing one would drop it from the output.
	ops := []editOp{
		{Type: editKeep, Line: "a"},
		{Type: editInsert, Line: "b"},
		{Type: editKeep, Line: "c"},
	}
	nearChange := []bool{false, false, false}

	chunks := collapseLineDiff(ops, nearChange)

	want := []lineDiffChunk{
		{Type: chunkCollapsed, Collapsed: 1},
		{Type: chunkInsert, Line: "b"},
		{Type: chunkCollapsed, Collapsed: 1},
	}
	if len(chunks) != len(want) {
		t.Fatalf("expected %d chunks, got %d: %+v", len(want), len(chunks), chunks)
	}
	for i, w := range want {
		if chunks[i] != w {
			t.Errorf("chunk %d = %+v, want %+v", i, chunks[i], w)
		}
	}
}

func TestCollapseLineDiff_NoChanges(t *testing.T) {
	// Identical inputs produce only keep ops, none near a change, so the whole
	// value collapses into a single marker.
	ops := computeLineDiff([]string{"a", "b", "c"}, []string{"a", "b", "c"})
	chunks := collapseLineDiff(ops, markNearChange(ops, 4))

	if len(chunks) != 1 {
		t.Fatalf("expected one chunk, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Type != chunkCollapsed || chunks[0].Collapsed != 3 {
		t.Errorf("expected a collapsed chunk of 3, got %+v", chunks[0])
	}
}

func TestResolveContextLines(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{-2, 4},
		{-1, 4},
		{0, 0},
		{1, 1},
		{7, 7},
	}

	for _, tt := range tests {
		if got := resolveContextLines(tt.in); got != tt.want {
			t.Errorf("resolveContextLines(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestCollapsedRunLabel(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1, "[1 line unchanged]"},
		{2, "[2 lines unchanged]"},
		{0, "[0 lines unchanged]"},
	}

	for _, tt := range tests {
		if got := collapsedRunLabel(tt.n); got != tt.want {
			t.Errorf("collapsedRunLabel(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
