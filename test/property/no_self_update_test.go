package property

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProperty22_NoSelfUpdateFunctionality tests that the codebase does not
// contain self-update functionality (no HTTP clients, no update downloads).
// **Validates: Requirements 10.1, 10.2**
func TestProperty22_NoSelfUpdateFunctionality(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	forbiddenPatterns := []string{
		"self-update",
		"selfupdate",
		"auto-update",
		"autoupdate",
		"checkForUpdates",
		"CheckForUpdates",
		"check_for_updates",
		"downloadUpdate",
		"DownloadUpdate",
	}

	goFiles, err := findGoSourceFiles(".")
	if err != nil {
		t.Fatalf("Failed to find Go source files: %v", err)
	}

	for _, file := range goFiles {
		if strings.HasSuffix(file, "_test.go") ||
			strings.Contains(file, "vendor/") ||
			strings.Contains(file, "test/") {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		for _, pattern := range forbiddenPatterns {
			if strings.Contains(string(content), pattern) {
				t.Fatalf("Found forbidden self-update pattern %q in %s", pattern, file)
			}
		}
	}
}

// TestProperty22_NoSelfUpdateFunctionality_NoBinaryDownload tests that the codebase
// does not contain code patterns for downloading and replacing binaries.
// **Validates: Requirements 10.1, 10.3**
func TestProperty22_NoSelfUpdateFunctionality_NoBinaryDownload(t *testing.T) {
	cleanup := chdirToRepoRoot(t)
	defer cleanup()

	binaryPatterns := []string{
		"downloadBinary",
		"DownloadBinary",
		"replaceBinary",
		"ReplaceBinary",
		"updateBinary",
		"UpdateBinary",
	}

	goFiles, err := findGoSourceFiles(".")
	if err != nil {
		t.Fatalf("Failed to find Go source files: %v", err)
	}

	for _, file := range goFiles {
		if strings.HasSuffix(file, "_test.go") ||
			strings.Contains(file, "vendor/") ||
			strings.Contains(file, "test/") {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		for _, pattern := range binaryPatterns {
			if strings.Contains(string(content), pattern) {
				t.Fatalf("Found forbidden binary download pattern %q in %s", pattern, file)
			}
		}
	}
}

// findGoSourceFiles recursively finds all .go files in a directory.
func findGoSourceFiles(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
