package diffyml

import (
	"strings"
	"testing"
)

// unchangedScalarDiff builds a DiffUnchanged entry for a map leaf.
func unchangedScalarDiff(path DiffPath, val any) Difference {
	return Difference{Path: path, Type: DiffUnchanged, From: val, To: val}
}

// inverseFormatOptions returns the format options the CLI sets in inverse mode
// (Options.Unchanged), which is what selects the "unchanged value(s)" wording.
func inverseFormatOptions() *FormatOptions {
	opts := DefaultFormatOptions()
	opts.Unchanged = true
	return opts
}

func TestInverseFormat_Compact(t *testing.T) {
	diffs := []Difference{unchangedScalarDiff(DiffPath{"image", "repo"}, "nginx")}
	out := (&CompactFormatter{}).Format(diffs, inverseFormatOptions())

	if !strings.Contains(out, "Found 1 unchanged value(s)") {
		t.Errorf("missing inverse header, got:\n%s", out)
	}
	if !strings.Contains(out, "= image.repo : nginx") {
		t.Errorf("missing '= image.repo : nginx', got:\n%s", out)
	}
}

func TestInverseFormat_CompactHeaderUnchangedZeroUnaffected(t *testing.T) {
	// A normal modified diff must keep the original header wording byte-for-byte.
	diffs := []Difference{{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: 2}}
	out := (&CompactFormatter{}).Format(diffs, DefaultFormatOptions())
	if !strings.Contains(out, "Found 1 difference(s) (0 removed, 0 added, 1 modified)") {
		t.Errorf("normal header changed unexpectedly, got:\n%s", out)
	}
}

func TestInverseFormat_CompactColor(t *testing.T) {
	opts := &FormatOptions{Color: true, Unchanged: true}
	ctx := resolvedPalette(opts).ColorCode(ColorRoleContext, false)
	out := (&CompactFormatter{}).Format(
		[]Difference{unchangedScalarDiff(DiffPath{"image", "repo"}, "nginx")}, opts,
	)

	// Header is color-terminated and followed by a blank line.
	if !strings.Contains(out, colorReset+"\n\n") {
		t.Errorf("expected reset+blank line after header, got %q", out)
	}
	// The "=" indicator is wrapped in the neutral context color.
	if !strings.Contains(out, ctx+"="+colorReset) {
		t.Errorf("expected colored '=' indicator, got %q", out)
	}
	// The value is wrapped in the neutral context color.
	if !strings.Contains(out, " : "+ctx+"nginx"+colorReset) {
		t.Errorf("expected colored value, got %q", out)
	}
}

func TestInverseFormat_BriefUnchangedCounting(t *testing.T) {
	// Exactly one unchanged → "1 unchanged" (kills the >1 boundary mutant).
	one := (&BriefFormatter{}).Format([]Difference{unchangedScalarDiff(DiffPath{"a"}, "x")}, nil)
	if strings.TrimSpace(one) != "1 unchanged" {
		t.Errorf("expected '1 unchanged', got %q", one)
	}
	// Zero unchanged among other diffs → no "unchanged" text (kills >=0 / >-1 mutants).
	zero := (&BriefFormatter{}).Format(
		[]Difference{{Path: DiffPath{"a"}, Type: DiffModified, From: 1, To: 2}}, nil,
	)
	if strings.Contains(zero, "unchanged") {
		t.Errorf("did not expect 'unchanged' with zero unchanged entries, got %q", zero)
	}
}

func TestInverseFormat_Brief(t *testing.T) {
	diffs := []Difference{
		unchangedScalarDiff(DiffPath{"a"}, "x"),
		unchangedScalarDiff(DiffPath{"b"}, "y"),
	}
	out := (&BriefFormatter{}).Format(diffs, nil)
	if strings.TrimSpace(out) != "2 unchanged" {
		t.Errorf("expected '2 unchanged', got %q", out)
	}
	single := (&BriefFormatter{}).FormatSingle(diffs[0], nil)
	if single != "= a\n" {
		t.Errorf("expected '= a', got %q", single)
	}
}

func TestInverseFormat_GitHub(t *testing.T) {
	diff := unchangedScalarDiff(DiffPath{"a"}, "x")
	out := (&GitHubFormatter{}).FormatSingle(diff, nil)
	if !strings.Contains(out, "::notice title=YAML Unchanged::") {
		t.Errorf("expected notice/YAML Unchanged, got %q", out)
	}
	if !strings.Contains(out, "Unchanged: a = x") {
		t.Errorf("expected 'Unchanged: a = x' description, got %q", out)
	}
}

