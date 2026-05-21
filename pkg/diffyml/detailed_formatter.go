// detailed_formatter.go - Detailed human-readable output formatter.
//
// Renders differences in a path-grouped style with descriptive labels,
// structured value display, and multiline text diffs.
//
// Split across three files:
//   - detailed_formatter.go (this file) — core orchestration and pure utilities
//   - detailed_formatter_render.go — YAML value rendering
//   - detailed_formatter_linediff.go — LCS line-diff algorithm
package diffyml

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DetailedFormatter implements the Formatter interface for detailed output.
type DetailedFormatter struct{}

// pathGroup holds a path and its associated differences for grouping.
type pathGroup struct {
	Path  DiffPath
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
		f.formatPathHeading(&sb, group.Path, group.Diffs[0].DocumentName, isMultiDoc, opts)
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
		key := diff.Path.String()
		if idx, exists := index[key]; exists {
			groups[idx].Diffs = append(groups[idx].Diffs, diff)
		} else {
			index[key] = len(groups)
			groups = append(groups, pathGroup{
				Path:  diff.Path,
				Diffs: []Difference{diff},
			})
		}
	}

	return groups
}

// documentLabel returns the display label for a document.
// Uses docName when available, otherwise falls back to "document N".
func documentLabel(idx int, docName string) string {
	if docName != "" {
		return docName
	}
	return fmt.Sprintf("document %d", idx)
}

// pathString returns the display string for a DiffPath, respecting go-patch style.
func pathString(path DiffPath, goPatch bool) string {
	if goPatch {
		return path.GoPatchString()
	}
	return path.String()
}

// formatPathHeading renders the path line for a group of diffs.
func (f *DetailedFormatter) formatPathHeading(sb *strings.Builder, path DiffPath, docName string, isMultiDoc bool, opts *FormatOptions) {
	if path.IsEmpty() {
		if opts.UseGoPatchStyle {
			f.writeBold(sb, "/", opts)
		} else {
			f.writeBold(sb, "(root level)", opts)
		}
	} else if path.IsBareDocIndex() {
		idx, _ := path.DocIndex()
		if isMultiDoc {
			f.writeBold(sb, "(root level)", opts)
			f.writeDocLabel(sb, documentLabel(idx, docName), opts)
		} else {
			f.writeBold(sb, k8sDocumentPath.String(), opts)
			if docName != "" {
				f.writeDocLabel(sb, docName, opts)
			}
		}
	} else if idx, rest, ok := path.DocIndexPrefix(); ok {
		f.writeBold(sb, pathString(rest, opts.UseGoPatchStyle), opts)
		f.writeDocLabel(sb, documentLabel(idx, docName), opts)
	} else {
		f.writeBold(sb, pathString(path, opts.UseGoPatchStyle), opts)
		if docName != "" {
			f.writeDocLabel(sb, docName, opts)
		}
	}
	sb.WriteString("\n")
}

// writeBold writes text in bold style.
func (f *DetailedFormatter) writeBold(sb *strings.Builder, text string, opts *FormatOptions) {
	sb.WriteString(colorStart(opts, styleBold))
	sb.WriteString(text)
	sb.WriteString(colorEnd(opts))
}

