package diffyml

import (
	"os"
	"testing"
)

func TestIsDirectory_WithDirectory(t *testing.T) {
	dir := t.TempDir()
	if !IsDirectory(dir) {
		t.Errorf("expected IsDirectory(%q) to be true for a directory", dir)
	}
}

func TestIsDirectory_WithFile(t *testing.T) {
	f, err := os.CreateTemp("", "testfile-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	_ = f.Close()

	if IsDirectory(f.Name()) {
		t.Errorf("expected IsDirectory(%q) to be false for a file", f.Name())
	}
}

func TestIsDirectory_WithNonExistentPath(t *testing.T) {
	if IsDirectory("/nonexistent/path/that/does/not/exist") {
		t.Error("expected IsDirectory to be false for a non-existent path")
	}
}

func TestIsDirectory_WithEmptyString(t *testing.T) {
	if IsDirectory("") {
		t.Error("expected IsDirectory to be false for an empty string")
	}
}

func TestIsDirectory_WithURL(t *testing.T) {
	if IsDirectory("https://example.com/file.yaml") {
		t.Error("expected IsDirectory to be false for a URL")
	}
}
