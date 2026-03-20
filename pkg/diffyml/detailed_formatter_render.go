// detailed_formatter_render.go - YAML value rendering for detailed output.
//
// Handles rendering of structured values (OrderedMap, maps, lists) and scalars
// as indented YAML-style text for the detailed formatter.
package diffyml

import (
	"fmt"
	"strings"
)

// renderEntryValue renders a value for an entry batch line.
// For list entries, renders values with "- " prefix. For map entries, renders as "key: value".
// The entire block is colored (green for adds, red for removes).
func (f *DetailedFormatter) renderEntryValue(sb *strings.Builder, val any, symbol string, indent int, path DiffPath, isList bool, opts *FormatOptions) {
	diffType := DiffRemoved
	if symbol == "+" {
		diffType = DiffAdded
	}
	palette := entryPalette(diffType, opts.TrueColor)

	// Map entries: render as key: value pairs
	if !isList {
		// When value is an OrderedMap (parent-level diff), render each key-value directly
		if om, ok := val.(*OrderedMap); ok {
			for _, k := range om.Keys {
				f.renderKeyValueYAML(sb, k, om.Values[k], indent, palette, opts)
			}
			return
		}
		key := path.Last()
		f.renderKeyValueYAML(sb, key, val, indent, palette, opts)
		return
	}

	// List entries: delegate to renderListItems which handles *OrderedMap,
	// map[string]any, and scalar fallback uniformly.
	// For []any values, pass items directly; otherwise wrap as single item.
	if v, ok := val.([]any); ok {
		f.renderListItems(sb, v, indent, palette, opts)
	} else {
		f.renderListItems(sb, []any{val}, indent, palette, opts)
	}
}

// renderDocumentValue renders a whole YAML document (top-level key-value pairs without list "- " prefix).
func (f *DetailedFormatter) renderDocumentValue(sb *strings.Builder, val any, symbol string, indent int, opts *FormatOptions) {
	diffType := DiffRemoved
	if symbol == "+" {
		diffType = DiffAdded
	}
	palette := entryPalette(diffType, opts.TrueColor)

	pad := strings.Repeat(" ", indent)
	whiteCode := colorWhite
	if opts.TrueColor {
		whiteCode = TrueColorCode(255, 255, 255)
	}
	f.writeColoredLine(sb, pad+"---", whiteCode, opts)

	switch v := val.(type) {
	case *OrderedMap:
		for _, key := range v.Keys {
			f.renderKeyValueYAML(sb, key, v.Values[key], indent, palette, opts)
		}
	case map[string]any:
		for _, key := range sortedMapKeys(v) {
			f.renderKeyValueYAML(sb, key, v[key], indent, palette, opts)
		}
	default:
		f.writeColoredLine(sb, fmt.Sprintf("%s%v", pad, formatDetailedValue(val)), palette.ScalarColor(val), opts)
	}
}

// renderKeyValueYAML renders a key: value pair in plain YAML style with color.
// Uses standard YAML indentation (2 spaces per level), no pipe guides.
func (f *DetailedFormatter) renderKeyValueYAML(sb *strings.Builder, key string, val any, indent int, palette *YAMLColorPalette, opts *FormatOptions) {
	f.renderKVCore(sb, key, val, indent, "", indent+2, palette, opts)
}

// renderFirstKeyValueYAML renders the first key of a list entry with "- " prefix.
// The key is rendered as "    - key: value" where indent is the base indentation.
// For nested values, continuation uses indent+4 to align under the key.
func (f *DetailedFormatter) renderFirstKeyValueYAML(sb *strings.Builder, key string, val any, indent int, palette *YAMLColorPalette, opts *FormatOptions) {
	f.renderKVCore(sb, key, val, indent, "- ", indent+4, palette, opts)
}

// renderKVCore is the shared implementation for renderKeyValueYAML and renderFirstKeyValueYAML.
// prefix is "" for plain keys or "- " for list-item first keys; childIndent is the
// indentation level used for nested children.
func (f *DetailedFormatter) renderKVCore(sb *strings.Builder, key string, val any, indent int, prefix string, childIndent int, palette *YAMLColorPalette, opts *FormatOptions) {
	pad := strings.Repeat(" ", indent)
	keyPrefix := pad + prefix + key
	switch v := val.(type) {
	case *OrderedMap:
		if len(v.Keys) == 0 {
			f.writeColoredLine(sb, keyPrefix+": {}", palette.EmptyStructure, opts)
		} else {
			f.writeColoredLine(sb, keyPrefix+":", palette.Key, opts)
			for _, k := range v.Keys {
				f.renderKeyValueYAML(sb, k, v.Values[k], childIndent, palette, opts)
			}
		}
	case map[string]any:
		if len(v) == 0 {
			f.writeColoredLine(sb, keyPrefix+": {}", palette.EmptyStructure, opts)
		} else {
			f.writeColoredLine(sb, keyPrefix+":", palette.Key, opts)
			for _, k := range sortedMapKeys(v) {
				f.renderKeyValueYAML(sb, k, v[k], childIndent, palette, opts)
			}
		}
	case []any:
		if len(v) == 0 {
			f.writeColoredLine(sb, keyPrefix+": []", palette.EmptyStructure, opts)
		} else {
			f.writeColoredLine(sb, keyPrefix+":", palette.Key, opts)
			f.renderListItems(sb, v, childIndent, palette, opts)
		}
	default:
		if str, ok := val.(string); ok && strings.Contains(str, "\n") {
			// For multiline, content indentation is always baseIndent+2 regardless of prefix
			mlIndent := indent
			if prefix != "" {
				mlIndent = indent + 2
			}
			f.renderMultilineValue(sb, keyPrefix+":", str, palette, mlIndent, opts)
		} else {
			f.writeKeyValueLine(sb, keyPrefix+":", formatDetailedValue(val), palette.Key, palette.ScalarColor(val), opts)
		}
	}
}

