package diffyml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFiles_AllRegularFiles(t *testing.T) {
	dir := t.TempDir()

	// Create various files — all should be discovered
	createFile(t, dir, "deploy.yaml", "key: value")
	createFile(t, dir, "service.yml", "key: value")
	createFile(t, dir, "readme.txt", "hello")
	createFile(t, dir, "config.json", "{}")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"config.json", "deploy.yaml", "readme.txt", "service.yml"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}

func TestDiscoverFiles_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected empty list, got %v", files)
	}
}

func TestDiscoverFiles_AlphabeticalOrder(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "z-config.yaml", "a: 1")
	createFile(t, dir, "a-deploy.yaml", "b: 2")
	createFile(t, dir, "m-service.yml", "c: 3")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"a-deploy.yaml", "m-service.yml", "z-config.yaml"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}

func TestDiscoverFiles_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "top.yaml", "key: value")
	// Create a subdirectory — should be skipped
	subdir := filepath.Join(dir, "nested")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != "top.yaml" {
		t.Errorf("expected [top.yaml], got %v", files)
	}
}

func TestDiscoverFiles_NonExistentDirectory(t *testing.T) {
	_, err := DiscoverFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestDiscoverFiles_ReturnsBaseNamesOnly(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "deploy.yaml", "key: value")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	// Should be just the base name, not a full path
	if files[0] != "deploy.yaml" {
		t.Errorf("expected base name 'deploy.yaml', got %q", files[0])
	}
}

func TestDiscoverFiles_ExtensionlessKubectlFiles(t *testing.T) {
	dir := t.TempDir()

	// Simulate kubectl temp file naming (no .yaml/.yml extension)
	createFile(t, dir, "apps.v1.Deployment.default.nginx", "apiVersion: apps/v1\nkind: Deployment")
	createFile(t, dir, "v1.Service.default.nginx", "apiVersion: v1\nkind: Service")
	createFile(t, dir, "regular.yaml", "key: value")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"apps.v1.Deployment.default.nginx", "regular.yaml", "v1.Service.default.nginx"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}
