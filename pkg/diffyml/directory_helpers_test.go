package diffyml

import (
	"os"
	"path/filepath"
	"testing"
)

// createFile creates a file in a directory.
func createFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
