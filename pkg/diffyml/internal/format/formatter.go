// formatter.go - Output formatting for differences.
//
// Key types: Formatter interface, FormatOptions, StructuredFormatter.
// Shared helpers: FormatValue, ConvertToGoPatchPath, DiffDescription.
// Small formatters: CompactFormatter, BriefFormatter.
// Larger formatters live in their own files:
//
//	github_formatter.go, gitlab_formatter.go, gitea_formatter.go, detailed_formatter.go.
package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// ValidFormatterNames lists all supported formatter names.
var ValidFormatterNames = []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}

// GetFormatter returns a formatter by name.
// Supported names: compact, brief, github, gitlab, gitea, detailed.
// Returns error for invalid formatter names with list of valid options.
func GetFormatter(name string) (types.Formatter, error) {
	// Normalize to lowercase for case-insensitive matching
	name = strings.ToLower(name)

	switch name {
	case "compact":
		return &CompactFormatter{}, nil
	case "brief":
		return &BriefFormatter{}, nil
	case "github":
		return &GitHubFormatter{}, nil
	case "gitlab":
		return &GitLabFormatter{}, nil
	case "gitea":
		return &GiteaFormatter{}, nil
	case "detailed":
		return &DetailedFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown output format %q, valid formats: %s",
			name, strings.Join(ValidFormatterNames, ", "))
	}
}

// --- Shared helpers ---

// FormatValue converts a value to string.
// Shows full values without truncation.
// Structured types (*types.OrderedMap, []interface{}) are serialized to inline YAML
// instead of Go's default %v representation.
func FormatValue(val interface{}) string {
	if val == nil {
		return "<nil>"
	}
	if t, ok := val.(time.Time); ok {
		return FormatTimestamp(t)
	}

	if s, ok := parse.MarshalStructuredYAML(val); ok {
		return strings.TrimSpace(s)
	}

	return fmt.Sprintf("%v", val)
}

// ConvertToGoPatchPath converts dot-notation path to Go-Patch style (/a/b/c).
func ConvertToGoPatchPath(path string) string {
	// Replace dots with slashes
	result := strings.ReplaceAll(path, ".", "/")
	// Convert array notation [n] to /n
	result = strings.ReplaceAll(result, "[", "/")
	result = strings.ReplaceAll(result, "]", "")
	// Ensure leading slash
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}
	return result
}

// DiffDescription returns a human-readable description of a difference.
// Shared by GitHub, GitLab, and Gitea formatters.
func DiffDescription(diff types.Difference) string {
	switch diff.Type {
	case types.DiffAdded:
		return fmt.Sprintf("Added: %s = %s", diff.Path, FormatValue(diff.To))
	case types.DiffRemoved:
		return fmt.Sprintf("Removed: %s = %s", diff.Path, FormatValue(diff.From))
	case types.DiffModified:
		return fmt.Sprintf("Modified: %s changed from %s to %s", diff.Path, FormatValue(diff.From), FormatValue(diff.To))
	default: // DiffOrderChanged
		return fmt.Sprintf("Order changed: %s", diff.Path)
	}
}

// --- CompactFormatter ---

// CompactFormatter renders differences in compact single-line-per-change format with color support.
// This was previously named HumanFormatter - preserved for backward compatibility with the compact output style.
type CompactFormatter struct{}

// FormatSingle renders a single difference in compact format.
func (f *CompactFormatter) FormatSingle(diff types.Difference, opts *types.FormatOptions) string {
	if opts == nil {
		opts = types.DefaultFormatOptions()
	}

	var sb strings.Builder
	f.formatDiff(&sb, diff, opts)
	return sb.String()
}

// Format renders differences in compact single-line-per-change format.
func (f *CompactFormatter) Format(diffs []types.Difference, opts *types.FormatOptions) string {
	if opts == nil {
		opts = types.DefaultFormatOptions()
	}

	if len(diffs) == 0 {
		return "no differences found\n"
	}

	var sb strings.Builder

	// Add header unless omitted
	if !opts.OmitHeader {
		f.formatHeader(&sb, diffs, opts)
	}

	for _, diff := range diffs {
		f.formatDiff(&sb, diff, opts)
	}

	return sb.String()
}

