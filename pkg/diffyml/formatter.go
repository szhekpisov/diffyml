// formatter.go - Output formatting for differences.
//
// Implements 6 output styles: compact, brief, github, gitlab, gitea, detailed.
// Key types: Formatter interface, FormatOptions.
// Each formatter implements Format(diffs, opts) string.
package diffyml

import (
	"fmt"
	"strings"
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
	// Width sets fixed terminal width (0 for auto-detect).
	Width int
	// OmitHeader skips the summary header in output.
	OmitHeader bool
	// NoTableStyle displays one text block per row instead of side-by-side.
	NoTableStyle bool
	// UseGoPatchStyle uses Go-Patch style paths in output.
	UseGoPatchStyle bool
	// ContextLines is the number of context lines for multi-line values.
	ContextLines int
	// MinorChangeThreshold is the threshold for minor change detection.
	MinorChangeThreshold float64
}

// DefaultFormatOptions returns FormatOptions with default values.
func DefaultFormatOptions() *FormatOptions {
	return &FormatOptions{
		Color:                false,
		TrueColor:            false,
		Width:                0, // Auto-detect
		OmitHeader:           false,
		NoTableStyle:         false,
		UseGoPatchStyle:      false,
		ContextLines:         4,
		MinorChangeThreshold: 0.1,
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
		sb.WriteString(colorYellow)
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
	var colorCode string

	switch diff.Type {
	case DiffAdded:
		indicator = "+"
		colorCode = colorGreen
	case DiffRemoved:
		indicator = "-"
		colorCode = colorRed
	case DiffModified:
		indicator = "±"
		colorCode = colorYellow
	case DiffOrderChanged:
		indicator = "⇆"
		colorCode = colorYellow
	}

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

	// Format values based on table style preference
	if opts.NoTableStyle {
		f.formatValuesSingleRow(sb, diff, opts)
	} else {
		f.formatValuesInline(sb, diff, opts)
	}

	sb.WriteString("\n")
}

func (f *CompactFormatter) formatValuesInline(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	switch diff.Type {
	case DiffModified:
		fromStr := formatValue(diff.From, opts)
		toStr := formatValue(diff.To, opts)

		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(colorRed)
		}
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
		sb.WriteString(" → ")
		if opts.Color {
			sb.WriteString(colorGreen)
		}
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffAdded:
		toStr := formatValue(diff.To, opts)
		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(colorGreen)
		}
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffRemoved:
		fromStr := formatValue(diff.From, opts)
		sb.WriteString(" : ")
		if opts.Color {
			sb.WriteString(colorRed)
		}
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffOrderChanged:
		sb.WriteString(" (order changed)")
	}
}

