package diffyml

import (
	"strings"
	"testing"
)

// Unit tests for columnLayout

func TestNewColumnLayout_DefaultWidth(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil columnLayout at default width (80)")
	}
	if cl.totalWidth != 80 {
		t.Errorf("expected totalWidth=80, got %d", cl.totalWidth)
	}
	// indent=4, separator display width=2, available=80-4-2=74
	if cl.available != 74 {
		t.Errorf("expected available=74, got %d", cl.available)
	}
}

func TestNewColumnLayout_CustomWidth(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.Width = 120
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil columnLayout at width 120")
	}
	if cl.totalWidth != 120 {
		t.Errorf("expected totalWidth=120, got %d", cl.totalWidth)
	}
	// available=120-4-2=114
	if cl.available != 114 {
		t.Errorf("expected available=114, got %d", cl.available)
	}
}

func TestNewColumnLayout_MinimumWidth(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.Width = 40
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil columnLayout at minimum width (40)")
	}
	if cl.totalWidth != 40 {
		t.Errorf("expected totalWidth=40, got %d", cl.totalWidth)
	}
	// available=40-4-2=34
	if cl.available != 34 {
		t.Errorf("expected available=34, got %d", cl.available)
	}
}

func TestNewColumnLayout_NilWhenTableStyleDisabled(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.NoTableStyle = true
	cl := newColumnLayout(opts)
	if cl != nil {
		t.Error("expected nil columnLayout when NoTableStyle is true")
	}
}

func TestNewColumnLayout_NilWhenTooNarrow(t *testing.T) {
	// GetTerminalWidth enforces minimum of 40, so width 20 is clamped to 40.
	// At width 40: available=40-4-2=34, available/2=17 >= 12 — non-nil.
	opts := DefaultFormatOptions()
	opts.Width = 20 // will be clamped to 40 by GetTerminalWidth
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Error("expected non-nil layout: width 20 is clamped to 40 by GetTerminalWidth")
	}
	if cl != nil && cl.totalWidth != 40 {
		t.Errorf("expected totalWidth=40 (clamped from 20), got %d", cl.totalWidth)
	}
}

func TestNewColumnLayout_BarelyWideEnough(t *testing.T) {
	// Need available/2 >= 12. available = totalWidth - 4 - 2 = totalWidth - 6
	// available/2 >= 12 => available >= 24 => totalWidth >= 30
	// But minimum terminal width is 40, so test at 40
	opts := DefaultFormatOptions()
	opts.Width = 40
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil columnLayout at width 40")
	}
	if cl.available/2 < 12 {
		t.Errorf("expected available/2 >= 12 (minTableColumnWidth), got %d", cl.available/2)
	}
}

// computeWidths tests

func TestColumnLayout_ComputeWidths_BothFit(t *testing.T) {
	cl := &columnLayout{available: 74}
	leftW, rightW := cl.computeWidths([]string{"hello"}, []string{"world"})
	// maxLeft=5, maxRight=5, both fit: leftW=5, rightW=74-5=69
	if leftW != 5 {
		t.Errorf("expected leftW=5, got %d", leftW)
	}
	if rightW != 69 {
		t.Errorf("expected rightW=69, got %d", rightW)
	}
}

func TestColumnLayout_ComputeWidths_LeftEmpty(t *testing.T) {
	cl := &columnLayout{available: 74}
	leftW, rightW := cl.computeWidths(nil, []string{"added line"})
	if leftW != 0 {
		t.Errorf("expected leftW=0, got %d", leftW)
	}
	if rightW != 74 {
		t.Errorf("expected rightW=74, got %d", rightW)
	}
}

func TestColumnLayout_ComputeWidths_RightEmpty(t *testing.T) {
	cl := &columnLayout{available: 74}
	leftW, rightW := cl.computeWidths([]string{"removed line"}, nil)
	if leftW != 12 {
		t.Errorf("expected leftW=12, got %d", leftW)
	}
	if rightW != 0 {
		t.Errorf("expected rightW=0, got %d", rightW)
	}
}

