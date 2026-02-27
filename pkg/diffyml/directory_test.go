package diffyml

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// --- Task 3.2: runDirectory tests ---

func TestRunDirectory_IdenticalFiles_Exit0(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: value\n"), []byte("key: value\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0, got %d", result.Code)
	}
	if stdout.String() != "" {
		t.Errorf("expected no output, got: %q", stdout.String())
	}
}

func TestRunDirectory_ModifiedFile_Exit1(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	if !strings.Contains(output, "--- a/deploy.yaml") {
		t.Errorf("expected file header in output, got: %q", output)
	}
	if !strings.Contains(output, "+++ b/deploy.yaml") {
		t.Errorf("expected file header in output, got: %q", output)
	}
}

func TestRunDirectory_AddedFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"new.yaml": {nil, []byte("key: value\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	if !strings.Contains(output, "--- /dev/null") {
		t.Errorf("expected /dev/null in from header, got: %q", output)
	}
	if !strings.Contains(output, "+++ b/new.yaml") {
		t.Errorf("expected '+++ b/new.yaml' in header, got: %q", output)
	}
}

func TestRunDirectory_RemovedFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"old.yaml": {[]byte("key: value\n"), nil},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	if !strings.Contains(output, "--- a/old.yaml") {
		t.Errorf("expected '--- a/old.yaml' in header, got: %q", output)
	}
	if !strings.Contains(output, "+++ /dev/null") {
		t.Errorf("expected '+++ /dev/null' in header, got: %q", output)
	}
}

func TestRunDirectory_MixedFiles(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"added.yaml":    {nil, []byte("new: true\n")},
		"modified.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
		"removed.yaml":  {[]byte("gone: true\n"), nil},
		"same.yaml":     {[]byte("no: change\n"), []byte("no: change\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	// Check alphabetical order of file headers
	addedIdx := strings.Index(output, "--- /dev/null")
	modifiedIdx := strings.Index(output, "--- a/modified.yaml")
	removedIdx := strings.Index(output, "--- a/removed.yaml")
	if addedIdx >= modifiedIdx || modifiedIdx >= removedIdx {
		t.Errorf("expected alphabetical file order: added < modified < removed, got indices: %d, %d, %d",
			addedIdx, modifiedIdx, removedIdx)
	}
	// same.yaml should have no output (no diffs)
	if strings.Contains(output, "same.yaml") {
		t.Errorf("expected no output for identical file 'same.yaml', got: %q", output)
	}
}

func TestRunDirectory_NoDiffs_NoSetExitCode_Exit0(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = false
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	// Without --set-exit-code, always exit 0
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 without --set-exit-code, got %d", result.Code)
	}
}

func TestRunDirectory_EmptyDirectories_Exit0(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for empty directories, got %d", result.Code)
	}
}

func TestRunDirectory_OmitHeader_SuppressesSummaryButKeepsFileHeaders(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.OmitHeader = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	// File headers should still be present
	if !strings.Contains(output, "--- a/deploy.yaml") {
		t.Errorf("expected file header present even with --omit-header, got: %q", output)
	}
}

func TestRunDirectory_ParseError_ContinuesProcessing(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"bad.yaml":  {[]byte("key: old\n"), []byte(":\nbad yaml [[[")},
		"good.yaml": {[]byte("a: 1\n"), []byte("a: 2\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	// good.yaml has diffs, so exit 1 (diffs take precedence over errors)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 (diffs found despite error), got %d", result.Code)
	}
	// Error should be logged to stderr
	if !strings.Contains(stderr.String(), "bad.yaml") {
		t.Errorf("expected error mentioning 'bad.yaml' in stderr, got: %q", stderr.String())
	}
	// good.yaml should still be processed
	if !strings.Contains(stdout.String(), "good.yaml") {
		t.Errorf("expected good.yaml output, got: %q", stdout.String())
	}
}

