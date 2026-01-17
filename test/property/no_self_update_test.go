package property

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty22_NoSelfUpdateFunctionality tests that the codebase does not
// contain self-update functionality (no HTTP clients, no update downloads).
// **Validates: Requirements 10.1, 10.2**
func TestProperty22_NoSelfUpdateFunctionality(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// Patterns that indicate self-update functionality
	forbiddenPatterns := []string{
		"self-update",       // Self-update functionality
		"selfupdate",        // Self-update functionality (no hyphen)
		"auto-update",       // Auto-update functionality
		"autoupdate",        // Auto-update functionality (no hyphen)
		"checkForUpdates",   // Update checking
		"CheckForUpdates",   // Update checking (exported)
		"check_for_updates", // Update checking (snake_case)
		"downloadUpdate",    // Download updates
		"DownloadUpdate",    // Download updates (exported)
	}

	properties.Property("codebase must not contain self-update patterns", prop.ForAll(
		func(patternIndex int) bool {
			pattern := forbiddenPatterns[patternIndex%len(forbiddenPatterns)]

			// Search all Go source files (excluding test files and vendor)
			goFiles, err := findGoSourceFiles(".")
			if err != nil {
				return false
			}

			for _, file := range goFiles {
				// Skip test files and vendor directory
				if strings.HasSuffix(file, "_test.go") ||
					strings.Contains(file, "vendor/") ||
					strings.Contains(file, "test/") {
					continue
				}

				content, err := os.ReadFile(file)
				if err != nil {
					continue
				}

				if strings.Contains(string(content), pattern) {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 99), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty22_NoSelfUpdateFunctionality_NoBinaryDownload tests that the codebase
// does not contain code patterns for downloading and replacing binaries.
// **Validates: Requirements 10.1, 10.3**
func TestProperty22_NoSelfUpdateFunctionality_NoBinaryDownload(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	properties := newProperties()

	// Patterns that indicate binary download/replacement
	binaryPatterns := []string{
		"os.Executable",   // Getting current executable path for replacement
		"ioutil.TempFile", // Creating temp files (for download)
		"os.Rename",       // Replacing binary
		"io.Copy",         // Could be used for download
		"downloadBinary",  // Download binary
		"DownloadBinary",  // Download binary (exported)
		"replaceBinary",   // Replace binary
		"ReplaceBinary",   // Replace binary (exported)
		"updateBinary",    // Update binary
		"UpdateBinary",    // Update binary (exported)
	}

	properties.Property("codebase must not contain binary download patterns", prop.ForAll(
		func(patternIndex int) bool {
			pattern := binaryPatterns[patternIndex%len(binaryPatterns)]

			// Search all Go source files (excluding test files and vendor)
			goFiles, err := findGoSourceFiles(".")
			if err != nil {
				return false
			}

			for _, file := range goFiles {
				// Skip test files and vendor directory
				if strings.HasSuffix(file, "_test.go") ||
					strings.Contains(file, "vendor/") ||
					strings.Contains(file, "test/") {
					continue
				}

				content, err := os.ReadFile(file)
				if err != nil {
					continue
				}

				if strings.Contains(string(content), pattern) {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 99), // Run 100 iterations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// findGoSourceFiles recursively finds all .go files in a directory.
func findGoSourceFiles(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Collect .go files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
