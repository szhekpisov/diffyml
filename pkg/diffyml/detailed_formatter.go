// detailed_formatter.go - Detailed human-readable output formatter.
//
// Renders differences in a path-grouped style with descriptive labels,
// structured value display, and multiline text diffs.
//
// Split across four files:
//   - detailed_formatter.go (this file) — core orchestration
//   - detailed_formatter_render.go — YAML value rendering
//   - detailed_formatter_linediff.go — LCS line-diff algorithm
//   - detailed_formatter_helpers.go — pure utility functions
package diffyml

import (
	"fmt"
	"strings"
)

// DetailedFormatter implements the Formatter interface for detailed output.
type DetailedFormatter struct{}

// pathGroup holds a path and its associated differences for grouping.
type pathGroup struct {
	Path  string
	Diffs []Difference
}

// Format renders differences in detailed human-readable style.
func (f *DetailedFormatter) Format(diffs []Difference, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
	}

	if len(diffs) == 0 {
		return "no differences found\n"
	}

	var sb strings.Builder

	if !opts.OmitHeader {
		f.formatHeader(&sb, diffs, opts)
	}

	isMultiDoc := f.detectMultiDoc(diffs)
	groups := f.groupByPath(diffs)
	for _, group := range groups {
		f.formatPathHeading(&sb, group.Path, isMultiDoc, opts)
		f.formatGroupDiffs(&sb, group, opts)
	}

	return sb.String()
}

// formatHeader renders a summary header line.
func (f *DetailedFormatter) formatHeader(sb *strings.Builder, diffs []Difference, opts *FormatOptions) {
	sb.WriteString(colorStart(opts, f.colorModified(opts)))
	fmt.Fprintf(sb, "Found %s %s",
		formatCount(len(diffs)),
		pluralize(len(diffs), "difference", "differences"))
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n\n")
}

// groupByPath groups diffs by their Path field, preserving order of first occurrence.
func (f *DetailedFormatter) groupByPath(diffs []Difference) []pathGroup {
	var groups []pathGroup
	index := make(map[string]int) // path -> index in groups

	for _, diff := range diffs {
		if idx, exists := index[diff.Path]; exists {
			groups[idx].Diffs = append(groups[idx].Diffs, diff)
		} else {
			index[diff.Path] = len(groups)
			groups = append(groups, pathGroup{
				Path:  diff.Path,
				Diffs: []Difference{diff},
			})
		}
	}

	return groups
}

// formatPathHeading renders the path line for a group of diffs.
func (f *DetailedFormatter) formatPathHeading(sb *strings.Builder, path string, isMultiDoc bool, opts *FormatOptions) {
	heading := path
	if path == "" {
		if opts.UseGoPatchStyle {
			heading = "/"
		} else {
			heading = "(root level)"
		}
	} else if idx, ok := parseBareDocIndex(path); ok {
		if isMultiDoc {
			heading = fmt.Sprintf("(document %d)", idx+1)
		} else {
			heading = k8sDocumentPath
		}
	} else if idx, rest, ok := parseDocIndexPrefix(path); ok {
		if opts.UseGoPatchStyle {
			heading = fmt.Sprintf("%d:%s", idx, convertToGoPatchPath(rest))
		} else {
			heading = fmt.Sprintf("%d:%s", idx, rest)
		}
	} else if opts.UseGoPatchStyle {
		heading = convertToGoPatchPath(path)
	}

	sb.WriteString(colorStart(opts, styleBold))
	sb.WriteString(heading)
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")
}

// formatGroupDiffs renders all diffs within a path group.
// Groups consecutive additions and removals for batched descriptors.
func (f *DetailedFormatter) formatGroupDiffs(sb *strings.Builder, group pathGroup, opts *FormatOptions) {
	var added, removed []Difference
	var others []Difference

	for _, diff := range group.Diffs {
		switch diff.Type {
		case DiffAdded:
			added = append(added, diff)
		case DiffRemoved:
			removed = append(removed, diff)
		default:
			others = append(others, diff)
		}
	}

	if len(removed) > 0 {
		f.formatEntryBatch(sb, removed, "removed", opts)
	}

	if len(added) > 0 {
		f.formatEntryBatch(sb, added, "added", opts)
	}

	for _, diff := range others {
		f.formatChangeDescriptor(sb, diff, opts)
	}
}

// formatEntryBatch renders a group of additions or removals with a count descriptor.
func (f *DetailedFormatter) formatEntryBatch(sb *strings.Builder, diffs []Difference, action string, opts *FormatOptions) {
	n := len(diffs)
	isListEntry := isListEntryDiff(diffs[0])
	entryType := "map"
	if isListEntry {
		entryType = "list"
	}

	countStr := formatCount(n)
	noun := pluralize(n, entryType+" entry", entryType+" entries")
	symbol := "+"
	colorFn := f.colorAdded
	if action == "removed" {
		symbol = "-"
		colorFn = f.colorRemoved
	}

	sb.WriteString("  ")
	sb.WriteString(colorStart(opts, colorFn(opts)))
	fmt.Fprintf(sb, "%s %s %s %s:", symbol, countStr, noun, action)
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")

	for _, diff := range diffs {
		var val any
		if diff.To != nil {
			val = diff.To
		} else {
			val = diff.From
		}
		if !opts.NoCertInspection {
			if s, ok := val.(string); ok && IsPEMCertificate(s) {
				val = FormatCertificate(s)
			}
		}
		f.renderEntryValue(sb, val, symbol, 4, diff.Path, isListEntry, opts)
	}
	sb.WriteString("\n")
}

