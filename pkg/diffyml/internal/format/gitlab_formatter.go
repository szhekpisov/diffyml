// gitlab_formatter.go - GitLab CI Code Quality output.
//
// Renders differences as GitLab Code Quality JSON with check_name,
// location.lines.begin, unique SHA-256 fingerprints, and severity per diff type.
package format

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// GitLabFormatter renders output compatible with GitLab CI Code Quality format.
// Complies with the GitLab Code Quality specification: includes check_name,
// location.lines.begin, unique SHA-256 fingerprints, and severity per diff type.
type GitLabFormatter struct{}

// GitLabSeverity returns the Code Quality severity for a diff type.
func GitLabSeverity(dt types.DiffType) string {
	switch dt {
	case types.DiffAdded:
		return "info"
	case types.DiffRemoved, types.DiffModified:
		return "major"
	default: // DiffOrderChanged
		return "minor"
	}
}

// GitLabCheckName returns the check_name for a diff type.
func GitLabCheckName(dt types.DiffType) string {
	switch dt {
	case types.DiffAdded:
		return "diffyml/added"
	case types.DiffRemoved:
		return "diffyml/removed"
	case types.DiffModified:
		return "diffyml/modified"
	default: // DiffOrderChanged
		return "diffyml/order-changed"
	}
}

// GitLabFingerprint returns a unique SHA-256 fingerprint.
// When filePath is non-empty, hashes filePath + ":" + description.
// When filePath is empty, hashes only description (backward compat).
func GitLabFingerprint(filePath, description string) string {
	input := description
	if filePath != "" {
		input = filePath + ":" + description
	}
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

// FormatSingle renders a single difference in GitLab CI JSON format (without array wrapper).
func (f *GitLabFormatter) FormatSingle(diff types.Difference, opts *types.FormatOptions) string {
	desc := DiffDescription(diff)
	return fmt.Sprintf(
		`{"description": %q, "check_name": %q, "fingerprint": %q, "severity": %q, "location": {"path": %q, "lines": {"begin": 1}}}`+"\n",
		desc, GitLabCheckName(diff.Type), GitLabFingerprint("", desc), GitLabSeverity(diff.Type), diff.Path)
}

// Format renders differences in GitLab CI format.
func (f *GitLabFormatter) Format(diffs []types.Difference, opts *types.FormatOptions) string {
	if len(diffs) == 0 {
		return "[]\n"
	}

	if opts == nil {
		opts = types.DefaultFormatOptions()
	}

	var sb strings.Builder
	sb.WriteString("[\n")

	for i, diff := range diffs {
		desc := DiffDescription(diff)
		locationPath := opts.FilePath
		if locationPath == "" {
			locationPath = diff.Path
		}
		fmt.Fprintf(&sb,
			`  {"description": %q, "check_name": %q, "fingerprint": %q, "severity": %q, "location": {"path": %q, "lines": {"begin": 1}}}`,
			desc, GitLabCheckName(diff.Type), GitLabFingerprint(opts.FilePath, desc), GitLabSeverity(diff.Type), locationPath)

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
func (f *GitLabFormatter) FormatAll(groups []types.DiffGroup, _ *types.FormatOptions) string {
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
			baseDesc := DiffDescription(diff)
			displayDesc := fmt.Sprintf("[%s] %s", group.FilePath, baseDesc)
			fmt.Fprintf(&sb,
				`  {"description": %q, "check_name": %q, "fingerprint": %q, "severity": %q, "location": {"path": %q, "lines": {"begin": 1}}}`,
				displayDesc, GitLabCheckName(diff.Type), GitLabFingerprint(group.FilePath, baseDesc), GitLabSeverity(diff.Type), group.FilePath)

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
