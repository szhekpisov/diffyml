package diffyml

import (
	"strings"
	"testing"
)

// Task 1.3: Unit tests for columnLayout

func TestNewColumnLayout_DefaultWidth(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil columnLayout at default width (80)")
	}
	if cl.totalWidth != 80 {
		t.Errorf("expected totalWidth=80, got %d", cl.totalWidth)
	}
	// indent=4, separator display width=3, available=80-4-3=73
	// leftWidth=73/2=36, rightWidth=73-36=37
	if cl.leftWidth != 36 {
		t.Errorf("expected leftWidth=36, got %d", cl.leftWidth)
	}
	if cl.rightWidth != 37 {
		t.Errorf("expected rightWidth=37, got %d", cl.rightWidth)
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
	// available=120-4-3=113, left=56, right=57
	if cl.leftWidth != 56 {
		t.Errorf("expected leftWidth=56, got %d", cl.leftWidth)
	}
	if cl.rightWidth != 57 {
		t.Errorf("expected rightWidth=57, got %d", cl.rightWidth)
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
	// available=40-4-3=33, left=16, right=17
	if cl.leftWidth != 16 {
		t.Errorf("expected leftWidth=16, got %d", cl.leftWidth)
	}
	if cl.rightWidth != 17 {
		t.Errorf("expected rightWidth=17, got %d", cl.rightWidth)
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
	// GetTerminalWidth enforces minimum of 40, so we test the guard logic
	// directly by creating a layout with a narrow width that bypasses GetTerminalWidth.
	// At width 40 (minimum), available=40-4-3=33, left=16 >= 12 — this is fine.
	// The nil guard protects against future changes to indent/separator/minimum.
	// Verify that width=40 (minimum enforceable) does NOT return nil:
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
	// Need leftWidth >= 12. available = totalWidth - 4 - 3 = totalWidth - 7
	// leftWidth = available/2 >= 12 => available >= 24 => totalWidth >= 31
	// But minimum terminal width is 40, so test at 40
	opts := DefaultFormatOptions()
	opts.Width = 40
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil columnLayout at width 40")
	}
	if cl.leftWidth < 12 {
		t.Errorf("expected leftWidth >= 12 (minTableColumnWidth), got %d", cl.leftWidth)
	}
}

// Truncation tests

func TestColumnLayout_Truncate_ShortString(t *testing.T) {
	cl := &columnLayout{leftWidth: 20, rightWidth: 20}
	result := cl.truncate("hello", 20)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestColumnLayout_Truncate_ExactFit(t *testing.T) {
	cl := &columnLayout{leftWidth: 5, rightWidth: 5}
	result := cl.truncate("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestColumnLayout_Truncate_Overflow(t *testing.T) {
	cl := &columnLayout{leftWidth: 10, rightWidth: 10}
	result := cl.truncate("hello world!", 5)
	// Should truncate to 4 chars + ellipsis = "hell…"
	if result != "hell…" {
		t.Errorf("expected 'hell…', got %q", result)
	}
}

func TestColumnLayout_Truncate_UnicodeSymbols(t *testing.T) {
	cl := &columnLayout{leftWidth: 10, rightWidth: 10}
	// "→±⇆" is 3 runes, should fit in width 5
	result := cl.truncate("→±⇆", 5)
	if result != "→±⇆" {
		t.Errorf("expected '→±⇆', got %q", result)
	}
}

func TestColumnLayout_Truncate_UnicodeOverflow(t *testing.T) {
	cl := &columnLayout{leftWidth: 10, rightWidth: 10}
	// "→±⇆↵·…" is 6 runes, truncate to width 4 => 3 runes + "…"
	result := cl.truncate("→±⇆↵·…", 4)
	if result != "→±⇆…" {
		t.Errorf("expected '→±⇆…', got %q", result)
	}
}

func TestColumnLayout_Truncate_EmptyString(t *testing.T) {
	cl := &columnLayout{leftWidth: 10, rightWidth: 10}
	result := cl.truncate("", 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestColumnLayout_Truncate_WidthOne(t *testing.T) {
	cl := &columnLayout{leftWidth: 1, rightWidth: 1}
	result := cl.truncate("hello", 1)
	if result != "…" {
		t.Errorf("expected '…', got %q", result)
	}
}

// PadRight tests

func TestColumnLayout_PadRight_ShortString(t *testing.T) {
	cl := &columnLayout{leftWidth: 10, rightWidth: 10}
	result := cl.padRight("hi", 5)
	if result != "hi   " {
		t.Errorf("expected 'hi   ', got %q", result)
	}
}

func TestColumnLayout_PadRight_ExactFit(t *testing.T) {
	cl := &columnLayout{leftWidth: 5, rightWidth: 5}
	result := cl.padRight("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestColumnLayout_PadRight_LongerString(t *testing.T) {
	cl := &columnLayout{leftWidth: 3, rightWidth: 3}
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

	var sb strings.Builder
	cl.formatRow(&sb, "old value", "new value", "", "", opts)
	output := sb.String()

	// Should contain indent (4 spaces), left value, separator (→), right value
	if !strings.Contains(output, "    ") {
		t.Errorf("expected 4-space indent in output: %q", output)
	}
	if !strings.Contains(output, "old value") {
		t.Errorf("expected 'old value' in output: %q", output)
	}
	if !strings.Contains(output, " → ") {
		t.Errorf("expected ' → ' separator in output: %q", output)
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

	var sb strings.Builder
	cl.formatRow(&sb, "old", "new", colorRed, colorGreen, opts)
	output := sb.String()

	// Left value should be wrapped in red
	if !strings.Contains(output, colorRed+"old") {
		t.Errorf("expected red-colored 'old' in output: %q", output)
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
	var sb strings.Builder
	cl.formatRow(&sb, longValue, longValue, "", "", opts)
	output := sb.String()

	// Output should contain ellipsis for truncated values
	if !strings.Contains(output, "…") {
		t.Errorf("expected ellipsis for truncated values in output: %q", output)
	}
}

func TestColumnLayout_FormatRow_EmptyLeft(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	var sb strings.Builder
	cl.formatRow(&sb, "", "added", "", colorGreen, opts)
	output := sb.String()

	if !strings.Contains(output, "added") {
		t.Errorf("expected 'added' in right column: %q", output)
	}
	if !strings.Contains(output, " → ") {
		t.Errorf("expected separator even with empty left: %q", output)
	}
}

func TestColumnLayout_FormatRow_EmptyRight(t *testing.T) {
	opts := DefaultFormatOptions()
	cl := newColumnLayout(opts)
	if cl == nil {
		t.Fatal("expected non-nil layout")
	}

	var sb strings.Builder
	cl.formatRow(&sb, "removed", "", colorRed, "", opts)
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

	var sb strings.Builder
	cl.formatRow(&sb, "A", "B", "", "", opts)
	output := sb.String()

	// The output should be: "    " + padded_left + " → " + right + "\n"
	// Check it starts with 4 spaces (indent)
	if !strings.HasPrefix(output, "    ") {
		t.Errorf("expected output to start with 4 spaces, got: %q", output)
	}

	// Arrow separator should be present
	arrowIdx := strings.Index(output, " → ")
	if arrowIdx < 0 {
		t.Fatalf("expected ' → ' separator in output: %q", output)
	}

	// Left value "A" should be before arrow
	leftPart := output[4:arrowIdx]
	if !strings.HasPrefix(leftPart, "A") {
		t.Errorf("expected left part to start with 'A', got: %q", leftPart)
	}

	// Right value "B" should be after arrow
	rightPart := output[arrowIdx+len(" → "):]
	rightPart = strings.TrimRight(rightPart, "\n")
	if !strings.HasPrefix(rightPart, "B") {
		t.Errorf("expected right part to start with 'B', got: %q", rightPart)
	}
}