func TestRunDirectory_ParseError_OnlyErrors_Exit255(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"bad.yaml": {[]byte("key: old\n"), []byte(":\nbad yaml [[[")},
	}

	result := runDirectory(cfg, rc, "", "")
	// Only errors, no diffs → exit 255
	if result.Code != ExitCodeError {
		t.Errorf("expected exit 255 (only errors), got %d", result.Code)
	}
}

// --- Task 5.1: Integration tests for flag compatibility in directory mode ---

func TestRunDirectory_IgnoreOrderChanges(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.IgnoreOrderChanges = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {
			[]byte("items:\n  - a\n  - b\n"),
			[]byte("items:\n  - b\n  - a\n"),
		},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 with --ignore-order-changes, got %d", result.Code)
	}
	if stdout.String() != "" {
		t.Errorf("expected no output when order changes are ignored, got: %q", stdout.String())
	}
}

func TestRunDirectory_FilterFlag(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Filter = []string{"config.a"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {
			[]byte("config:\n  a: 1\n  b: 2\n"),
			[]byte("config:\n  a: 10\n  b: 20\n"),
		},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 with filtered diffs, got %d", result.Code)
	}
	output := stdout.String()
	if !strings.Contains(output, "config.a") {
		t.Errorf("expected config.a in filtered output, got: %q", output)
	}
	if strings.Contains(output, "config.b") {
		t.Errorf("expected config.b to be filtered out, got: %q", output)
	}
}

func TestRunDirectory_ExcludeFlag(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Exclude = []string{"config.secret"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {
			[]byte("config:\n  name: app\n  secret: old-pass\n"),
			[]byte("config:\n  name: app-v2\n  secret: new-pass\n"),
		},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 with non-excluded diffs, got %d", result.Code)
	}
	output := stdout.String()
	if strings.Contains(output, "secret") {
		t.Errorf("expected config.secret to be excluded, got: %q", output)
	}
	if !strings.Contains(output, "config.name") {
		t.Errorf("expected config.name in output, got: %q", output)
	}
}

func TestRunDirectory_OutputFormat_Compact(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	// Compact format uses ± for modifications
	if !strings.Contains(output, "±") {
		t.Errorf("expected compact format with ± indicator, got: %q", output)
	}
}

func TestRunDirectory_OutputFormat_Brief(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "brief"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	// Brief format shows counts
	if !strings.Contains(output, "±") && !strings.Contains(output, "modified") {
		t.Errorf("expected brief format output, got: %q", output)
	}
}

func TestRunDirectory_OutputFormat_GitHub(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "github"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	// GitHub format uses ::warning :: annotations
	if !strings.Contains(output, "::warning") {
		t.Errorf("expected GitHub Actions format with ::warning, got: %q", output)
	}
}

func TestRunDirectory_SwapDirectories(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Swap = true
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: from_val\n"), []byte("key: to_val\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 with swap, got %d", result.Code)
	}
	output := stdout.String()
	// With swap, from and to are reversed
	if !strings.Contains(output, "from_val") || !strings.Contains(output, "to_val") {
		t.Errorf("expected both values in swapped output, got: %q", output)
	}
}

