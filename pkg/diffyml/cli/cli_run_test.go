package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for main execution flow (Task 5.4)

func TestRunConfig_Defaults(t *testing.T) {
	rc := NewRunConfig()
	if rc == nil {
		t.Fatal("NewRunConfig() returned nil")
	}
	if rc.Stdout == nil {
		t.Error("expected Stdout to be initialized")
	}
	if rc.Stderr == nil {
		t.Error("expected Stderr to be initialized")
	}
}

func TestRun_MissingFromFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "/nonexistent/from.yaml"
	cfg.ToFile = "/nonexistent/to.yaml"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code %d for missing file, got %d", ExitCodeError, result.Code)
	}
	if result.Err == nil {
		t.Error("expected error for missing file")
	}
}

func TestRun_InvalidOutputFormat(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.Output = "invalid"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code %d for invalid format, got %d", ExitCodeError, result.Code)
	}
}

func TestRun_CompareIdenticalContent(t *testing.T) {
	yaml1 := "key: value\n"
	yaml2 := "key: value\n"

	cfg := NewCLIConfig()
	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d for identical content, got %d", ExitCodeSuccess, result.Code)
	}
}

func TestRun_CompareWithDifferences_NoSetExitCode(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.SetExitCode = false

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	// Without -s flag, should return 0 even with differences
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d without -s flag, got %d", ExitCodeSuccess, result.Code)
	}
}

func TestRun_CompareWithDifferences_WithSetExitCode(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	// With -s flag, should return 1 when differences found
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with -s flag and differences, got %d", ExitCodeDifferences, result.Code)
	}
}

func TestRun_OutputToStdout(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	Run(cfg, rc)

	output := stdout.String()
	if output == "" {
		t.Error("expected output to be written to stdout")
	}
	// Should contain difference info
	if !containsSubstr(output, "key") {
		t.Error("expected output to contain path 'key'")
	}
}

func TestRun_WithFiltering(t *testing.T) {
	yaml1 := "config:\n  key1: a\n  key2: b\n"
	yaml2 := "config:\n  key1: x\n  key2: y\n"

	cfg := NewCLIConfig()
	cfg.Filter = []string{"config.key1"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	Run(cfg, rc)

	output := stdout.String()
	// Should contain key1 but not key2
	if !containsSubstr(output, "key1") {
		t.Error("expected output to contain filtered path 'key1'")
	}
	if containsSubstr(output, "key2") {
		t.Error("expected output to NOT contain excluded path 'key2'")
	}
}

func TestRun_OmitHeader(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.OmitHeader = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	Run(cfg, rc)

	output := stdout.String()
	// Should NOT contain the header (which contains "Found X difference(s)")
	if containsSubstr(output, "Found") && containsSubstr(output, "difference(s)") {
		t.Error("expected header to be omitted")
	}
}

func TestRun_BriefOutput(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "brief"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	Run(cfg, rc)

	output := stdout.String()
	// Brief format should indicate a modification (± in streaming, "modified" in batch)
	if !containsSubstr(output, "±") && !containsSubstr(output, "modified") {
		t.Errorf("expected brief output to contain '±' or 'modified', got: %s", output)
	}
}

func TestRun_InvalidYAML(t *testing.T) {
	invalidYAML := "invalid: yaml: content:\n  - not valid"
	validYAML := "key: value\n"

	cfg := NewCLIConfig()
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(invalidYAML)
	rc.ToContent = []byte(validYAML)

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code %d for invalid YAML, got %d", ExitCodeError, result.Code)
	}
}

func TestRun_IgnoreOrderChanges(t *testing.T) {
	yaml1 := "items:\n  - a\n  - b\n"
	yaml2 := "items:\n  - b\n  - a\n"

	cfg := NewCLIConfig()
	cfg.IgnoreOrderChanges = true
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	// With ignore order, same elements in different order = no difference
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d when ignoring order changes, got %d", ExitCodeSuccess, result.Code)
	}
}

func TestRun_ShowHelp(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.ShowHelp = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d for help, got %d", ExitCodeSuccess, result.Code)
	}
	output := stdout.String()
	if !containsSubstr(output, "Usage:") {
		t.Error("expected help output to contain 'Usage:'")
	}
}

// --- Task 4: Wire directory mode into CLI entry point ---

