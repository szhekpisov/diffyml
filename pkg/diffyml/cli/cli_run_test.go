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
	cfg.ToFile = "/tmp/outside/deploy.yaml"

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
	// Should warn on stderr if absolute path used
	if strings.HasPrefix(path, "/") && !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr when using absolute path, stderr: %q", stderr.String())
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
