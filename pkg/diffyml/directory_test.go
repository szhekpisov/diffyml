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
	f.Close()

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

// --- Task 1.2: DiscoverYAMLFiles tests ---

func TestDiscoverYAMLFiles_MixedFileTypes(t *testing.T) {
	dir := t.TempDir()

	// Create YAML files
	createFile(t, dir, "deploy.yaml", "key: value")
	createFile(t, dir, "service.yml", "key: value")
	// Create non-YAML files (should be skipped)
	createFile(t, dir, "readme.txt", "hello")
	createFile(t, dir, "config.json", "{}")

	files, err := DiscoverYAMLFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"deploy.yaml", "service.yml"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i, name := range expected {
		if files[i] != name {
			t.Errorf("expected files[%d]=%q, got %q", i, name, files[i])
		}
	}
}

func TestDiscoverYAMLFiles_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	files, err := DiscoverYAMLFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected empty list, got %v", files)
	}
}

func TestDiscoverYAMLFiles_AlphabeticalOrder(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "z-config.yaml", "a: 1")
	createFile(t, dir, "a-deploy.yaml", "b: 2")
	createFile(t, dir, "m-service.yml", "c: 3")

	files, err := DiscoverYAMLFiles(dir)
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

func TestDiscoverYAMLFiles_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "top.yaml", "key: value")
	// Create a subdirectory with a YAML-like name
	subdir := filepath.Join(dir, "nested.yaml")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverYAMLFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != "top.yaml" {
		t.Errorf("expected [top.yaml], got %v", files)
	}
}

func TestDiscoverYAMLFiles_NonExistentDirectory(t *testing.T) {
	_, err := DiscoverYAMLFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestDiscoverYAMLFiles_ReturnsBaseNamesOnly(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "deploy.yaml", "key: value")

	files, err := DiscoverYAMLFiles(dir)
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

func TestDiscoverYAMLFiles_OnlyYamlAndYmlExtensions(t *testing.T) {
	dir := t.TempDir()

	createFile(t, dir, "good.yaml", "a: 1")
	createFile(t, dir, "good.yml", "b: 2")
	createFile(t, dir, "bad.YAML", "c: 3")  // uppercase - should be skipped
	createFile(t, dir, "bad.YML", "d: 4")   // uppercase - should be skipped
	createFile(t, dir, "bad.yamlx", "e: 5") // wrong extension

	files, err := DiscoverYAMLFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"good.yaml", "good.yml"}
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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"

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
	cfg.Color = "off"
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
	cfg.Color = "off"
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
	cfg.Color = "off"
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
	cfg.Color = "off"
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
	cfg.Color = "off"
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
	cfg.Color = "off"
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
	cfg.Color = "off"
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
	cfg.Color = "off"
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
	cfg.Color = "off"
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

// Helper to create a file in a directory.
func createFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
