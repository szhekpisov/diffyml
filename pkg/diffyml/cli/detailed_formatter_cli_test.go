package cli

import (
	"strings"
	"testing"
)

func TestCLI_DetailedOutputFormat(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("timeout: 30\nhost: localhost\n"),
		ToContent:   []byte("timeout: 60\nhost: localhost\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	// Should use detailed-style formatting
	if !strings.Contains(output, "± value change") {
		t.Errorf("expected detailed-style '± value change' in CLI output, got: %q", output)
	}
	if !strings.Contains(output, "timeout") {
		t.Errorf("expected path 'timeout' in CLI output, got: %q", output)
	}
}

func TestCLI_DetailedIdenticalFiles(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	content := []byte("timeout: 30\nhost: localhost\n")
	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: content,
		ToContent:   content,
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if !strings.Contains(output, "no differences found") {
		t.Errorf("expected 'no differences found' for identical files, got: %q", output)
	}
}

func TestCLI_DetailedWithOmitHeader(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.OmitHeader = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("key: old\n"),
		ToContent:   []byte("key: new\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if strings.Contains(output, "Found") {
		t.Errorf("expected no header with --omit-header, got: %q", output)
	}
	if !strings.Contains(output, "± value change") {
		t.Errorf("expected diff content even with --omit-header, got: %q", output)
	}
}

func TestCLI_DetailedWithGoPatchStyle(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.UseGoPatchStyle = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("config:\n  timeout: 30\n"),
		ToContent:   []byte("config:\n  timeout: 60\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path '/config/timeout' in CLI output, got: %q", output)
	}
}

func TestCLI_DetailedWithAllFlags(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.OmitHeader = true
	cfg.UseGoPatchStyle = true
	cfg.MultiLineContextLines = 2
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("config:\n  timeout: 30\n"),
		ToContent:   []byte("config:\n  timeout: 60\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if strings.Contains(output, "Found") {
		t.Errorf("expected no header, got: %q", output)
	}
	if !strings.Contains(output, "/config/timeout") {
		t.Errorf("expected go-patch path, got: %q", output)
	}
}

func TestCLI_DetailedWithSetExitCode(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.SetExitCode = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("key: old\n"),
		ToContent:   []byte("key: new\n"),
	}

	result := Run(cfg, rc)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d for differences with -s, got %d", ExitCodeDifferences, result.Code)
	}
}

func TestCLI_DetailedWithSetExitCodeNoDiffs(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.SetExitCode = true
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	content := []byte("key: same\n")
	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: content,
		ToContent:   content,
	}

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d for no differences, got %d", ExitCodeSuccess, result.Code)
	}
}

func TestCLI_DetailedMultipleChanges(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("timeout: 30\nhost: localhost\nport: 8080\n"),
		ToContent:   []byte("timeout: 60\nhost: production\nport: 8080\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	// Should have header mentioning two differences
	if !strings.Contains(output, "Found two differences") {
		t.Errorf("expected 'Found two differences' in header, got: %q", output)
	}
}

func TestCLI_DetailedStructuredAddition(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.FromFile = "a.yaml"
	cfg.ToFile = "b.yaml"

	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("items: []\n"),
		ToContent:   []byte("items:\n  - name: nginx\n    port: 80\n"),
	}

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("Run returned error: %v", result.Err)
	}

	output := stdout.String()
	if !strings.Contains(output, "added") {
		t.Errorf("expected 'added' for new list entry, got: %q", output)
	}
}
