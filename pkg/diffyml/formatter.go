// formatter.go - Output formatting for differences.
//
// Implements 7 output styles: compact, brief, github, gitlab, gitea, json, detailed.
// Key types: Formatter interface, FormatOptions.
// Each formatter implements Format(diffs, opts) string.
package diffyml

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
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

// DiffGroup pairs differences from a single file with its path.
// Used by StructuredFormatter.FormatAll for aggregated directory-mode output.
type DiffGroup struct {
	FilePath string       // Relative file path (e.g., "deploy.yaml")
	Diffs    []Difference // Differences for this file
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
var validFormatterNames = []string{"compact", "brief", "github", "gitlab", "gitea", "json", "detailed"}

// FormatterByName returns a formatter by name.
// Supported names: compact, brief, github, gitlab, gitea, json, detailed.
// Returns error for invalid formatter names with list of valid options.
func FormatterByName(name string) (Formatter, error) {
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
	case "json":
		return &JSONFormatter{}, nil
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

	sb.WriteString(colorStart(opts, colorYellow))
	fmt.Fprintf(sb, "Found %d difference(s)", len(diffs))
	sb.WriteString(colorEnd(opts))
	fmt.Fprintf(sb, " (%d removed, %d added, %d modified)\n\n", removed, added, modified)
}

func (f *CompactFormatter) formatDiff(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	var path string
	if opts.UseGoPatchStyle {
		path = diff.Path.GoPatchString()
	} else {
		path = diff.Path.String()
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
	sb.WriteString(colorStart(opts, colorCode))
	sb.WriteString(indicator)
	sb.WriteString(colorEnd(opts))

	sb.WriteString(" ")

	sb.WriteString(path)

	if diff.DocumentName != "" {
		sb.WriteString(" ")
		sb.WriteString(colorStart(opts, DocNameColorCode(opts.TrueColor)))
		fmt.Fprintf(sb, "(%s)", diff.DocumentName)
		sb.WriteString(colorEnd(opts))
	}

	f.formatValuesInline(sb, diff, opts)

	sb.WriteString("\n")
}

func (f *CompactFormatter) formatValuesInline(sb *strings.Builder, diff Difference, opts *FormatOptions) {
	switch diff.Type {
	case DiffModified:
		fromStr := formatValue(diff.From)
		toStr := formatValue(diff.To)

		sb.WriteString(" : ")
		sb.WriteString(colorStart(opts, colorRed))
		sb.WriteString(fromStr)
		sb.WriteString(colorEnd(opts))
		sb.WriteString(" → ")
		sb.WriteString(colorStart(opts, colorGreen))
		sb.WriteString(toStr)
		sb.WriteString(colorEnd(opts))
	case DiffAdded:
		toStr := formatValue(diff.To)
		sb.WriteString(" : ")
		sb.WriteString(colorStart(opts, colorGreen))
		sb.WriteString(toStr)
		sb.WriteString(colorEnd(opts))
	case DiffRemoved:
		fromStr := formatValue(diff.From)
		sb.WriteString(" : ")
		sb.WriteString(colorStart(opts, colorRed))
		sb.WriteString(fromStr)
		sb.WriteString(colorEnd(opts))
	case DiffOrderChanged:
		sb.WriteString(" (order changed)")
	}
}

// formatValue converts a value to string.
// Shows full values without truncation.
// Structured types (*OrderedMap, map[string]any, []any) are
// serialized to inline YAML instead of Go's default %v representation.
func formatValue(val any) string {
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
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", removed))
	}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", added))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}

	return strings.Join(parts, ", ") + "\n"
}

// GitHubFormatter renders output compatible with GitHub Actions workflow commands.
// Uses differentiated commands (notice/warning/error) and title parameter per diff type.
type GitHubFormatter struct{}

// gitHubCommand returns the workflow command and title for a diff type.
func gitHubCommand(dt DiffType) (command, title string) {
	switch dt {
	case DiffAdded:
		return "notice", "YAML Added"
	case DiffRemoved:
		return "error", "YAML Removed"
	case DiffModified:
		return "warning", "YAML Modified"
	default: // DiffOrderChanged
		return "notice", "YAML Order Changed"
	}
}

// diffDescription returns a human-readable description of a difference.
// Shared by GitHub, GitLab, and Gitea formatters.
func diffDescription(diff Difference) string {
	docSuffix := ""
	if diff.DocumentName != "" {
		docSuffix = fmt.Sprintf(" (%s)", diff.DocumentName)
	}
	switch diff.Type {
	case DiffAdded:
		return fmt.Sprintf("Added: %s%s = %s", diff.Path, docSuffix, formatValue(diff.To))
	case DiffRemoved:
		return fmt.Sprintf("Removed: %s%s = %s", diff.Path, docSuffix, formatValue(diff.From))
	case DiffModified:
		return fmt.Sprintf("Modified: %s%s changed from %s to %s", diff.Path, docSuffix, formatValue(diff.From), formatValue(diff.To))
	default: // DiffOrderChanged
		return fmt.Sprintf("Order changed: %s%s", diff.Path, docSuffix)
	}
}

// gitHubAnnotationLimit is the maximum number of annotations per command type
// per step, as enforced by GitHub Actions.
const gitHubAnnotationLimit = 10

// FormatSingle renders a single difference in GitHub Actions format.
func (f *GitHubFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	cmd, title := gitHubCommand(diff.Type)
	msg := diffDescription(diff)
	return fmt.Sprintf("::%s title=%s::%s\n", cmd, title, msg)
}

// Format renders differences in GitHub Actions format.
// When opts.FilePath is non-empty: ::<cmd> file=<path>,title=<title>::<message>
// When opts.FilePath is empty:     ::<cmd> title=<title>::<message>
// Tracks per-type counts; truncates at 10 per command type.
// Appends summary annotation per truncated type.
func (f *GitHubFormatter) Format(diffs []Difference, opts *FormatOptions) string {
	if len(diffs) == 0 {
		return ""
	}

	filePath := ""
	if opts != nil {
		filePath = opts.FilePath
	}

	var sb strings.Builder
	counts := map[string]int{}
	omitted := map[string]int{}

	for _, diff := range diffs {
		cmd, title := gitHubCommand(diff.Type)
		if counts[cmd] < gitHubAnnotationLimit {
			gitHubWriteCommand(&sb, cmd, title, diffDescription(diff), filePath)
			counts[cmd]++
		} else {
			omitted[cmd]++
		}
	}

	gitHubWriteSummaries(&sb, omitted)
	return sb.String()
}

// gitHubWriteCommand writes a single GitHub Actions workflow command to the builder.
func gitHubWriteCommand(sb *strings.Builder, cmd, title, msg, filePath string) {
	if filePath != "" {
		fmt.Fprintf(sb, "::%s file=%s,title=%s::%s\n", cmd, filePath, title, msg)
	} else {
		fmt.Fprintf(sb, "::%s title=%s::%s\n", cmd, title, msg)
	}
}

// gitHubWriteSummaries appends summary annotations for each truncated command type.
// Summary annotations do not include the file= parameter and do not count toward the limit.
func gitHubWriteSummaries(sb *strings.Builder, omitted map[string]int) {
	for _, cmd := range []string{"notice", "warning", "error"} {
		if n := omitted[cmd]; n > 0 {
			fmt.Fprintf(sb, "::%s title=diffyml::%d additional %s annotations omitted due to GitHub Actions limit\n", cmd, n, cmd)
		}
	}
}

// FormatAll produces GitHub Actions workflow commands for multiple file groups.
// Each diff uses file=<group.FilePath> for file-specific annotations.
// Annotation limits (10 per type) apply across ALL groups combined.
// Summary annotations omit the file= parameter.
// Returns empty string when all groups have zero diffs.
func (f *GitHubFormatter) FormatAll(groups []DiffGroup, opts *FormatOptions) string {
	var sb strings.Builder
	counts := map[string]int{}
	omitted := map[string]int{}

	for _, group := range groups {
		for _, diff := range group.Diffs {
			cmd, title := gitHubCommand(diff.Type)
			if counts[cmd] < gitHubAnnotationLimit {
				gitHubWriteCommand(&sb, cmd, title, diffDescription(diff), group.FilePath)
				counts[cmd]++
			} else {
				omitted[cmd]++
			}
		}
	}

	gitHubWriteSummaries(&sb, omitted)
	return sb.String()
}

// GitLabFormatter renders output compatible with GitLab CI Code Quality format.
// Complies with the GitLab Code Quality specification: includes check_name,
// location.lines.begin, unique SHA-256 fingerprints, and severity per diff type.
type GitLabFormatter struct{}

// gitLabSeverity returns the Code Quality severity for a diff type.
func gitLabSeverity(dt DiffType) string {
	switch dt {
	case DiffAdded:
		return "info"
	case DiffRemoved, DiffModified:
		return "major"
	default: // DiffOrderChanged
		return "minor"
	}
}

// gitLabCheckName returns the check_name for a diff type.
func gitLabCheckName(dt DiffType) string {
	switch dt {
	case DiffAdded:
		return "diffyml/added"
	case DiffRemoved:
		return "diffyml/removed"
	case DiffModified:
		return "diffyml/modified"
	default: // DiffOrderChanged
		return "diffyml/order-changed"
	}
}

// gitLabFingerprint returns a unique SHA-256 fingerprint.
// When filePath is non-empty, hashes filePath + ":" + description.
// When filePath is empty, hashes only description (backward compat).
func gitLabFingerprint(filePath, description string) string {
	input := description
	if filePath != "" {
		input = filePath + ":" + description
	}
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

// FormatSingle renders a single difference in GitLab CI JSON format (without array wrapper).
func (f *GitLabFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	desc := diffDescription(diff)
	return fmt.Sprintf(
		`{"description": %q, "check_name": %q, "fingerprint": %q, "severity": %q, "location": {"path": %q, "lines": {"begin": 1}}}`+"\n",
		desc, gitLabCheckName(diff.Type), gitLabFingerprint("", desc), gitLabSeverity(diff.Type), diff.Path)
}

// Format renders differences in GitLab CI format.
func (f *GitLabFormatter) Format(diffs []Difference, opts *FormatOptions) string {
	if len(diffs) == 0 {
		return "[]\n"
	}

	if opts == nil {
		opts = DefaultFormatOptions()
	}

	var sb strings.Builder
	sb.WriteString("[\n")

	for i, diff := range diffs {
		desc := diffDescription(diff)
		locationPath := opts.FilePath
		if locationPath == "" {
			locationPath = diff.Path.String()
		}
		fmt.Fprintf(&sb,
			`  {"description": %q, "check_name": %q, "fingerprint": %q, "severity": %q, "location": {"path": %q, "lines": {"begin": 1}}}`,
			desc, gitLabCheckName(diff.Type), gitLabFingerprint(opts.FilePath, desc), gitLabSeverity(diff.Type), locationPath)

		if i < len(diffs)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("]\n")
	return sb.String()
}

// FormatAll renders all diff groups as a single JSON array for directory mode.
// Implements StructuredFormatter interface.
func (f *GitLabFormatter) FormatAll(groups []DiffGroup, _ *FormatOptions) string {
	// Count total diffs for comma handling
	total := 0
	for _, g := range groups {
		total += len(g.Diffs)
	}

	if total == 0 {
		return "[]\n"
	}

	var sb strings.Builder
	sb.WriteString("[\n")

	idx := 0
	for _, group := range groups {
		for _, diff := range group.Diffs {
			baseDesc := diffDescription(diff)
			displayDesc := fmt.Sprintf("[%s] %s", group.FilePath, baseDesc)
			fmt.Fprintf(&sb,
				`  {"description": %q, "check_name": %q, "fingerprint": %q, "severity": %q, "location": {"path": %q, "lines": {"begin": 1}}}`,
				displayDesc, gitLabCheckName(diff.Type), gitLabFingerprint(group.FilePath, baseDesc), gitLabSeverity(diff.Type), group.FilePath)

			if idx < total-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
			idx++
		}
	}

	sb.WriteString("]\n")
	return sb.String()
}

// GiteaFormatter renders output compatible with Gitea CI/CD.
// Uses GitHub Actions compatible format. Note: Gitea Actions silently ignores
// workflow command annotations (see gitea/gitea#23722), so annotations may not
// appear in the Gitea UI. The output is still valid for log parsing.
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

// FormatAll delegates to GitHubFormatter for directory mode support.
func (f *GiteaFormatter) FormatAll(groups []DiffGroup, opts *FormatOptions) string {
	gh := &GitHubFormatter{}
	return gh.FormatAll(groups, opts)
}

// JSONFormatter renders differences as machine-readable JSON.
// Output is a JSON array of objects, one per difference, with typed values.
type JSONFormatter struct{}

// jsonDiff is the JSON representation of a single difference.
type jsonDiff struct {
	Path          string `json:"path"`
	Type          string `json:"type"`
	From          any    `json:"from"`
	To            any    `json:"to"`
	DocumentIndex int    `json:"document_index"`
	DocumentName  string `json:"document_name,omitempty"`
}

// jsonDirDiff extends jsonDiff with a file path for directory mode.
type jsonDirDiff struct {
	File string `json:"file"`
	jsonDiff
}

// jsonDiffTypeName returns the string name for a DiffType.
func jsonDiffTypeName(dt DiffType) string {
	switch dt {
	case DiffAdded:
		return "added"
	case DiffRemoved:
		return "removed"
	case DiffModified:
		return "modified"
	default: // DiffOrderChanged
		return "order_changed"
	}
}

// jsonPrepareValue converts a Difference value to a JSON-serializable form.
// *OrderedMap is converted to map[string]any since JSON has no ordered-map concept.
// float64 Inf/NaN values are converted to strings since encoding/json cannot marshal them.
func jsonPrepareValue(val any) any {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case *OrderedMap:
		return orderedMapToPlain(v)
	case float64:
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return fmt.Sprintf("%v", v)
		}
		return v
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = jsonPrepareValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = jsonPrepareValue(item)
		}
		return out
	default:
		return val
	}
}

