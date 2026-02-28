// detailed_formatter.go - Detailed human-readable output formatter.
//
// Renders differences in a path-grouped style with descriptive labels,
// structured value display, and multiline text diffs.
package diffyml

import (
	"fmt"
	"strconv"
	"strings"
)

// DetailedFormatter implements the Formatter interface for detailed output.
type DetailedFormatter struct{}

// pathGroup holds a path and its associated differences for grouping.
type pathGroup struct {
	Path  string
	Diffs []Difference
}

// editOpType represents a type of edit operation in a line diff.
type editOpType int

const (
	editKeep editOpType = iota
	editInsert
	editDelete
)

// editOp represents a single edit operation in a line diff.
type editOp struct {
	Type editOpType
	Line string
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
	if opts.Color {
		sb.WriteString(f.colorModified(opts))
	}
	fmt.Fprintf(sb, "Found %s %s",
		formatCount(len(diffs)),
		pluralize(len(diffs), "difference", "differences"))
	if opts.Color {
		sb.WriteString(GetColorReset())
	}
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
			heading = "(document)"
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

	if opts.Color {
		sb.WriteString(styleBold)
	}
	sb.WriteString(heading)
	if opts.Color {
		sb.WriteString(colorReset)
	}
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

	if len(added) > 0 {
		f.formatEntryBatch(sb, added, "added", opts)
	}

	if len(removed) > 0 {
		f.formatEntryBatch(sb, removed, "removed", opts)
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

	if opts.Color {
		sb.WriteString("  ")
		sb.WriteString(colorFn(opts))
		fmt.Fprintf(sb, "%s %s %s %s:", symbol, countStr, noun, action)
		sb.WriteString(f.colorReset())
		sb.WriteString("\n")
	} else {
		fmt.Fprintf(sb, "  %s %s %s %s:\n", symbol, countStr, noun, action)
	}

	for _, diff := range diffs {
		var val interface{}
		if diff.To != nil {
			val = diff.To
		} else {
			val = diff.From
		}
		f.renderEntryValue(sb, val, symbol, 4, diff.Path, isListEntry, opts)
	}
	sb.WriteString("\n")
}

// renderEntryValue renders a value for an entry batch line.
// For list entries, renders values with "- " prefix. For map entries, renders as "key: value".
// The entire block is colored (green for adds, red for removes).
func (f *DetailedFormatter) renderEntryValue(sb *strings.Builder, val interface{}, symbol string, indent int, path string, isList bool, opts *FormatOptions) {
	colorCode := ""
	if opts.Color {
		if symbol == "+" {
			colorCode = f.colorAdded(opts)
		} else {
			colorCode = f.colorRemoved(opts)
		}
	}

	// Map entries: extract key from path and render as key: value
	if !isList {
		key := path
		if idx := strings.LastIndex(path, "."); idx >= 0 {
			key = path[idx+1:]
		}
		f.renderKeyValueYAML(sb, key, val, indent, colorCode, opts)
		return
	}

	// List entries: use "- " prefix for scalars, YAML block with dash prefix for structured
	pad := strings.Repeat(" ", indent)
	switch v := val.(type) {
	case *OrderedMap:
		for i, key := range v.Keys {
			if i == 0 {
				// First key: "    - key: value" (dash prefix at indent level)
				f.renderFirstKeyValueYAML(sb, key, v.Values[key], indent, colorCode, opts)
			} else {
				// Continuation keys: indent+2 to align under first key's content
				f.renderKeyValueYAML(sb, key, v.Values[key], indent+2, colorCode, opts)
			}
		}
	case map[string]interface{}:
		first := true
		for key, value := range v {
			if first {
				f.renderFirstKeyValueYAML(sb, key, value, indent, colorCode, opts)
				first = false
			} else {
				f.renderKeyValueYAML(sb, key, value, indent+2, colorCode, opts)
			}
		}
	case []interface{}:
		for _, item := range v {
			f.writeColoredLine(sb, fmt.Sprintf("%s- %v", pad, formatDetailedValue(item)), colorCode, opts)
		}
	default:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %v", pad, formatDetailedValue(val)), colorCode, opts)
	}
}

// renderKeyValueYAML renders a key: value pair in plain YAML style with color.
// Uses standard YAML indentation (2 spaces per level), no pipe guides.
func (f *DetailedFormatter) renderKeyValueYAML(sb *strings.Builder, key string, val interface{}, indent int, colorCode string, opts *FormatOptions) {
	pad := strings.Repeat(" ", indent)
	switch v := val.(type) {
	case *OrderedMap:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		for _, k := range v.Keys {
			f.renderKeyValueYAML(sb, k, v.Values[k], indent+2, colorCode, opts)
		}
	case map[string]interface{}:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		for k, value := range v {
			f.renderKeyValueYAML(sb, k, value, indent+2, colorCode, opts)
		}
	case []interface{}:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		for _, item := range v {
			f.writeColoredLine(sb, fmt.Sprintf("%s  - %v", pad, formatDetailedValue(item)), colorCode, opts)
		}
	default:
		if str, ok := val.(string); ok && strings.Contains(str, "\n") {
			f.renderMultilineValue(sb, fmt.Sprintf("%s%s:", pad, key), str, colorCode, indent, opts)
		} else {
			f.writeColoredLine(sb, fmt.Sprintf("%s%s: %v", pad, key, formatDetailedValue(val)), colorCode, opts)
		}
	}
}

// renderFirstKeyValueYAML renders the first key of a list entry with "- " prefix.
// The key is rendered as "    - key: value" where indent is the base indentation.
// For nested values, continuation uses indent+2 to align under the key.
func (f *DetailedFormatter) renderFirstKeyValueYAML(sb *strings.Builder, key string, val interface{}, indent int, colorCode string, opts *FormatOptions) {
	pad := strings.Repeat(" ", indent)
	switch v := val.(type) {
	case *OrderedMap:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		for _, k := range v.Keys {
			f.renderKeyValueYAML(sb, k, v.Values[k], indent+4, colorCode, opts)
		}
	case map[string]interface{}:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		for k, value := range v {
			f.renderKeyValueYAML(sb, k, value, indent+4, colorCode, opts)
		}
	case []interface{}:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		for _, item := range v {
			f.writeColoredLine(sb, fmt.Sprintf("%s    - %v", pad, formatDetailedValue(item)), colorCode, opts)
		}
	default:
		if str, ok := val.(string); ok && strings.Contains(str, "\n") {
			f.renderMultilineValue(sb, fmt.Sprintf("%s- %s:", pad, key), str, colorCode, indent+2, opts)
		} else {
			f.writeColoredLine(sb, fmt.Sprintf("%s- %s: %v", pad, key, formatDetailedValue(val)), colorCode, opts)
		}
	}
}

// renderMultilineValue renders a multiline string in YAML block literal style (|).
func (f *DetailedFormatter) renderMultilineValue(sb *strings.Builder, prefix, value, colorCode string, indent int, opts *FormatOptions) {
	f.writeColoredLine(sb, prefix+" |", colorCode, opts)
	pad := strings.Repeat(" ", indent+2)
	for _, line := range strings.Split(strings.TrimRight(value, "\n"), "\n") {
		f.writeColoredLine(sb, pad+line, colorCode, opts)
	}
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
		f.writeColoredLine(sb, fmt.Sprintf("    - %v", formatDetailedValue(diff.From)), f.colorRemoved(opts), opts)
		f.writeColoredLine(sb, fmt.Sprintf("    + %v", formatDetailedValue(diff.To)), f.colorAdded(opts), opts)
		sb.WriteString("\n")
		return
	}

	// Both strings — check for multiline and whitespace-only
	fromStr, fromOk := diff.From.(string)
	toStr, toOk := diff.To.(string)

	if fromOk && toOk {
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
	}

	// Default: scalar value change
	f.writeDescriptorLine(sb, "  ± value change", f.colorModified, opts)
	f.writeColoredLine(sb, fmt.Sprintf("    - %v", formatDetailedValue(diff.From)), f.colorRemoved(opts), opts)
	f.writeColoredLine(sb, fmt.Sprintf("    + %v", formatDetailedValue(diff.To)), f.colorAdded(opts), opts)
	sb.WriteString("\n")
}

