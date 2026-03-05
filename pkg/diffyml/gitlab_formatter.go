// gitlab_formatter.go - GitLab CI Code Quality output.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
)

type GitLabFormatter = format.GitLabFormatter

func gitLabFingerprint(filePath, description string) string {
	return format.GitLabFingerprint(filePath, description)
}
