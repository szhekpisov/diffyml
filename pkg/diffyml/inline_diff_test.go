package diffyml

import (
	"strings"
	"testing"
)

// --- tokenize tests ---

func TestTokenize_WordsAndPunctuation(t *testing.T) {
	got := tokenize("192.168.10.1")
	want := []string{"192", ".", "168", ".", "10", ".", "1"}
	assertTokens(t, got, want)
}

func TestTokenize_VersionString(t *testing.T) {
	got := tokenize("demo:v1.20.1")
	want := []string{"demo", ":", "v1", ".", "20", ".", "1"}
	assertTokens(t, got, want)
}

func TestTokenize_IPRange(t *testing.T) {
	got := tokenize("192.168.10.1-192.168.10.25")
	want := []string{"192", ".", "168", ".", "10", ".", "1", "-", "192", ".", "168", ".", "10", ".", "25"}
	assertTokens(t, got, want)
}

func TestTokenize_PureWord(t *testing.T) {
	got := tokenize("hello")
	want := []string{"hello"}
	assertTokens(t, got, want)
}

func TestTokenize_PurePunctuation(t *testing.T) {
	got := tokenize(".:!")
	want := []string{".", ":", "!"}
	assertTokens(t, got, want)
}

func TestTokenize_Empty(t *testing.T) {
	got := tokenize("")
	if len(got) != 0 {
		t.Errorf("expected empty tokens, got %v", got)
	}
}

func TestTokenize_Unicode(t *testing.T) {
	got := tokenize("café.résumé")
	want := []string{"caf", "é", ".", "r", "é", "sum", "é"}
	assertTokens(t, got, want)
}

func TestTokenize_LeadingTrailingPunctuation(t *testing.T) {
	got := tokenize("--name--")
	want := []string{"-", "-", "name", "-", "-"}
	assertTokens(t, got, want)
}

// --- computeInlineDiff tests ---

func TestComputeInlineDiff_VersionBump(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff("demo:v1.20.1", "demo:v1.21.1")
	if fromSegs == nil || toSegs == nil {
		t.Fatal("expected non-nil segments")
	}

	// "demo:v1." unchanged, "20" changed, ".1" unchanged
	assertSegments(t, "from", fromSegs, []inlineSegment{
		{Text: "demo:v1.", Changed: false},
		{Text: "20", Changed: true},
		{Text: ".1", Changed: false},
	})
	assertSegments(t, "to", toSegs, []inlineSegment{
		{Text: "demo:v1.", Changed: false},
		{Text: "21", Changed: true},
		{Text: ".1", Changed: false},
	})
}

func TestComputeInlineDiff_IPChange(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff("192.168.10.1", "192.168.11.1")
	if fromSegs == nil || toSegs == nil {
		t.Fatal("expected non-nil segments")
	}

	assertSegments(t, "from", fromSegs, []inlineSegment{
		{Text: "192.168.", Changed: false},
		{Text: "10", Changed: true},
		{Text: ".1", Changed: false},
	})
	assertSegments(t, "to", toSegs, []inlineSegment{
		{Text: "192.168.", Changed: false},
		{Text: "11", Changed: true},
		{Text: ".1", Changed: false},
	})
}

func TestComputeInlineDiff_IPRangeMultipleChanges(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff(
		"192.168.10.1-192.168.10.25",
		"192.168.11.1-192.168.11.25",
	)
	if fromSegs == nil || toSegs == nil {
		t.Fatal("expected non-nil segments")
	}

	// Two "10" → "11" changes with unchanged parts between
	assertSegments(t, "from", fromSegs, []inlineSegment{
		{Text: "192.168.", Changed: false},
		{Text: "10", Changed: true},
		{Text: ".1-192.168.", Changed: false},
		{Text: "10", Changed: true},
		{Text: ".25", Changed: false},
	})
	assertSegments(t, "to", toSegs, []inlineSegment{
		{Text: "192.168.", Changed: false},
		{Text: "11", Changed: true},
		{Text: ".1-192.168.", Changed: false},
		{Text: "11", Changed: true},
		{Text: ".25", Changed: false},
	})
}