func TestRunDirectory_MultipleFilesWithFilter(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Filter = []string{"name"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {
			[]byte("name: app1\nreplicas: 1\n"),
			[]byte("name: app2\nreplicas: 3\n"),
		},
		"service.yaml": {
			[]byte("name: svc1\nport: 80\n"),
			[]byte("name: svc2\nport: 443\n"),
		},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	// Both files should have name diffs
	if !strings.Contains(output, "deploy.yaml") {
		t.Errorf("expected deploy.yaml header in output, got: %q", output)
	}
	if !strings.Contains(output, "service.yaml") {
		t.Errorf("expected service.yaml header in output, got: %q", output)
	}
	// replicas and port should be filtered out
	if strings.Contains(output, "replicas") {
		t.Errorf("expected replicas to be filtered out, got: %q", output)
	}
	if strings.Contains(output, "port") {
		t.Errorf("expected port to be filtered out, got: %q", output)
	}
}

func TestRunDirectory_ExcludeAllDiffs_Exit0(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Exclude = []string{"key"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	// All diffs excluded → exit 0
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 when all diffs excluded, got %d", result.Code)
	}
}

// --- Task 3.2: Structured formatter in directory mode ---

func TestRunDirectory_GitLab_SingleJSONArray(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml":  {[]byte("key: old\n"), []byte("key: new\n")},
		"service.yaml": {[]byte("port: 80\n"), []byte("port: 443\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}

	output := stdout.String()
	// Must be a single valid JSON array
	var findings []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &findings); err != nil {
		t.Fatalf("output is not valid JSON array: %v\noutput: %s", err, output)
	}
	if len(findings) != 2 {
		t.Errorf("expected 2 findings in single array, got %d", len(findings))
	}
}

func TestRunDirectory_GitLab_NoFileHeaders(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	// Should NOT contain unified-diff-style headers
	if strings.Contains(output, "--- a/") {
		t.Errorf("expected no file headers for GitLab format, got: %s", output)
	}
	if strings.Contains(output, "+++ b/") {
		t.Errorf("expected no file headers for GitLab format, got: %s", output)
	}
}

func TestRunDirectory_GitLab_EmptyProducesEmptyArray(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: same\n"), []byte("key: same\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0, got %d", result.Code)
	}

	output := stdout.String()
	if strings.TrimSpace(output) != "[]" {
		t.Errorf("expected empty JSON array for no diffs, got: %q", output)
	}
}

func TestRunDirectory_GitLab_EmptyDirectoriesProducesEmptyArray(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0, got %d", result.Code)
	}

	output := stdout.String()
	if strings.TrimSpace(output) != "[]" {
		t.Errorf("expected empty JSON array for empty directories, got: %q", output)
	}
}

func TestRunDirectory_GitLab_LocationPathIsFilePath(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	var findings []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &findings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	location := findings[0]["location"].(map[string]interface{})
	path := location["path"].(string)
	if path != "deploy.yaml" {
		t.Errorf("expected location.path 'deploy.yaml', got %q", path)
	}
}

func TestRunDirectory_GitLab_DescriptionIncludesFilename(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	var findings []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &findings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	desc := findings[0]["description"].(string)
	if !strings.Contains(desc, "deploy.yaml") {
		t.Errorf("expected filename in description, got: %s", desc)
	}
	if !strings.Contains(desc, "key") {
		t.Errorf("expected YAML path in description, got: %s", desc)
	}
}

