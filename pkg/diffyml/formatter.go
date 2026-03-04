// formatter.go - Output formatting for differences.
//
// Key types: Formatter interface, FormatOptions, StructuredFormatter.
// Shared helpers: formatValue, convertToGoPatchPath, diffDescription.
// Small formatters: CompactFormatter, BriefFormatter.
// Larger formatters live in their own files:
//   github_formatter.go, gitlab_formatter.go, gitea_formatter.go, detailed_formatter.go.
package diffyml

import (
	"fmt"
	"strings"
	"time"
)

// Formatter formats differences for output.
type Formatter interface {
	// Format renders differences as a string according to the formatter's style.
	Format(diffs []Difference, opts *FormatOptions) string
}

// FormatOptions configures output formatting.
type FormatOptions struct {
	// Color enables ANSI color codes in output.
	Color bool
	// TrueColor enables 24-bit true color if supported.
	TrueColor bool
	// OmitHeader skips the summary header in output.
	OmitHeader bool
	// UseGoPatchStyle uses Go-Patch style paths in output.
	UseGoPatchStyle bool
	// ContextLines is the number of context lines for multi-line values.
	ContextLines int
	// NoCertInspection disables x509 certificate inspection in output.
	NoCertInspection bool
	// FilePath is the source file path set by the CLI layer.
	// Used by GitLabFormatter for location.path and fingerprint generation.
	// Defaults to empty string (backward compatible).
	FilePath string
}

// StructuredFormatter is an opt-in interface for formatters that need
// aggregated output across all files in directory mode.
// Checked via type assertion in runDirectory.
type StructuredFormatter interface {
	FormatAll(groups []DiffGroup, opts *FormatOptions) string
}

// DefaultFormatOptions returns FormatOptions with default values.
func DefaultFormatOptions() *FormatOptions {
	return &FormatOptions{
		Color:           false,
		TrueColor:       false,
		OmitHeader:      false,
		UseGoPatchStyle: false,
		ContextLines:    4,
	}
}

// validFormatterNames lists all supported formatter names.
var validFormatterNames = []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}

// GetFormatter returns a formatter by name.
// Supported names: compact, brief, github, gitlab, gitea, detailed.
// Returns error for invalid formatter names with list of valid options.
func GetFormatter(name string) (Formatter, error) {
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
			name, strings.Join(validFormatterNames, ", "))
	}
}

// --- Shared helpers ---

// formatValue converts a value to string.
// Shows full values without truncation.
// Structured types (*OrderedMap, []interface{}) are serialized to inline YAML
// instead of Go's default %v representation.
func formatValue(val interface{}) string {
	if val == nil {
		return "<nil>"
	}
	if t, ok := val.(time.Time); ok {
		return formatTimestamp(t)
	}

	if s, ok := marshalStructuredYAML(val); ok {
		return strings.TrimSpace(s)
	}

	return fmt.Sprintf("%v", val)
}

// convertToGoPatchPath converts dot-notation path to Go-Patch style (/a/b/c).
func convertToGoPatchPath(path string) string {
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

// diffDescription returns a human-readable description of a difference.
// Shared by GitHub, GitLab, and Gitea formatters.
func diffDescription(diff Difference) string {
	switch diff.Type {
	case DiffAdded:
		return fmt.Sprintf("Added: %s = %s", diff.Path, formatValue(diff.To))
	case DiffRemoved:
		return fmt.Sprintf("Removed: %s = %s", diff.Path, formatValue(diff.From))
	case DiffModified:
		return fmt.Sprintf("Modified: %s changed from %s to %s", diff.Path, formatValue(diff.From), formatValue(diff.To))
	default: // DiffOrderChanged
		return fmt.Sprintf("Order changed: %s", diff.Path)
	}
}

// --- CompactFormatter ---

// CompactFormatter renders differences in compact single-line-per-change format with color support.
// This was previously named HumanFormatter - preserved for backward compatibility with the compact output style.
type CompactFormatter struct{}

// FormatSingle renders a single difference in compact format.
func (f *CompactFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
	}

	var sb strings.Builder
	f.formatDiff(&sb, diff, opts)
	return sb.String()
}

// Format renders differences in compact single-line-per-change format.
func (f *CompactFormatter) Format(diffs []Difference, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
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

func (f *CompactFormatter) formatHeader(sb *strings.Builder, diffs []Difference, opts *FormatOptions) {
	var added, removed, modified int
	for _, d := range diffs {
		switch d.Type {
		case DiffAdded:
			added++
		case DiffRemoved:
			removed++
		case DiffModified, DiffOrderChanged:
			modified++
		}
	}

	if opts.Color {
		sb.WriteString(CompactColor(DiffModified))
	}
	fmt.Fprintf(sb, "Found %d difference(s)", len(diffs))
	if opts.Color {
		sb.WriteString(colorReset)
	}
	fmt.Fprintf(sb, " (%d added, %d removed, %d modified)\n\n", added, removed, modified)
}

func (f *CompactFormatter) formatDiff(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	path := diff.Path
	if opts.UseGoPatchStyle {
		path = convertToGoPatchPath(path)
	}

	var indicator string

	switch diff.Type {
	case DiffAdded:
		indicator = "+"
	case DiffRemoved:
		indicator = "-"
	case DiffModified:
		indicator = "±"
	case DiffOrderChanged:
		indicator = "⇆"
	}

	colorCode := CompactColor(diff.Type)

	// Apply color for the indicator
	if opts.Color {
		sb.WriteString(colorCode)
	}
	sb.WriteString(indicator)
	if opts.Color {
		sb.WriteString(colorReset)
	}

	sb.WriteString(" ")

	sb.WriteString(path)

	f.formatValuesInline(sb, diff, opts)

	sb.WriteString("\n")
}

func (f *CompactFormatter) formatValuesInline(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	switch diff.Type {
	case DiffModified:
		fromStr := formatValue(diff.From)
		toStr := formatValue(diff.To)

		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(CompactColor(DiffRemoved))
		}
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
		sb.WriteString(" → ")
		if opts.Color {
			sb.WriteString(CompactColor(DiffAdded))
		}
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffAdded:
		toStr := formatValue(diff.To)
		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(CompactColor(DiffAdded))
		}
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffRemoved:
		fromStr := formatValue(diff.From)
		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(CompactColor(DiffRemoved))
		}
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffOrderChanged:
		sb.WriteString(" (order changed)")
	}
}

// --- BriefFormatter ---

// BriefFormatter renders a concise summary of changes.
type BriefFormatter struct{}

// FormatSingle renders a single difference as a brief line.
func (f *BriefFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	switch diff.Type {
	case DiffAdded:
		return fmt.Sprintf("+ %s\n", diff.Path)
	case DiffRemoved:
		return fmt.Sprintf("- %s\n", diff.Path)
	case DiffModified:
		return fmt.Sprintf("± %s\n", diff.Path)
	default: // DiffOrderChanged
		return fmt.Sprintf("⇆ %s\n", diff.Path)
	}
}

// Format renders a brief summary of differences.
func (f *BriefFormatter) Format(diffs []Difference, _ *FormatOptions) string {
	if len(diffs) == 0 {
		return "no differences\n"
	}

	var added, removed, modified int
	for _, diff := range diffs {
		switch diff.Type {
		case DiffAdded:
			added++
		case DiffRemoved:
			removed++
		case DiffModified, DiffOrderChanged:
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