// formatMultilineDiff renders an inline line-by-line diff for multiline strings.
func (f *DetailedFormatter) formatMultilineDiff(sb *strings.Builder, from, to string, opts *FormatOptions) {
	fromLines := strings.Split(from, "\n")
	toLines := strings.Split(to, "\n")
	ops := computeLineDiff(fromLines, toLines)

	// Count additions and deletions
	additions := 0
	deletions := 0
	for _, op := range ops {
		switch op.Type {
		case editInsert:
			additions++
		case editDelete:
			deletions++
		}
	}

	descriptor := fmt.Sprintf("  ± value change in multiline text (%s %s, %s %s)",
		formatCount(additions), pluralize(additions, "insert", "inserts"),
		formatCount(deletions), pluralize(deletions, "deletion", "deletions"))
	f.writeDescriptorLine(sb, descriptor, f.colorModified, opts)

	// Apply context collapsing
	contextLines := opts.ContextLines
	if contextLines < 0 {
		contextLines = 4
	}

	// Mark which ops are near a change
	nearChange := make([]bool, len(ops))
	for i, op := range ops {
		if op.Type != editKeep {
			// Mark surrounding context
			for j := max(0, i-contextLines); j <= min(len(ops)-1, i+contextLines); j++ {
				nearChange[j] = true
			}
		}
	}

	// Render with collapsing
	i := 0
	for i < len(ops) {
		op := ops[i]
		if op.Type != editKeep || nearChange[i] {
			switch op.Type {
			case editKeep:
				f.writeColoredLine(sb, fmt.Sprintf("      %s", op.Line), f.colorContext(opts), opts)
			case editInsert:
				f.writeColoredLine(sb, fmt.Sprintf("    + %s", op.Line), f.colorAdded(opts), opts)
			case editDelete:
				f.writeColoredLine(sb, fmt.Sprintf("    - %s", op.Line), f.colorRemoved(opts), opts)
			}
			i++
		} else {
			// Count consecutive non-near-change keep ops
			collapsed := 0
			for i < len(ops) && ops[i].Type == editKeep && !nearChange[i] {
				collapsed++
				i++
			}
			f.writeColoredLine(sb, fmt.Sprintf("    [%d %s unchanged]", collapsed, pluralize(collapsed, "line", "lines")), f.colorContext(opts), opts)
		}
	}
	sb.WriteString("\n")
}

