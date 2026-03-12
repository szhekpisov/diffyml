package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeFilePath_Empty(t *testing.T) {
	got := normalizeFilePath("", nil)
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
			var stderr bytes.Buffer
			got := normalizeFilePath(tt.path, &stderr)
			if got != tt.path {
				t.Errorf("expected %q, got %q", tt.path, got)
			}
			if stderr.Len() > 0 {
				t.Errorf("expected no warning for %s, got %q", tt.path, stderr.String())
			}
		})
	}
}

func TestNormalizeFilePath_RelativePath(t *testing.T) {
	got := normalizeFilePath("./some/file.yaml", nil)
	if got != "some/file.yaml" {
		t.Errorf("expected 'some/file.yaml', got %q", got)
	}
}

func TestNormalizeFilePath_AlreadyRelative(t *testing.T) {
	got := normalizeFilePath("some/file.yaml", nil)
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

	var stderr bytes.Buffer
	got := normalizeFilePath(absPath, &stderr)
	if got != "testfile.yaml" {
		t.Errorf("expected 'testfile.yaml', got %q", got)
	}
	if stderr.Len() > 0 {
		t.Errorf("expected no warning, got %q", stderr.String())
	}
}

func TestNormalizeFilePath_AbsoluteOutsideCwd(t *testing.T) {
	var stderr bytes.Buffer
	got := normalizeFilePath("/nonexistent/path/file.yaml", &stderr)
	if got != "/nonexistent/path/file.yaml" {
		t.Errorf("expected absolute path back, got %q", got)
	}
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning, got %q", stderr.String())
	}
}

func TestNormalizeFilePath_AbsoluteOutsideCwd_NilStderr(t *testing.T) {
	got := normalizeFilePath("/nonexistent/path/file.yaml", nil)
	if got != "/nonexistent/path/file.yaml" {
		t.Errorf("expected absolute path back, got %q", got)
	}
}