func TestRunDirectory_GitLab_UniqueFingerprintsAcrossFiles(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	// Same change in two different files
	rc.FilePairs = map[string][2][]byte{
		"file1.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
		"file2.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	var findings []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &findings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	fp1 := findings[0]["fingerprint"].(string)
	fp2 := findings[1]["fingerprint"].(string)
	if fp1 == fp2 {
		t.Errorf("fingerprints should be unique across files, both got: %s", fp1)
	}
}

func TestRunDirectory_GitLab_StripsDotSlashFromPairName(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"./deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	// Should not contain ./ prefix in the output
	if strings.Contains(output, `"./deploy.yaml"`) {
		t.Errorf("expected ./ prefix stripped, got: %s", output)
	}
}

func TestRunDirectory_NonGitLab_StillHasFileHeaders(t *testing.T) {
	// Non-structured formatters should continue to get per-file headers
	formatters := []string{"compact", "brief"}

	for _, format := range formatters {
		t.Run(format, func(t *testing.T) {
			cfg := NewCLIConfig()
			cfg.SetExitCode = true
			cfg.Color = "never"
			cfg.Output = format

			rc := NewRunConfig()
			var stdout, stderr strings.Builder
			rc.Stdout = &stdout
			rc.Stderr = &stderr
			rc.FilePairs = map[string][2][]byte{
				"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
			}

			runDirectory(cfg, rc, "", "")

			output := stdout.String()
			if !strings.Contains(output, "--- a/deploy.yaml") {
				t.Errorf("expected file header for %s format, got: %q", format, output)
			}
		})
	}
}

func TestRunDirectory_GitLab_ExitCodePrecedence(t *testing.T) {
	// When diffs exist alongside errors, diffs take precedence
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"bad.yaml":  {[]byte("key: old\n"), []byte(":\nbad yaml [[[")},
		"good.yaml": {[]byte("a: 1\n"), []byte("a: 2\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	// good.yaml has diffs → exit 1 (diffs take precedence)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 (diffs found), got %d", result.Code)
	}
}

func TestRunDirectory_GitLab_OnlyErrors_Exit255(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"bad.yaml": {[]byte("key: old\n"), []byte(":\nbad yaml [[[")},
	}

	result := runDirectory(cfg, rc, "", "")
	// Only errors, no diffs → exit 255
	if result.Code != ExitCodeError {
		t.Errorf("expected exit 255 (only errors), got %d", result.Code)
	}
	// Should still output empty array
	output := stdout.String()
	if strings.TrimSpace(output) != "[]" {
		t.Errorf("expected empty JSON array when only errors, got: %q", output)
	}
}

// --- Task 4.2: Verify other formatters are unaffected in directory mode ---

func TestRunDirectory_NonGitLab_FileHeadersPresent(t *testing.T) {
	// All non-structured formatters must receive per-file headers in directory mode.
	formatters := []string{"compact", "brief", "detailed"}

	for _, format := range formatters {
		t.Run(format, func(t *testing.T) {
			cfg := NewCLIConfig()
			cfg.SetExitCode = true
			cfg.Color = "never"
			cfg.Output = format

			rc := NewRunConfig()
			var stdout, stderr strings.Builder
			rc.Stdout = &stdout
			rc.Stderr = &stderr
			rc.FilePairs = map[string][2][]byte{
				"deploy.yaml":  {[]byte("key: old\n"), []byte("key: new\n")},
				"service.yaml": {[]byte("port: 80\n"), []byte("port: 443\n")},
			}

			runDirectory(cfg, rc, "", "")

			output := stdout.String()
			// Both file headers should be present
			if !strings.Contains(output, "--- a/deploy.yaml") {
				t.Errorf("[%s] expected '--- a/deploy.yaml' header, got: %q", format, output)
			}
			if !strings.Contains(output, "+++ b/deploy.yaml") {
				t.Errorf("[%s] expected '+++ b/deploy.yaml' header, got: %q", format, output)
			}
			if !strings.Contains(output, "--- a/service.yaml") {
				t.Errorf("[%s] expected '--- a/service.yaml' header, got: %q", format, output)
			}
			if !strings.Contains(output, "+++ b/service.yaml") {
				t.Errorf("[%s] expected '+++ b/service.yaml' header, got: %q", format, output)
			}
		})
	}
}

func TestRunDirectory_NonGitLab_NotStructuredFormatter(t *testing.T) {
	// Verify that non-structured formatters do NOT implement StructuredFormatter.
	nonStructured := []string{"compact", "brief", "detailed"}

	for _, name := range nonStructured {
		t.Run(name, func(t *testing.T) {
			f, err := GetFormatter(name)
			if err != nil {
				t.Fatalf("failed to get formatter: %v", err)
			}
			if _, ok := f.(StructuredFormatter); ok {
				t.Errorf("formatter %q should NOT implement StructuredFormatter", name)
			}
		})
	}
}

