// formatter.go - Output formatting for differences.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Type aliases for public API
type Formatter = types.Formatter
type FormatOptions = types.FormatOptions
type StructuredFormatter = types.StructuredFormatter
type CompactFormatter = format.CompactFormatter
type BriefFormatter = format.BriefFormatter

func DefaultFormatOptions() *FormatOptions { return types.DefaultFormatOptions() }

func GetFormatter(name string) (Formatter, error) { return format.GetFormatter(name) }

func formatValue(val interface{}) string      { return format.FormatValue(val) }
func convertToGoPatchPath(path string) string { return format.ConvertToGoPatchPath(path) }
func diffDescription(diff Difference) string  { return format.DiffDescription(diff) }