func TestComputeInlineDiff_Identical(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff("same-value", "same-value")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil for identical strings")
	}
}

func TestComputeInlineDiff_Empty(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff("", "something")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil when from is empty")
	}

	fromSegs, toSegs = computeInlineDiff("something", "")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil when to is empty")
	}
}

func TestComputeInlineDiff_TooShort(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff("ab", "cd")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil for very short strings")
	}
}

func TestComputeInlineDiff_Multiline(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff("line1\nline2", "line1\nline3")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil for multiline strings")
	}
}

func TestComputeInlineDiff_CompletelyDifferent(t *testing.T) {
	fromSegs, toSegs := computeInlineDiff("alpha-beta-gamma", "one-two-three")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil when strings are too different")
	}
}

func TestComputeInlineDiff_ThresholdBoundary(t *testing.T) {
	// 10 tokens total, 3 kept = exactly 30% — should pass threshold
	// "a.b.c.d.e" → tokens: a . b . c . d . e (9 tokens)
	// Change most but keep 3+: "a.b.c.x.y" keeps a, ., b, ., c, . = 6 kept — well above
	fromSegs, toSegs := computeInlineDiff("a.b.c.d.e", "a.b.c.x.y")
	if fromSegs == nil || toSegs == nil {
		t.Error("expected segments when similarity is above threshold")
	}
}

func TestComputeInlineDiff_BelowThreshold(t *testing.T) {
	// Strings with very little in common
	fromSegs, toSegs := computeInlineDiff("aaa.bbb.ccc", "xxx.yyy.zzz")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil when similarity is below threshold")
	}
}

func TestComputeInlineDiff_ExactlyTwoTokens(t *testing.T) {
	// Both sides have exactly 2 tokens — should be skipped as too short.
	// "a." → ["a", "."], "b." → ["b", "."]
	fromSegs, toSegs := computeInlineDiff("a.", "b.")
	if fromSegs != nil || toSegs != nil {
		t.Error("expected nil for 2-token strings")
	}
}

func TestComputeInlineDiff_ExactThreshold30Percent(t *testing.T) {
	// Exactly 30% character similarity — on the boundary, should still produce
	// segments (the guard is strict less-than, not less-than-or-equal).
	// "ab-cd.ef-g" vs "xy-zw.mn-q": shared tokens are "-", ".", "-" = 3 chars.
	// longer = 10, keepChars*10 = 30 == longer*3 = 30 → not < → passes.
	fromSegs, toSegs := computeInlineDiff("ab-cd.ef-g", "xy-zw.mn-q")
	if fromSegs == nil || toSegs == nil {
		t.Error("expected segments at exactly 30% similarity boundary")
	}
}

// --- renderInlineSegments tests ---

func TestRenderInlineSegments_NoColor(t *testing.T) {
	segments := []inlineSegment{
		{Text: "hello.", Changed: false},
		{Text: "world", Changed: true},
	}
	var sb strings.Builder
	opts := &FormatOptions{Color: false}
	renderInlineSegments(&sb, segments, "", "", opts)

	got := sb.String()
	want := "hello.world"
	if got != want {
		t.Errorf("no-color: got %q, want %q", got, want)
	}
}

func TestRenderInlineSegments_WithColor(t *testing.T) {
	segments := []inlineSegment{
		{Text: "unchanged", Changed: false},
		{Text: "CHANGED", Changed: true},
		{Text: "tail", Changed: false},
	}
	var sb strings.Builder
	opts := &FormatOptions{Color: true}
	baseColor := colorRed
	dimColor := colorGray
	renderInlineSegments(&sb, segments, baseColor, dimColor, opts)

	got := sb.String()
	// Unchanged parts get dim color
	if !strings.Contains(got, colorGray+"unchanged") {
		t.Errorf("expected dim color on unchanged segment, got %q", got)
	}
	// Changed parts get bold + base color
	if !strings.Contains(got, styleBold+colorRed+"CHANGED"+styleBoldOff) {
		t.Errorf("expected bold+base on changed segment, got %q", got)
	}
	// Tail unchanged
	if !strings.Contains(got, colorGray+"tail") {
		t.Errorf("expected dim color on tail segment, got %q", got)
	}
}