// computeLineDiff computes line-level diff using LCS algorithm.
func computeLineDiff(fromLines, toLines []string) []editOp {
	m := len(fromLines)
	n := len(toLines)

	// Build LCS table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if fromLines[i-1] == toLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to produce edit operations
	var ops []editOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && fromLines[i-1] == toLines[j-1] {
			ops = append(ops, editOp{Type: editKeep, Line: fromLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, editOp{Type: editInsert, Line: toLines[j-1]})
			j--
		} else {
			ops = append(ops, editOp{Type: editDelete, Line: fromLines[i-1]})
			i--
		}
	}

	// Reverse to get correct order
	for left, right := 0, len(ops)-1; left < right; left, right = left+1, right-1 {
		ops[left], ops[right] = ops[right], ops[left]
	}

	return ops
}

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
func yamlTypeName(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case int, int64:
		return "int"
	case float64:
		return "float"
	case bool:
		return "bool"
	case *OrderedMap, map[string]interface{}:
		return "map"
	case []interface{}:
		return "list"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// formatCommaSeparated formats a slice value as comma-separated items.
// For non-slice values, falls back to formatDetailedValue for scalar display.
func formatCommaSeparated(val interface{}) string {
	if items, ok := val.([]interface{}); ok {
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = formatDetailedValue(item)
		}
		return strings.Join(parts, ", ")
	}
	return formatDetailedValue(val)
}

