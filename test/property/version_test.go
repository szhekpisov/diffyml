package property

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildTestBinary compiles diffyml into a per-test temp directory with the
// given ldflags and returns the absolute binary path. Using t.TempDir() keeps
// concurrent test workers from trampling each other's output files.
func buildTestBinary(t *testing.T, name, ldflags string) string {
	t.Helper()
	repoRoot, err := getRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	binaryPath := filepath.Join(t.TempDir(), name)
	args := []string{"build"}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, "-o", binaryPath)

	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary for testing: %v", err)
	}
	return binaryPath
}

// TestProperty23_VersionFlagFunctionality tests that running the binary with
// the --version flag outputs version information and exits successfully.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagFunctionality(t *testing.T) {
	binaryPath := buildTestBinary(t, "diffyml_test", "")

	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version flag failed: %v", err)
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "version") {
		t.Error("output missing 'version' keyword")
	}
	if !strings.Contains(outputStr, "diffyml") {
		t.Error("output missing 'diffyml' name")
	}
	if !strings.Contains(outputStr, "commit:") {
		t.Error("output missing commit info")
	}
	if !strings.Contains(outputStr, "built:") {
		t.Error("output missing build date info")
	}
	if !cmd.ProcessState.Success() {
		t.Error("command did not exit successfully")
	}
}

// TestProperty23_VersionFlagVariations tests various version flag formats
// to ensure all common variations work correctly.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagVariations(t *testing.T) {
	binaryPath := buildTestBinary(t, "diffyml_test_variations", "")

	versionFlags := []string{"--version", "-version", "-V"}

	for _, flag := range versionFlags {
		t.Run(flag, func(t *testing.T) {
			cmd := exec.Command(binaryPath, flag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("flag %s failed: %v", flag, err)
			}

			outputStr := string(output)

			if !strings.Contains(outputStr, "diffyml version") {
				t.Error("output missing 'diffyml version'")
			}
			if !strings.Contains(outputStr, "commit:") {
				t.Error("output missing commit info")
			}
			if !strings.Contains(outputStr, "built:") {
				t.Error("output missing build date info")
			}
			if !cmd.ProcessState.Success() {
				t.Error("command did not exit successfully")
			}
		})
	}
}

// TestProperty23_VersionFlagWithLdflags tests that version information can be
// injected at build time using ldflags.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagWithLdflags(t *testing.T) {
	testVersion := "1.2.3"
	testCommit := "abc123def"
	testBuildDate := "2024-01-15T10:30:00Z"

	ldflags := "-X main.version=" + testVersion +
		" -X main.commit=" + testCommit +
		" -X main.buildDate=" + testBuildDate

	binaryPath := buildTestBinary(t, "diffyml_test_ldflags", ldflags)

	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version flag failed: %v", err)
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, testVersion) {
		t.Errorf("output missing injected version %q", testVersion)
	}
	if !strings.Contains(outputStr, testCommit) {
		t.Errorf("output missing injected commit %q", testCommit)
	}
	if !strings.Contains(outputStr, testBuildDate) {
		t.Errorf("output missing injected build date %q", testBuildDate)
	}
	if !cmd.ProcessState.Success() {
		t.Error("command did not exit successfully")
	}
}

// TestProperty23_VersionFlagOutputFormat tests that the version output
// follows the expected format.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagOutputFormat(t *testing.T) {
	binaryPath := buildTestBinary(t, "diffyml_test_format", "")

	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version flag failed: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))

	// Expected format: "diffyml version X.Y.Z (commit: abc123, built: date)"
	if !strings.HasPrefix(outputStr, "diffyml version") {
		t.Error("output does not start with 'diffyml version'")
	}
	if !strings.Contains(outputStr, "(") || !strings.Contains(outputStr, ")") {
		t.Error("output missing parentheses")
	}
	if !strings.Contains(outputStr, "commit:") {
		t.Error("output missing 'commit:' separator")
	}
	if !strings.Contains(outputStr, "built:") {
		t.Error("output missing 'built:' separator")
	}
	if !strings.Contains(outputStr, ",") {
		t.Error("output missing comma separator")
	}
	lines := strings.Split(outputStr, "\n")
	if len(lines) != 1 {
		t.Errorf("expected single line output, got %d lines", len(lines))
	}
}

// TestProperty23_VersionFlagExitsWithoutProcessing tests that the version flag
// causes the program to exit without processing other arguments.
// **Validates: Requirements 10.5**
func TestProperty23_VersionFlagExitsWithoutProcessing(t *testing.T) {
	binaryPath := buildTestBinary(t, "diffyml_test_exit", "")

	cmd := exec.Command(binaryPath, "--version", "invalid_file.yaml", "another_invalid.yaml")
	output, err := cmd.CombinedOutput()
	// Should exit successfully (exit code 0) despite invalid arguments
	if err != nil {
		t.Fatalf("--version with extra args failed: %v", err)
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "diffyml version") {
		t.Error("output missing version info")
	}
	if strings.Contains(outputStr, "Error") || strings.Contains(outputStr, "error") || strings.Contains(outputStr, "invalid") {
		t.Error("output contains error messages despite --version flag")
	}
	if !cmd.ProcessState.Success() {
		t.Error("command did not exit successfully")
	}
}
