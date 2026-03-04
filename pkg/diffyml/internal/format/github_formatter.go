// github_formatter.go - GitHub Actions workflow command output.
//
// Renders differences as GitHub Actions annotations using differentiated
// commands (notice/warning/error) with per-type truncation at 10 annotations.
package format

import (
	"fmt"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// GitHubFormatter renders output compatible with GitHub Actions workflow commands.
// Uses differentiated commands (notice/warning/error) and title parameter per diff type.
type GitHubFormatter struct{}

// gitHubCommand returns the workflow command and title for a diff type.
func gitHubCommand(dt types.DiffType) (command, title string) {
	switch dt {
	case types.DiffAdded:
		return "notice", "YAML Added"
	case types.DiffRemoved:
		return "error", "YAML Removed"
	case types.DiffModified:
		return "warning", "YAML Modified"
	default: // DiffOrderChanged
		return "notice", "YAML Order Changed"
	}
}

// gitHubAnnotationLimit is the maximum number of annotations per command type
// per step, as enforced by GitHub Actions.
const gitHubAnnotationLimit = 10

// FormatSingle renders a single difference in GitHub Actions format.
func (f *GitHubFormatter) FormatSingle(diff types.Difference, opts *types.FormatOptions) string {
	cmd, title := gitHubCommand(diff.Type)
	msg := DiffDescription(diff)
	return fmt.Sprintf("::%s title=%s::%s\n", cmd, title, msg)
}

// Format renders differences in GitHub Actions format.
// When opts.FilePath is non-empty: ::<cmd> file=<path>,title=<title>::<message>
// When opts.FilePath is empty:     ::<cmd> title=<title>::<message>
// Tracks per-type counts; truncates at 10 per command type.
// Appends summary annotation per truncated type.
func (f *GitHubFormatter) Format(diffs []types.Difference, opts *types.FormatOptions) string {
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

	gitHubFormatDiffs(diffs, filePath, &sb, counts, omitted)

	gitHubWriteSummaries(&sb, omitted)
	return sb.String()
}

// gitHubFormatDiffs writes GitHub Actions annotations for a slice of diffs,
// tracking per-command-type counts and omitting beyond the limit.
func gitHubFormatDiffs(diffs []types.Difference, filePath string, sb *strings.Builder, counts, omitted map[string]int) {
	for _, diff := range diffs {
		cmd, title := gitHubCommand(diff.Type)
		if counts[cmd] < gitHubAnnotationLimit {
			gitHubWriteCommand(sb, cmd, title, DiffDescription(diff), filePath)
			counts[cmd]++
		} else {
			omitted[cmd]++
		}
	}
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
func (f *GitHubFormatter) FormatAll(groups []types.DiffGroup, opts *types.FormatOptions) string {
	var sb strings.Builder
	counts := map[string]int{}
	omitted := map[string]int{}

	for _, group := range groups {
		gitHubFormatDiffs(group.Diffs, group.FilePath, &sb, counts, omitted)
	}

	gitHubWriteSummaries(&sb, omitted)
	return sb.String()
}