// writeDocLabel writes a document label suffix in light steel blue.
func (f *DetailedFormatter) writeDocLabel(sb *strings.Builder, label string, opts *FormatOptions) {
	sb.WriteString("  ")
	sb.WriteString(colorStart(opts, resolvedPalette(opts).ColorCode(ColorRoleDocName, opts.TrueColor)))
	fmt.Fprintf(sb, "(%s)", label)
	sb.WriteString(colorEnd(opts))
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

	// Detect document-level diffs (path is bare "[N]")
	// All diffs in a batch share the same path structure; checking the first is sufficient.
	isDocLevel := diffs[0].Path.IsBareDocIndex()

	isListEntry := isListEntryDiff(diffs[0])
	entryType := "map"
	if isListEntry {
		entryType = "list"
	}

	countStr := formatCount(n)
	var noun string
	if isDocLevel {
		noun = pluralize(n, "document", "documents")
	} else {
		noun = pluralize(n, entryType+" entry", entryType+" entries")
	}
	symbol := "+"
	if action == "removed" {
		symbol = "-"
	}

	sb.WriteString("  ")
	sb.WriteString(colorStart(opts, f.colorModified(opts)))
	fmt.Fprintf(sb, "%s %s %s %s:", symbol, countStr, noun, action)
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")

	for _, diff := range diffs {
		var val any
		anchor := diff.LineTo
		if diff.To != nil {
			val = diff.To
		} else {
			val = diff.From
			anchor = diff.LineFrom
		}
		if !opts.NoCertInspection {
			if s, ok := val.(string); ok && IsPEMCertificate(s) {
				val = FormatCertificate(s)
			}
		}
		// Render into a sub-builder so the entry's first line can be prefixed with
		// its source line number; renderers themselves stay line-number-agnostic.
		var eb strings.Builder
		if isDocLevel {
			f.renderDocumentValue(&eb, val, symbol, 4, opts)
		} else {
			f.renderEntryValue(&eb, val, symbol, 4, diff.Path, isListEntry, opts)
		}
		f.writeEntryWithLineNumber(sb, eb.String(), anchor, opts)
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
		f.writeTypeChangeValue(sb, diff.From, "-", diff.LineFrom, f.colorRemoved(opts), opts)
		f.writeTypeChangeValue(sb, diff.To, "+", diff.LineTo, f.colorAdded(opts), opts)
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

		// Multiline detection (before whitespace-only check, so multiline
		// strings get a readable line-by-line diff instead of a single
		// unreadable line with ↵ markers)
		if strings.Contains(fromStr, "\n") || strings.Contains(toStr, "\n") {
			f.formatMultilineDiff(sb, fromStr, toStr, diff.LineFrom, diff.LineTo, opts)
			return
		}

		// Whitespace-only change detection (single-line strings only)
		if isWhitespaceOnlyChange(fromStr, toStr) {
			f.writeDescriptorLine(sb, "  ± whitespace only change", f.colorModified, opts)
			f.writeColoredLine(sb, fmt.Sprintf("    - %s%s", linePrefix(opts, diff.LineFrom), visualizeWhitespace(fromStr)), f.colorRemoved(opts), opts)
			f.writeColoredLine(sb, fmt.Sprintf("    + %s%s", linePrefix(opts, diff.LineTo), visualizeWhitespace(toStr)), f.colorAdded(opts), opts)
			sb.WriteString("\n")
			return
		}

		// Scalar string value change (may be cert-transformed)
		f.writeValueChange(sb, fromStr, toStr, diff.LineFrom, diff.LineTo, opts)
		return
	}

	// Default: non-string scalar value change
	f.writeValueChange(sb, formatDetailedValue(diff.From), formatDetailedValue(diff.To), diff.LineFrom, diff.LineTo, opts)
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

// writeValueChange writes a "± value change" block with inline diff highlighting
// when color is enabled and the values are similar enough, otherwise falls back
// to plain colored lines.
func (f *DetailedFormatter) writeValueChange(sb *strings.Builder, from, to string, fromLine, toLine int, opts *FormatOptions) {
	f.writeDescriptorLine(sb, "  ± value change", f.colorModified, opts)
	if opts.Color {
		if fromSegs, toSegs := computeInlineDiff(from, to); fromSegs != nil {
			f.writeInlineDiffLine(sb, "    - "+linePrefix(opts, fromLine), fromSegs, ColorRoleRemoved, opts)
			f.writeInlineDiffLine(sb, "    + "+linePrefix(opts, toLine), toSegs, ColorRoleAdded, opts)
			sb.WriteString("\n")
			return
		}
	}
	f.writeColoredLine(sb, fmt.Sprintf("    - %s%s", linePrefix(opts, fromLine), from), f.colorRemoved(opts), opts)
	f.writeColoredLine(sb, fmt.Sprintf("    + %s%s", linePrefix(opts, toLine), to), f.colorAdded(opts), opts)
	sb.WriteString("\n")
}

// linePrefix returns "N: " when line numbers are enabled and line is known (>0),
// otherwise "". Used to prefix detailed-output value lines with their source line.
func linePrefix(opts *FormatOptions, line int) string {
	if opts == nil || !opts.LineNumbers || line <= 0 {
		return ""
	}
	return strconv.Itoa(line) + ": "
}

// advanceLine returns line+1 for a known line (>0), leaving 0 (unknown) unchanged.
func advanceLine(line int) int {
	if line > 0 {
		return line + 1
	}
	return 0
}

// insertLineNumber injects "N: " into a rendered entry line, after any leading ANSI
// color escape, indentation spaces, and an optional "- " list marker — so the number
// sits where a value begins (e.g. "    - 8: key: value").
func insertLineNumber(line string, num int) string {
	i := 0
	if strings.HasPrefix(line[i:], "\x1b[") {
		if m := strings.IndexByte(line[i:], 'm'); m >= 0 {
			i += m + 1
		}
	}
	for i < len(line) && line[i] == ' ' {
		i++
	}
	if strings.HasPrefix(line[i:], "- ") {
		i += 2
	}
	return line[:i] + strconv.Itoa(num) + ": " + line[i:]
}

// writeEntryWithLineNumber writes a rendered entry block, prefixing its first line
// with the source line number when enabled and known.
func (f *DetailedFormatter) writeEntryWithLineNumber(sb *strings.Builder, block string, line int, opts *FormatOptions) {
	if !opts.LineNumbers || line <= 0 {
		sb.WriteString(block)
		return
	}
	if nl := strings.IndexByte(block, '\n'); nl >= 0 {
		sb.WriteString(insertLineNumber(block[:nl], line))
		sb.WriteString(block[nl:])
		return
	}
	sb.WriteString(insertLineNumber(block, line))
}

// writeColoredLine writes a line with color code and newline.
func (f *DetailedFormatter) writeColoredLine(sb *strings.Builder, text string, code string, opts *FormatOptions) {
	sb.WriteString(colorStart(opts, code))
	sb.WriteString(text)
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")
}

// writeKeyValueLine writes a line with the key portion in one color and the value in another.
func (f *DetailedFormatter) writeKeyValueLine(sb *strings.Builder, keyText string, valueText string, keyCode string, valueCode string, opts *FormatOptions) {
	sb.WriteString(colorStart(opts, keyCode))
	sb.WriteString(keyText)
	sb.WriteString(colorEnd(opts))
	sb.WriteString(colorStart(opts, valueCode))
	sb.WriteString(" ")
	sb.WriteString(valueText)
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

// writeInlineDiffLine writes a prefixed line with inline diff highlighting.
// Changed segments are rendered in bold with the base color; unchanged segments
// use a dimmed color for visual contrast.
func (f *DetailedFormatter) writeInlineDiffLine(sb *strings.Builder, prefix string, segments []inlineSegment, role ColorRole, opts *FormatOptions) {
	p := resolvedPalette(opts)
	baseColor := p.ColorCode(role, opts.TrueColor)
	dim := dimColorCode(role, opts)
	sb.WriteString(colorStart(opts, dim))
	sb.WriteString(prefix)
	renderInlineSegments(sb, segments, baseColor, dim, opts)
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")
}

// Color helper methods for DetailedFormatter

func (f *DetailedFormatter) colorAdded(opts *FormatOptions) string {
	return resolvedPalette(opts).ColorCode(ColorRoleAdded, opts.TrueColor)
}

func (f *DetailedFormatter) colorRemoved(opts *FormatOptions) string {
	return resolvedPalette(opts).ColorCode(ColorRoleRemoved, opts.TrueColor)
}

func (f *DetailedFormatter) colorModified(opts *FormatOptions) string {
	return resolvedPalette(opts).ColorCode(ColorRoleModified, opts.TrueColor)
}

func (f *DetailedFormatter) colorContext(opts *FormatOptions) string {
	return resolvedPalette(opts).ColorCode(ColorRoleContext, opts.TrueColor)
}

// --- Pure utility helpers used across the detailed formatter ---

// isWhitespaceOnlyChange checks if the difference between two strings is only whitespace.
func isWhitespaceOnlyChange(from, to string) bool {
	if from == to {
		return false // no change at all
	}
	// Strip all whitespace and compare
	fromStripped := stripWhitespace(from)
	toStripped := stripWhitespace(to)
	return fromStripped == toStripped
}

// stripWhitespace removes all whitespace characters from a string.
func stripWhitespace(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// visualizeWhitespace replaces invisible characters with visible symbols.
func visualizeWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\n", "↵")
	s = strings.ReplaceAll(s, " ", "·")
	return s
}

// yamlTypeName returns a human-readable YAML type name for a value.
func yamlTypeName(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case int, int64:
		return "int"
	case float64:
		return "float"
	case bool:
		return "bool"
	case *OrderedMap, map[string]any:
		return "map"
	case []any:
		return "list"
	case time.Time:
		return "timestamp"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// formatCommaSeparated formats a slice value as comma-separated items.
// For non-slice values, falls back to formatDetailedValue for scalar display.
func formatCommaSeparated(val any) string {
	if items, ok := val.([]any); ok {
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = formatDetailedValue(item)
		}
		return strings.Join(parts, ", ")
	}
	return formatDetailedValue(val)
}

// formatDetailedValue formats a value for display.
// Nil values render as "<nil>" via fmt.Sprintf's default verb.
func formatDetailedValue(val any) string {
	if t, ok := val.(time.Time); ok {
		return formatTimestamp(t)
	}
	return fmt.Sprintf("%v", val)
}

// formatTimestamp formats a time.Time as a YAML-friendly date or datetime string.
func formatTimestamp(t time.Time) string {
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}

// formatCount returns a human-readable count string.
// Numbers 1-12 are spelled out as English words.
func formatCount(n int) string {
	words := []string{
		"zero", "one", "two", "three", "four", "five",
		"six", "seven", "eight", "nine", "ten", "eleven", "twelve",
	}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%d", n)
}

// pluralize returns singular or plural form based on count.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
