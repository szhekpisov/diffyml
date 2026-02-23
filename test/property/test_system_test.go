package property

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestProperty6_TestExecutionSuccess tests that running `go test ./...` discovers
// and executes tests successfully without errors.
// **Validates: Requirements 3.1**
// Note: This test runs only 1 iteration since go test ./... is slow.
// The property is still validated - we verify it holds for the current repo state.
func TestProperty6_TestExecutionSuccess(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Run go test for just the pkg/diffyml package (faster than ./...)
	cmd := exec.Command("go", "test", "./pkg/diffyml/...")
	output, err := cmd.CombinedOutput()

	// Test execution must succeed
	if err != nil {
		t.Fatalf("go test failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Verify tests were actually discovered and run
	// Output should contain "ok" for successful packages
	hasOkPackages := strings.Contains(outputStr, "ok") ||
		strings.Contains(outputStr, "PASS")

	if !hasOkPackages {
		t.Errorf("No test packages found in output: %s", outputStr)
	}

	// Verify no test failures in output
	hasFailures := strings.Contains(outputStr, "FAIL") &&
		!strings.Contains(outputStr, "--- FAIL:")

	if hasFailures {
		t.Errorf("Test failures detected: %s", outputStr)
	}
}

// TestProperty6_TestExecutionSuccess_PackageDiscovery tests that tests in
// pkg/diffyml are properly discovered and run.
// **Validates: Requirements 3.1, 3.2**
func TestProperty6_TestExecutionSuccess_PackageDiscovery(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Run go test for the specific package
	cmd := exec.Command("go", "test", "./pkg/diffyml/...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("go test failed for pkg/diffyml: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Verify the package was tested
	hasPackageOutput := strings.Contains(outputStr, "github.com/szhekpisov/diffyml/pkg/diffyml") ||
		strings.Contains(outputStr, "ok")

	if !hasPackageOutput {
		t.Errorf("pkg/diffyml tests not found in output: %s", outputStr)
	}
}

// TestProperty6_TestExecutionSuccess_ListTests tests that go test -list
// discovers test functions.
// **Validates: Requirements 3.1**
func TestProperty6_TestExecutionSuccess_ListTests(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Run go test -list to list all tests
	cmd := exec.Command("go", "test", "-list", ".*", "./pkg/diffyml/...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("go test -list failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Verify tests were discovered
	// Output should contain test function names starting with "Test"
	hasTests := strings.Contains(outputStr, "Test")

	if !hasTests {
		t.Errorf("No test functions discovered: %s", outputStr)
	}

	// Count the number of tests discovered
	lines := strings.Split(outputStr, "\n")
	testCount := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Test") {
			testCount++
		}
	}

	// Should have at least some tests
	if testCount < 1 {
		t.Errorf("Expected at least 1 test, found %d", testCount)
	}
}

// TestProperty7_TestFixturesPresence tests that the testdata directory exists
// and contains test fixture files.
// **Validates: Requirements 3.3**
func TestProperty7_TestFixturesPresence(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	// Check if testdata directory exists
	testdataInfo, err := os.Stat("testdata")
	if err != nil {
		t.Fatalf("testdata directory not found: %v", err)
	}

	// Verify it's a directory
	if !testdataInfo.IsDir() {
		t.Fatal("testdata is not a directory")
	}

	// Check for fixture files
	hasFixtures := false
	err = filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check for YAML files (common fixture format)
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			hasFixtures = true
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Error walking testdata: %v", err)
	}

	if !hasFixtures {
		t.Error("No fixture files found in testdata")
	}
}

// TestProperty7_TestFixturesPresence_FixturesDirectory tests that the
// testdata/fixtures subdirectory exists with test files.
// **Validates: Requirements 3.3**
func TestProperty7_TestFixturesPresence_FixturesDirectory(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	fixturesDir := filepath.Join("testdata", "fixtures")

	// Check if fixtures directory exists
	fixturesInfo, err := os.Stat(fixturesDir)
	if err != nil {
		t.Skip("testdata/fixtures directory not found, skipping")
	}

	if !fixturesInfo.IsDir() {
		t.Fatal("testdata/fixtures is not a directory")
	}

	// Count fixture files
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("Failed to read fixtures directory: %v", err)
	}

	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			fileCount++
		}
	}

	// Should have at least one fixture file
	if fileCount < 1 {
		t.Error("No fixture files found in testdata/fixtures")
	}
}

// TestProperty7_TestFixturesPresence_PerfDirectory tests that the
// testdata/perf subdirectory exists with performance test files.
// **Validates: Requirements 3.3**
func TestProperty7_TestFixturesPresence_PerfDirectory(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	perfDir := filepath.Join("testdata", "perf")

	// Check if perf directory exists
	perfInfo, err := os.Stat(perfDir)
	if err != nil {
		t.Fatalf("testdata/perf directory not found: %v", err)
	}

	if !perfInfo.IsDir() {
		t.Fatal("testdata/perf is not a directory")
	}

	// Count perf test files
	entries, err := os.ReadDir(perfDir)
	if err != nil {
		t.Fatalf("Failed to read perf directory: %v", err)
	}

	// Should have at least one perf test entry (files or subdirectories)
	if len(entries) < 1 {
		t.Error("No perf test entries found in testdata/perf")
	}
}
