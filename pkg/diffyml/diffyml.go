// Package diffyml provides YAML diff functionality for comparing YAML documents.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Type aliases for public API
type YAMLValue = types.YAMLValue
type YAMLKind = types.YAMLKind
type DiffType = types.DiffType
type Difference = types.Difference
type DiffGroup = types.DiffGroup
type Options = types.Options

// YAMLKind constants
const (
	KindNull      = types.KindNull
	KindString    = types.KindString
	KindInt       = types.KindInt
	KindFloat     = types.KindFloat
	KindBool      = types.KindBool
	KindTimestamp = types.KindTimestamp
	KindMap       = types.KindMap
	KindList      = types.KindList
	KindUnknown   = types.KindUnknown
)

// DiffType constants
const (
	DiffAdded        = types.DiffAdded
	DiffRemoved      = types.DiffRemoved
	DiffModified     = types.DiffModified
	DiffOrderChanged = types.DiffOrderChanged
)

// Compare compares two YAML documents and returns the differences.
func Compare(from, to []byte, opts *Options) ([]Difference, error) {
	return compare.Compare(from, to, opts)
}

// yamlKindOf returns the YAMLKind for a parsed YAML value.
func yamlKindOf(v YAMLValue) YAMLKind {
	return types.YamlKindOf(v)
}

// extractPathOrder extracts the order of all paths from parsed documents.
func extractPathOrder(fromDocs, toDocs []interface{}, opts *Options) map[string]int {
	return compare.ExtractPathOrder(fromDocs, toDocs, opts)
}

// isListEntryDiff checks if a difference represents a list entry.
func isListEntryDiff(diff Difference) bool {
	return types.IsListEntryDiff(diff)
}

// sortDiffsWithOrder sorts differences for consistent output.
func sortDiffsWithOrder(diffs []Difference, pathOrder map[string]int) {
	types.SortDiffsWithOrder(diffs, pathOrder)
}

// formatCount returns a human-readable count string.
func formatCount(n int) string {
	return types.FormatCount(n)
}

// pluralize returns singular or plural form based on count.
func pluralize(n int, singular, plural string) string {
	return types.Pluralize(n, singular, plural)
}

// joinPath joins path segments with a dot.
func joinPath(base, key string) string {
	return types.JoinPath(base, key)
}

// cleanPath removes leading dots from path.
func cleanPath(path string) string {
	return types.CleanPath(path)
}