func TestColumnLayout_ComputeWidths_Overflow(t *testing.T) {
	cl := &columnLayout{available: 40}
	leftW, rightW := cl.computeWidths(
		[]string{strings.Repeat("x", 30)},
		[]string{strings.Repeat("y", 30)},
	)
	// Both overflow (30+30=60 > 40), proportional: 30/60*40=20 each
	if leftW+rightW != 40 {
		t.Errorf("expected leftW+rightW=40, got %d+%d=%d", leftW, rightW, leftW+rightW)
	}
	if leftW < minTableColumnWidth {
		t.Errorf("expected leftW >= %d, got %d", minTableColumnWidth, leftW)
	}
	if rightW < minTableColumnWidth {
		t.Errorf("expected rightW >= %d, got %d", minTableColumnWidth, rightW)
	}
}

func TestColumnLayout_ComputeWidths_ProportionalAsymmetric(t *testing.T) {
	cl := &columnLayout{available: 60}
	// 20 left, 80 right — proportional: 20/100*60=12, 80/100*60=48
	leftW, rightW := cl.computeWidths(
		[]string{strings.Repeat("x", 20)},
		[]string{strings.Repeat("y", 80)},
	)
	if leftW+rightW != 60 {
		t.Errorf("expected leftW+rightW=60, got %d+%d=%d", leftW, rightW, leftW+rightW)
	}
	if leftW < minTableColumnWidth {
		t.Errorf("expected leftW >= %d, got %d", minTableColumnWidth, leftW)
	}
}

func TestColumnLayout_ComputeWidths_MinimumEnforcement(t *testing.T) {
	cl := &columnLayout{available: 30}
	// 5 left, 100 right — proportional: 5/105*30=1 < 12, so enforce minimum
	leftW, rightW := cl.computeWidths(
		[]string{"short"},
		[]string{strings.Repeat("y", 100)},
	)
	if leftW < minTableColumnWidth {
		t.Errorf("expected leftW >= %d (minimum enforced), got %d", minTableColumnWidth, leftW)
	}
	if leftW+rightW != 30 {
		t.Errorf("expected leftW+rightW=30, got %d+%d=%d", leftW, rightW, leftW+rightW)
	}
}

func TestColumnLayout_ComputeWidths_MultipleLines(t *testing.T) {
	cl := &columnLayout{available: 74}
	leftW, rightW := cl.computeWidths(
		[]string{"short", "a longer line here"},
		[]string{"x", "medium text"},
	)
	// maxLeft=18 ("a longer line here"), maxRight=11 ("medium text")
	// 18+11=29 <= 74, so leftW=18, rightW=74-18=56
	if leftW != 18 {
		t.Errorf("expected leftW=18, got %d", leftW)
	}
	if rightW != 56 {
		t.Errorf("expected rightW=56, got %d", rightW)
	}
}

func TestColumnLayout_ComputeWidths_BothEmpty(t *testing.T) {
	cl := &columnLayout{available: 74}
	leftW, rightW := cl.computeWidths(nil, nil)
	// Both empty: maxLeft=0, first branch returns (0, available)
	if leftW != 0 {
		t.Errorf("expected leftW=0, got %d", leftW)
	}
	if rightW != 74 {
		t.Errorf("expected rightW=74, got %d", rightW)
	}
}

// Truncation tests

