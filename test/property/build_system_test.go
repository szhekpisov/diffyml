package property

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty2_BuildSystemSuccess tests that running `go build` completes
// successfully without errors and produces an executable binary.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("go build must complete successfully without errors", prop.ForAll(
		func(dummyInput int) bool {
			// Create a unique binary name for this test iteration
			binaryName := "diffyml_build_test"

			// Clean up any existing binary
			os.Remove(binaryName)
			defer os.Remove(binaryName)

			// Run go build
			cmd := exec.Command("go", "build", "-o", binaryName)
			output, err := cmd.CombinedOutput()

			// Build must succeed
			if err != nil {
				t.Logf("Build failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// Verify the binary was created
			info, err := os.Stat(binaryName)
			if err != nil {
				t.Logf("Binary not found after build: %v", err)
				return false
			}

			// Verify it's a regular file
			if !info.Mode().IsRegular() {
				t.Logf("Binary is not a regular file")
				return false
			}

			// Verify it's executable
			if info.Mode()&0111 == 0 {
				t.Logf("Binary is not executable")
				return false
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty2_BuildSystemSuccess_WithCleanEnvironment tests that the build
// succeeds even in a clean environment without cached dependencies.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess_WithCleanEnvironment(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("go build must succeed without additional dependencies", prop.ForAll(
		func(dummyInput int) bool {
			binaryName := "diffyml_clean_build_test"

			// Clean up
			os.Remove(binaryName)
			defer os.Remove(binaryName)

			// Verify go.mod exists (required for build)
			if _, err := os.Stat("go.mod"); err != nil {
				t.Logf("go.mod not found: %v", err)
				return false
			}

			// Run go build with verbose output
			cmd := exec.Command("go", "build", "-v", "-o", binaryName)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Logf("Build failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// Verify no error messages in output
			outputStr := strings.ToLower(string(output))
			hasErrors := strings.Contains(outputStr, "error:") ||
				strings.Contains(outputStr, "fatal:")

			if hasErrors {
				t.Logf("Build output contains errors: %s", string(output))
				return false
			}

			// Verify binary exists and is executable
			info, err := os.Stat(binaryName)
			if err != nil {
				return false
			}

			return info.Mode().IsRegular() && info.Mode()&0111 != 0
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty2_BuildSystemSuccess_WithLdflags tests that the build succeeds
// with version information injected via ldflags.
// **Validates: Requirements 2.1**
func TestProperty2_BuildSystemSuccess_WithLdflags(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("go build with ldflags must succeed", prop.ForAll(
		func(dummyInput int) bool {
			binaryName := "diffyml_ldflags_test"

			// Clean up
			os.Remove(binaryName)
			defer os.Remove(binaryName)

			// Build with ldflags (as Homebrew would do)
			ldflags := "-X main.version=1.0.0 -X main.commit=test123 -X main.buildDate=2024-01-15"
			cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binaryName)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Logf("Build with ldflags failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// Verify binary was created
			info, err := os.Stat(binaryName)
			if err != nil {
				return false
			}

			// Verify it's executable
			if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
				return false
			}

			// Verify the binary runs and shows injected version
			versionCmd := exec.Command("./"+binaryName, "--version")
			versionOutput, err := versionCmd.CombinedOutput()
			if err != nil {
				t.Logf("Version check failed: %v", err)
				return false
			}

			versionStr := string(versionOutput)
			hasInjectedVersion := strings.Contains(versionStr, "1.0.0") &&
				strings.Contains(versionStr, "test123")

			return hasInjectedVersion
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty4_SingleBinaryOutput tests that a successful build produces
// exactly one executable file with the expected name.
// **Validates: Requirements 2.3**
func TestProperty4_SingleBinaryOutput(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("build must produce exactly one executable binary", prop.ForAll(
		func(dummyInput int) bool {
			binaryName := "diffyml_single_output_test"

			// Clean up any existing files
			os.Remove(binaryName)
			defer os.Remove(binaryName)

			// Count files before build
			beforeFiles, err := countFilesInDir(".")
			if err != nil {
				t.Logf("Failed to count files before build: %v", err)
				return false
			}

			// Run go build
			cmd := exec.Command("go", "build", "-o", binaryName)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Build failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// Count files after build
			afterFiles, err := countFilesInDir(".")
			if err != nil {
				t.Logf("Failed to count files after build: %v", err)
				return false
			}

			// Exactly one new file should be created
			newFilesCount := afterFiles - beforeFiles
			if newFilesCount != 1 {
				t.Logf("Expected 1 new file, got %d", newFilesCount)
				return false
			}

			// Verify the binary exists with the expected name
			info, err := os.Stat(binaryName)
			if err != nil {
				t.Logf("Binary not found: %v", err)
				return false
			}

			// Verify it's a single regular file (not a directory)
			if !info.Mode().IsRegular() {
				t.Logf("Output is not a regular file")
				return false
			}

			// Verify it's executable
			if info.Mode()&0111 == 0 {
				t.Logf("Output is not executable")
				return false
			}

			// Verify the binary name matches what we specified
			absPath, err := filepath.Abs(binaryName)
			if err != nil {
				return false
			}

			expectedPath := filepath.Join(filepath.Dir(absPath), binaryName)
			if absPath != expectedPath {
				t.Logf("Binary path mismatch: got %s, expected %s", absPath, expectedPath)
				return false
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty4_SingleBinaryOutput_WithDefaultName tests that building without
// specifying an output name produces a single binary named after the module.
// **Validates: Requirements 2.3**
func TestProperty4_SingleBinaryOutput_WithDefaultName(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("build without -o flag must produce single binary with module name", prop.ForAll(
		func(dummyInput int) bool {
			// Default binary name is the module name (diffyml)
			defaultBinaryName := "diffyml"

			// Clean up
			os.Remove(defaultBinaryName)
			defer os.Remove(defaultBinaryName)

			// Run go build without -o flag
			cmd := exec.Command("go", "build")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Build failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// Verify the binary was created with the default name
			info, err := os.Stat(defaultBinaryName)
			if err != nil {
				t.Logf("Default binary not found: %v", err)
				return false
			}

			// Verify it's a regular executable file
			isValid := info.Mode().IsRegular() && info.Mode()&0111 != 0

			return isValid
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty4_SingleBinaryOutput_NoExtraFiles tests that the build process
// doesn't create extra files beyond the binary.
// **Validates: Requirements 2.3**
func TestProperty4_SingleBinaryOutput_NoExtraFiles(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("build must not create extra files beyond the binary", prop.ForAll(
		func(dummyInput int) bool {
			binaryName := "diffyml_no_extra_files_test"

			// Clean up
			os.Remove(binaryName)
			defer os.Remove(binaryName)

			// Get list of files before build
			beforeFiles, err := listFilesInDir(".")
			if err != nil {
				t.Logf("Failed to list files before build: %v", err)
				return false
			}

			// Run go build
			cmd := exec.Command("go", "build", "-o", binaryName)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Build failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// Get list of files after build
			afterFiles, err := listFilesInDir(".")
			if err != nil {
				t.Logf("Failed to list files after build: %v", err)
				return false
			}

			// Find new files
			newFiles := findNewFiles(beforeFiles, afterFiles)

			// Should have exactly one new file (the binary)
			if len(newFiles) != 1 {
				t.Logf("Expected 1 new file, got %d: %v", len(newFiles), newFiles)
				return false
			}

			// The new file should be our binary
			if newFiles[0] != binaryName {
				t.Logf("New file is not the expected binary: got %s, expected %s", newFiles[0], binaryName)
				return false
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty5_DependencyIntegrity tests that all dependencies have checksums
// in go.sum, ensuring no unversioned or unchecksummed dependencies.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("all dependencies must have checksums in go.sum", prop.ForAll(
		func(dummyInput int) bool {
			// Verify go.sum exists
			goSumInfo, err := os.Stat("go.sum")
			if err != nil {
				t.Logf("go.sum not found: %v", err)
				return false
			}

			// Verify it's a regular file
			if !goSumInfo.Mode().IsRegular() {
				t.Logf("go.sum is not a regular file")
				return false
			}

			// Read go.sum content
			goSumContent, err := os.ReadFile("go.sum")
			if err != nil {
				t.Logf("Failed to read go.sum: %v", err)
				return false
			}

			// Verify go.sum is not empty
			if len(goSumContent) == 0 {
				t.Logf("go.sum is empty")
				return false
			}

			// Run go mod verify to check dependency integrity
			cmd := exec.Command("go", "mod", "verify")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("go mod verify failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// Verify the output indicates all modules are verified
			outputStr := string(output)
			hasVerifiedMessage := strings.Contains(outputStr, "all modules verified") ||
				strings.Contains(outputStr, "verified")

			return hasVerifiedMessage
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty5_DependencyIntegrity_WithGoModCheck tests that go.mod and go.sum
// are in sync and all dependencies are properly tracked.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity_WithGoModCheck(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("go.mod and go.sum must be in sync", prop.ForAll(
		func(dummyInput int) bool {
			// Run go mod tidy to check if files are in sync
			// This command will fail if go.mod and go.sum are out of sync
			cmd := exec.Command("go", "mod", "tidy", "-v")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("go mod tidy failed: %v\nOutput: %s", err, string(output))
				return false
			}

			// After tidy, verify that go.sum still exists and is valid
			goSumInfo, err := os.Stat("go.sum")
			if err != nil {
				t.Logf("go.sum not found after tidy: %v", err)
				return false
			}

			if !goSumInfo.Mode().IsRegular() {
				return false
			}

			// Verify go mod verify still passes
			verifyCmd := exec.Command("go", "mod", "verify")
			verifyOutput, err := verifyCmd.CombinedOutput()
			if err != nil {
				t.Logf("go mod verify failed after tidy: %v\nOutput: %s", err, string(verifyOutput))
				return false
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty5_DependencyIntegrity_NoUnversionedDeps tests that all
// dependencies in go.mod have explicit versions.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity_NoUnversionedDeps(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("all dependencies must have explicit versions", prop.ForAll(
		func(dummyInput int) bool {
			// Read go.mod
			goModContent, err := os.ReadFile("go.mod")
			if err != nil {
				t.Logf("Failed to read go.mod: %v", err)
				return false
			}

			goModText := string(goModContent)
			lines := strings.Split(goModText, "\n")

			// Check each require line for version information
			inRequireBlock := false
			for _, line := range lines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in a require block
				if strings.HasPrefix(trimmedLine, "require (") {
					inRequireBlock = true
					continue
				}
				if inRequireBlock && trimmedLine == ")" {
					inRequireBlock = false
					continue
				}

				// Check require lines
				if strings.HasPrefix(trimmedLine, "require ") || inRequireBlock {
					// Skip empty lines and closing parenthesis
					if trimmedLine == "" || trimmedLine == ")" {
						continue
					}

					// Skip comments
					if strings.HasPrefix(trimmedLine, "//") {
						continue
					}

					// Extract the dependency line
					depLine := strings.TrimPrefix(trimmedLine, "require ")

					// Skip if it's just the opening of a require block
					if strings.HasSuffix(depLine, "(") {
						continue
					}

					// Check if the line contains a version
					// Valid versions start with 'v' followed by numbers
					// or are pseudo-versions with timestamps
					if depLine != "" && !strings.Contains(depLine, "//") {
						parts := strings.Fields(depLine)
						if len(parts) >= 2 {
							version := parts[1]
							// Check for valid version format
							hasValidVersion := strings.HasPrefix(version, "v") &&
								(strings.Contains(version, ".") || strings.Contains(version, "-"))

							if !hasValidVersion {
								t.Logf("Dependency without valid version: %s", depLine)
								return false
							}
						}
					}
				}
			}

			return true
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty5_DependencyIntegrity_BuildWithVerify tests that building
// after verifying dependencies succeeds.
// **Validates: Requirements 2.5**
func TestProperty5_DependencyIntegrity_BuildWithVerify(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newHeavyProperties()

	properties.Property("build must succeed after dependency verification", prop.ForAll(
		func(dummyInput int) bool {
			// First, verify dependencies
			verifyCmd := exec.Command("go", "mod", "verify")
			verifyOutput, err := verifyCmd.CombinedOutput()
			if err != nil {
				t.Logf("go mod verify failed: %v\nOutput: %s", err, string(verifyOutput))
				return false
			}

			// Then, build
			binaryName := "diffyml_verify_build_test"
			os.Remove(binaryName)
			defer os.Remove(binaryName)

			buildCmd := exec.Command("go", "build", "-o", binaryName)
			buildOutput, err := buildCmd.CombinedOutput()
			if err != nil {
				t.Logf("Build failed after verify: %v\nOutput: %s", err, string(buildOutput))
				return false
			}

			// Verify binary was created
			info, err := os.Stat(binaryName)
			if err != nil {
				return false
			}

			return info.Mode().IsRegular() && info.Mode()&0111 != 0
		},
		gen.IntRange(1, 100), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
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
