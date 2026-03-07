package diffyml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Task 1.1: IsDirectory tests ---

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

// --- Task 1.2: DiscoverFiles tests ---

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

// --- Task 2: BuildFilePairPlan tests ---

func TestBuildFilePairPlan_BothExist(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "a: 1")
	createFile(t, toDir, "deploy.yaml", "a: 2")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	p := pairs[0]
	if p.Name != "deploy.yaml" {
		t.Errorf("expected Name='deploy.yaml', got %q", p.Name)
	}
	if p.Type != FilePairBothExist {
		t.Errorf("expected Type=FilePairBothExist, got %d", p.Type)
	}
	if p.FromPath == "" || p.ToPath == "" {
		t.Errorf("expected both paths non-empty, got FromPath=%q, ToPath=%q", p.FromPath, p.ToPath)
	}
}

func TestBuildFilePairPlan_OnlyFrom(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "removed.yaml", "gone: true")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	p := pairs[0]
	if p.Name != "removed.yaml" {
		t.Errorf("expected Name='removed.yaml', got %q", p.Name)
	}
	if p.Type != FilePairOnlyFrom {
		t.Errorf("expected Type=FilePairOnlyFrom, got %d", p.Type)
	}
	if p.FromPath == "" {
		t.Error("expected FromPath non-empty")
	}
	if p.ToPath != "" {
		t.Errorf("expected ToPath empty, got %q", p.ToPath)
	}
}

func TestBuildFilePairPlan_OnlyTo(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, toDir, "added.yml", "new: true")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	p := pairs[0]
	if p.Name != "added.yml" {
		t.Errorf("expected Name='added.yml', got %q", p.Name)
	}
	if p.Type != FilePairOnlyTo {
		t.Errorf("expected Type=FilePairOnlyTo, got %d", p.Type)
	}
	if p.FromPath != "" {
		t.Errorf("expected FromPath empty, got %q", p.FromPath)
	}
	if p.ToPath == "" {
		t.Error("expected ToPath non-empty")
	}
}

func TestBuildFilePairPlan_MixedScenario(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "both.yaml", "a: 1")
	createFile(t, toDir, "both.yaml", "a: 2")
	createFile(t, fromDir, "removed.yaml", "b: 1")
	createFile(t, toDir, "added.yaml", "c: 1")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d: %+v", len(pairs), pairs)
	}

	// Should be sorted alphabetically
	expected := []struct {
		name     string
		pairType FilePairType
	}{
		{"added.yaml", FilePairOnlyTo},
		{"both.yaml", FilePairBothExist},
		{"removed.yaml", FilePairOnlyFrom},
	}

	for i, exp := range expected {
		if pairs[i].Name != exp.name {
			t.Errorf("pairs[%d].Name = %q, want %q", i, pairs[i].Name, exp.name)
		}
		if pairs[i].Type != exp.pairType {
			t.Errorf("pairs[%d].Type = %d, want %d", i, pairs[i].Type, exp.pairType)
		}
	}
}

func TestBuildFilePairPlan_AlphabeticalSorting(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "z.yaml", "a: 1")
	createFile(t, fromDir, "a.yaml", "b: 2")
	createFile(t, fromDir, "m.yaml", "c: 3")
	createFile(t, toDir, "z.yaml", "a: 1")
	createFile(t, toDir, "a.yaml", "b: 2")
	createFile(t, toDir, "m.yaml", "c: 3")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	expectedNames := []string{"a.yaml", "m.yaml", "z.yaml"}
	for i, name := range expectedNames {
		if pairs[i].Name != name {
			t.Errorf("pairs[%d].Name = %q, want %q", i, pairs[i].Name, name)
		}
	}
}

func TestBuildFilePairPlan_BothDirectoriesEmpty(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 0 {
		t.Errorf("expected empty pairs, got %v", pairs)
	}
}

func TestBuildFilePairPlan_OneDirectoryEmpty(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "a: 1")
	createFile(t, fromDir, "service.yaml", "b: 2")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	for _, p := range pairs {
		if p.Type != FilePairOnlyFrom {
			t.Errorf("expected FilePairOnlyFrom for %q, got %d", p.Name, p.Type)
		}
	}
}

