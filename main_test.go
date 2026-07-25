package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// testBinaryName returns the build output name for the test binary, with the
// .exe extension Windows requires to recognize and execute it.
func testBinaryName() string {
	name := "diffyml_test"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// TestVersionFlag tests that the --version flag displays version information
func TestVersionFlag(t *testing.T) {
	// Build the binary for testing
	bin := testBinaryName()
	cmd := exec.Command("go", "build", "-o", bin)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove(bin)

	// Test --version flag
	cmd = exec.Command("./"+bin, "--version")
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
	bin := testBinaryName()
	cmd := exec.Command("go", "build", "-o", bin)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove(bin)

	// Test -V flag
	cmd = exec.Command("./"+bin, "-V")
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
	bin := testBinaryName()
	cmd := exec.Command("go", "build",
		"-ldflags", "-X main.version=1.2.3 -X main.commit=abc123def -X main.buildDate=2024-01-15",
		"-o", bin)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary with ldflags: %v", err)
	}
	defer os.Remove(bin)

	// Test --version flag
	cmd = exec.Command("./"+bin, "--version")
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

// TestBadFlagReportsOnceWithHelpPointer covers the fallout of dropping the
// os.Args pre-scan: "--typo --help" used to print usage and exit 0, because the
// scan saw --help before anything was parsed. Parsing now happens first, so the
// typo is reported instead. This runs as a subprocess because the flag package
// writes its own error and usage dump to the FlagSet's output, which is the
// process stderr rather than the writer run() is handed — an in-process test
// capturing only run()'s stderr could not see the dump at all.
func TestBadFlagReportsOnceWithHelpPointer(t *testing.T) {
	bin := filepath.Join(t.TempDir(), testBinaryName())
	if out, err := exec.Command("go", "build", "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("build test binary: %v\n%s", err, out)
	}

	for _, args := range [][]string{{"--typo"}, {"--typo", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := exec.Command(bin, args...)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected a non-zero exit for %v, got success:\n%s", args, out)
			}
			got := string(out)

			// The flag package's own listing must not compete with Usage().
			if strings.Contains(got, "Usage of diffyml:") {
				t.Errorf("flag package usage dump leaked into output:\n%s", got)
			}
			// ...and the failure is reported exactly once, not echoed by both
			// the flag package and main.
			if n := strings.Count(got, "flag provided but not defined"); n != 1 {
				t.Errorf("expected the error reported once, got %d times:\n%s", n, got)
			}
			if !strings.Contains(got, "Error: flag provided but not defined: -typo") {
				t.Errorf("expected the curated error, got:\n%s", got)
			}
			if !strings.Contains(got, "Run 'diffyml --help' for usage.") {
				t.Errorf("expected a pointer to --help, got:\n%s", got)
			}
		})
	}
}

// TestFlagValuesNotMistakenForHelpOrVersion is a regression test: main used to
// pre-scan os.Args for "-h"/"-V"/"--version", which also matched those strings
// when they appeared as the *value* of a preceding flag. "--mask-placeholder -h"
// printed usage instead of running the diff. Both are real flags now, so the
// flag package binds the value to the flag that owns it.
func TestFlagValuesNotMistakenForHelpOrVersion(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "from.yaml")
	to := filepath.Join(dir, "to.yaml")
	if err := os.WriteFile(from, []byte("a: 1\n"), 0o600); err != nil {
		t.Fatalf("write from: %v", err)
	}
	if err := os.WriteFile(to, []byte("a: 2\n"), 0o600); err != nil {
		t.Fatalf("write to: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"placeholder is -h", []string{from, to, "--mask-placeholder", "-h"}},
		{"placeholder is --help", []string{from, to, "--mask-placeholder", "--help"}},
		{"summary-model is -V", []string{from, to, "--summary-model", "-V"}},
		{"summary-model is --version", []string{from, to, "--summary-model", "--version"}},
		{"filter is -h", []string{from, to, "--filter", "-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			code := run(tt.args, &stdout, &stderr)

			if code != 0 {
				t.Errorf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
			}
			out := stdout.String()
			if strings.Contains(out, "A diff tool for YAML files") {
				t.Errorf("flag value was treated as --help; got usage output:\n%s", out)
			}
			if strings.Contains(out, "diffyml version") {
				t.Errorf("flag value was treated as --version; got:\n%s", out)
			}
		})
	}
}

// TestHelpAndVersionWithoutFileArgs verifies --help and --version still work
// with no file arguments, which is why the os.Args pre-scan existed. ParseArgs
// now short-circuits before the "requires two file arguments" check instead.
func TestHelpAndVersionWithoutFileArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"short help", []string{"-h"}, "A diff tool for YAML files"},
		{"long help", []string{"--help"}, "A diff tool for YAML files"},
		{"single-dash help", []string{"-help"}, "A diff tool for YAML files"},
		{"short version", []string{"-V"}, "diffyml version"},
		{"long version", []string{"--version"}, "diffyml version"},
		{"single-dash version", []string{"-version"}, "diffyml version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			code := run(tt.args, &stdout, &stderr)

			if code != 0 {
				t.Errorf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.want) {
				t.Errorf("expected stdout to contain %q, got:\n%s", tt.want, stdout.String())
			}
		})
	}
}

// TestVersionTakesPrecedenceOverHelp preserves the original ordering: main
// checked the version flag before the help flag.
func TestVersionTakesPrecedenceOverHelp(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run([]string{"--help", "--version"}, &stdout, &stderr); code != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "diffyml version") {
		t.Errorf("expected version output, got:\n%s", out)
	}
	if strings.Contains(out, "A diff tool for YAML files") {
		t.Errorf("expected version to win over help, got usage output:\n%s", out)
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