// formatChangeDescriptor renders the descriptor line for a single diff.
func (f *DetailedFormatter) formatChangeDescriptor(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	switch diff.Type {
	case DiffModified:
		f.formatModified(sb, diff, opts)
	case DiffOrderChanged:
		f.writeDescriptorLine(sb, "  ⇆ order changed", f.colorModified, opts)
		if diff.From != nil {
			f.writeColoredLine(sb, fmt.Sprintf("    - %s", formatCommaSeparated(diff.From)), f.colorRemoved(opts), opts)
		}
		if diff.To != nil {
			f.writeColoredLine(sb, fmt.Sprintf("    + %s", formatCommaSeparated(diff.To)), f.colorAdded(opts), opts)
		}
		sb.WriteString("\n")
	}
}

// formatModified renders a modification descriptor with type change, multiline, and whitespace detection.
func (f *DetailedFormatter) formatModified(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	fromType := yamlTypeName(diff.From)
	toType := yamlTypeName(diff.To)

	// Type change detection
	if fromType != toType {
		if opts.Color {
			f.writeDescriptorLine(sb, fmt.Sprintf("  ± type change from %s%s%s to %s%s%s",
				styleItalic, fromType, styleItalicOff,
				styleItalic, toType, styleItalicOff), f.colorModified, opts)
		} else {
			f.writeDescriptorLine(sb, fmt.Sprintf("  ± type change from %s to %s", fromType, toType), f.colorModified, opts)
		}
		f.writeTypeChangeValue(sb, diff.From, "-", f.colorRemoved(opts), opts)
		f.writeTypeChangeValue(sb, diff.To, "+", f.colorAdded(opts), opts)
		sb.WriteString("\n")
		return
	}

	// Both strings — check for multiline and whitespace-only
	fromStr, fromOk := diff.From.(string)
	toStr, toOk := diff.To.(string)

	if fromOk && toOk {
		// Certificate inspection: transform PEM certs to single-line summaries
		if !opts.NoCertInspection && IsPEMCertificate(fromStr) && IsPEMCertificate(toStr) {
			fromStr = FormatCertificate(fromStr)
			toStr = FormatCertificate(toStr)
		}

		// Whitespace-only change detection
		if isWhitespaceOnlyChange(fromStr, toStr) {
			f.writeDescriptorLine(sb, "  ± whitespace only change", f.colorModified, opts)
			f.writeColoredLine(sb, fmt.Sprintf("    - %s", visualizeWhitespace(fromStr)), f.colorRemoved(opts), opts)
			f.writeColoredLine(sb, fmt.Sprintf("    + %s", visualizeWhitespace(toStr)), f.colorAdded(opts), opts)
			sb.WriteString("\n")
			return
		}

		// Multiline detection
		if strings.Contains(fromStr, "\n") || strings.Contains(toStr, "\n") {
			f.formatMultilineDiff(sb, fromStr, toStr, opts)
			return
		}

		// Scalar string value change (may be cert-transformed)
		f.writeDescriptorLine(sb, "  ± value change", f.colorModified, opts)
		f.writeColoredLine(sb, fmt.Sprintf("    - %s", fromStr), f.colorRemoved(opts), opts)
		f.writeColoredLine(sb, fmt.Sprintf("    + %s", toStr), f.colorAdded(opts), opts)
		sb.WriteString("\n")
		return
	}

	// Default: non-string scalar value change
	f.writeDescriptorLine(sb, "  ± value change", f.colorModified, opts)
	f.writeColoredLine(sb, fmt.Sprintf("    - %v", formatDetailedValue(diff.From)), f.colorRemoved(opts), opts)
	f.writeColoredLine(sb, fmt.Sprintf("    + %v", formatDetailedValue(diff.To)), f.colorAdded(opts), opts)
	sb.WriteString("\n")
}

// detectMultiDoc checks if diffs span multiple documents by examining DocumentIndex values.
func (f *DetailedFormatter) detectMultiDoc(diffs []Difference) bool {
	seen := -1
	for _, d := range diffs {
		if seen == -1 {
			seen = d.DocumentIndex
		} else if d.DocumentIndex != seen {
			return true
		}
	}
	return false
}

// writeColoredLine writes a line with color code and newline.
func (f *DetailedFormatter) writeColoredLine(sb *strings.Builder, text string, code string, opts *FormatOptions) {
	sb.WriteString(colorStart(opts, code))
	sb.WriteString(text)
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")
}

// writeDescriptorLine writes a descriptor line using a color function.
func (f *DetailedFormatter) writeDescriptorLine(sb *strings.Builder, text string, colorFn func(*FormatOptions) string, opts *FormatOptions) {
	sb.WriteString(colorStart(opts, colorFn(opts)))
	sb.WriteString(text)
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")
}

// Color helper methods for DetailedFormatter

func (f *DetailedFormatter) colorAdded(opts *FormatOptions) string {
	return DetailedColorCode(DiffAdded, opts.TrueColor)
}

func (f *DetailedFormatter) colorRemoved(opts *FormatOptions) string {
	return DetailedColorCode(DiffRemoved, opts.TrueColor)
}

func (f *DetailedFormatter) colorModified(opts *FormatOptions) string {
	return DetailedColorCode(DiffModified, opts.TrueColor)
}

func (f *DetailedFormatter) colorContext(opts *FormatOptions) string {
	return ContextColorCode(opts.TrueColor)
}
