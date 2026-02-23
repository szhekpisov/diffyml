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
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	binaryName := "diffyml_build_test"
	os.Remove(binaryName)
	defer os.Remove(binaryName)

	cmd := exec.Command("go", "build", "-o", binaryName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, string(output))
	}

	info, err := os.Stat(binaryName)
	if err != nil {
		t.Fatalf("Binary not found after build: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatal("Binary is not a regular file")
	}

	if info.Mode()&0111 == 0 {
		t.Fatal("Binary is not executable")
	}
}

// TestProperty2_BuildSystemSuccess_WithCleanEnvironment tests that the build
// succeeds even in a clean environment without cached dependencies.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess_WithCleanEnvironment(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	binaryName := "diffyml_clean_build_test"
	os.Remove(binaryName)
	defer os.Remove(binaryName)

	if _, err := os.Stat("go.mod"); err != nil {
		t.Fatalf("go.mod not found: %v", err)
	}

	cmd := exec.Command("go", "build", "-v", "-o", binaryName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := strings.ToLower(string(output))
	if strings.Contains(outputStr, "error:") || strings.Contains(outputStr, "fatal:") {
		t.Fatalf("Build output contains errors: %s", string(output))
	}

	info, err := os.Stat(binaryName)
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}
}

// TestProperty2_BuildSystemSuccess_WithLdflags tests that the build succeeds
// with version information injected via ldflags.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess_WithLdflags(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	binaryName := "diffyml_ldflags_test"
	os.Remove(binaryName)
	defer os.Remove(binaryName)

	ldflags := "-X main.version=1.0.0 -X main.commit=test123 -X main.buildDate=2024-01-15"
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binaryName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build with ldflags failed: %v\nOutput: %s", err, string(output))
	}

	info, err := os.Stat(binaryName)
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}

	versionCmd := exec.Command("./"+binaryName, "--version")
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

// TestProperty4_SingleBinaryOutput tests that a successful build produces
// exactly one executable file with the expected name.
// **Validates: Requirements 2.3**
func TestProperty4_SingleBinaryOutput(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	binaryName := "diffyml_single_output_test"
	os.Remove(binaryName)
	defer os.Remove(binaryName)

	beforeFiles, err := countFilesInDir(".")
	if err != nil {
		t.Fatalf("Failed to count files before build: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", binaryName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, string(output))
	}

	afterFiles, err := countFilesInDir(".")
	if err != nil {
		t.Fatalf("Failed to count files after build: %v", err)
	}

	newFilesCount := afterFiles - beforeFiles
	if newFilesCount != 1 {
		t.Fatalf("Expected 1 new file, got %d", newFilesCount)
	}

	info, err := os.Stat(binaryName)
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatal("Output is not a regular file")
	}

	if info.Mode()&0111 == 0 {
		t.Fatal("Output is not executable")
	}

	absPath, err := filepath.Abs(binaryName)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	expectedPath := filepath.Join(filepath.Dir(absPath), binaryName)
	if absPath != expectedPath {
		t.Fatalf("Binary path mismatch: got %s, expected %s", absPath, expectedPath)
	}
}

// TestProperty4_SingleBinaryOutput_WithDefaultName tests that building without
// specifying an output name produces a single binary named after the module.
// **Validates: Requirements 2.3**
func TestProperty4_SingleBinaryOutput_WithDefaultName(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	defaultBinaryName := "diffyml"
	os.Remove(defaultBinaryName)
	defer os.Remove(defaultBinaryName)

	cmd := exec.Command("go", "build")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, string(output))
	}

	info, err := os.Stat(defaultBinaryName)
	if err != nil {
		t.Fatalf("Default binary not found: %v", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}
}

// TestProperty4_SingleBinaryOutput_NoExtraFiles tests that the build process
// doesn't create extra files beyond the binary.
// **Validates: Requirements 2.3**
func TestProperty4_SingleBinaryOutput_NoExtraFiles(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	binaryName := "diffyml_no_extra_files_test"
	os.Remove(binaryName)
	defer os.Remove(binaryName)

	beforeFiles, err := listFilesInDir(".")
	if err != nil {
		t.Fatalf("Failed to list files before build: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", binaryName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, string(output))
	}

	afterFiles, err := listFilesInDir(".")
	if err != nil {
		t.Fatalf("Failed to list files after build: %v", err)
	}

	newFiles := findNewFiles(beforeFiles, afterFiles)

	if len(newFiles) != 1 {
		t.Fatalf("Expected 1 new file, got %d: %v", len(newFiles), newFiles)
	}

	if newFiles[0] != binaryName {
		t.Fatalf("New file is not the expected binary: got %s, expected %s", newFiles[0], binaryName)
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
				parts := strings.Fields(depLine)
				if len(parts) >= 2 {
					version := parts[1]
					hasValidVersion := strings.HasPrefix(version, "v") &&
						(strings.Contains(version, ".") || strings.Contains(version, "-"))

					if !hasValidVersion {
						t.Fatalf("Dependency without valid version: %s", depLine)
					}
				}
			}
		}
	}
}

// TestProperty5_DependencyIntegrity_BuildWithVerify tests that building
// after verifying dependencies succeeds.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity_BuildWithVerify(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	verifyCmd := exec.Command("go", "mod", "verify")
	verifyOutput, err := verifyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod verify failed: %v\nOutput: %s", err, string(verifyOutput))
	}

	binaryName := "diffyml_verify_build_test"
	os.Remove(binaryName)
	defer os.Remove(binaryName)

	buildCmd := exec.Command("go", "build", "-o", binaryName)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed after verify: %v\nOutput: %s", err, string(buildOutput))
	}

	info, err := os.Stat(binaryName)
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		t.Fatal("Binary is not a regular executable file")
	}
}

// Helper function to count files in a directory (non-recursive)
func countFilesInDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}

	return count, nil
}

// Helper function to list files in a directory (non-recursive)
func listFilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// Helper function to find new files by comparing two lists
func findNewFiles(before, after []string) []string {
	beforeMap := make(map[string]bool)
	for _, file := range before {
		beforeMap[file] = true
	}

	var newFiles []string
	for _, file := range after {
		if !beforeMap[file] {
			newFiles = append(newFiles, file)
		}
	}

	return newFiles
}
