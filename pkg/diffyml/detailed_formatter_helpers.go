// detailed_formatter_helpers.go - Pure utility functions for detailed output.
//
// String manipulation, type naming, formatting helpers, and path parsing
// used by the detailed formatter. All functions are pure (no receiver).
package diffyml

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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

// formatDetailedValue formats a value for display, handling nil.
func formatDetailedValue(val any) string {
	if val == nil {
		return "<nil>"
	}
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
