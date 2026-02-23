package property

import (
	"os"
	"path/filepath"
	"testing"
)

// getRepoRoot returns the repository root directory by walking up from the
// current working directory until it finds a go.mod file.
func getRepoRoot() (string, error) {
	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root without finding go.mod
			break
		}
		dir = parent
	}

	// If we're running from test/property, go up two levels
	cwd, _ := os.Getwd()
	repoRoot := filepath.Join(cwd, "..", "..")
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		return filepath.Abs(repoRoot)
	}

	return "", os.ErrNotExist
}

// chdir changes to the repository root directory for the duration of the test.
// It returns a cleanup function that restores the original directory.
func chdirToRepoRoot(t *testing.T) func() {
	t.Helper()

	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Find repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	// Change to repository root
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}

	// Return cleanup function
	return func() {
		os.Chdir(origDir)
	}
}
