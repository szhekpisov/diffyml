package property

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// TestProperty1_SemanticVersionTagFormat tests that all release tags in the
// repository follow the semantic versioning pattern vMAJOR.MINOR.PATCH.
// **Validates: Requirements 1.1**
func TestProperty1_SemanticVersionTagFormat(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Semantic version pattern: vMAJOR.MINOR.PATCH (e.g., v1.0.0, v2.1.3)
	semverPattern := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

	// Get all tags from git
	cmd := exec.Command("git", "tag", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If git fails, it might not be a git repository
		t.Skip("not a git repository")
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")

	// If no tags exist, the test passes (no invalid tags)
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
		return
	}

	// Check each tag
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		// Only check version tags (those starting with 'v')
		if strings.HasPrefix(tag, "v") {
			if !semverPattern.MatchString(tag) {
				t.Errorf("tag %q does not match semantic versioning format", tag)
			}
		}
	}
}

// TestProperty1_SemanticVersionTagFormat_ValidPatterns tests that generated
// semantic version patterns are recognized correctly.
// **Validates: Requirements 1.1**
func TestProperty1_SemanticVersionTagFormat_ValidPatterns(t *testing.T) {
	// Semantic version pattern
	semverPattern := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

	cases := []struct{ major, minor, patch int }{
		{0, 0, 0},
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
		{1, 2, 3},
		{10, 20, 30},
		{100, 0, 99},
		{0, 99, 100},
	}

	for _, tc := range cases {
		tag := fmt.Sprintf("v%d.%d.%d", tc.major, tc.minor, tc.patch)
		t.Run(tag, func(t *testing.T) {
			if !semverPattern.MatchString(tag) {
				t.Errorf("generated semantic version %q does not match pattern", tag)
			}
		})
	}
}

// TestProperty1_SemanticVersionTagFormat_InvalidPatterns tests that invalid
// version patterns are correctly rejected.
// **Validates: Requirements 1.1**
func TestProperty1_SemanticVersionTagFormat_InvalidPatterns(t *testing.T) {
	// Semantic version pattern
	semverPattern := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

	// Invalid patterns that should NOT match
	invalidPatterns := []string{
		"1.0.0",        // Missing 'v' prefix
		"v1.0",         // Missing patch version
		"v1",           // Missing minor and patch
		"v1.0.0-alpha", // Pre-release suffix (not allowed for stable)
		"v1.0.0.0",     // Too many version parts
		"va.b.c",       // Non-numeric version
		"V1.0.0",       // Uppercase V
		"version1.0.0", // Wrong prefix
		"v1.0.0-rc1",   // Release candidate suffix
		"v1.0.0+build", // Build metadata suffix
	}

	for _, pattern := range invalidPatterns {
		t.Run(pattern, func(t *testing.T) {
			if semverPattern.MatchString(pattern) {
				t.Errorf("invalid pattern %q should not match semantic versioning format", pattern)
			}
		})
	}
}
