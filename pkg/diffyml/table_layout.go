// table_layout.go - Column layout for side-by-side (table-style) rendering.
//
// Encapsulates terminal width allocation and provides column-aware
// string formatting utilities for the detailed formatter's table mode.
package diffyml

import (
	"strings"
	"unicode/utf8"
)

// minTableColumnWidth is the minimum column width for table mode.
// If either column would be narrower than this, table mode is disabled.
const minTableColumnWidth = 12

// separatorDisplay is the visual separator between left and right columns.
const separatorDisplay = " → "

// separatorDisplayWidth is the display width of the separator in terminal columns.
// Note: len(" → ") returns 5 bytes because → is U+2192 (3 bytes in UTF-8),
// but it occupies only 1 terminal column. Display width = space + arrow + space = 3.
const separatorDisplayWidth = 3

// tableIndent is the number of spaces for left indentation in table rows.
const tableIndent = 4

// columnLayout holds computed column widths for side-by-side rendering.
type columnLayout struct {
	totalWidth int    // Total available terminal width
	indent     int    // Left indentation (spaces)
	separator  string // Visual separator between columns (e.g., " → ")
	leftWidth  int    // Computed left column width
	rightWidth int    // Computed right column width
}

// newColumnLayout creates a column layout from format options.
// Returns nil when table style is disabled or the terminal is too narrow.
func newColumnLayout(opts *FormatOptions) *columnLayout {
	if opts.NoTableStyle {
		return nil
	}

	totalWidth := GetTerminalWidth(opts.Width)
	available := totalWidth - tableIndent - separatorDisplayWidth
	leftWidth := available / 2
	rightWidth := available - leftWidth

	if leftWidth < minTableColumnWidth {
		return nil
	}

	return &columnLayout{
		totalWidth: totalWidth,
		indent:     tableIndent,
		separator:  separatorDisplay,
		leftWidth:  leftWidth,
		rightWidth: rightWidth,
	}
}

// truncate truncates a plain-text string to fit within the given column width.
// Appends "…" if truncation occurs.
// CONSTRAINT: s must be plain text with no ANSI escape sequences.
func (cl *columnLayout) truncate(s string, width int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount <= width {
		return s
	}
	if width <= 0 {
		return ""
	}
	if width == 1 {
		return "…"
	}
	// Keep width-1 runes + ellipsis
	i := 0
	kept := 0
	for kept < width-1 {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
		kept++
	}
	return s[:i] + "…"
}

// padRight pads a string with spaces to fill the given width.
// Uses rune count for width measurement.
func (cl *columnLayout) padRight(s string, width int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runeCount)
}

// formatRow renders a single side-by-side row with left value, separator, and right value.
// Color codes are applied after truncation and padding to prevent width miscalculation.
func (cl *columnLayout) formatRow(sb *strings.Builder, left, right, leftColor, rightColor string, opts *FormatOptions) {
	// Truncate plain text first
	left = cl.truncate(left, cl.leftWidth)
	right = cl.truncate(right, cl.rightWidth)

	// Pad left column to fixed width for alignment
	left = cl.padRight(left, cl.leftWidth)

	// Write indent
	sb.WriteString(strings.Repeat(" ", cl.indent))

	// Write left value with optional color
	if opts.Color && leftColor != "" {
		sb.WriteString(leftColor)
		sb.WriteString(left)
		sb.WriteString(colorReset)
	} else {
		sb.WriteString(left)
	}

	// Write separator
	sb.WriteString(cl.separator)

	// Write right value with optional color
	if opts.Color && rightColor != "" {
		sb.WriteString(rightColor)
		sb.WriteString(right)
		sb.WriteString(colorReset)
	} else {
		sb.WriteString(right)
	}

	sb.WriteString("\n")
}

// formatContextRow renders a context line spanning both columns.
func (cl *columnLayout) formatContextRow(sb *strings.Builder, text, colorCode string, opts *FormatOptions) {
	sb.WriteString(strings.Repeat(" ", cl.indent))
	if opts.Color && colorCode != "" {
		sb.WriteString(colorCode)
		sb.WriteString(text)
		sb.WriteString(colorReset)
	} else {
		sb.WriteString(text)
	}
	sb.WriteString("\n")
}

// formatAnnotationRow renders a centered annotation spanning the full width.
func (cl *columnLayout) formatAnnotationRow(sb *strings.Builder, text, colorCode string, opts *FormatOptions) {
	sb.WriteString(strings.Repeat(" ", cl.indent))
	if opts.Color && colorCode != "" {
		sb.WriteString(colorCode)
		sb.WriteString(text)
		sb.WriteString(colorReset)
	} else {
		sb.WriteString(text)
	}
	sb.WriteString("\n")
}