func TestBuildFilePairPlan_ErrorOnInvalidFromDir(t *testing.T) {
	toDir := t.TempDir()
	_, err := BuildFilePairPlan("/nonexistent/dir", toDir)
	if err == nil {
		t.Error("expected error for non-existent from directory")
	}
}

func TestBuildFilePairPlan_ErrorOnInvalidToDir(t *testing.T) {
	fromDir := t.TempDir()
	_, err := BuildFilePairPlan(fromDir, "/nonexistent/dir")
	if err == nil {
		t.Error("expected error for non-existent to directory")
	}
}

func TestBuildFilePairPlan_FullPaths(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "a: 1")
	createFile(t, toDir, "deploy.yaml", "a: 2")

	pairs, err := BuildFilePairPlan(fromDir, toDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}

	expectedFrom := fromDir + "/deploy.yaml"
	expectedTo := toDir + "/deploy.yaml"
	if pairs[0].FromPath != expectedFrom {
		t.Errorf("FromPath = %q, want %q", pairs[0].FromPath, expectedFrom)
	}
	if pairs[0].ToPath != expectedTo {
		t.Errorf("ToPath = %q, want %q", pairs[0].ToPath, expectedTo)
	}
}

// --- Task 3.1: FormatFileHeader tests ---

func TestFormatFileHeader_BothExist_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("deploy.yaml", FilePairBothExist, opts)

	if !strings.Contains(header, "--- a/deploy.yaml") {
		t.Errorf("expected '--- a/deploy.yaml' in header, got: %q", header)
	}
	if !strings.Contains(header, "+++ b/deploy.yaml") {
		t.Errorf("expected '+++ b/deploy.yaml' in header, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyFrom_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("removed.yaml", FilePairOnlyFrom, opts)

	if !strings.Contains(header, "--- a/removed.yaml") {
		t.Errorf("expected '--- a/removed.yaml' in header, got: %q", header)
	}
	if !strings.Contains(header, "+++ /dev/null") {
		t.Errorf("expected '+++ /dev/null' in header, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyTo_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("added.yaml", FilePairOnlyTo, opts)

	if !strings.Contains(header, "--- /dev/null") {
		t.Errorf("expected '--- /dev/null' in header, got: %q", header)
	}
	if !strings.Contains(header, "+++ b/added.yaml") {
		t.Errorf("expected '+++ b/added.yaml' in header, got: %q", header)
	}
}

func TestFormatFileHeader_BothExist_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	header := FormatFileHeader("deploy.yaml", FilePairBothExist, opts)

	// Should contain ANSI bold+white for both "---" and "+++"
	if !strings.Contains(header, "\033[1m") {
		t.Errorf("expected bold ANSI code in colored header, got: %q", header)
	}
	if !strings.Contains(header, "\033[37m") {
		t.Errorf("expected white ANSI code in colored header, got: %q", header)
	}
	if !strings.Contains(header, "--- a/deploy.yaml") {
		t.Errorf("expected '--- a/deploy.yaml' in header, got: %q", header)
	}
	if !strings.Contains(header, "\033[0m") {
		t.Errorf("expected reset ANSI code in colored header, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyFrom_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	header := FormatFileHeader("removed.yaml", FilePairOnlyFrom, opts)

	if !strings.Contains(header, "+++ /dev/null") {
		t.Errorf("expected '+++ /dev/null' in header, got: %q", header)
	}
	if !strings.Contains(header, "\033[1m") {
		t.Errorf("expected bold ANSI code, got: %q", header)
	}
}

func TestFormatFileHeader_OnlyTo_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	header := FormatFileHeader("added.yaml", FilePairOnlyTo, opts)

	if !strings.Contains(header, "--- /dev/null") {
		t.Errorf("expected '--- /dev/null' in header, got: %q", header)
	}
	if !strings.Contains(header, "\033[1m") {
		t.Errorf("expected bold ANSI code, got: %q", header)
	}
}

func TestFormatFileHeader_EndsWithNewline(t *testing.T) {
	opts := &FormatOptions{Color: false}
	header := FormatFileHeader("test.yaml", FilePairBothExist, opts)

	if !strings.HasSuffix(header, "\n") {
		t.Errorf("expected header to end with newline, got: %q", header)
	}
}

// Helper to create a file in a directory.
func createFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
