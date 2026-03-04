// gitea_formatter.go - Gitea CI/CD output.
//
// Delegates to GitHubFormatter since Gitea uses GitHub Actions compatible format.
// Note: Gitea Actions silently ignores workflow command annotations
// (see gitea/gitea#23722), so annotations may not appear in the Gitea UI.
// The output is still valid for log parsing.
package format

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// GiteaFormatter renders output compatible with Gitea CI/CD.
type GiteaFormatter struct{}

// FormatSingle renders a single difference in Gitea CI format (GitHub Actions compatible).
func (f *GiteaFormatter) FormatSingle(diff types.Difference, opts *types.FormatOptions) string {
	gh := &GitHubFormatter{}
	return gh.FormatSingle(diff, opts)
}

// Format renders differences in Gitea CI format (GitHub Actions compatible).
func (f *GiteaFormatter) Format(diffs []types.Difference, opts *types.FormatOptions) string {
	gh := &GitHubFormatter{}
	return gh.Format(diffs, opts)
}

// FormatAll delegates to GitHubFormatter for directory mode support.
func (f *GiteaFormatter) FormatAll(groups []types.DiffGroup, opts *types.FormatOptions) string {
	gh := &GitHubFormatter{}
	return gh.FormatAll(groups, opts)
}
