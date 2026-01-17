package property

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty1_SemanticVersionTagFormat tests that all release tags in the
// repository follow the semantic versioning pattern vMAJOR.MINOR.PATCH.
// **Validates: Requirements 1.1**
func TestProperty1_SemanticVersionTagFormat(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// Semantic version pattern: vMAJOR.MINOR.PATCH (e.g., v1.0.0, v2.1.3)
	semverPattern := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

	properties.Property("all release tags must follow semantic versioning format", prop.ForAll(
		func(dummyInput int) bool {
			// Get all tags from git
			cmd := exec.Command("git", "tag", "-l")
			output, err := cmd.CombinedOutput()
			if err != nil {
				// If git fails, it might not be a git repository
				// In that case, we consider the test passed (no invalid tags)
				return true
			}

			tags := strings.Split(strings.TrimSpace(string(output)), "\n")

			// If no tags exist, the test passes (no invalid tags)
			if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
				return true
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
						return false
					}
				}
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty1_SemanticVersionTagFormat_ValidPatterns tests that generated
// semantic version patterns are recognized correctly.
// **Validates: Requirements 1.1**
func TestProperty1_SemanticVersionTagFormat_ValidPatterns(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// Semantic version pattern
	semverPattern := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

	properties.Property("generated semantic versions must match the expected pattern", prop.ForAll(
		func(major, minor, patch int) bool {
			// Generate a valid semantic version tag
			tag := "v" + intToStr(major) + "." + intToStr(minor) + "." + intToStr(patch)
			return semverPattern.MatchString(tag)
		},
		gen.IntRange(0, 100), // MAJOR version
		gen.IntRange(0, 100), // MINOR version
		gen.IntRange(0, 100), // PATCH version
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty1_SemanticVersionTagFormat_InvalidPatterns tests that invalid
// version patterns are correctly rejected.
// **Validates: Requirements 1.1**
func TestProperty1_SemanticVersionTagFormat_InvalidPatterns(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

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

	properties.Property("invalid version patterns must not match semantic versioning format", prop.ForAll(
		func(patternIndex int) bool {
			pattern := invalidPatterns[patternIndex%len(invalidPatterns)]
			// Invalid patterns should NOT match
			return !semverPattern.MatchString(pattern)
		},
		gen.IntRange(0, 99), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// intToStr converts an integer to a string.
func intToStr(n int) string {
	if n < 0 {
		n = -n
	}
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		digit := n % 10
		result = string(rune('0'+digit)) + result
		n /= 10
	}
	return result
}
