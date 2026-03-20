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
		}
	} else if idx, rest, ok := path.DocIndexPrefix(); ok {
		f.writeBold(sb, pathString(rest, opts.UseGoPatchStyle), opts)
		f.writeDocLabel(sb, documentLabel(idx, docName), opts)
	} else {
		f.writeBold(sb, pathString(path, opts.UseGoPatchStyle), opts)
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
	sb.WriteString(colorStart(opts, DocNameColorCode(opts.TrueColor)))
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
		if isDocLevel {
			f.renderDocumentValue(sb, val, symbol, 4, opts)
		} else {
			f.renderEntryValue(sb, val, symbol, 4, diff.Path, isListEntry, opts)
		}
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

		// Multiline detection (before whitespace-only check, so multiline
		// strings get a readable line-by-line diff instead of a single
		// unreadable line with ↵ markers)
		if strings.Contains(fromStr, "\n") || strings.Contains(toStr, "\n") {
			f.formatMultilineDiff(sb, fromStr, toStr, opts)
			return
		}

		// Whitespace-only change detection (single-line strings only)
		if isWhitespaceOnlyChange(fromStr, toStr) {
			f.writeDescriptorLine(sb, "  ± whitespace only change", f.colorModified, opts)
			f.writeColoredLine(sb, fmt.Sprintf("    - %s", visualizeWhitespace(fromStr)), f.colorRemoved(opts), opts)
			f.writeColoredLine(sb, fmt.Sprintf("    + %s", visualizeWhitespace(toStr)), f.colorAdded(opts), opts)
			sb.WriteString("\n")
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