func (f *CompactFormatter) formatValuesSingleRow(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	// Single-row display mode - one block per change
	sb.WriteString("\n")

	switch diff.Type {
	case DiffModified:
		fromStr := formatValue(diff.From, opts)
		toStr := formatValue(diff.To, opts)

		if opts.Color {
			sb.WriteString(colorRed)
		}
		sb.WriteString("  - ")
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
		sb.WriteString("\n")

		if opts.Color {
			sb.WriteString(colorGreen)
		}
		sb.WriteString("  + ")
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffAdded:
		toStr := formatValue(diff.To, opts)
		if opts.Color {
			sb.WriteString(colorGreen)
		}
		sb.WriteString("  + ")
		sb.WriteString(toStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	case DiffRemoved:
		fromStr := formatValue(diff.From, opts)
		if opts.Color {
			sb.WriteString(colorRed)
		}
		sb.WriteString("  - ")
		sb.WriteString(fromStr)
		if opts.Color {
			sb.WriteString(colorReset)
		}
	}
}

// formatValue converts a value to string.
// Shows full values without truncation.
func formatValue(val interface{}, opts *FormatOptions) string {
	if val == nil {
		return "<nil>"
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
	case DiffOrderChanged:
		return fmt.Sprintf("⇆ %s\n", diff.Path)
	default:
		return fmt.Sprintf("? %s\n", diff.Path)
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

// GitHubFormatter renders output compatible with GitHub Actions workflow commands.
type GitHubFormatter struct{}

// FormatSingle renders a single difference in GitHub Actions format.
func (f *GitHubFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	var msg string
	switch diff.Type {
	case DiffAdded:
		msg = fmt.Sprintf("Added: %s = %v", diff.Path, diff.To)
	case DiffRemoved:
		msg = fmt.Sprintf("Removed: %s = %v", diff.Path, diff.From)
	case DiffModified:
		msg = fmt.Sprintf("Modified: %s changed from %v to %v", diff.Path, diff.From, diff.To)
	case DiffOrderChanged:
		msg = fmt.Sprintf("Order changed: %s", diff.Path)
	}
	return fmt.Sprintf("::warning ::%s\n", msg)
}

// Format renders differences in GitHub Actions format.
func (f *GitHubFormatter) Format(diffs []Difference, _ *FormatOptions) string {
	if len(diffs) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, diff := range diffs {
		var msg string
		switch diff.Type {
		case DiffAdded:
			msg = fmt.Sprintf("Added: %s = %v", diff.Path, diff.To)
		case DiffRemoved:
			msg = fmt.Sprintf("Removed: %s = %v", diff.Path, diff.From)
		case DiffModified:
			msg = fmt.Sprintf("Modified: %s changed from %v to %v", diff.Path, diff.From, diff.To)
		case DiffOrderChanged:
			msg = fmt.Sprintf("Order changed: %s", diff.Path)
		}
		fmt.Fprintf(&sb, "::warning ::%s\n", msg)
	}
	return sb.String()
}

// GitLabFormatter renders output compatible with GitLab CI Code Quality format.
type GitLabFormatter struct{}

// FormatSingle renders a single difference in GitLab CI JSON format (without array wrapper).
func (f *GitLabFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	var description string
	switch diff.Type {
	case DiffAdded:
		description = fmt.Sprintf("Added: %s = %v", diff.Path, diff.To)
	case DiffRemoved:
		description = fmt.Sprintf("Removed: %s = %v", diff.Path, diff.From)
	case DiffModified:
		description = fmt.Sprintf("Modified: %s changed from %v to %v", diff.Path, diff.From, diff.To)
	case DiffOrderChanged:
		description = fmt.Sprintf("Order changed: %s", diff.Path)
	}
	return fmt.Sprintf(`{"description": %q, "fingerprint": %q, "severity": "minor", "location": {"path": %q}}`+"\n",
		description, diff.Path, diff.Path)
}

// Format renders differences in GitLab CI format.
func (f *GitLabFormatter) Format(diffs []Difference, _ *FormatOptions) string {
	if len(diffs) == 0 {
		return "[]\n"
	}

	var sb strings.Builder
	sb.WriteString("[\n")

	for i, diff := range diffs {
		var description string
		switch diff.Type {
		case DiffAdded:
			description = fmt.Sprintf("Added: %s = %v", diff.Path, diff.To)
		case DiffRemoved:
			description = fmt.Sprintf("Removed: %s = %v", diff.Path, diff.From)
		case DiffModified:
			description = fmt.Sprintf("Modified: %s changed from %v to %v", diff.Path, diff.From, diff.To)
		case DiffOrderChanged:
			description = fmt.Sprintf("Order changed: %s", diff.Path)
		}

		fmt.Fprintf(&sb, `  {"description": %q, "fingerprint": %q, "severity": "minor", "location": {"path": %q}}`,
			description, diff.Path, diff.Path)

		if i < len(diffs)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("]\n")
	return sb.String()
}

// GiteaFormatter renders output compatible with Gitea CI/CD.
// Uses GitHub Actions compatible format.
type GiteaFormatter struct{}

// FormatSingle renders a single difference in Gitea CI format (GitHub Actions compatible).
func (f *GiteaFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	// Gitea uses GitHub Actions compatible format
	gh := &GitHubFormatter{}
	return gh.FormatSingle(diff, opts)
}

// Format renders differences in Gitea CI format (GitHub Actions compatible).
func (f *GiteaFormatter) Format(diffs []Difference, opts *FormatOptions) string {
	// Gitea uses GitHub Actions compatible format
	gh := &GitHubFormatter{}
	return gh.Format(diffs, opts)
}
