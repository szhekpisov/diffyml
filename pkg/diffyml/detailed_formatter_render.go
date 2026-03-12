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
func (f *DetailedFormatter) renderEntryValue(sb *strings.Builder, val any, symbol string, indent int, path string, isList bool, opts *FormatOptions) {
	code := f.colorRemoved(opts)
	if symbol == "+" {
		code = f.colorAdded(opts)
	}

	// Map entries: extract key from path and render as key: value
	if !isList {
		key := path
		if idx := strings.LastIndex(path, "."); idx >= 0 {
			key = path[idx+1:]
		}
		f.renderKeyValueYAML(sb, key, val, indent, code, opts)
		return
	}

	// List entries: delegate to renderListItems which handles *OrderedMap,
	// map[string]any, and scalar fallback uniformly.
	// For []any values, pass items directly; otherwise wrap as single item.
	if v, ok := val.([]any); ok {
		f.renderListItems(sb, v, indent, code, opts)
	} else {
		f.renderListItems(sb, []any{val}, indent, code, opts)
	}
}

// renderDocumentValue renders a whole YAML document (top-level key-value pairs without list "- " prefix).
func (f *DetailedFormatter) renderDocumentValue(sb *strings.Builder, val any, symbol string, indent int, opts *FormatOptions) {
	code := f.colorRemoved(opts)
	if symbol == "+" {
		code = f.colorAdded(opts)
	}

	pad := strings.Repeat(" ", indent)
	whiteCode := colorWhite
	if opts.TrueColor {
		whiteCode = TrueColorCode(255, 255, 255)
	}
	f.writeColoredLine(sb, pad+"---", whiteCode, opts)

	switch v := val.(type) {
	case *OrderedMap:
		for _, key := range v.Keys {
			f.renderKeyValueYAML(sb, key, v.Values[key], indent, code, opts)
		}
	case map[string]any:
		for _, key := range sortedMapKeys(v) {
			f.renderKeyValueYAML(sb, key, v[key], indent, code, opts)
		}
	default:
		f.writeColoredLine(sb, fmt.Sprintf("%s%v", pad, formatDetailedValue(val)), code, opts)
	}
}

// renderKeyValueYAML renders a key: value pair in plain YAML style with color.
// Uses standard YAML indentation (2 spaces per level), no pipe guides.
func (f *DetailedFormatter) renderKeyValueYAML(sb *strings.Builder, key string, val any, indent int, colorCode string, opts *FormatOptions) {
	pad := strings.Repeat(" ", indent)
	switch v := val.(type) {
	case *OrderedMap:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		for _, k := range v.Keys {
			f.renderKeyValueYAML(sb, k, v.Values[k], indent+2, colorCode, opts)
		}
	case map[string]any:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		for _, k := range sortedMapKeys(v) {
			f.renderKeyValueYAML(sb, k, v[k], indent+2, colorCode, opts)
		}
	case []any:
		f.writeColoredLine(sb, fmt.Sprintf("%s%s:", pad, key), colorCode, opts)
		f.renderListItems(sb, v, indent+2, colorCode, opts)
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
func (f *DetailedFormatter) renderFirstKeyValueYAML(sb *strings.Builder, key string, val any, indent int, colorCode string, opts *FormatOptions) {
	pad := strings.Repeat(" ", indent)
	switch v := val.(type) {
	case *OrderedMap:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		for _, k := range v.Keys {
			f.renderKeyValueYAML(sb, k, v.Values[k], indent+4, colorCode, opts)
		}
	case map[string]any:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		for _, k := range sortedMapKeys(v) {
			f.renderKeyValueYAML(sb, k, v[k], indent+4, colorCode, opts)
		}
	case []any:
		f.writeColoredLine(sb, fmt.Sprintf("%s- %s:", pad, key), colorCode, opts)
		f.renderListItems(sb, v, indent+4, colorCode, opts)
	default:
		if str, ok := val.(string); ok && strings.Contains(str, "\n") {
			f.renderMultilineValue(sb, fmt.Sprintf("%s- %s:", pad, key), str, colorCode, indent+2, opts)
		} else {
			f.writeColoredLine(sb, fmt.Sprintf("%s- %s: %v", pad, key, formatDetailedValue(val)), colorCode, opts)
		}
	}
}

// renderListItems renders items of a []any list with proper type dispatch.
// Structured items (*OrderedMap, map[string]any) are rendered using YAML-style
// key-value methods. Scalars use formatDetailedValue().
func (f *DetailedFormatter) renderListItems(sb *strings.Builder, items []any, indent int, colorCode string, opts *FormatOptions) {
	for _, item := range items {
		switch v := item.(type) {
		case *OrderedMap:
			for i, key := range v.Keys {
				if i == 0 {
					f.renderFirstKeyValueYAML(sb, key, v.Values[key], indent, colorCode, opts)
				} else {
					f.renderKeyValueYAML(sb, key, v.Values[key], indent+2, colorCode, opts)
				}
			}
		case map[string]any:
			for i, key := range sortedMapKeys(v) {
				if i == 0 {
					f.renderFirstKeyValueYAML(sb, key, v[key], indent, colorCode, opts)
				} else {
					f.renderKeyValueYAML(sb, key, v[key], indent+2, colorCode, opts)
				}
			}
		default:
			pad := strings.Repeat(" ", indent)
			f.writeColoredLine(sb, fmt.Sprintf("%s- %v", pad, formatDetailedValue(item)), colorCode, opts)
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