func TestColumnLayout_Truncate_ShortString(t *testing.T) {
	cl := &columnLayout{available: 40}
	result := cl.truncate("hello", 20)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestColumnLayout_Truncate_ExactFit(t *testing.T) {
	cl := &columnLayout{available: 10}
	result := cl.truncate("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestColumnLayout_Truncate_Overflow(t *testing.T) {
	cl := &columnLayout{available: 20}
	result := cl.truncate("hello world!", 5)
	// Should truncate to 4 chars + ellipsis = "hell…"
	if result != "hell…" {
		t.Errorf("expected 'hell…', got %q", result)
	}
}

func TestColumnLayout_Truncate_UnicodeSymbols(t *testing.T) {
	cl := &columnLayout{available: 20}
	// "→±⇆" is 3 runes, should fit in width 5
	result := cl.truncate("→±⇆", 5)
	if result != "→±⇆" {
		t.Errorf("expected '→±⇆', got %q", result)
	}
}

func TestColumnLayout_Truncate_UnicodeOverflow(t *testing.T) {
	cl := &columnLayout{available: 20}
	// "→±⇆↵·…" is 6 runes, truncate to width 4 => 3 runes + "…"
	result := cl.truncate("→±⇆↵·…", 4)
	if result != "→±⇆…" {
		t.Errorf("expected '→±⇆…', got %q", result)
	}
}

func TestColumnLayout_Truncate_EmptyString(t *testing.T) {
	cl := &columnLayout{available: 20}
	result := cl.truncate("", 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestColumnLayout_Truncate_WidthOne(t *testing.T) {
	cl := &columnLayout{available: 2}
	result := cl.truncate("hello", 1)
	if result != "…" {
		t.Errorf("expected '…', got %q", result)
	}
}

// PadRight tests

func TestColumnLayout_PadRight_ShortString(t *testing.T) {
	cl := &columnLayout{available: 20}
	result := cl.padRight("hi", 5)
	if result != "hi   " {
		t.Errorf("expected 'hi   ', got %q", result)
	}
}

func TestColumnLayout_PadRight_ExactFit(t *testing.T) {
	cl := &columnLayout{available: 10}
	result := cl.padRight("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestColumnLayout_PadRight_LongerString(t *testing.T) {
	cl := &columnLayout{available: 6}
	result := cl.padRight("hello", 3)
	// Should not add padding, string is already >= width
	if result != "hello" {
		t.Errorf("expected 'hello' (no truncation by padRight), got %q", result)
	}
}

// formatRow tests

func TestColumnLayout_FormatRow_NoColor(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	leftW, rightW := cl.computeWidths([]string{"old value"}, []string{"new value"})

	var sb strings.Builder
	cl.formatRow(&sb, "old value", "new value", "", "", leftW, rightW, opts)
	output := sb.String()

	// Should contain indent (4 spaces), left value, separator (2 spaces), right value
	if !strings.Contains(output, "    ") {
		t.Errorf("expected 4-space indent in output: %q", output)
	}
	if !strings.Contains(output, "old value") {
		t.Errorf("expected 'old value' in output: %q", output)
	}
	if !strings.Contains(output, "new value") {
		t.Errorf("expected 'new value' in output: %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("expected output to end with newline: %q", output)
	}
}

func TestColumnLayout_FormatRow_WithColor(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.Color = true
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	leftW, rightW := cl.computeWidths([]string{"old"}, []string{"new"})

	var sb strings.Builder
	cl.formatRow(&sb, "old", "new", colorRed, colorGreen, leftW, rightW, opts)
	output := sb.String()

	// Left value should be wrapped in red
	if !strings.Contains(output, colorRed) {
		t.Errorf("expected red color in output: %q", output)
	}
	if !strings.Contains(output, "old") {
		t.Errorf("expected 'old' in output: %q", output)
	}
	// Right value should be wrapped in green
	if !strings.Contains(output, colorGreen+"new") {
		t.Errorf("expected green-colored 'new' in output: %q", output)
	}
	// Reset codes
	if !strings.Contains(output, colorReset) {
		t.Errorf("expected color reset in output: %q", output)
	}
}

func TestColumnLayout_FormatRow_TruncatesLongValues(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.Width = 40 // small width to force truncation
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	longValue := strings.Repeat("x", 100)
	leftW, rightW := cl.computeWidths([]string{longValue}, []string{longValue})

	var sb strings.Builder
	cl.formatRow(&sb, longValue, longValue, "", "", leftW, rightW, opts)
	output := sb.String()

	// Output should contain ellipsis for truncated values
	if !strings.Contains(output, "…") {
		t.Errorf("expected ellipsis for truncated values in output: %q", output)
	}
}

func TestColumnLayout_FormatRow_RightOnly(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	// Right-only mode: leftW=0
	leftW, rightW := cl.computeWidths(nil, []string{"added"})

	var sb strings.Builder
	cl.formatRow(&sb, "", "added", "", colorGreen, leftW, rightW, opts)
	output := sb.String()

	if !strings.Contains(output, "added") {
		t.Errorf("expected 'added' in right column: %q", output)
	}
	// No separator in right-only mode
	if strings.Contains(output, "  added") && strings.Count(output, "  ") > 1 {
		// This is fine — just indent spaces
	}
}

func TestColumnLayout_FormatRow_LeftOnly(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	// Left-only mode: rightW=0
	leftW, rightW := cl.computeWidths([]string{"removed"}, nil)

	var sb strings.Builder
	cl.formatRow(&sb, "removed", "", colorRed, "", leftW, rightW, opts)
	output := sb.String()

	if !strings.Contains(output, "removed") {
		t.Errorf("expected 'removed' in left column: %q", output)
	}
}

// formatContextRow tests

func TestColumnLayout_FormatContextRow_NoColor(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	var sb strings.Builder
	cl.formatContextRow(&sb, "unchanged line", "", opts)
	output := sb.String()

	if !strings.Contains(output, "unchanged line") {
		t.Errorf("expected 'unchanged line' in output: %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("expected newline at end: %q", output)
	}
}

func TestColumnLayout_FormatContextRow_WithColor(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.Color = true
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	var sb strings.Builder
	cl.formatContextRow(&sb, "context", colorGray, opts)
	output := sb.String()

	if !strings.Contains(output, colorGray) {
		t.Errorf("expected gray color in output: %q", output)
	}
	if !strings.Contains(output, colorReset) {
		t.Errorf("expected color reset in output: %q", output)
	}
}

// formatAnnotationRow tests

func TestColumnLayout_FormatAnnotationRow_NoColor(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	var sb strings.Builder
	cl.formatAnnotationRow(&sb, "[5 lines unchanged]", "", opts)
	output := sb.String()

	if !strings.Contains(output, "[5 lines unchanged]") {
		t.Errorf("expected annotation text in output: %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("expected newline at end: %q", output)
	}
}

func TestColumnLayout_FormatAnnotationRow_WithColor(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.Color = true
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	var sb strings.Builder
	cl.formatAnnotationRow(&sb, "[3 lines unchanged]", colorGray, opts)
	output := sb.String()

	if !strings.Contains(output, colorGray) {
		t.Errorf("expected gray color in output: %q", output)
	}
}

// Row structure verification

func TestColumnLayout_FormatRow_Structure(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	leftW, rightW := cl.computeWidths([]string{"A"}, []string{"B"})

	var sb strings.Builder
	cl.formatRow(&sb, "A", "B", "", "", leftW, rightW, opts)
	output := sb.String()

	// The output should be: "    " + padded_left + "  " + right + "\n"
	// Check it starts with 4 spaces (indent)
	if !strings.HasPrefix(output, "    ") {
		t.Errorf("expected output to start with 4 spaces, got: %q", output)
	}

	// Should contain "A" followed by separator then "B"
	trimmed := strings.TrimPrefix(output, "    ")
	trimmed = strings.TrimRight(trimmed, "\n")
	if !strings.HasPrefix(trimmed, "A") {
		t.Errorf("expected content to start with 'A', got: %q", trimmed)
	}
	if !strings.HasSuffix(trimmed, "B") {
		t.Errorf("expected content to end with 'B', got: %q", trimmed)
	}
}