func TestRun_BothDirectories_DispatchesToDirectoryMode(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "key: old\n")
	createFile(t, toDir, "deploy.yaml", "key: new\n")

	cfg := NewCLIConfig()
	cfg.FromFile = fromDir
	cfg.ToFile = toDir
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 for directory diffs, got %d; stderr: %s", result.Code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "--- a/deploy.yaml") {
		t.Errorf("expected directory-mode file header in output, got: %q", output)
	}
}

func TestRun_BothDirectories_NoDiffs_Exit0(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	createFile(t, fromDir, "deploy.yaml", "key: same\n")
	createFile(t, toDir, "deploy.yaml", "key: same\n")

	cfg := NewCLIConfig()
	cfg.FromFile = fromDir
	cfg.ToFile = toDir
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for identical directory content, got %d", result.Code)
	}
	if stdout.String() != "" {
		t.Errorf("expected no output for identical content, got: %q", stdout.String())
	}
}

func TestRun_MixedTypes_DirAndFile_Error(t *testing.T) {
	dir := t.TempDir()

	// Create a temporary file
	f, err := os.CreateTemp("", "testfile-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("key: value\n")
	_ = f.Close()
	defer os.Remove(f.Name())

	cfg := NewCLIConfig()
	cfg.FromFile = dir
	cfg.ToFile = f.Name()
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit 255 for mixed types, got %d", result.Code)
	}
	if !strings.Contains(stderr.String(), "both") || !strings.Contains(stderr.String(), "same type") {
		t.Errorf("expected error mentioning both arguments must be same type, got: %q", stderr.String())
	}
}

func TestRun_MixedTypes_FileAndDir_Error(t *testing.T) {
	dir := t.TempDir()

	f, err := os.CreateTemp("", "testfile-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("key: value\n")
	_ = f.Close()
	defer os.Remove(f.Name())

	cfg := NewCLIConfig()
	cfg.FromFile = f.Name()
	cfg.ToFile = dir
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit 255 for mixed types, got %d", result.Code)
	}
}

func TestRun_BothFiles_NoRegression(t *testing.T) {
	// Existing file-mode behavior should be completely unchanged
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 for file diffs, got %d", result.Code)
	}
	// Should NOT have directory-mode file headers
	output := stdout.String()
	if strings.Contains(output, "--- a/") {
		t.Errorf("expected no directory-mode headers in file mode, got: %q", output)
	}
}

func TestRun_PreloadedContent_SkipsDirectoryDetection(t *testing.T) {
	// When FromContent/ToContent are pre-loaded, directory detection should be skipped
	// even if FromFile/ToFile happen to be directories
	dir := t.TempDir()

	cfg := NewCLIConfig()
	cfg.FromFile = dir
	cfg.ToFile = dir
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte("key: value1\n")
	rc.ToContent = []byte("key: value2\n")

	result := Run(cfg, rc)
	// Should use file-mode with pre-loaded content, not directory mode
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit 1 for preloaded content diffs, got %d", result.Code)
	}
	// No directory-mode headers
	output := stdout.String()
	if strings.Contains(output, "--- a/") {
		t.Errorf("expected no directory-mode headers with preloaded content, got: %q", output)
	}
}

// --- Task 3.1: File path normalization in single-file mode ---

func TestRun_GitLab_SetsFilePathFromToFile(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	cfg.ToFile = "deploy.yaml"

	Run(cfg, rc)

	output := stdout.String()
	// location.path should be the file path, not the YAML key path
	if !strings.Contains(output, `"path": "deploy.yaml"`) {
		t.Errorf("expected location.path 'deploy.yaml' in output, got: %s", output)
	}
}

func TestRun_GitLab_StripsDotSlashPrefix(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	cfg.ToFile = "./deploy.yaml"

	Run(cfg, rc)

	output := stdout.String()
	// Should strip ./ prefix
	if !strings.Contains(output, `"path": "deploy.yaml"`) {
		t.Errorf("expected ./ prefix stripped from path, got: %s", output)
	}
	if strings.Contains(output, `"path": "./deploy.yaml"`) {
		t.Errorf("expected ./ prefix to be stripped, got: %s", output)
	}
}

func TestRun_GitLab_ConvertsAbsoluteToRelative(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	cfg.ToFile = filepath.Join(cwd, "deploy.yaml")

	Run(cfg, rc)

	output := stdout.String()
	// Should be relative path, not absolute
	if !strings.Contains(output, `"path": "deploy.yaml"`) {
		t.Errorf("expected absolute path converted to relative, got: %s", output)
	}
}

