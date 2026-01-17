package property

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty23_VersionFlagFunctionality tests that running the binary with
// the --version flag outputs version information and exits successfully.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagFunctionality(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// First, ensure the binary is built
	buildCmd := exec.Command("go", "build", "-o", "diffyml_test")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary for testing: %v", err)
	}
	defer os.Remove("diffyml_test") // Clean up after test

	properties.Property("binary with --version flag must output version info and exit successfully", prop.ForAll(
		func(dummyInput int) bool {
			// Test --version flag
			cmd := exec.Command("./diffyml_test", "--version")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return false
			}

			outputStr := string(output)

			// Verify output contains version information
			hasVersionKeyword := strings.Contains(outputStr, "version")
			hasDiffymlName := strings.Contains(outputStr, "diffyml")

			// Verify output contains commit information
			hasCommitInfo := strings.Contains(outputStr, "commit:")

			// Verify output contains build date information
			hasBuildInfo := strings.Contains(outputStr, "built:")

			// Verify the command exited successfully (exit code 0)
			exitedSuccessfully := cmd.ProcessState.Success()

			return hasVersionKeyword && hasDiffymlName && hasCommitInfo &&
				hasBuildInfo && exitedSuccessfully
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty23_VersionFlagVariations tests various version flag formats
// to ensure all common variations work correctly.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagVariations(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// Build the binary once for all tests
	buildCmd := exec.Command("go", "build", "-o", "diffyml_test_variations")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary for testing: %v", err)
	}
	defer os.Remove("diffyml_test_variations")

	// Test different version flag variations
	versionFlags := []string{"--version", "-version", "-V"}

	properties.Property("all version flag variations must work correctly", prop.ForAll(
		func(flagIndex int) bool {
			// Use the flag index to select which flag to test
			flag := versionFlags[flagIndex%len(versionFlags)]

			cmd := exec.Command("./diffyml_test_variations", flag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return false
			}

			outputStr := string(output)

			// Verify output format
			hasVersionInfo := strings.Contains(outputStr, "diffyml version")
			hasCommit := strings.Contains(outputStr, "commit:")
			hasBuilt := strings.Contains(outputStr, "built:")

			// Verify successful exit
			exitedSuccessfully := cmd.ProcessState.Success()

			return hasVersionInfo && hasCommit && hasBuilt && exitedSuccessfully
		},
		gen.IntRange(0, 99), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty23_VersionFlagWithLdflags tests that version information can be
// injected at build time using ldflags.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagWithLdflags(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	properties.Property("version flag must display injected version information", prop.ForAll(
		func(dummyInput int) bool {
			// Build with custom version information
			testVersion := "1.2.3"
			testCommit := "abc123def"
			testBuildDate := "2024-01-15T10:30:00Z"

			ldflags := "-X main.version=" + testVersion +
				" -X main.commit=" + testCommit +
				" -X main.buildDate=" + testBuildDate

			buildCmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", "diffyml_test_ldflags")
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				return false
			}
			defer os.Remove("diffyml_test_ldflags")

			// Run with --version flag
			cmd := exec.Command("./diffyml_test_ldflags", "--version")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return false
			}

			outputStr := string(output)

			// Verify the injected values appear in the output
			hasInjectedVersion := strings.Contains(outputStr, testVersion)
			hasInjectedCommit := strings.Contains(outputStr, testCommit)
			hasInjectedBuildDate := strings.Contains(outputStr, testBuildDate)

			// Verify successful exit
			exitedSuccessfully := cmd.ProcessState.Success()

			return hasInjectedVersion && hasInjectedCommit &&
				hasInjectedBuildDate && exitedSuccessfully
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty23_VersionFlagOutputFormat tests that the version output
// follows the expected format.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagOutputFormat(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "diffyml_test_format")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary for testing: %v", err)
	}
	defer os.Remove("diffyml_test_format")

	properties.Property("version output must follow expected format", prop.ForAll(
		func(dummyInput int) bool {
			cmd := exec.Command("./diffyml_test_format", "--version")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return false
			}

			outputStr := strings.TrimSpace(string(output))

			// Expected format: "diffyml version X.Y.Z (commit: abc123, built: date)"
			// Check for the basic structure
			hasProperFormat := strings.HasPrefix(outputStr, "diffyml version")

			// Check for parentheses containing commit and built info
			hasParentheses := strings.Contains(outputStr, "(") && strings.Contains(outputStr, ")")

			// Check for proper separators
			hasCommitSeparator := strings.Contains(outputStr, "commit:")
			hasBuiltSeparator := strings.Contains(outputStr, "built:")
			hasCommaSeparator := strings.Contains(outputStr, ",")

			// Verify output is a single line (no extra newlines in the middle)
			lines := strings.Split(outputStr, "\n")
			isSingleLine := len(lines) == 1

			return hasProperFormat && hasParentheses && hasCommitSeparator &&
				hasBuiltSeparator && hasCommaSeparator && isSingleLine
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty23_VersionFlagExitsWithoutProcessing tests that the version flag
// causes the program to exit without processing other arguments.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagExitsWithoutProcessing(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "diffyml_test_exit")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary for testing: %v", err)
	}
	defer os.Remove("diffyml_test_exit")

	properties.Property("version flag must cause immediate exit without processing other args", prop.ForAll(
		func(dummyInput int) bool {
			// Run with --version and other arguments that would normally cause errors
			cmd := exec.Command("./diffyml_test_exit", "--version", "invalid_file.yaml", "another_invalid.yaml")
			output, err := cmd.CombinedOutput()

			// Should exit successfully (exit code 0) despite invalid arguments
			if err != nil {
				return false
			}

			outputStr := string(output)

			// Should only show version info, not error messages about invalid files
			hasVersionInfo := strings.Contains(outputStr, "diffyml version")
			hasNoErrorMessages := !strings.Contains(outputStr, "Error") &&
				!strings.Contains(outputStr, "error") &&
				!strings.Contains(outputStr, "invalid")

			// Verify successful exit
			exitedSuccessfully := cmd.ProcessState.Success()

			return hasVersionInfo && hasNoErrorMessages && exitedSuccessfully
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
