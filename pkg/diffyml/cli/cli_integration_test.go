package cli

import (
	"strings"
	"testing"
)

// CLI Integration Tests (Task 6.4)

func TestCLI_EndToEnd_ParseAndRun(t *testing.T) {
	// Test complete flow: parse args -> run -> get result
	yaml1 := "config:\n  name: test\n  value: 100\n"
	yaml2 := "config:\n  name: test\n  value: 200\n"

	cfg := NewCLIConfig()
	args := []string{"-o", "compact", "from.yaml", "to.yaml"}
	if err := cfg.ParseArgs(args); err != nil {
		t.Fatalf("failed to parse args: %v", err)
	}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	if !containsSubstr(output, "config.value") {
		t.Errorf("expected path in output, got: %s", output)
	}
}

func TestCLI_EndToEnd_AllFormatters(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	formats := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := NewCLIConfig()
			cfg.Output = format

			rc := NewRunConfig()
			var stdout, stderr strings.Builder
			rc.Stdout = &stdout
			rc.Stderr = &stderr
			rc.FromContent = []byte(yaml1)
			rc.ToContent = []byte(yaml2)

			result := Run(cfg, rc)
			if result.Err != nil {
				t.Errorf("unexpected error for format %s: %v", format, result.Err)
			}
			if stdout.String() == "" && format != "github" {
				// GitHub can be empty for no diffs, but we have diffs
				t.Errorf("expected output for format %s", format)
			}
		})
	}
}

func TestCLI_ExitCode_NoDifferences_WithSetExitCode(t *testing.T) {
	yaml := "key: value\n"

	cfg := NewCLIConfig()
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code 0 for no differences with -s, got %d", result.Code)
	}
}

func TestCLI_ExitCode_HasDifferences_WithSetExitCode(t *testing.T) {
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
		t.Errorf("expected exit code 1 for differences with -s, got %d", result.Code)
	}
}

func TestCLI_ExitCode_HasDifferences_WithoutSetExitCode(t *testing.T) {
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
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code 0 without -s regardless of differences, got %d", result.Code)
	}
}

func TestCLI_ExitCode_Error_WithSetExitCode(t *testing.T) {
	invalidYAML := "invalid: yaml: content:\n  - bad"
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
		t.Errorf("expected exit code 255 for error with -s, got %d", result.Code)
	}
}

func TestCLI_ErrorHandling_MissingFromFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "/nonexistent/path/file.yaml"
	cfg.ToFile = "/another/nonexistent/file.yaml"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code 255 for missing file, got %d", result.Code)
	}
	if stderr.String() == "" {
		t.Error("expected error message in stderr")
	}
}

func TestCLI_ErrorHandling_InvalidRegex(t *testing.T) {
	yaml := "key: value\n"

	cfg := NewCLIConfig()
	cfg.FilterRegexp = []string{"[invalid"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code 255 for invalid regex, got %d", result.Code)
	}
}

func TestCLI_FlagCombinations_IgnoreOrderAndWhitespace(t *testing.T) {
	yaml1 := "items:\n  - a\n  - b\ntext: \"  hello  \"\n"
	yaml2 := "items:\n  - b\n  - a\ntext: \"hello\"\n"

	cfg := NewCLIConfig()
	cfg.IgnoreOrderChanges = true
	cfg.IgnoreWhitespaceChanges = true
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	// With both ignore flags, the only differences are order and whitespace
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected no differences when ignoring order and whitespace, got code %d", result.Code)
	}
}

func TestCLI_FlagCombinations_SwapAndFilter(t *testing.T) {
	yaml1 := "config:\n  a: 1\n  b: 2\n"
	yaml2 := "config:\n  a: 10\n  b: 20\n"

	cfg := NewCLIConfig()
	cfg.Swap = true
	cfg.Filter = []string{"config.a"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// With swap, from becomes to and vice versa
	// With filter, only config.a should be shown
	if !containsSubstr(output, "config.a") {
		t.Error("expected config.a in filtered output")
	}
	if containsSubstr(output, "config.b") {
		t.Error("expected config.b to be filtered out")
	}
}

func TestCLI_OutputFormat_CompactWithColor(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Color = "always"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	Run(cfg, rc)

	output := stdout.String()
	// Should contain ANSI color codes when color is forced always
	if !containsSubstr(output, "\033[") {
		t.Error("expected ANSI color codes in output with color=always")
	}
}

func TestCLI_OutputFormat_CompactWithoutColor(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	Run(cfg, rc)

	output := stdout.String()
	// Should NOT contain ANSI color codes when color is never
	if containsSubstr(output, "\033[") {
		t.Error("expected no ANSI color codes in output with color=never")
	}
}

func TestCLI_Chroot_BothFiles(t *testing.T) {
	yaml1 := "root:\n  data:\n    value: 1\n"
	yaml2 := "root:\n  data:\n    value: 2\n"

	cfg := NewCLIConfig()
	cfg.Chroot = "root.data"
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	result := Run(cfg, rc)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected differences in chroot path, got code %d", result.Code)
	}

	output := stdout.String()
	// Path should be relative to chroot
	if !containsSubstr(output, "value") {
		t.Error("expected 'value' path in output")
	}
}

func TestCLI_MultiDocument_Comparison(t *testing.T) {
	yaml1 := "---\ndoc: one\n---\ndoc: two\n"
	yaml2 := "---\ndoc: one\n---\ndoc: three\n"

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
		t.Errorf("expected differences in multi-doc, got code %d", result.Code)
	}
}

func TestCLI_ComplexYAML_NestedStructures(t *testing.T) {
	yaml1 := `
config:
  database:
    host: localhost
    port: 5432
  cache:
    enabled: true
    ttl: 300
`
	yaml2 := `
config:
  database:
    host: production
    port: 5432
  cache:
    enabled: true
    ttl: 600
`

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
		t.Errorf("expected differences in complex YAML, got code %d", result.Code)
	}

	output := stdout.String()
	if !containsSubstr(output, "config.database.host") {
		t.Error("expected config.database.host difference")
	}
	if !containsSubstr(output, "config.cache.ttl") {
		t.Error("expected config.cache.ttl difference")
	}
}