func TestFormatter_FilePathFieldIgnoredByNonGitLab(t *testing.T) {
	// Setting FilePath on FormatOptions should not change the output
	// of formatters that do not use file-aware annotations.
	// GitHub and Gitea formatters intentionally use FilePath for file= parameter.
	nonGitLab := []string{"compact", "brief", "detailed"}

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: DiffAdded, To: 8080},
	}

	for _, name := range nonGitLab {
		t.Run(name, func(t *testing.T) {
			f, err := GetFormatter(name)
			if err != nil {
				t.Fatalf("failed to get formatter: %v", err)
			}

			optsWithout := DefaultFormatOptions()
			outputWithout := f.Format(diffs, optsWithout)

			optsWith := DefaultFormatOptions()
			optsWith.FilePath = "deploy.yaml"
			outputWith := f.Format(diffs, optsWith)

			if outputWithout != outputWith {
				t.Errorf("formatter %q output changed when FilePath is set\nwithout: %q\nwith:    %q",
					name, outputWithout, outputWith)
			}
		})
	}
}

func TestRunDirectory_NonGitLab_PerFileOutput(t *testing.T) {
	// Verify that non-GitLab formatters produce per-file formatted output
	// (not a single aggregated output) in directory mode.
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"a.yaml": {[]byte("key: old1\n"), []byte("key: new1\n")},
		"b.yaml": {[]byte("key: old2\n"), []byte("key: new2\n")},
	}

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	// Each file's output should be preceded by its header
	aHeaderIdx := strings.Index(output, "--- a/a.yaml")
	bHeaderIdx := strings.Index(output, "--- a/b.yaml")
	if aHeaderIdx < 0 {
		t.Fatalf("missing header for a.yaml")
	}
	if bHeaderIdx < 0 {
		t.Fatalf("missing header for b.yaml")
	}
	// a.yaml header should come before b.yaml header (alphabetical)
	if aHeaderIdx >= bHeaderIdx {
		t.Errorf("expected a.yaml header before b.yaml header, got indices: a=%d, b=%d", aHeaderIdx, bHeaderIdx)
	}
}

func TestRunDirectory_DetailedFormatter_FileHeadersInDirectoryMode(t *testing.T) {
	// Detailed formatter specifically should have file headers in directory mode.
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "detailed"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"config.yaml": {[]byte("app:\n  name: old\n"), []byte("app:\n  name: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}
	output := stdout.String()
	if !strings.Contains(output, "--- a/config.yaml") {
		t.Errorf("expected file header for detailed format, got: %q", output)
	}
}

// --- Task 3.1/3.2: GitHub/Gitea structured formatter in directory mode ---

func TestRunDirectory_GitHub_StructuredOutput(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "github"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml":  {[]byte("key: old\n"), []byte("key: new\n")},
		"service.yaml": {[]byte("port: 80\n"), []byte("port: 443\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}

	output := stdout.String()

	// Should NOT have unified-diff file headers (structured mode)
	if strings.Contains(output, "--- a/") {
		t.Errorf("expected no file headers for GitHub structured format, got: %s", output)
	}
	if strings.Contains(output, "+++ b/") {
		t.Errorf("expected no file headers for GitHub structured format, got: %s", output)
	}

	// Should include file= parameter with correct file names
	if !strings.Contains(output, "file=deploy.yaml") {
		t.Errorf("expected file=deploy.yaml in output, got: %s", output)
	}
	if !strings.Contains(output, "file=service.yaml") {
		t.Errorf("expected file=service.yaml in output, got: %s", output)
	}

	// Should have ::warning commands
	if !strings.Contains(output, "::warning") {
		t.Errorf("expected ::warning in output, got: %s", output)
	}
}

func TestRunDirectory_GitHub_EmptyProducesEmptyString(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "github"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: same\n"), []byte("key: same\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0, got %d", result.Code)
	}

	output := stdout.String()
	if output != "" {
		t.Errorf("expected empty output for no diffs, got: %q", output)
	}
}

func TestRunDirectory_Gitea_StructuredOutput(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "gitea"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1, got %d", result.Code)
	}

	output := stdout.String()

	// Should NOT have unified-diff file headers
	if strings.Contains(output, "--- a/") {
		t.Errorf("expected no file headers for Gitea structured format, got: %s", output)
	}

	// Should include file= parameter
	if !strings.Contains(output, "file=deploy.yaml") {
		t.Errorf("expected file=deploy.yaml in output, got: %s", output)
	}
}

func TestRunDirectory_GitHub_StripsDotSlashFromPairName(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.SetExitCode = true
	cfg.Color = "never"
	cfg.Output = "github"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"./deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	if strings.Contains(output, "file=./deploy.yaml") {
		t.Errorf("expected ./ prefix stripped from file parameter, got: %s", output)
	}
	if !strings.Contains(output, "file=deploy.yaml") {
		t.Errorf("expected file=deploy.yaml (without ./), got: %s", output)
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

// --- Task 3.2: Wire summarizer into directory comparison mode ---

func TestRunDirectory_WithSummary_NonStructured(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Multiple files were modified."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml":  {[]byte("key: old\n"), []byte("key: new\n")},
		"service.yaml": {[]byte("port: 80\n"), []byte("port: 443\n")},
	}
	rc.SummaryAPIURL = server.URL

	result := runDirectory(cfg, rc, "", "")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should have per-file diff output
	if !strings.Contains(output, "deploy.yaml") {
		t.Error("expected deploy.yaml in output")
	}
	// Should have AI summary
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header, got: %s", output)
	}
	if !strings.Contains(output, "Multiple files were modified.") {
		t.Errorf("expected summary text, got: %s", output)
	}
}

func TestRunDirectory_WithSummary_Structured(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Structured summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "github"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	result := runDirectory(cfg, rc, "", "")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header for structured formatter, got: %s", output)
	}
}

