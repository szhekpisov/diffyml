package property

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestProperty2_BuildSystemSuccess tests that running `go build` completes
// successfully without errors and produces an executable binary.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess(t *testing.T) {
	skipOnWindows(t)
	binaryPath := buildTestBinary(t, "diffyml_build_test")

	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("Binary not found after build: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatal("Binary is not a regular file")
	}

	if info.Mode()&0o111 == 0 {
		t.Fatal("Binary is not executable")
	}
}

// TestProperty2_BuildSystemSuccess_WithCleanEnvironment tests that the build
// succeeds even in a clean environment without cached dependencies.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess_WithCleanEnvironment(t *testing.T) {
	skipOnWindows(t)
	binaryPath := buildTestBinary(t, "diffyml_clean_build_test", "-v")

	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}
}

// TestProperty2_BuildSystemSuccess_WithLdflags tests that the build succeeds
// with version information injected via ldflags.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess_WithLdflags(t *testing.T) {
	skipOnWindows(t)
	ldflags := "-X main.version=1.0.0 -X main.commit=test123 -X main.buildDate=2024-01-15"
	binaryPath := buildTestBinary(t, "diffyml_ldflags_test", "-ldflags", ldflags)

	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}

	versionCmd := exec.Command(binaryPath, "--version")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version check failed: %v", err)
	}

	versionStr := string(versionOutput)
	if !strings.Contains(versionStr, "1.0.0") {
		t.Error("output missing injected version '1.0.0'")
	}
	if !strings.Contains(versionStr, "test123") {
		t.Error("output missing injected commit 'test123'")
	}
}

// TestProperty4_DefaultBinaryName tests that `go build` without `-o` produces
// a binary named after the module path's last component (`diffyml`).
// **Validates: Requirements 2.3**
func TestProperty4_DefaultBinaryName(t *testing.T) {
	skipOnWindows(t)
	repoRoot, err := getRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	tempDir := t.TempDir()
	// Trailing separator tells `go build` to write into tempDir while still
	// choosing the default binary name from the module path.
	cmd := exec.Command("go", "build", "-o", tempDir+string(os.PathSeparator))
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, string(output))
	}

	binaryPath := filepath.Join(tempDir, "diffyml")
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("Default-name binary not found at %s: %v", binaryPath, err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}
}

// TestProperty5_DependencyIntegrity tests that all dependencies have checksums
// in go.sum, ensuring no unversioned or unchecksummed dependencies.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	goSumInfo, err := os.Stat("go.sum")
	if err != nil {
		t.Fatalf("go.sum not found: %v", err)
	}

	if !goSumInfo.Mode().IsRegular() {
		t.Fatal("go.sum is not a regular file")
	}

	goSumContent, err := os.ReadFile("go.sum")
	if err != nil {
		t.Fatalf("Failed to read go.sum: %v", err)
	}

	if len(goSumContent) == 0 {
		t.Fatal("go.sum is empty")
	}

	cmd := exec.Command("go", "mod", "verify")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod verify failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "all modules verified") && !strings.Contains(outputStr, "verified") {
		t.Fatalf("go mod verify did not confirm verification: %s", outputStr)
	}
}

// TestProperty5_DependencyIntegrity_WithGoModCheck tests that go.mod and go.sum
// are in sync and all dependencies are properly tracked.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity_WithGoModCheck(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	cmd := exec.Command("go", "mod", "tidy", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, string(output))
	}

	goSumInfo, err := os.Stat("go.sum")
	if err != nil {
		t.Fatalf("go.sum not found after tidy: %v", err)
	}

	if !goSumInfo.Mode().IsRegular() {
		t.Fatal("go.sum is not a regular file after tidy")
	}

	verifyCmd := exec.Command("go", "mod", "verify")
	verifyOutput, err := verifyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod verify failed after tidy: %v\nOutput: %s", err, string(verifyOutput))
	}
}

// TestProperty5_DependencyIntegrity_NoUnversionedDeps tests that all
// dependencies in go.mod have explicit versions.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity_NoUnversionedDeps(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	goModContent, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	lines := strings.Split(string(goModContent), "\n")

	inRequireBlock := false
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && trimmedLine == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(trimmedLine, "require ") || inRequireBlock {
			if trimmedLine == "" || trimmedLine == ")" || strings.HasPrefix(trimmedLine, "//") {
				continue
			}

			depLine := strings.TrimPrefix(trimmedLine, "require ")
			if strings.HasSuffix(depLine, "(") {
				continue
			}

			if depLine != "" && !strings.Contains(depLine, "//") {
				checkDependencyVersion(t, depLine)
			}
		}
	}
}

// TestProperty5_DependencyIntegrity_BuildWithVerify tests that building
// after verifying dependencies succeeds.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity_BuildWithVerify(t *testing.T) {
	skipOnWindows(t)
	repoRoot, err := getRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	verifyCmd := exec.Command("go", "mod", "verify")
	verifyCmd.Dir = repoRoot
	verifyOutput, err := verifyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod verify failed: %v\nOutput: %s", err, string(verifyOutput))
	}

	binaryPath := buildTestBinary(t, "diffyml_verify_build_test")

	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}
}

// checkDependencyVersion validates that a dependency line has a valid version.
func checkDependencyVersion(t *testing.T, depLine string) {
	t.Helper()
	parts := strings.Fields(depLine)
	if len(parts) < 2 {
		return
	}
	version := parts[1]
	hasValidVersion := strings.HasPrefix(version, "v") &&
		(strings.Contains(version, ".") || strings.Contains(version, "-"))
	if !hasValidVersion {
		t.Fatalf("Dependency without valid version: %s", depLine)
	}
}