func TestRun_GitLab_FallbackOnParentTraversingPath(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	// Use an absolute path that's outside CWD, which will produce ../..
	cfg.ToFile = "/nonexistent/outside/deploy.yaml"

	Run(cfg, rc)

	output := stdout.String()
	// Parse the JSON to verify it's valid and has a path
	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}
	location := result[0]["location"].(map[string]any)
	path := location["path"].(string)
	// Path should either be the original or converted — but never empty
	if path == "" {
		t.Error("location.path should not be empty")
	}
	// Absolute path outside CWD should not produce a warning
	if strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected no warning on stderr, got: %q", stderr.String())
	}
}

// --- GIT_EXTERNAL_DIFF integration tests ---

func TestRun_GitExternalDiff_YAMLWithDiffs_UsesDisplayPath(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "charts/deploy.yaml"
	cfg.Output = "gitlab"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "charts/deploy.yaml") {
		t.Errorf("expected display path 'charts/deploy.yaml' in output, got: %s", output)
	}
}

func TestRun_GitExternalDiff_FileHeader(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "charts/deploy.yaml"
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "--- a/charts/deploy.yaml") {
		t.Errorf("expected '--- a/' header, got: %s", output)
	}
	if !strings.Contains(output, "+++ b/charts/deploy.yaml") {
		t.Errorf("expected '+++ b/' header, got: %s", output)
	}
}

func TestRun_GitExternalDiff_FileHeader_NewFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.FromFile = "/dev/null"
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte{}
	rc.ToContent = []byte("key: value\n")

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "--- /dev/null") {
		t.Errorf("expected '--- /dev/null' for new file, got: %s", output)
	}
	if !strings.Contains(output, "+++ b/deploy.yaml") {
		t.Errorf("expected '+++ b/' header for new file, got: %s", output)
	}
}

func TestRun_GitExternalDiff_FileHeader_DeletedFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.ToFile = "/dev/null"
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte("key: value\n")
	rc.ToContent = []byte{}

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "--- a/deploy.yaml") {
		t.Errorf("expected '--- a/' header for deleted file, got: %s", output)
	}
	if !strings.Contains(output, "+++ /dev/null") {
		t.Errorf("expected '+++ /dev/null' for deleted file, got: %s", output)
	}
}

func TestRun_GitExternalDiff_NonYAML_SkippedWithWarning(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "Makefile"
	cfg.FromFile = "/tmp/old"
	cfg.ToFile = "/tmp/new"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for non-YAML file, got %d", result.Code)
	}
	if stdout.String() != "" {
		t.Errorf("expected no stdout for non-YAML file, got: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Warning: skipping non-YAML file Makefile") {
		t.Errorf("expected warning on stderr, got: %q", stderr.String())
	}
}

func TestRun_GitExternalDiff_NonYAML_Extensions(t *testing.T) {
	for _, ext := range []string{".go", ".json", ".txt", ".md", ""} {
		t.Run("ext="+ext, func(t *testing.T) {
			cfg := NewCLIConfig()
			cfg.GitExternalDiff = true
			cfg.GitDisplayPath = "file" + ext

			rc := NewRunConfig()
			var stdout, stderr strings.Builder
			rc.Stdout = &stdout
			rc.Stderr = &stderr

			result := Run(cfg, rc)
			if result.Code != ExitCodeSuccess {
				t.Errorf("expected exit 0 for %q file, got %d", ext, result.Code)
			}
		})
	}
}

func TestRun_GitExternalDiff_DevNull_NewFile(t *testing.T) {
	yaml2 := "key: newvalue\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.FromFile = "/dev/null"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte{} // empty = /dev/null
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for new file diff, got %d; stderr: %s", result.Code, stderr.String())
	}
	if stdout.String() == "" {
		t.Error("expected diff output for new file")
	}
}

func TestRun_GitExternalDiff_DevNull_DeletedFile(t *testing.T) {
	yaml1 := "key: oldvalue\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.ToFile = "/dev/null"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte{} // empty = /dev/null

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for deleted file diff, got %d; stderr: %s", result.Code, stderr.String())
	}
	if stdout.String() == "" {
		t.Error("expected diff output for deleted file")
	}
}

func TestRun_GitExternalDiff_SetExitCode_Suppressed(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.SetExitCode = true // should be suppressed in git external diff mode

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 (--set-exit-code suppressed in git mode), got %d", result.Code)
	}
	if stdout.String() == "" {
		t.Error("expected diff output despite exit 0")
	}
}

func TestRun_GitExternalDiff_NoDiffs_Exit0(t *testing.T) {
	yaml := "key: same\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for no diffs, got %d", result.Code)
	}
}