func TestInverseFormat_GitLab(t *testing.T) {
	if got := gitLabSeverity(DiffUnchanged); got != "info" {
		t.Errorf("gitLabSeverity(DiffUnchanged) = %q, want info", got)
	}
	if got := gitLabCheckName(DiffUnchanged); got != "diffyml/unchanged" {
		t.Errorf("gitLabCheckName(DiffUnchanged) = %q, want diffyml/unchanged", got)
	}
}

func TestInverseFormat_JSON(t *testing.T) {
	if got := jsonDiffTypeName(DiffUnchanged); got != "unchanged" {
		t.Errorf("jsonDiffTypeName(DiffUnchanged) = %q, want unchanged", got)
	}
	out := (&JSONFormatter{}).Format([]Difference{unchangedScalarDiff(DiffPath{"a"}, "x")}, DefaultFormatOptions())
	if !strings.Contains(out, `"type": "unchanged"`) {
		t.Errorf("expected unchanged type in JSON, got:\n%s", out)
	}
}

func TestInverseFormat_JSONPatchSkips(t *testing.T) {
	if got := rfc6902OpName(DiffUnchanged); got != "" {
		t.Errorf("rfc6902OpName(DiffUnchanged) = %q, want empty", got)
	}
	out := (&JSONPatchFormatter{}).Format([]Difference{unchangedScalarDiff(DiffPath{"a"}, "x")}, nil)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("expected empty patch for unchanged, got %q", out)
	}
}

func TestInverseFormat_DetailedHeaderAndBatch(t *testing.T) {
	diffs := []Difference{unchangedScalarDiff(DiffPath{"image", "repo"}, "nginx")}
	out := (&DetailedFormatter{}).Format(diffs, inverseFormatOptions())
	if !strings.Contains(out, "Found one unchanged value") {
		t.Errorf("missing inverse header, got:\n%s", out)
	}
	if !strings.Contains(out, "= one map entry unchanged:") {
		t.Errorf("missing unchanged batch descriptor, got:\n%s", out)
	}
	if !strings.Contains(out, "repo: nginx") {
		t.Errorf("missing rendered value, got:\n%s", out)
	}
}