// formatDetailedValue formats a value for display, handling nil.
func formatDetailedValue(val interface{}) string {
	if val == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", val)
}

// formatCount returns a human-readable count string.
// Numbers 1-12 are spelled out as English words.
func formatCount(n int) string {
	words := []string{"zero", "one", "two", "three", "four", "five",
		"six", "seven", "eight", "nine", "ten", "eleven", "twelve"}
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

// writeColoredLine writes a line with color code and newline.
func (f *DetailedFormatter) writeColoredLine(sb *strings.Builder, text string, colorCode string, opts *FormatOptions) {
	if opts.Color {
		sb.WriteString(colorCode)
		sb.WriteString(text)
		sb.WriteString(f.colorReset())
	} else {
		sb.WriteString(text)
	}
	sb.WriteString("\n")
}

// writeDescriptorLine writes a descriptor line using a color function.
func (f *DetailedFormatter) writeDescriptorLine(sb *strings.Builder, text string, colorFn func(*FormatOptions) string, opts *FormatOptions) {
	if opts.Color {
		sb.WriteString(colorFn(opts))
		sb.WriteString(text)
		sb.WriteString(f.colorReset())
	} else {
		sb.WriteString(text)
	}
	sb.WriteString("\n")
}

// parseBareDocIndex extracts the index from a bare document index path like "[0]", "[1]".
// Returns the index and true if the path is a bare document index, false otherwise.
// Does NOT match paths like "items[0]" or "[0].spec".
func parseBareDocIndex(path string) (int, bool) {
	if !strings.HasPrefix(path, "[") || !strings.HasSuffix(path, "]") {
		return 0, false
	}
	inner := path[1 : len(path)-1]
	idx, err := strconv.Atoi(inner)
	if err != nil {
		return 0, false
	}
	return idx, true
}

// parseDocIndexPrefix extracts a leading [N] document index from a path like "[0].spec.field".
// Returns (index, remainingPath, true) if found, (0, originalPath, false) otherwise.
// Only matches paths starting with "[N]." — bare "[N]" is handled by parseBareDocIndex.
func parseDocIndexPrefix(path string) (int, string, bool) {
	if !strings.HasPrefix(path, "[") {
		return 0, path, false
	}
	closeBracket := strings.Index(path, "]")
	if closeBracket < 0 {
		return 0, path, false
	}
	// Must have a dot after the closing bracket
	if closeBracket+1 >= len(path) || path[closeBracket+1] != '.' {
		return 0, path, false
	}
	inner := path[1:closeBracket]
	idx, err := strconv.Atoi(inner)
	if err != nil {
		return 0, path, false
	}
	rest := path[closeBracket+2:] // skip "]."
	return idx, rest, true
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

// Color helper methods for DetailedFormatter

func (f *DetailedFormatter) colorAdded(opts *FormatOptions) string {
	return GetDetailedColorCode(DiffAdded, opts.TrueColor)
}

func (f *DetailedFormatter) colorRemoved(opts *FormatOptions) string {
	return GetDetailedColorCode(DiffRemoved, opts.TrueColor)
}

func (f *DetailedFormatter) colorModified(opts *FormatOptions) string {
	return GetDetailedColorCode(DiffModified, opts.TrueColor)
}

func (f *DetailedFormatter) colorContext(opts *FormatOptions) string {
	return GetContextColorCode(opts.TrueColor)
}

func (f *DetailedFormatter) colorReset() string {
	return GetColorReset()
}
