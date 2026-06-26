package property

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// skipOnWindows skips tests whose assertions rely on Unix executable-bit
// semantics (Go's os.Stat reports no 0o111 bit for binaries on Windows).
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("executable-bit assertions are not meaningful on Windows")
	}
}

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

// buildTestBinary compiles diffyml into a per-test temp directory and returns
// the absolute binary path. Per-test temp dirs keep concurrent test workers
// from trampling each other's output. Extra args are passed to `go build`
// between `build` and `-o <path>`.
func buildTestBinary(t *testing.T, name string, extraArgs ...string) string {
	t.Helper()
	repoRoot, err := getRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	// Windows requires the .exe extension to recognize and execute the binary.
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binaryPath := filepath.Join(t.TempDir(), name)
	args := append([]string{"build"}, extraArgs...)
	args = append(args, "-o", binaryPath)

	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, output)
	}
	return binaryPath
}
