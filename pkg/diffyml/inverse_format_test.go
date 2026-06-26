package diffyml

import (
	"strings"
	"testing"
)

// unchangedScalarDiff builds a DiffUnchanged entry for a map leaf.
func unchangedScalarDiff(path DiffPath, val any) Difference {
	return Difference{Path: path, Type: DiffUnchanged, From: val, To: val}
}

func TestInverseFormat_Compact(t *testing.T) {
	diffs := []Difference{unchangedScalarDiff(DiffPath{"image", "repo"}, "nginx")}
	out := (&CompactFormatter{}).Format(diffs, DefaultFormatOptions())

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
	opts := &FormatOptions{Color: true}
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
	out := (&DetailedFormatter{}).Format(diffs, DefaultFormatOptions())
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

func TestInverseFormat_CountUnchanged(t *testing.T) {
	diffs := []Difference{
		unchangedScalarDiff(DiffPath{"a"}, 1),
		{Path: DiffPath{"b"}, Type: DiffModified, From: 1, To: 2},
		unchangedScalarDiff(DiffPath{"c"}, 3),
	}
	if got := countUnchanged(diffs); got != 2 {
		t.Errorf("countUnchanged = %d, want 2", got)
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