func TestRunDirectory_WithSummary_NoDiffs_NoAPICall(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: same\n"), []byte("key: same\n")},
	}
	rc.SummaryAPIURL = server.URL

	runDirectory(cfg, rc, "", "")

	if apiCalled {
		t.Error("API should not be called when no files have differences")
	}
}

func TestRunDirectory_WithSummary_APIFailure_Warning(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"internal error"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	result := runDirectory(cfg, rc, "", "")

	// Exit code should not be affected
	if result.Code == ExitCodeError {
		t.Errorf("expected exit code to not be error, got %d", result.Code)
	}
	// Warning on stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
	// Standard output should still be present
	if !strings.Contains(stdout.String(), "deploy.yaml") {
		t.Error("expected standard directory output despite API failure")
	}
}

func TestRunDirectory_BriefSummary_ReplacesOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Directory changes summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "brief"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	// Should have AI summary
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header, got: %s", output)
	}
	// Should NOT have brief format markers (per-file)
	if strings.Contains(output, "±") {
		t.Errorf("expected brief output suppressed, but found '±' in: %s", output)
	}
}

func TestRunDirectory_BriefSummary_FallbackOnAPIFailure(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"fail"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "brief"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	// Should fall back to brief output
	if !strings.Contains(output, "deploy.yaml") {
		t.Errorf("expected brief fallback output with file header, got: %s", output)
	}
	// Warning on stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
}

func TestRunDirectory_WithSummary_PreservesExitCode(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --set-exit-code, got %d", ExitCodeDifferences, result.Code)
	}
}

// --- Task 4.2: End-to-end integration tests for summary in directory mode ---

func TestRunDirectory_WithoutSummary_NoAPICall(t *testing.T) {
	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = false
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	runDirectory(cfg, rc, "", "")

	if apiCalled {
		t.Error("API should not be called when --summary is not set in directory mode")
	}
	// Standard output should still be present
	if !strings.Contains(stdout.String(), "deploy.yaml") {
		t.Error("expected standard directory output")
	}
}

