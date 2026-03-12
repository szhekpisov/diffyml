package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeFilePath_Empty(t *testing.T) {
	got := normalizeFilePath("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestNormalizeFilePath_DevPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"dev_fd", "/dev/fd/12"},
		{"dev_stdin", "/dev/stdin"},
		{"dev_null", "/dev/null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFilePath(tt.path)
			if got != tt.path {
				t.Errorf("expected %q, got %q", tt.path, got)
			}
		})
	}
}

func TestNormalizeFilePath_RelativePath(t *testing.T) {
	got := normalizeFilePath("./some/file.yaml")
	if got != "some/file.yaml" {
		t.Errorf("expected 'some/file.yaml', got %q", got)
	}
}

func TestNormalizeFilePath_AlreadyRelative(t *testing.T) {
	got := normalizeFilePath("some/file.yaml")
	if got != "some/file.yaml" {
		t.Errorf("expected 'some/file.yaml', got %q", got)
	}
}

func TestNormalizeFilePath_AbsoluteInCwd(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	absPath := filepath.Join(cwd, "testfile.yaml")

	got := normalizeFilePath(absPath)
	if got != "testfile.yaml" {
		t.Errorf("expected 'testfile.yaml', got %q", got)
	}
}

func TestNormalizeFilePath_AbsoluteOutsideCwd(t *testing.T) {
	got := normalizeFilePath("/nonexistent/path/file.yaml")
	if got != "/nonexistent/path/file.yaml" {
		t.Errorf("expected absolute path back, got %q", got)
	}
}