// orderedMapToPlain recursively converts an OrderedMap to a plain map[string]any.
func orderedMapToPlain(om *OrderedMap) map[string]any {
	out := make(map[string]any, len(om.Keys))
	for _, key := range om.Keys {
		out[key] = jsonPrepareValue(om.Values[key])
	}
	return out
}

// buildJSONDiff converts a single Difference into a jsonDiff struct.
func buildJSONDiff(diff Difference, opts *FormatOptions) jsonDiff {
	path := diff.Path.String()
	if opts.UseGoPatchStyle {
		path = diff.Path.GoPatchString()
	}
	return jsonDiff{
		Path:          path,
		Type:          jsonDiffTypeName(diff.Type),
		From:          jsonPrepareValue(diff.From),
		To:            jsonPrepareValue(diff.To),
		DocumentIndex: diff.DocumentIndex,
		DocumentName:  diff.DocumentName,
	}
}

// FormatSingle renders a single difference as a JSON object (without array wrapper).
func (f *JSONFormatter) FormatSingle(diff Difference, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
	}
	d := buildJSONDiff(diff, opts)
	out, err := json.Marshal(d)
	if err != nil {
		return "{}\n"
	}
	return string(out) + "\n"
}

// jsonMarshalIndent marshals v as indented JSON, returning "[]" on error.
func jsonMarshalIndent(v any) string {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "[]\n"
	}
	return string(out) + "\n"
}

// Format renders differences as a JSON array.
func (f *JSONFormatter) Format(diffs []Difference, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
	}

	items := make([]jsonDiff, len(diffs))
	for i, diff := range diffs {
		items[i] = buildJSONDiff(diff, opts)
	}

	return jsonMarshalIndent(items)
}

// FormatAll renders all diff groups as a single JSON array for directory mode.
// Implements StructuredFormatter interface.
func (f *JSONFormatter) FormatAll(groups []DiffGroup, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
	}

	total := 0
	for _, g := range groups {
		total += len(g.Diffs)
	}

	items := make([]jsonDirDiff, 0, total)
	for _, group := range groups {
		for _, diff := range group.Diffs {
			d := buildJSONDiff(diff, opts)
			items = append(items, jsonDirDiff{
				File:     group.FilePath,
				jsonDiff: d,
			})
		}
	}

	return jsonMarshalIndent(items)
}