func TestRunDirectory_WithSummary_GitLab_AppendsSummary(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"GitLab structured summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "gitlab"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml":  {[]byte("key: old\n"), []byte("key: new\n")},
		"service.yaml": {[]byte("port: 80\n"), []byte("port: 443\n")},
	}
	rc.SummaryAPIURL = server.URL

	result := runDirectory(cfg, rc, "", "")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// GitLab JSON output should be valid
	var findings []map[string]interface{}
	// The output may contain both the JSON array and the AI summary
	// Split at the AI Summary header
	jsonEnd := strings.Index(output, "\nAI Summary:")
	jsonPart := output
	if jsonEnd > 0 {
		jsonPart = output[:jsonEnd]
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &findings); err != nil {
		t.Fatalf("output JSON is not valid: %v\noutput: %s", err, output)
	}
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
	// Should contain AI summary
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header, got: %s", output)
	}
	if !strings.Contains(output, "GitLab structured summary.") {
		t.Errorf("expected summary text, got: %s", output)
	}
}

func TestRunDirectory_WithSummary_APIFailure_PreservesExitCodeWithSetExitCode(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"fail"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	result := runDirectory(cfg, rc, "", "")
	// Exit code should be 1 (differences), not 255 (error) despite API failure
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --set-exit-code and API failure, got %d",
			ExitCodeDifferences, result.Code)
	}
	// Warning should be on stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
	// Standard diff output should still be present
	if !strings.Contains(stdout.String(), "deploy.yaml") {
		t.Error("expected standard diff output despite API failure")
	}
}

func TestRunDirectory_WithSummary_MultipleFiles_SingleSummary(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	var promptContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 0 {
			promptContent = req.Messages[0].Content
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Multi-file summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml":  {[]byte("replicas: 1\n"), []byte("replicas: 3\n")},
		"service.yaml": {[]byte("port: 80\n"), []byte("port: 443\n")},
	}
	rc.SummaryAPIURL = server.URL

	runDirectory(cfg, rc, "", "")

	// Both files should be mentioned in the prompt
	if !strings.Contains(promptContent, "deploy.yaml") {
		t.Errorf("expected deploy.yaml in prompt, got: %s", promptContent)
	}
	if !strings.Contains(promptContent, "service.yaml") {
		t.Errorf("expected service.yaml in prompt, got: %s", promptContent)
	}
	// Only one summary section in output
	output := stdout.String()
	summaryCount := strings.Count(output, "AI Summary:")
	if summaryCount != 1 {
		t.Errorf("expected exactly 1 AI Summary header, got %d in: %s", summaryCount, output)
	}
}

// --- Mutation testing: directory.go ---

func TestRunDirectory_StructuredFormatter_ZeroDiffs_ExitCode(t *testing.T) {
	// directory.go:362 — when there are zero diffs with a structured formatter
	// and SetExitCode, exit code should be 0 and no summary should appear
	cfg := NewCLIConfig()
	cfg.Output = "github"
	cfg.SetExitCode = true
	cfg.Summary = true // summary enabled but should NOT trigger with 0 diffs
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	// Identical files → zero diffs
	rc.FilePairs = map[string][2][]byte{
		"same.yaml": {[]byte("key: val\n"), []byte("key: val\n")},
	}

	result := runDirectory(cfg, rc, "", "")

	if result.Code != 0 {
		t.Errorf("expected exit code 0 for identical files, got %d", result.Code)
	}
	if strings.Contains(stdout.String(), "AI Summary:") {
		t.Error("no AI Summary should appear when there are zero diffs")
	}
}

func TestRunDirectory_WithSummary_Gitea_AppendsSummary(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Gitea summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "gitea"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FilePairs = map[string][2][]byte{
		"deploy.yaml": {[]byte("key: old\n"), []byte("key: new\n")},
	}
	rc.SummaryAPIURL = server.URL

	runDirectory(cfg, rc, "", "")

	output := stdout.String()
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header for Gitea format, got: %s", output)
	}
	if !strings.Contains(output, "Gitea summary.") {
		t.Errorf("expected summary text, got: %s", output)
	}
}
