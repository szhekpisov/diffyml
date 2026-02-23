package diffyml

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const fixturesDir = "../../testdata/fixtures"

func TestCLI_Fixtures(t *testing.T) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("failed to read fixtures directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(fixturesDir, entry.Name())

		// Detect fixture type: directory layout (dir1/ + dir2/) or file layout (file1.yaml)
		isDir := isDirectoryFixture(dir)
		isFile := isFileFixture(dir)
		if !isDir && !isFile {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			if isDir {
				runDirectoryFixture(t, dir)
			} else {
				runFixture(t, dir)
			}
		})
	}
}

// isFileFixture checks if a fixture uses the file1.yaml/file2.yaml layout.
func isFileFixture(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "file1.yaml"))
	return err == nil
}

// isDirectoryFixture checks if a fixture uses the dir1/dir2 directory layout.
func isDirectoryFixture(dir string) bool {
	info1, err := os.Stat(filepath.Join(dir, "dir1"))
	if err != nil || !info1.IsDir() {
		return false
	}
	info2, err := os.Stat(filepath.Join(dir, "dir2"))
	if err != nil || !info2.IsDir() {
		return false
	}
	return true
}

func runFixture(t *testing.T, dir string) {
	t.Helper()

	fromContent := readFixtureFile(t, dir, "file1.yaml")
	toContent := readFixtureFile(t, dir, "file2.yaml")
	expectedOutput := readOptionalFixtureFile(t, dir, "expected_output.yaml")
	expectedExitCodeStr := readOptionalFixtureFile(t, dir, "expected_exit_code")
	paramsCfg := readOptionalFixtureFile(t, dir, "params.cfg")

	if expectedOutput == "" && expectedExitCodeStr == "" {
		t.Fatalf("fixture must have at least expected_output.yaml or expected_exit_code")
	}

	cfg := NewCLIConfig()

	// Apply params.cfg if present
	if paramsCfg != "" {
		args := parseParamsCfg(paramsCfg)
		// Append dummy positional args to satisfy ParseArgs
		args = append(args, "dummy-from", "dummy-to")
		if err := cfg.ParseArgs(args); err != nil {
			t.Fatalf("failed to parse params.cfg: %v", err)
		}
	}

	// Force runner invariants after ParseArgs (so params.cfg cannot override)
	cfg.Color = "off"
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = fromContent
	rc.ToContent = toContent

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run() returned error: %v\nstderr: %s", result.Err, stderr.String())
	}

	// Assert expected output
	if expectedOutput != "" {
		got := stdout.String()
		if got != expectedOutput {
			t.Errorf("output does not match expected_output.yaml\n--- got ---\n%s\n--- expected ---\n%s", got, expectedOutput)
		}
	}

	// Assert expected exit code
	if expectedExitCodeStr != "" {
		wantCode, err := strconv.Atoi(strings.TrimSpace(expectedExitCodeStr))
		if err != nil {
			t.Fatalf("invalid expected_exit_code content: %v", err)
		}
		if result.Code != wantCode {
			t.Errorf("expected exit code %d, got %d\nstdout: %s", wantCode, result.Code, stdout.String())
		}
		// When differences are expected, stdout should be non-empty
		if wantCode == ExitCodeDifferences && stdout.String() == "" {
			t.Error("expected non-empty stdout when differences are present")
		}
	}
}

// runDirectoryFixture runs a fixture that uses dir1/dir2 directory layout.
func runDirectoryFixture(t *testing.T, dir string) {
	t.Helper()

	dir1 := filepath.Join(dir, "dir1")
	dir2 := filepath.Join(dir, "dir2")
	expectedOutput := readOptionalFixtureFile(t, dir, "expected_output.yaml")
	expectedExitCodeStr := readOptionalFixtureFile(t, dir, "expected_exit_code")
	paramsCfg := readOptionalFixtureFile(t, dir, "params.cfg")

	if expectedOutput == "" && expectedExitCodeStr == "" {
		t.Fatalf("fixture must have at least expected_output.yaml or expected_exit_code")
	}

	cfg := NewCLIConfig()
	cfg.FromFile = dir1
	cfg.ToFile = dir2

	// Apply params.cfg if present
	if paramsCfg != "" {
		args := parseParamsCfg(paramsCfg)
		args = append(args, "dummy-from", "dummy-to")
		if err := cfg.ParseArgs(args); err != nil {
			t.Fatalf("failed to parse params.cfg: %v", err)
		}
		// Restore directory paths after ParseArgs
		cfg.FromFile = dir1
		cfg.ToFile = dir2
	}

	// Force runner invariants
	cfg.Color = "off"
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)

	// Assert expected output
	if expectedOutput != "" {
		got := stdout.String()
		if got != expectedOutput {
			t.Errorf("output does not match expected_output.yaml\n--- got ---\n%s\n--- expected ---\n%s", got, expectedOutput)
		}
	}

	// Assert expected exit code
	if expectedExitCodeStr != "" {
		wantCode, err := strconv.Atoi(strings.TrimSpace(expectedExitCodeStr))
		if err != nil {
			t.Fatalf("invalid expected_exit_code content: %v", err)
		}
		if result.Code != wantCode {
			t.Errorf("expected exit code %d, got %d\nstdout: %s\nstderr: %s",
				wantCode, result.Code, stdout.String(), stderr.String())
		}
		if wantCode == ExitCodeDifferences && stdout.String() == "" {
			t.Error("expected non-empty stdout when differences are present")
		}
	}
}

// parseParamsCfg parses a params.cfg file content into CLI args.
// Supports shell-style quoting (double and single quotes) and # comments.
func parseParamsCfg(content string) []string {
	var args []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		args = append(args, tokenizeLine(line)...)
	}
	return args
}

// tokenizeLine splits a line into tokens respecting double/single quotes.
func tokenizeLine(line string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == ' ' && !inSingle && !inDouble:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func readFixtureFile(t *testing.T, dir, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("failed to read %s: %v", name, err)
	}
	return data
}

func readOptionalFixtureFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("failed to read %s: %v", name, err)
	}
	return string(data)
}
