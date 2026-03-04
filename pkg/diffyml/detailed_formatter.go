// detailed_formatter.go - Detailed human-readable output formatter.
package diffyml

import (
	"time"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
)

type DetailedFormatter = format.DetailedFormatter

// Type aliases for unexported types used in tests.
type editOpType = format.EditOpType
type editOp = format.EditOp

const (
	editKeep   = format.EditKeep
	editInsert = format.EditInsert
	editDelete = format.EditDelete
)

// Wrapper functions for unexported names used in tests.
func computeLineDiff(fromLines, toLines []string) []editOp {
	return format.ComputeLineDiff(fromLines, toLines)
}
func isWhitespaceOnlyChange(from, to string) bool { return format.IsWhitespaceOnlyChange(from, to) }
func stripWhitespace(s string) string              { return format.StripWhitespace(s) }
func visualizeWhitespace(s string) string          { return format.VisualizeWhitespace(s) }
func yamlTypeName(v interface{}) string            { return format.YamlTypeName(v) }
func formatCommaSeparated(val interface{}) string  { return format.FormatCommaSeparated(val) }
func formatDetailedValue(val interface{}) string   { return format.FormatDetailedValue(val) }
func formatTimestamp(t time.Time) string           { return format.FormatTimestamp(t) }
func parseBareDocIndex(path string) (int, bool)    { return format.ParseBareDocIndex(path) }
func parseDocIndexPrefix(path string) (int, string, bool) {
	return format.ParseDocIndexPrefix(path)
}
func formatValueAsYAMLLines(val interface{}) []string { return format.FormatValueAsYAMLLines(val) }
func isStructured(val interface{}) bool               { return format.IsStructured(val) }
