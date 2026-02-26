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
const separatorDisplay = "  "

// separatorDisplayWidth is the display width of the separator in terminal columns.
const separatorDisplayWidth = 2

// tableIndent is the number of spaces for left indentation in table rows.
const tableIndent = 4

// columnLayout holds computed column widths for side-by-side rendering.
type columnLayout struct {
	totalWidth int    // Total available terminal width
	indent     int    // Left indentation (spaces)
	separator  string // Visual separator between columns
	available  int    // Usable width for content (totalWidth - indent - separatorDisplayWidth)
}

// newColumnLayout creates a column layout from format options.
// Returns nil when table style is disabled or the terminal is too narrow.
func newColumnLayout(opts *FormatOptions) *columnLayout {
	if opts.NoTableStyle {
		return nil
	}

	totalWidth := GetTerminalWidth(opts.Width)
	available := totalWidth - tableIndent - separatorDisplayWidth

	if available/2 < minTableColumnWidth {
		return nil
	}

	return &columnLayout{
		totalWidth: totalWidth,
		indent:     tableIndent,
		separator:  separatorDisplay,
		available:  available,
	}
}

// computeWidths calculates adaptive left and right column widths
// based on the actual content of the lines to be rendered.
func (cl *columnLayout) computeWidths(leftLines, rightLines []string) (leftW, rightW int) {
	maxLeft := 0
	for _, line := range leftLines {
		if n := utf8.RuneCountInString(line); n > maxLeft {
			maxLeft = n
		}
	}
	maxRight := 0
	for _, line := range rightLines {
		if n := utf8.RuneCountInString(line); n > maxRight {
			maxRight = n
		}
	}

	// One side empty
	if maxLeft == 0 {
		return 0, cl.available
	}
	if maxRight == 0 {
		return maxLeft, 0
	}

	// Both fit
	if maxLeft+maxRight <= cl.available {
		return maxLeft, cl.available - maxLeft
	}

	// Overflow: proportional allocation with minimum enforcement
	total := maxLeft + maxRight
	leftW = cl.available * maxLeft / total
	if leftW < minTableColumnWidth {
		leftW = minTableColumnWidth
	}
	rightW = cl.available - leftW
	if rightW < minTableColumnWidth {
		rightW = minTableColumnWidth
		leftW = cl.available - rightW
	}
	return leftW, rightW
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

// formatRow renders a single side-by-side row with adaptive column widths.
// leftW and rightW are the computed column widths from computeWidths.
// Three rendering modes:
//   - Both sides (leftW > 0 && rightW > 0): indent + padRight(left, leftW) + separator + right
//   - Left only (rightW == 0): indent + left (no padding, no separator)
//   - Right only (leftW == 0): indent + right (no separator)
func (cl *columnLayout) formatRow(sb *strings.Builder, left, right, leftColor, rightColor string, leftW, rightW int, opts *FormatOptions) {
	sb.WriteString(strings.Repeat(" ", cl.indent))

	if leftW > 0 && rightW > 0 {
		// Both sides
		left = cl.truncate(left, leftW)
		right = cl.truncate(right, rightW)
		left = cl.padRight(left, leftW)

		if opts.Color && leftColor != "" {
			sb.WriteString(leftColor)
			sb.WriteString(left)
			sb.WriteString(colorReset)
		} else {
			sb.WriteString(left)
		}
		sb.WriteString(cl.separator)
		if opts.Color && rightColor != "" {
			sb.WriteString(rightColor)
			sb.WriteString(right)
			sb.WriteString(colorReset)
		} else {
			sb.WriteString(right)
		}
	} else if rightW == 0 {
		// Left only — no padding, no separator
		left = cl.truncate(left, leftW)
		if opts.Color && leftColor != "" {
			sb.WriteString(leftColor)
			sb.WriteString(left)
			sb.WriteString(colorReset)
		} else {
			sb.WriteString(left)
		}
	} else {
		// Right only — no separator
		right = cl.truncate(right, rightW)
		if opts.Color && rightColor != "" {
			sb.WriteString(rightColor)
			sb.WriteString(right)
			sb.WriteString(colorReset)
		} else {
			sb.WriteString(right)
		}
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
