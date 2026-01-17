package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestVersionFlag tests that the --version flag displays version information
func TestVersionFlag(t *testing.T) {
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "diffyml_test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("diffyml_test")

	// Test --version flag
	cmd = exec.Command("./diffyml_test", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run --version: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "diffyml version") {
		t.Errorf("Expected version output to contain 'diffyml version', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "commit:") {
		t.Errorf("Expected version output to contain 'commit:', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "built:") {
		t.Errorf("Expected version output to contain 'built:', got: %s", outputStr)
	}

	// Verify exit code is 0
	if cmd.ProcessState.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got: %d", cmd.ProcessState.ExitCode())
	}
}

// TestVersionFlagShortForm tests that the -V flag displays version information
func TestVersionFlagShortForm(t *testing.T) {
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "diffyml_test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("diffyml_test")

	// Test -V flag
	cmd = exec.Command("./diffyml_test", "-V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run -V: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "diffyml version") {
		t.Errorf("Expected version output to contain 'diffyml version', got: %s", outputStr)
	}
}

// TestVersionFlagWithLdflags tests that version information can be injected via ldflags
func TestVersionFlagWithLdflags(t *testing.T) {
	// Build the binary with version injection
	cmd := exec.Command("go", "build",
		"-ldflags", "-X main.version=1.2.3 -X main.commit=abc123def -X main.buildDate=2024-01-15",
		"-o", "diffyml_test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary with ldflags: %v", err)
	}
	defer os.Remove("diffyml_test")

	// Test --version flag
	cmd = exec.Command("./diffyml_test", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run --version: %v", err)
	}

	outputStr := string(output)
	expectedParts := []string{
		"diffyml version 1.2.3",
		"commit: abc123def",
		"built: 2024-01-15",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected version output to contain '%s', got: %s", part, outputStr)
		}
	}
}

// TestFormatVersion tests the formatVersion function
func TestFormatVersion(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test with custom values
	version = "1.0.0"
	commit = "abc123"
	buildDate = "2024-01-15"

	result := formatVersion()
	expected := "diffyml version 1.0.0 (commit: abc123, built: 2024-01-15)\n"

	if result != expected {
		t.Errorf("Expected: %q, got: %q", expected, result)
	}
}

// TestFormatVersionDefaults tests the formatVersion function with default values
func TestFormatVersionDefaults(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test with default values
	version = "dev"
	commit = "none"
	buildDate = "unknown"

	result := formatVersion()
	expected := "diffyml version dev (commit: none, built: unknown)\n"

	if result != expected {
		t.Errorf("Expected: %q, got: %q", expected, result)
	}
}
