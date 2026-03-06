// detailed_formatter.go - Detailed human-readable output formatter.
//
// Renders differences in a path-grouped style with descriptive labels,
// structured value display, and multiline text diffs.
package format

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// K8sDocumentPath is the heading used for single-document Kubernetes paths.
const K8sDocumentPath = "(document)"

// DetailedFormatter implements the Formatter interface for detailed output.
type DetailedFormatter struct{}

// pathGroup holds a path and its associated differences for grouping.
type pathGroup struct {
	Path  string
	Diffs []types.Difference
}

// EditOpType represents a type of edit operation in a line diff.
type EditOpType int

const (
	EditKeep EditOpType = iota
	EditInsert
	EditDelete
)

// EditOp represents a single edit operation in a line diff.
type EditOp struct {
	Type EditOpType
	Line string
}

// Format renders differences in detailed human-readable style.
func (f *DetailedFormatter) Format(diffs []types.Difference, opts *types.FormatOptions) string {
	if opts == nil {
		opts = types.DefaultFormatOptions()
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
func (f *DetailedFormatter) formatHeader(sb *strings.Builder, diffs []types.Difference, opts *types.FormatOptions) {
	clr := Colorizer{TrueColor: opts.TrueColor}
	if opts.Color {
		sb.WriteString(clr.Modified())
	}
	fmt.Fprintf(sb, "Found %s %s",
		types.FormatCount(len(diffs)),
		types.Pluralize(len(diffs), "difference", "differences"))
	if opts.Color {
		sb.WriteString(clr.Reset())
	}
	sb.WriteString("\n\n")
}

// groupByPath groups diffs by their Path field, preserving order of first occurrence.
func (f *DetailedFormatter) groupByPath(diffs []types.Difference) []pathGroup {
	var groups []pathGroup
	index := make(map[string]int) // path -> index in groups

	for _, diff := range diffs {
		if idx, exists := index[diff.Path]; exists {
			groups[idx].Diffs = append(groups[idx].Diffs, diff)
		} else {
			index[diff.Path] = len(groups)
			groups = append(groups, pathGroup{
				Path:  diff.Path,
				Diffs: []types.Difference{diff},
			})
		}
	}

	return groups
}

// formatPathHeading renders the path line for a group of diffs.
func (f *DetailedFormatter) formatPathHeading(sb *strings.Builder, path string, isMultiDoc bool, opts *types.FormatOptions) {
	heading := path
	if path == "" {
		if opts.UseGoPatchStyle {
			heading = "/"
		} else {
			heading = "(root level)"
		}
	} else if idx, ok := ParseBareDocIndex(path); ok {
		if isMultiDoc {
			heading = fmt.Sprintf("(document %d)", idx+1)
		} else {
			heading = K8sDocumentPath
		}
	} else if idx, rest, ok := ParseDocIndexPrefix(path); ok {
		if opts.UseGoPatchStyle {
			heading = fmt.Sprintf("%d:%s", idx, ConvertToGoPatchPath(rest))
		} else {
			heading = fmt.Sprintf("%d:%s", idx, rest)
		}
	} else if opts.UseGoPatchStyle {
		heading = ConvertToGoPatchPath(path)
	}

	if opts.Color {
		sb.WriteString(StyleBold)
	}
	sb.WriteString(heading)
	if opts.Color {
		sb.WriteString(ColorReset)
	}
	sb.WriteString("\n")
}

// formatGroupDiffs renders all diffs within a path group.
// Groups consecutive additions and removals for batched descriptors.
func (f *DetailedFormatter) formatGroupDiffs(sb *strings.Builder, group pathGroup, opts *types.FormatOptions) {
	var added, removed []types.Difference
	var others []types.Difference

	for _, diff := range group.Diffs {
		switch diff.Type {
		case types.DiffAdded:
			added = append(added, diff)
		case types.DiffRemoved:
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
func (f *DetailedFormatter) formatEntryBatch(sb *strings.Builder, diffs []types.Difference, action string, opts *types.FormatOptions) {
	clr := Colorizer{TrueColor: opts.TrueColor}
	n := len(diffs)
	isListEntry := types.IsListEntryDiff(diffs[0])
	entryType := "map"
	if isListEntry {
		entryType = "list"
	}

	countStr := types.FormatCount(n)
	noun := types.Pluralize(n, entryType+" entry", entryType+" entries")
	symbol := "+"
	cc := clr.Added()
	if action == "removed" {
		symbol = "-"
		cc = clr.Removed()
	}

	if opts.Color {
		sb.WriteString("  ")
		sb.WriteString(cc)
		fmt.Fprintf(sb, "%s %s %s %s:", symbol, countStr, noun, action)
		sb.WriteString(clr.Reset())
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
		if !opts.NoCertInspection {
			if s, ok := val.(string); ok && IsPEMCertificate(s) {
				val = FormatCertificate(s)
			}
		}
		f.renderEntryValue(sb, val, symbol, 4, diff.Path, isListEntry, opts)
	}
	sb.WriteString("\n")
}

// renderEntryValue renders a value for an entry batch line.
// For list entries, renders values with "- " prefix. For map entries, renders as "key: value".
// The entire block is colored (green for adds, red for removes).
func (f *DetailedFormatter) renderEntryValue(sb *strings.Builder, val interface{}, symbol string, indent int, path string, isList bool, opts *types.FormatOptions) {
	clr := Colorizer{TrueColor: opts.TrueColor}
	colorCode := ""
	if opts.Color {
		if symbol == "+" {
			colorCode = clr.Added()
		} else {
			colorCode = clr.Removed()
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

	// List entries: delegate to renderListItems which handles *types.OrderedMap
	// and scalar fallback uniformly.
	// For []interface{} values, pass items directly; otherwise wrap as single item.
	if v, ok := val.([]interface{}); ok {
		f.renderListItems(sb, v, indent, colorCode, opts)
	} else {
		f.renderListItems(sb, []interface{}{val}, indent, colorCode, opts)
	}
}

// renderKeyValueYAML renders a key: value pair in plain YAML style with color.
// Uses standard YAML indentation (2 spaces per level), no pipe guides.
func (f *DetailedFormatter) renderKeyValueYAML(sb *strings.Builder, key string, val interface{}, indent int, colorCode string, opts *types.FormatOptions) {
	if om := types.ToOrderedMap(val); om != nil {
		val = om
	}
	pad := strings.Repeat(" ", indent)
	switch v := val.(type) {
	case *types.OrderedMap:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		for _, k := range v.Keys {
			f.renderKeyValueYAML(sb, k, v.Values[k], indent+2, colorCode, opts)
		}
	case []interface{}:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		f.renderListItems(sb, v, indent+2, colorCode, opts)
	default:
		if str, ok := val.(string); ok && strings.Contains(str, "\n") {
			f.renderMultilineValue(sb, fmt.Sprintf("%s%s:", pad, key), str, colorCode, indent, opts)
		} else {
			f.writeColoredLine(sb, fmt.Sprintf("%s%s: %v", pad, key, FormatDetailedValue(val)), colorCode, opts)
		}
	}
}

// renderFirstKeyValueYAML renders the first key of a list entry with "- " prefix.
// The key is rendered as "    - key: value" where indent is the base indentation.
// For nested values, continuation uses indent+2 to align under the key.
func (f *DetailedFormatter) renderFirstKeyValueYAML(sb *strings.Builder, key string, val interface{}, indent int, colorCode string, opts *types.FormatOptions) {
	if om := types.ToOrderedMap(val); om != nil {
		val = om
	}
	pad := strings.Repeat(" ", indent)
	switch v := val.(type) {
	case *types.OrderedMap:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		for _, k := range v.Keys {
			f.renderKeyValueYAML(sb, k, v.Values[k], indent+4, colorCode, opts)
		}
	case []interface{}:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		f.renderListItems(sb, v, indent+4, colorCode, opts)
	default:
		if str, ok := val.(string); ok && strings.Contains(str, "\n") {
			f.renderMultilineValue(sb, fmt.Sprintf("%s- %s:", pad, key), str, colorCode, indent+2, opts)
		} else {
			f.writeColoredLine(sb, fmt.Sprintf("%s- %s: %v", pad, key, FormatDetailedValue(val)), colorCode, opts)
		}
	}
}

// renderListItems renders items of a []interface{} list with proper type dispatch.
// Structured items (*types.OrderedMap) are rendered using YAML-style key-value methods.
// Scalars use FormatDetailedValue().
func (f *DetailedFormatter) renderListItems(sb *strings.Builder, items []interface{}, indent int, colorCode string, opts *types.FormatOptions) {
	for _, item := range items {
		if om := types.ToOrderedMap(item); om != nil {
			for i, key := range om.Keys {
				if i == 0 {
					f.renderFirstKeyValueYAML(sb, key, om.Values[key], indent, colorCode, opts)
				} else {
					f.renderKeyValueYAML(sb, key, om.Values[key], indent+2, colorCode, opts)
				}
			}
		} else {
			pad := strings.Repeat(" ", indent)
			f.writeColoredLine(sb, fmt.Sprintf("%s- %v", pad, FormatDetailedValue(item)), colorCode, opts)
		}
	}
}

// renderMultilineValue renders a multiline string in YAML block literal style (|).
func (f *DetailedFormatter) renderMultilineValue(sb *strings.Builder, prefix, value, colorCode string, indent int, opts *types.FormatOptions) {
	f.writeColoredLine(sb, prefix+" |", colorCode, opts)
	pad := strings.Repeat(" ", indent+2)
	for _, line := range strings.Split(strings.TrimRight(value, "\n"), "\n") {
		f.writeColoredLine(sb, pad+line, colorCode, opts)
	}
}

// formatChangeDescriptor renders the descriptor line for a single diff.
func (f *DetailedFormatter) formatChangeDescriptor(sb *strings.Builder, diff types.Difference, opts *types.FormatOptions) {
	clr := Colorizer{TrueColor: opts.TrueColor}
	switch diff.Type {
	case types.DiffModified:
		f.formatModified(sb, diff, opts)
	case types.DiffOrderChanged:
		f.writeDescriptorLine(sb, "  ⇆ order changed", clr.Modified(), opts)
		if diff.From != nil {
			f.writeColoredLine(sb, fmt.Sprintf("    - %s", FormatCommaSeparated(diff.From)), clr.Removed(), opts)
		}
		if diff.To != nil {
			f.writeColoredLine(sb, fmt.Sprintf("    + %s", FormatCommaSeparated(diff.To)), clr.Added(), opts)
		}
		sb.WriteString("\n")
	}
}

// formatModified renders a modification descriptor with type change, multiline, and whitespace detection.
func (f *DetailedFormatter) formatModified(sb *strings.Builder, diff types.Difference, opts *types.FormatOptions) {
	clr := Colorizer{TrueColor: opts.TrueColor}
	fromType := YamlTypeName(diff.From)
	toType := YamlTypeName(diff.To)

	// Type change detection
	if fromType != toType {
		if opts.Color {
			f.writeDescriptorLine(sb, fmt.Sprintf("  ± type change from %s%s%s to %s%s%s",
				StyleItalic, fromType, StyleItalicOff,
				StyleItalic, toType, StyleItalicOff), clr.Modified(), opts)
		} else {
			f.writeDescriptorLine(sb, fmt.Sprintf("  ± type change from %s to %s", fromType, toType), clr.Modified(), opts)
		}
		f.writeTypeChangeValue(sb, diff.From, "-", clr.Removed(), opts)
		f.writeTypeChangeValue(sb, diff.To, "+", clr.Added(), opts)
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
		if IsWhitespaceOnlyChange(fromStr, toStr) {
			f.writeDescriptorLine(sb, "  ± whitespace only change", clr.Modified(), opts)
			f.writeColoredLine(sb, fmt.Sprintf("    - %s", VisualizeWhitespace(fromStr)), clr.Removed(), opts)
			f.writeColoredLine(sb, fmt.Sprintf("    + %s", VisualizeWhitespace(toStr)), clr.Added(), opts)
			sb.WriteString("\n")
			return
		}

		// Multiline detection
		if strings.Contains(fromStr, "\n") || strings.Contains(toStr, "\n") {
			f.formatMultilineDiff(sb, fromStr, toStr, opts)
			return
		}

		// Scalar string value change (may be cert-transformed)
		f.writeDescriptorLine(sb, "  ± value change", clr.Modified(), opts)
		f.writeColoredLine(sb, fmt.Sprintf("    - %s", fromStr), clr.Removed(), opts)
		f.writeColoredLine(sb, fmt.Sprintf("    + %s", toStr), clr.Added(), opts)
		sb.WriteString("\n")
		return
	}

	// Default: non-string scalar value change
	f.writeDescriptorLine(sb, "  ± value change", clr.Modified(), opts)
	f.writeColoredLine(sb, fmt.Sprintf("    - %v", FormatDetailedValue(diff.From)), clr.Removed(), opts)
	f.writeColoredLine(sb, fmt.Sprintf("    + %v", FormatDetailedValue(diff.To)), clr.Added(), opts)
	sb.WriteString("\n")
}

// formatMultilineDiff renders an inline line-by-line diff for multiline strings.
func (f *DetailedFormatter) formatMultilineDiff(sb *strings.Builder, from, to string, opts *types.FormatOptions) {
	clr := Colorizer{TrueColor: opts.TrueColor}
	fromLines := strings.Split(from, "\n")
	toLines := strings.Split(to, "\n")
	ops := ComputeLineDiff(fromLines, toLines)

	// Count additions and deletions
	additions := 0
	deletions := 0
	for _, op := range ops {
		switch op.Type {
		case EditInsert:
			additions++
		case EditDelete:
			deletions++
		}
	}

	descriptor := fmt.Sprintf("  ± value change in multiline text (%s %s, %s %s)",
		types.FormatCount(additions), types.Pluralize(additions, "insert", "inserts"),
		types.FormatCount(deletions), types.Pluralize(deletions, "deletion", "deletions"))
	f.writeDescriptorLine(sb, descriptor, clr.Modified(), opts)

	// Apply context collapsing
	contextLines := opts.ContextLines
	if contextLines < 0 {
		contextLines = 4
	}

	// Mark which ops are near a change
	nearChange := make([]bool, len(ops))
	for i, op := range ops {
		if op.Type != EditKeep {
			// Mark surrounding context
			for j := max(0, i-contextLines); j <= min(len(ops)-1, i+contextLines); j++ {
				nearChange[j] = true
			}
		}
	}

	// Render with collapsing
	skipUntil := 0
	for i, op := range ops {
		if i < skipUntil {
			continue
		}
		if op.Type != EditKeep || nearChange[i] {
			switch op.Type {
			case EditKeep:
				f.writeColoredLine(sb, fmt.Sprintf("      %s", op.Line), clr.Context(), opts)
			case EditInsert:
				f.writeColoredLine(sb, fmt.Sprintf("    + %s", op.Line), clr.Added(), opts)
			case EditDelete:
				f.writeColoredLine(sb, fmt.Sprintf("    - %s", op.Line), clr.Removed(), opts)
			}
		} else {
			// Count consecutive non-near-change keep ops
			collapsed := 0
			for _, sub := range ops[i:] {
				if sub.Type != EditKeep || nearChange[i+collapsed] {
					break
				}
				collapsed++
			}
			skipUntil = i + collapsed
			f.writeColoredLine(sb, fmt.Sprintf("    [%d %s unchanged]", collapsed, types.Pluralize(collapsed, "line", "lines")), clr.Context(), opts)
		}
	}
	sb.WriteString("\n")
}

// ComputeLineDiff computes line-level diff using LCS algorithm.
func ComputeLineDiff(fromLines, toLines []string) []EditOp {
	m := len(fromLines)
	n := len(toLines)

	// Build LCS table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
			if fromLines[i-1] == toLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to produce edit operations
	var ops []EditOp
	i, j := m, n
	for i > 0 || j > 0 {
		//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
		if i > 0 && j > 0 && fromLines[i-1] == toLines[j-1] {
			ops = append(ops, EditOp{Type: EditKeep, Line: fromLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, EditOp{Type: EditInsert, Line: toLines[j-1]})
			j--
		} else {
			ops = append(ops, EditOp{Type: EditDelete, Line: fromLines[i-1]})
			i--
		}
	}

	// Reverse to get correct order
	slices.Reverse(ops)

	return ops
}

// IsWhitespaceOnlyChange checks if the difference between two strings is only whitespace.
func IsWhitespaceOnlyChange(from, to string) bool {
	if from == to {
		return false // no change at all
	}
	// Strip all whitespace and compare
	fromStripped := StripWhitespace(from)
	toStripped := StripWhitespace(to)
	return fromStripped == toStripped
}

// StripWhitespace removes all whitespace characters from a string.
func StripWhitespace(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// VisualizeWhitespace replaces invisible characters with visible symbols.
func VisualizeWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\n", "↵")
	s = strings.ReplaceAll(s, " ", "·")
	return s
}

// YamlTypeName returns a human-readable YAML type name for a value.
func YamlTypeName(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case int, int64:
		return "int"
	case float64:
		return "float"
	case bool:
		return "bool"
	case *types.OrderedMap:
		return "map"
	case []interface{}:
		return "list"
	case time.Time:
		return "timestamp"
	case nil:
		return "null"
	default:
		if types.ToOrderedMap(v) != nil {
			return "map"
		}
		return fmt.Sprintf("%T", v)
	}
}

// FormatCommaSeparated formats a slice value as comma-separated items.
// For non-slice values, falls back to FormatDetailedValue for scalar display.
func FormatCommaSeparated(val interface{}) string {
	if items, ok := val.([]interface{}); ok {
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = FormatDetailedValue(item)
		}
		return strings.Join(parts, ", ")
	}
	return FormatDetailedValue(val)
}

// FormatDetailedValue formats a value for display, handling nil.
func FormatDetailedValue(val interface{}) string {
	if val == nil {
		return "<nil>"
	}
	if t, ok := val.(time.Time); ok {
		return FormatTimestamp(t)
	}
	return fmt.Sprintf("%v", val)
}

// FormatTimestamp formats a time.Time as a YAML-friendly date or datetime string.
func FormatTimestamp(t time.Time) string {
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}

// writeTypeChangeValue renders a value for type-change display.
// For structured values (maps, lists), renders as indented YAML lines.
// For scalars, renders as a single line.
func (f *DetailedFormatter) writeTypeChangeValue(sb *strings.Builder, val interface{}, symbol string, colorCode string, opts *types.FormatOptions) {
	if IsStructured(val) {
		for _, line := range FormatValueAsYAMLLines(val) {
			f.writeColoredLine(sb, fmt.Sprintf("    %s %s", symbol, line), colorCode, opts)
		}
	} else {
		f.writeColoredLine(sb, fmt.Sprintf("    %s %v", symbol, FormatDetailedValue(val)), colorCode, opts)
	}
}

// FormatValueAsYAMLLines formats a structured value as YAML lines for type-change display.
func FormatValueAsYAMLLines(val interface{}) []string {
	var lines []string
	formatValueAsYAMLRecurse(val, "", &lines)
	return lines
}

func formatValueAsYAMLRecurse(val interface{}, indent string, lines *[]string) {
	if om := types.ToOrderedMap(val); om != nil {
		val = om
	}
	switch v := val.(type) {
	case *types.OrderedMap:
		for _, key := range v.Keys {
			child := v.Values[key]
			if IsStructured(child) {
				*lines = append(*lines, fmt.Sprintf("%s%s:", indent, key))
				formatValueAsYAMLRecurse(child, indent+"  ", lines)
			} else {
				*lines = append(*lines, fmt.Sprintf("%s%s: %v", indent, key, FormatDetailedValue(child)))
			}
		}
	case []interface{}:
		for _, item := range v {
			if IsStructured(item) {
				*lines = append(*lines, fmt.Sprintf("%s- ...", indent))
				formatValueAsYAMLRecurse(item, indent+"  ", lines)
			} else {
				*lines = append(*lines, fmt.Sprintf("%s- %v", indent, FormatDetailedValue(item)))
			}
		}
	default:
		*lines = append(*lines, fmt.Sprintf("%s%v", indent, FormatDetailedValue(val)))
	}
}

// IsStructured reports whether val is a structured value (*types.OrderedMap or []interface{}).
func IsStructured(val interface{}) bool {
	switch val.(type) {
	case *types.OrderedMap, []interface{}:
		return true
	default:
		return types.ToOrderedMap(val) != nil
	}
}

// writeColoredLine writes a line with color code and newline.
func (f *DetailedFormatter) writeColoredLine(sb *strings.Builder, text string, colorCode string, opts *types.FormatOptions) {
	if opts.Color {
		sb.WriteString(colorCode)
		sb.WriteString(text)
		sb.WriteString(ColorReset)
	} else {
		sb.WriteString(text)
	}
	sb.WriteString("\n")
}

// writeDescriptorLine writes a descriptor line with color.
func (f *DetailedFormatter) writeDescriptorLine(sb *strings.Builder, text string, colorCode string, opts *types.FormatOptions) {
	if opts.Color {
		sb.WriteString(colorCode)
		sb.WriteString(text)
		sb.WriteString(ColorReset)
	} else {
		sb.WriteString(text)
	}
	sb.WriteString("\n")
}

// ParseBareDocIndex extracts the index from a bare document index path like "[0]", "[1]".
// Returns the index and true if the path is a bare document index, false otherwise.
// Does NOT match paths like "items[0]" or "[0].spec".
func ParseBareDocIndex(path string) (int, bool) {
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

// ParseDocIndexPrefix extracts a leading [N] document index from a path like "[0].spec.field".
// Returns (index, remainingPath, true) if found, (0, originalPath, false) otherwise.
// Only matches paths starting with "[N]." — bare "[N]" is handled by ParseBareDocIndex.
func ParseDocIndexPrefix(path string) (int, string, bool) {
	if !strings.HasPrefix(path, "[") {
		return 0, path, false
	}
	inner, afterBracket, found := strings.Cut(path[1:], "]")
	if !found {
		return 0, path, false
	}
	// Must have a dot after the closing bracket
	if !strings.HasPrefix(afterBracket, ".") {
		return 0, path, false
	}
	idx, err := strconv.Atoi(inner)
	if err != nil {
		return 0, path, false
	}
	rest := afterBracket[1:] // skip "."
	return idx, rest, true
}

// detectMultiDoc checks if diffs span multiple documents by examining DocumentIndex values.
func (f *DetailedFormatter) detectMultiDoc(diffs []types.Difference) bool {
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
