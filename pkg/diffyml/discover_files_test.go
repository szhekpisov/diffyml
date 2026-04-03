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

func TestDiscoverFiles_RecursesIntoSubdirectories(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "top.yaml", "key: value")
	createFile(t, dir, "nested/inner.yaml", "key: nested")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"nested/inner.yaml", "top.yaml"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}

func TestDiscoverFiles_NonExistentDirectory(t *testing.T) {
	_, err := DiscoverFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestDiscoverFiles_ReturnsRelativePaths(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "deploy.yaml", "key: value")
	createFile(t, dir, "sub/nested.yaml", "key: nested")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	// Flat files return base name (which is the relative path)
	if files[0] != "deploy.yaml" {
		t.Errorf("expected 'deploy.yaml', got %q", files[0])
	}
	// Nested files return forward-slash relative paths
	if files[1] != "sub/nested.yaml" {
		t.Errorf("expected 'sub/nested.yaml', got %q", files[1])
	}
}

func TestDiscoverFiles_NestedFiles(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "ns-a/deploy.yaml", "a: 1")
	createFile(t, dir, "ns-a/service.yaml", "b: 2")
	createFile(t, dir, "ns-b/deploy.yaml", "c: 3")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"ns-a/deploy.yaml", "ns-a/service.yaml", "ns-b/deploy.yaml"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}

func TestDiscoverFiles_DeepNesting(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "a/b/c/deep.yaml", "key: deep")
	createFile(t, dir, "a/b/mid.yaml", "key: mid")
	createFile(t, dir, "a/top.yaml", "key: top")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"a/b/c/deep.yaml", "a/b/mid.yaml", "a/top.yaml"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}

func TestDiscoverFiles_MixedTopAndNestedFiles(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "top.yaml", "a: 1")
	createFile(t, dir, "ns-a/deploy.yaml", "b: 2")
	createFile(t, dir, "ns-a/service.yaml", "c: 3")
	createFile(t, dir, "zebra.yaml", "d: 4")

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Lexicographic sort: "/" (0x2F) < lowercase letters,
	// so "ns-a/..." sorts before "top.yaml" — consistent with git diff.
	expected := []string{"ns-a/deploy.yaml", "ns-a/service.yaml", "top.yaml", "zebra.yaml"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}

func TestDiscoverFiles_UnreadableSubdirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	dir := t.TempDir()
	createFile(t, dir, "top.yaml", "key: value")

	restricted := filepath.Join(dir, "restricted")
	if err := os.Mkdir(restricted, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(restricted, 0o755) })

	_, err := DiscoverFiles(dir)
	if err == nil {
		t.Error("expected error for unreadable subdirectory")
	}
}

func TestDiscoverFiles_SkipsSymlinks(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "real.yaml", "a: 1")

	// Create a circular symlink directory — must not cause infinite recursion
	if err := os.Symlink(dir, filepath.Join(dir, "loop")); err != nil {
		t.Fatal(err)
	}
	// Create a symlink to a regular file — should also be skipped
	if err := os.Symlink(filepath.Join(dir, "real.yaml"), filepath.Join(dir, "link.yaml")); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != "real.yaml" {
		t.Errorf("expected [real.yaml], got %v", files)
	}
}

func TestDiscoverFiles_EmptySubdirectories(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "top.yaml", "key: value")
	if err := os.MkdirAll(filepath.Join(dir, "empty-sub"), 0o755); err != nil {
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