// renderListItems renders items of a []any list with proper type dispatch.
// Structured items (*OrderedMap, map[string]any) are rendered using YAML-style
// key-value methods. Scalars use formatDetailedValue().
func (f *DetailedFormatter) renderListItems(sb *strings.Builder, items []any, indent int, palette *YAMLColorPalette, opts *FormatOptions) {
	for _, item := range items {
		switch v := item.(type) {
		case *OrderedMap:
			for i, key := range v.Keys {
				if i == 0 {
					f.renderFirstKeyValueYAML(sb, key, v.Values[key], indent, palette, opts)
				} else {
					f.renderKeyValueYAML(sb, key, v.Values[key], indent+2, palette, opts)
				}
			}
		case map[string]any:
			for i, key := range sortedMapKeys(v) {
				if i == 0 {
					f.renderFirstKeyValueYAML(sb, key, v[key], indent, palette, opts)
				} else {
					f.renderKeyValueYAML(sb, key, v[key], indent+2, palette, opts)
				}
			}
		default:
			pad := strings.Repeat(" ", indent)
			f.writeKeyValueLine(sb, pad+"-", formatDetailedValue(item), palette.Key, palette.ScalarColor(item), opts)
		}
	}
}

// renderMultilineValue renders a multiline string in YAML block literal style (|).
func (f *DetailedFormatter) renderMultilineValue(sb *strings.Builder, prefix, value string, palette *YAMLColorPalette, indent int, opts *FormatOptions) {
	f.writeColoredLine(sb, prefix+" |", palette.Key, opts)
	pad := strings.Repeat(" ", indent+2)
	for _, line := range strings.Split(strings.TrimRight(value, "\n"), "\n") {
		f.writeColoredLine(sb, pad+line, palette.MultilineText, opts)
	}
}

// writeTypeChangeValue renders a value for type-change display.
// For structured values (maps, lists), renders as indented YAML lines.
// For scalars, renders as a single line.
func (f *DetailedFormatter) writeTypeChangeValue(sb *strings.Builder, val any, symbol string, colorCode string, opts *FormatOptions) {
	if isStructured(val) {
		for _, line := range formatValueAsYAMLLines(val) {
			f.writeColoredLine(sb, fmt.Sprintf("    %s %s", symbol, line), colorCode, opts)
		}
	} else {
		f.writeColoredLine(sb, fmt.Sprintf("    %s %v", symbol, formatDetailedValue(val)), colorCode, opts)
	}
}

// formatValueAsYAMLLines formats a structured value as YAML lines for type-change display.
func formatValueAsYAMLLines(val any) []string {
	var lines []string
	formatValueAsYAMLRecurse(val, "", &lines)
	return lines
}

func formatValueAsYAMLRecurse(val any, indent string, lines *[]string) {
	switch v := val.(type) {
	case *OrderedMap:
		for _, key := range v.Keys {
			child := v.Values[key]
			if isStructured(child) {
				*lines = append(*lines, fmt.Sprintf("%s%s:", indent, key))
				formatValueAsYAMLRecurse(child, indent+"  ", lines)
			} else {
				*lines = append(*lines, fmt.Sprintf("%s%s: %v", indent, key, formatDetailedValue(child)))
			}
		}
	case map[string]any:
		for key, child := range v {
			if isStructured(child) {
				*lines = append(*lines, fmt.Sprintf("%s%s:", indent, key))
				formatValueAsYAMLRecurse(child, indent+"  ", lines)
			} else {
				*lines = append(*lines, fmt.Sprintf("%s%s: %v", indent, key, formatDetailedValue(child)))
			}
		}
	case []any:
		for _, item := range v {
			if isStructured(item) {
				*lines = append(*lines, fmt.Sprintf("%s- ...", indent))
				formatValueAsYAMLRecurse(item, indent+"  ", lines)
			} else {
				*lines = append(*lines, fmt.Sprintf("%s- %v", indent, formatDetailedValue(item)))
			}
		}
	default:
		*lines = append(*lines, fmt.Sprintf("%s%v", indent, formatDetailedValue(val)))
	}
}

func isStructured(val any) bool {
	switch val.(type) {
	case *OrderedMap, map[string]any, []any:
		return true
	default:
		return false
	}
}
