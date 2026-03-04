// gitlab_formatter.go - GitLab CI Code Quality output.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
)

type GitLabFormatter = format.GitLabFormatter

// Wrapper functions for unexported names used in tests.
func gitLabSeverity(dt DiffType) string                     { return format.GitLabSeverity(dt) }
func gitLabCheckName(dt DiffType) string                    { return format.GitLabCheckName(dt) }
func gitLabFingerprint(filePath, description string) string { return format.GitLabFingerprint(filePath, description) }