func TestRenderInlineSegments_AllChanged(t *testing.T) {
	segments := []inlineSegment{
		{Text: "everything", Changed: true},
	}
	var sb strings.Builder
	opts := &FormatOptions{Color: true}
	renderInlineSegments(&sb, segments, colorGreen, colorGray, opts)

	got := sb.String()
	want := styleBold + colorGreen + "everything" + styleBoldOff
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderInlineSegments_AllUnchanged(t *testing.T) {
	segments := []inlineSegment{
		{Text: "nothing changed", Changed: false},
	}
	var sb strings.Builder
	opts := &FormatOptions{Color: true}
	renderInlineSegments(&sb, segments, colorGreen, colorGray, opts)

	got := sb.String()
	want := colorGray + "nothing changed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- dimColorCode tests ---

func TestDimColorCode_TrueColor(t *testing.T) {
	opts := &FormatOptions{Color: true, TrueColor: true}
	got := dimColorCode(ColorRoleRemoved, opts)
	// Default removed: RGB(185, 49, 27) → dim: ((185+128)/2, (49+128)/2, (27+128)/2) = (156, 88, 77)
	want := TrueColorCode(156, 88, 77)
	if got != want {
		t.Errorf("TrueColor dim removed: got %q, want %q", got, want)
	}
}

func TestDimColorCode_TrueColorAdded(t *testing.T) {
	opts := &FormatOptions{Color: true, TrueColor: true}
	got := dimColorCode(ColorRoleAdded, opts)
	// Default added: RGB(88, 191, 56) → dim: ((88+128)/2, (191+128)/2, (56+128)/2) = (108, 159, 92)
	want := TrueColorCode(108, 159, 92)
	if got != want {
		t.Errorf("TrueColor dim added: got %q, want %q", got, want)
	}
}

func TestDimColorCode_8Color(t *testing.T) {
	opts := &FormatOptions{Color: true, TrueColor: false}
	got := dimColorCode(ColorRoleRemoved, opts)
	if got != colorRed {
		t.Errorf("8-color dim removed: got %q, want %q", got, colorRed)
	}
}

func TestDimColorCode_CustomPalette(t *testing.T) {
	custom := &CustomColor{R: 200, G: 100, B: 50, ANSICode: colorCyan, IsCustom: true}
	palette := DefaultCustomColorPalette()
	palette.Removed = custom
	opts := &FormatOptions{Color: true, TrueColor: true, Palette: palette}

	got := dimColorCode(ColorRoleRemoved, opts)
	want := TrueColorCode((200+128)/2, (100+128)/2, (50+128)/2)
	if got != want {
		t.Errorf("custom TrueColor dim: got %q, want %q", got, want)
	}
}

// --- appendSegment tests ---

func TestAppendSegment_Coalesces(t *testing.T) {
	segs := []inlineSegment{{Text: "a", Changed: true}}
	segs = appendSegment(segs, "b", true)
	if len(segs) != 1 || segs[0].Text != "ab" {
		t.Errorf("expected coalesced segment 'ab', got %v", segs)
	}
}

func TestAppendSegment_NewSegment(t *testing.T) {
	segs := []inlineSegment{{Text: "a", Changed: true}}
	segs = appendSegment(segs, "b", false)
	if len(segs) != 2 {
		t.Errorf("expected 2 segments, got %d", len(segs))
	}
}

func TestAppendSegment_Empty(t *testing.T) {
	var segs []inlineSegment
	segs = appendSegment(segs, "a", false)
	if len(segs) != 1 || segs[0].Text != "a" || segs[0].Changed {
		t.Errorf("expected single unchanged segment 'a', got %v", segs)
	}
}

// --- helpers ---

func assertTokens(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("token count: got %d %v, want %d %v", len(got), got, len(want), want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func assertSegments(t *testing.T, label string, got, want []inlineSegment) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s segment count: got %d, want %d\ngot:  %+v\nwant: %+v", label, len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i].Text != want[i].Text || got[i].Changed != want[i].Changed {
			t.Errorf("%s segment[%d]: got %+v, want %+v", label, i, got[i], want[i])
		}
	}
}
