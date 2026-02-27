package diffyml

import (
	"strings"
	"testing"
)

// Task 5.2: Integration tests for CLI end-to-end

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

// Task 3 (colored-output): Integration and no-regression tests

func TestDetailedFormatter_Integration_AllDiffTypesColored(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		// Added: structured map entry (exercises pipe guides)
		{Path: "services.0", Type: DiffAdded, To: om},
		// Removed: scalar
		{Path: "config.oldKey", Type: DiffRemoved, From: "deprecated"},
		// Modified: type change (exercises italic type names)
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		// Modified: scalar value change
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		// Order changed (exercises colored was/now)
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b", "c"},
			To:   []interface{}{"c", "b", "a"}},
	}

	output := f.Format(diffs, opts)

	// 1. Bold path headings: all path headings should be wrapped in bold
	for _, path := range []string{"services.0", "config.oldKey", "config.port", "config.timeout", "items"} {
		if !strings.Contains(output, styleBold+path+colorReset) {
			t.Errorf("expected bold path heading for %q, got: %q", path, output)
		}
	}

	// 2. Italic type names in type-change descriptor
	if !strings.Contains(output, styleItalic+"int"+styleItalicOff) {
		t.Errorf("expected italic 'int' in type change descriptor, got: %q", output)
	}
	if !strings.Contains(output, styleItalic+"string"+styleItalicOff) {
		t.Errorf("expected italic 'string' in type change descriptor, got: %q", output)
	}

	// 3. Entry values colored (structured map added has green-colored YAML lines)
	addedColor := GetDetailedColorCode(DiffAdded, false)
	if !strings.Contains(output, addedColor+"    - name: nginx") {
		t.Errorf("expected green colored entry value lines, got: %q", output)
	}

	// 4. Red on - line, green on + line (order change)
	removedColor := GetDetailedColorCode(DiffRemoved, false)
	if !strings.Contains(output, removedColor+"    - ") {
		t.Errorf("expected red color on '- ' line, got: %q", output)
	}
	if !strings.Contains(output, addedColor+"    + ") {
		t.Errorf("expected green color on '+ ' line, got: %q", output)
	}

	// 5. Reset codes present
	if !strings.Contains(output, colorReset) {
		t.Errorf("expected color reset codes, got: %q", output)
	}
}

func TestDetailedFormatter_Integration_AllDiffTypesUncolored(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false

	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: om},
		{Path: "config.oldKey", Type: DiffRemoved, From: "deprecated"},
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b", "c"},
			To:   []interface{}{"c", "b", "a"}},
	}

	output := f.Format(diffs, opts)

	// No ANSI escape codes whatsoever when color is disabled
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI escape codes in uncolored output, got: %q", output)
	}

	// Content should still be present
	for _, expected := range []string{
		"services.0", "config.oldKey", "config.port", "config.timeout", "items",
		"- name: nginx", "port: 80",
		"type change from int to string",
		"± value change",
		"⇆ order changed",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in uncolored output, got: %q", expected, output)
		}
	}
}

func TestDetailedFormatter_Integration_TrueColorBoldItalicCombination(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.TrueColor = true

	diffs := []Difference{
		// Type change to exercise italic + yellow true color
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		// Added structured map to exercise pipe guides + true color context
		{Path: "services.0", Type: DiffAdded, To: func() *OrderedMap {
			om := NewOrderedMap()
			om.Keys = append(om.Keys, "name", "port")
			om.Values["name"] = "nginx"
			om.Values["port"] = 80
			return om
		}()},
		// Order change to exercise true color red/green on was/now
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"x", "y"},
			To:   []interface{}{"y", "x"}},
	}

	output := f.Format(diffs, opts)

	// Bold path headings should still work with true color
	if !strings.Contains(output, styleBold+"config.port"+colorReset) {
		t.Errorf("expected bold path heading in true color mode, got: %q", output)
	}

	// Italic type names within true color yellow descriptor
	trueYellow := GetDetailedColorCode(DiffModified, true)
	if !strings.Contains(output, trueYellow) {
		t.Errorf("expected true color yellow for type change descriptor, got: %q", output)
	}
	if !strings.Contains(output, styleItalic+"int"+styleItalicOff) {
		t.Errorf("expected italic type names in true color mode, got: %q", output)
	}

	// True color green on entry value lines
	trueGreen := GetDetailedColorCode(DiffAdded, true)
	if !strings.Contains(output, trueGreen+"    - name: nginx") {
		t.Errorf("expected true color green for entry value lines, got: %q", output)
	}

	// True color red/green on -/+
	trueRed := GetDetailedColorCode(DiffRemoved, true)
	if !strings.Contains(output, trueRed+"    - ") {
		t.Errorf("expected true color red on '- ' line, got: %q", output)
	}
	if !strings.Contains(output, trueGreen+"    + ") {
		t.Errorf("expected true color green on '+ ' line, got: %q", output)
	}
}