func TestInverseFormat_DetailedRootDocuments(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{name: "scalar", yaml: "same\n", want: "(root level)\n  = one document unchanged:\n    ---\n    same\n\n"},
		{name: "sequence", yaml: "- alpha\n- beta\n", want: "(root level)\n  = one document unchanged:\n    ---\n    - alpha\n    - beta\n\n"},
		{name: "empty map", yaml: "{}\n", want: "(root level)\n  = one document unchanged:\n    ---\n    {}\n\n"},
		{name: "empty sequence", yaml: "[]\n", want: "(root level)\n  = one document unchanged:\n    ---\n    []\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs, err := Compare([]byte(tt.yaml), []byte(tt.yaml), &Options{Unchanged: true})
			if err != nil {
				t.Fatalf("Compare: %v", err)
			}
			got := (&DetailedFormatter{}).Format(diffs, &FormatOptions{OmitHeader: true, Unchanged: true})
			if got != tt.want {
				t.Fatalf("unexpected detailed output\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestInverseFormat_DetailedRootPlainMap(t *testing.T) {
	value := map[string]any{}
	diffs := []Difference{{Path: DiffPath{}, Type: DiffUnchanged, From: value, To: value}}

	got := (&DetailedFormatter{}).Format(diffs, &FormatOptions{OmitHeader: true, Unchanged: true})
	want := "(root level)\n  = one document unchanged:\n    ---\n    {}\n\n"
	if got != want {
		t.Fatalf("unexpected detailed output\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestInverseFormat_DetailedRootNonEmptyMaps(t *testing.T) {
	ordered := &OrderedMap{
		Keys:   []string{"name"},
		Values: map[string]any{"name": "same"},
	}
	tests := []struct {
		name  string
		value any
	}{
		{name: "ordered map", value: ordered},
		{name: "plain map", value: map[string]any{"name": "same"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := []Difference{{Path: DiffPath{}, Type: DiffUnchanged, From: tt.value, To: tt.value}}
			got := (&DetailedFormatter{}).Format(diffs, &FormatOptions{OmitHeader: true, Unchanged: true})
			want := "(root level)\n  = one document unchanged:\n    ---\n    name: same\n\n"
			if got != want {
				t.Fatalf("unexpected detailed output\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestRenderDocumentValue_NormalEmptyMapCompatibility(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "ordered map", value: NewOrderedMap()},
		{name: "plain map", value: map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			(&DetailedFormatter{}).renderDocumentValue(&sb, tt.value, "+", 4, DefaultFormatOptions())
			if got, want := sb.String(), "    ---\n"; got != want {
				t.Fatalf("normal empty-map rendering changed\ngot: %q\nwant: %q", got, want)
			}
		})
	}
}

func TestDetailedFormat_EmptyPathAdditionRemainsMapEntry(t *testing.T) {
	diffs := []Difference{{Path: DiffPath{}, Type: DiffAdded, To: "value"}}
	got := (&DetailedFormatter{}).Format(diffs, &FormatOptions{OmitHeader: true})
	if !strings.Contains(got, "+ one map entry added:") {
		t.Fatalf("empty-path addition must remain a map entry, got:\n%s", got)
	}
	if strings.Contains(got, "document added") {
		t.Fatalf("empty-path addition was misclassified as a document, got:\n%s", got)
	}
}

func TestRenderDocumentValue_NormalSequenceCompatibility(t *testing.T) {
	var sb strings.Builder
	opts := DefaultFormatOptions()
	(&DetailedFormatter{}).renderDocumentValue(&sb, []any{"alpha"}, "+", 4, opts)

	if got, want := sb.String(), "    ---\n    [alpha]\n"; got != want {
		t.Fatalf("normal sequence rendering changed\ngot: %q\nwant: %q", got, want)
	}
}

func TestInverseFormat_DetailedNeutralPalette(t *testing.T) {
	diffs := []Difference{unchangedScalarDiff(DiffPath{"image", "repo"}, "nginx")}
	// Exercise both the true-color neutral palette and the flat neutral palette
	// branches reached only by DiffUnchanged entry batches.
	for _, tc := range []struct {
		name    string
		opts    *FormatOptions
		neutral string // a color code unique to the neutral palette
	}{
		{"truecolor", &FormatOptions{Color: true, TrueColor: true}, TrueColorCode(220, 220, 220)},
		{"flat", &FormatOptions{Color: true, TrueColor: false}, colorWhite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := (&DetailedFormatter{}).Format(diffs, tc.opts)
			if !strings.Contains(out, "nginx") {
				t.Errorf("expected rendered value, got:\n%s", out)
			}
			if !strings.Contains(out, tc.neutral) {
				t.Errorf("expected neutral palette code %q in output, got:\n%s", tc.neutral, out)
			}
		})
	}
}

func TestInverseFormat_DiffTypeForSymbol(t *testing.T) {
	cases := map[string]DiffType{"+": DiffAdded, "=": DiffUnchanged, "-": DiffRemoved}
	for sym, want := range cases {
		if got := diffTypeForSymbol(sym); got != want {
			t.Errorf("diffTypeForSymbol(%q) = %v, want %v", sym, got, want)
		}
	}
}

func TestInverseFormat_DetailedListEntry(t *testing.T) {
	// A numeric-last path is a list entry; covers the list branch of the batch.
	diffs := []Difference{{Path: DiffPath{"ports", "0"}, Type: DiffUnchanged, From: 80, To: 80}}
	out := (&DetailedFormatter{}).Format(diffs, DefaultFormatOptions())
	if !strings.Contains(out, "one list entry unchanged:") {
		t.Errorf("expected list entry batch, got:\n%s", out)
	}
	if !strings.Contains(out, "- 80") {
		t.Errorf("expected '- 80', got:\n%s", out)
	}
}

// TestIsListEntryDiff_UnchangedTrustsContainerKind pins the DiffUnchanged branch
// in isListEntryDiff. Inverse mode emits raw collapsed values, so a map subtree
// whose value carries a name/id key must NOT be treated as a list entry (the
// hasIdentifierField heuristic would misfire), while a sequence-collapsed entry
// at a non-numeric identifier path must be. Kills BRANCH_IF on the
// `diff.Type == DiffUnchanged` guard and the boolean mutants on
// `return diff.listEntry`.
func TestIsListEntryDiff_UnchangedTrustsContainerKind(t *testing.T) {
	// Value carries a top-level "name" key, which would trip hasIdentifierField.
	withName := &OrderedMap{Keys: []string{"name"}, Values: map[string]any{"name": "foo"}}

	// Map collapse (listEntry=false) at a non-numeric path: must be a map entry.
	mapEntry := Difference{Path: DiffPath{"config"}, Type: DiffUnchanged, From: withName, To: withName, listEntry: false}
	if isListEntryDiff(mapEntry) {
		t.Error("collapsed map subtree (listEntry=false) must not be a list entry")
	}

	// Sequence collapse (listEntry=true) at an identifier path: must be a list.
	listEntry := Difference{Path: DiffPath{"containers", "app"}, Type: DiffUnchanged, From: withName, To: withName, listEntry: true}
	if !isListEntryDiff(listEntry) {
		t.Error("collapsed sequence element (listEntry=true) must be a list entry")
	}
}