func (f *CompactFormatter) formatHeader(sb *strings.Builder, diffs []types.Difference, opts *types.FormatOptions) {
	var added, removed, modified int
	for _, d := range diffs {
		switch d.Type {
		case types.DiffAdded:
			added++
		case types.DiffRemoved:
			removed++
		case types.DiffModified, types.DiffOrderChanged:
			modified++
		}
	}

	if opts.Color {
		sb.WriteString(types.CompactColor(types.DiffModified))
	}
	fmt.Fprintf(sb, "Found %d difference(s)", len(diffs))
	if opts.Color {
		sb.WriteString(types.ColorReset)
	}
	fmt.Fprintf(sb, " (%d added, %d removed, %d modified)\n\n", added, removed, modified)
}

func (f *CompactFormatter) formatDiff(sb *strings.Builder, diff types.Difference, opts *types.FormatOptions) {
	path := diff.Path
	if opts.UseGoPatchStyle {
		path = ConvertToGoPatchPath(path)
	}

	var indicator string

	switch diff.Type {
	case types.DiffAdded:
		indicator = "+"
	case types.DiffRemoved:
		indicator = "-"
	case types.DiffModified:
		indicator = "±"
	case types.DiffOrderChanged:
		indicator = "⇆"
	}

	colorCode := types.CompactColor(diff.Type)

	// Apply color for the indicator
	if opts.Color {
		sb.WriteString(colorCode)
	}
	sb.WriteString(indicator)
	if opts.Color {
		sb.WriteString(types.ColorReset)
	}

	sb.WriteString(" ")

	sb.WriteString(path)

	f.formatValuesInline(sb, diff, opts)

	sb.WriteString("\n")
}

func (f *CompactFormatter) formatValuesInline(sb *strings.Builder, diff types.Difference, opts *types.FormatOptions) {
	switch diff.Type {
	case types.DiffModified:
		fromStr := FormatValue(diff.From)
		toStr := FormatValue(diff.To)

		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(types.CompactColor(types.DiffRemoved))
		}
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(types.ColorReset)
		}
		sb.WriteString(" → ")
		if opts.Color {
			sb.WriteString(types.CompactColor(types.DiffAdded))
		}
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(types.ColorReset)
		}
	case types.DiffAdded:
		toStr := FormatValue(diff.To)
		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(types.CompactColor(types.DiffAdded))
		}
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(types.ColorReset)
		}
	case types.DiffRemoved:
		fromStr := FormatValue(diff.From)
		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(types.CompactColor(types.DiffRemoved))
		}
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(types.ColorReset)
		}
	case types.DiffOrderChanged:
		sb.WriteString(" (order changed)")
	}
}

// --- BriefFormatter ---

// BriefFormatter renders a concise summary of changes.
type BriefFormatter struct{}

// FormatSingle renders a single difference as a brief line.
func (f *BriefFormatter) FormatSingle(diff types.Difference, opts *types.FormatOptions) string {
	switch diff.Type {
	case types.DiffAdded:
		return fmt.Sprintf("+ %s\n", diff.Path)
	case types.DiffRemoved:
		return fmt.Sprintf("- %s\n", diff.Path)
	case types.DiffModified:
		return fmt.Sprintf("± %s\n", diff.Path)
	default: // DiffOrderChanged
		return fmt.Sprintf("⇆ %s\n", diff.Path)
	}
}

// Format renders a brief summary of differences.
func (f *BriefFormatter) Format(diffs []types.Difference, _ *types.FormatOptions) string {
	if len(diffs) == 0 {
		return "no differences\n"
	}

	var added, removed, modified int
	for _, diff := range diffs {
		switch diff.Type {
		case types.DiffAdded:
			added++
		case types.DiffRemoved:
			removed++
		case types.DiffModified, types.DiffOrderChanged:
			modified++
		}
	}

	var parts []string
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", added))
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", removed))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}

	return strings.Join(parts, ", ") + "\n"
}