func TestDetailedFormatter_Integration_AutoColorModeNoTerminal(t *testing.T) {
	// When auto color mode resolves to no-color (stdout is not a terminal),
	// the output should contain zero ANSI escape sequences
	cfg := NewColorConfig(ColorModeAuto, true, 0)
	cfg.SetIsTerminal(false)

	opts := DefaultFormatOptions()
	cfg.ToFormatOptions(opts)

	// Verify auto mode resolved to no color
	if opts.Color {
		t.Fatal("expected Color=false when auto mode with non-terminal")
	}

	f, _ := GetFormatter("detailed")
	diffs := []Difference{
		{Path: "services.0", Type: DiffAdded, To: func() *OrderedMap {
			om := NewOrderedMap()
			om.Keys = append(om.Keys, "name", "port")
			om.Values["name"] = "nginx"
			om.Values["port"] = 80
			return om
		}()},
		{Path: "config.port", Type: DiffModified, From: 8080, To: "8080"},
		{Path: "items", Type: DiffOrderChanged,
			From: []interface{}{"a", "b"},
			To:   []interface{}{"b", "a"}},
	}

	output := f.Format(diffs, opts)

	if strings.Contains(output, "\033[") {
		t.Errorf("auto color mode with non-terminal should emit no ANSI codes, got: %q", output)
	}
}

func TestDetailedFormatter_Integration_NoRegressionSnapshots(t *testing.T) {
	// Verify uncolored output is byte-identical to expected baseline for all diff types
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	tests := []struct {
		name     string
		diffs    []Difference
		expected string
	}{
		{
			name:     "scalar modification",
			diffs:    []Difference{{Path: "key", Type: DiffModified, From: "old", To: "new"}},
			expected: "key\n  ± value change\n    - old\n    + new\n\n",
		},
		{
			name:     "type change",
			diffs:    []Difference{{Path: "port", Type: DiffModified, From: 8080, To: "8080"}},
			expected: "port\n  ± type change from int to string\n    - 8080\n    + 8080\n\n",
		},
		{
			name:     "list entry added",
			diffs:    []Difference{{Path: "items.0", Type: DiffAdded, To: "newItem"}},
			expected: "items.0\n  + one list entry added:\n    - newItem\n\n",
		},
		{
			name:     "map entry removed",
			diffs:    []Difference{{Path: "config.key", Type: DiffRemoved, From: "value"}},
			expected: "config.key\n  - one map entry removed:\n    key: value\n\n",
		},
		{
			name: "order change",
			diffs: []Difference{{Path: "items", Type: DiffOrderChanged,
				From: []interface{}{"a", "b"}, To: []interface{}{"b", "a"}}},
			expected: "items\n  ⇆ order changed\n    - a, b\n    + b, a\n\n",
		},
		{
			name:     "whitespace change",
			diffs:    []Difference{{Path: "key", Type: DiffModified, From: "a b", To: "a  b"}},
			expected: "key\n  ± whitespace only change\n    - a·b\n    + a··b\n\n",
		},
		{
			name: "structured map added",
			diffs: func() []Difference {
				om := NewOrderedMap()
				om.Keys = append(om.Keys, "name", "port")
				om.Values["name"] = "nginx"
				om.Values["port"] = 80
				return []Difference{{Path: "services.0", Type: DiffAdded, To: om}}
			}(),
			expected: "services.0\n  + one list entry added:\n    - name: nginx\n      port: 80\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format(tt.diffs, opts)
			if output != tt.expected {
				t.Errorf("no-regression snapshot mismatch.\nExpected:\n%s\nGot:\n%s", tt.expected, output)
			}
		})
	}
}