func TestRun_GitExternalDiff_AutoForcesColor(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.Color = "auto" // default

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\033[") {
		t.Error("expected ANSI color codes in git external diff mode with color=auto")
	}
}

func TestRun_GitExternalDiff_ColorNeverRespected(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}
	if strings.Contains(stdout.String(), "\033[") {
		t.Error("expected no ANSI color codes with --color never")
	}
}

func TestRun_GitExternalDiff_AutoForcesTrueColor(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.TrueColor = "auto" // default

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\033[38;2;") {
		t.Error("expected 24-bit ANSI color codes in git external diff mode with truecolor=auto")
	}
}

func TestRun_GitExternalDiff_TrueColorNeverRespected(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "deploy.yaml"
	cfg.TrueColor = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v; stderr: %s", result.Err, stderr.String())
	}
	if strings.Contains(stdout.String(), "\033[38;2;") {
		t.Error("expected no 24-bit ANSI color codes with --truecolor never")
	}
}

func TestRun_GitExternalDiff_FullPipeline_ParseArgsThenRun(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	err := cfg.ParseArgs([]string{
		"--set-exit-code",
		"--output", "gitlab",
		"charts/deploy.yaml",       // name
		"/tmp/old-content",         // old-file
		"abc1234abc1234abc1234",    // old-hex
		"100644",                   // old-mode
		"/work/charts/deploy.yaml", // new-file
		"def5678def5678def5678",    // new-hex
		"100644",                   // new-mode
	})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}

	if !cfg.GitExternalDiff {
		t.Fatal("expected GitExternalDiff=true after ParseArgs")
	}
	if !cfg.SetExitCode {
		t.Fatal("expected SetExitCode=true after ParseArgs")
	}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	// --set-exit-code is suppressed in git external diff mode (git aborts on non-zero)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 (--set-exit-code suppressed), got %d; stderr: %s", result.Code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "charts/deploy.yaml") {
		t.Errorf("expected display path in output, got: %s", output)
	}
}

func TestRun_GitExternalDiff_ParseError_NonFatal(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "broken.yaml"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte("valid: yaml\n")
	rc.ToContent = []byte(":\n  bad:\n    - [unmatched")

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for parse error in git mode, got %d", result.Code)
	}
	if !strings.Contains(stderr.String(), "Warning: skipping broken.yaml") {
		t.Errorf("expected warning on stderr, got: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "Error:") {
		t.Errorf("expected no 'Error:' prefix in git mode, got: %q", stderr.String())
	}
}

func TestRun_GitExternalDiff_ReadError_NonFatal(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "missing.yaml"
	cfg.FromFile = "/nonexistent/path/old.yaml"
	cfg.ToFile = "/nonexistent/path/new.yaml"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit 0 for read error in git mode, got %d", result.Code)
	}
	if !strings.Contains(stderr.String(), "Warning: skipping missing.yaml") {
		t.Errorf("expected warning on stderr, got: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "Error:") {
		t.Errorf("expected no 'Error:' prefix in git mode, got: %q", stderr.String())
	}
}

func TestRun_GitExternalDiff_RenameHeader(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitOriginalPath = "old-name.yaml"
	cfg.GitDisplayPath = "new-name.yaml"
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Fatalf("expected exit 0, got %d: %v", result.Code, result.Err)
	}
	out := stdout.String()
	if !strings.Contains(out, "--- a/old-name.yaml") {
		t.Errorf("expected '--- a/old-name.yaml' in header, got: %q", out)
	}
	if !strings.Contains(out, "+++ b/new-name.yaml") {
		t.Errorf("expected '+++ b/new-name.yaml' in header, got: %q", out)
	}
}

func TestRun_GitExternalDiff_NonYAML_Warning(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.GitExternalDiff = true
	cfg.GitDisplayPath = "config.json"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Fatalf("expected exit 0, got %d", result.Code)
	}
	if stdout.String() != "" {
		t.Errorf("expected no stdout, got: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Warning: skipping non-YAML file config.json") {
		t.Errorf("expected warning on stderr, got: %q", stderr.String())
	}
}

func TestRun_TrueColorAlways(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.Color = "always"
	cfg.TrueColor = "always"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code == ExitCodeError {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// With TrueColor=always, output should contain 24-bit ANSI codes (38;2;)
	if !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected 24-bit ANSI color codes in output with TrueColor=always, got: %s", output)
	}
}
