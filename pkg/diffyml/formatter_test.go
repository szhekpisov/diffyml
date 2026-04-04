package diffyml

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func TestFormatterByName_Compact(t *testing.T) {
	f, err := FormatterByName("compact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_Brief(t *testing.T) {
	f, err := FormatterByName("brief")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_GitHub(t *testing.T) {
	f, err := FormatterByName("github")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_GitLab(t *testing.T) {
	f, err := FormatterByName("gitlab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_Gitea(t *testing.T) {
	f, err := FormatterByName("gitea")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_JSON(t *testing.T) {
	f, err := FormatterByName("json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_Invalid(t *testing.T) {
	_, err := FormatterByName("invalid")
	if err == nil {
		t.Error("expected error for invalid formatter name")
	}
}

func TestFormatterByName_EmptyName(t *testing.T) {
	_, err := FormatterByName("")
	if err == nil {
		t.Error("expected error for empty formatter name")
	}
}

func TestFormatterByName_CaseInsensitive(t *testing.T) {
	// Formatter names should be case-insensitive for convenience
	f, err := FormatterByName("COMPACT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestFormatterByName_ListsValidFormats(t *testing.T) {
	_, err := FormatterByName("badname")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	// Error message should list valid formats
	errStr := err.Error()
	expectedFormats := []string{"compact", "brief", "github", "gitlab", "gitea", "json", "detailed"}
	for _, format := range expectedFormats {
		if !contains(errStr, format) {
			t.Errorf("error message should list valid format '%s', got: %s", format, errStr)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFormatOptions_Defaults(t *testing.T) {
	opts := DefaultFormatOptions()

	if opts.OmitHeader {
		t.Error("expected default OmitHeader to be false")
	}
	if opts.UseGoPatchStyle {
		t.Error("expected default UseGoPatchStyle to be false")
	}
	if opts.ContextLines != 4 {
		t.Errorf("expected default ContextLines 4, got %d", opts.ContextLines)
	}
	if opts.FilePath != "" {
		t.Errorf("expected default FilePath to be empty, got %q", opts.FilePath)
	}
}

func TestDiffGroup_Construction(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "old", To: "new"},
	}
	group := DiffGroup{
		FilePath: "deploy.yaml",
		Diffs:    diffs,
	}
	if group.FilePath != "deploy.yaml" {
		t.Errorf("expected FilePath %q, got %q", "deploy.yaml", group.FilePath)
	}
	if len(group.Diffs) != 1 {
		t.Errorf("expected 1 diff, got %d", len(group.Diffs))
	}
}

func TestStructuredFormatter_InterfaceCompile(t *testing.T) {
	// Verify StructuredFormatter interface can be used in type assertions.
	var f Formatter = &GitLabFormatter{}
	_, ok := f.(StructuredFormatter)
	if !ok {
		t.Fatal("GitLabFormatter should implement StructuredFormatter")
	}
}

func TestFormatter_Interface(t *testing.T) {
	// Verify all formatters implement the Formatter interface correctly
	formatters := []string{"compact", "brief", "github", "gitlab", "gitea", "json", "detailed"}

	diffs := []Difference{
		{Path: DiffPath{"test", "path"}, Type: DiffModified, From: "old", To: "new"},
	}
	opts := DefaultFormatOptions()

	for _, name := range formatters {
		t.Run(name, func(t *testing.T) {
			f, err := FormatterByName(name)
			if err != nil {
				t.Fatalf("failed to get formatter: %v", err)
			}

			// Format should not panic and should return something
			output := f.Format(diffs, opts)
			if output == "" {
				t.Errorf("formatter %s returned empty output", name)
			}
		})
	}
}

func TestFormatter_EmptyDiffs(t *testing.T) {
	formatters := []string{"compact", "brief", "github", "gitlab", "gitea", "json", "detailed"}

	diffs := []Difference{}
	opts := DefaultFormatOptions()

	for _, name := range formatters {
		t.Run(name, func(t *testing.T) {
			f, err := FormatterByName(name)
			if err != nil {
				t.Fatalf("failed to get formatter: %v", err)
			}

			// Should handle empty diffs gracefully (no panic)
			_ = f.Format(diffs, opts)
		})
	}
}

func TestFormatter_NilOptions(t *testing.T) {
	f, _ := FormatterByName("compact")

	diffs := []Difference{
		{Path: DiffPath{"test"}, Type: DiffAdded, From: nil, To: "value"},
	}

	// Should handle nil options gracefully (no panic)
	output := f.Format(diffs, nil)
	if output == "" {
		t.Error("formatter should produce output even with nil options")
	}
}

func TestDiffPath_GoPatchString(t *testing.T) {
	tests := []struct {
		path     DiffPath
		expected string
	}{
		{DiffPath{"config", "name"}, "/config/name"},
		{DiffPath{"items", "[0]", "value"}, "/items/0/value"},
		{DiffPath{"root"}, "/root"},
		{DiffPath{"a", "b", "c", "d"}, "/a/b/c/d"},
		{DiffPath{"labels", "helm.sh/chart"}, "/labels/helm.sh/chart"},
		{nil, "/"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.path.GoPatchString()
			if result != tt.expected {
				t.Errorf("DiffPath%v.GoPatchString() = %q, want %q", []string(tt.path), result, tt.expected)
			}
		})
	}
}

// Color configuration tests

func TestColorMode_Always(t *testing.T) {
	mode := ColorModeAlways
	// ColorModeAlways should always enable color
	enabled := ResolveColorMode(mode, false)
	if !enabled {
		t.Error("ColorModeAlways should enable color even when not a terminal")
	}

	enabled = ResolveColorMode(mode, true)
	if !enabled {
		t.Error("ColorModeAlways should enable color when terminal")
	}
}

func TestColorMode_Never(t *testing.T) {
	mode := ColorModeNever
	// ColorModeNever should never enable color
	enabled := ResolveColorMode(mode, true)
	if enabled {
		t.Error("ColorModeNever should disable color even when terminal")
	}

	enabled = ResolveColorMode(mode, false)
	if enabled {
		t.Error("ColorModeNever should disable color when not a terminal")
	}
}

func TestColorMode_Auto(t *testing.T) {
	mode := ColorModeAuto

	// Auto should enable color when terminal
	enabled := ResolveColorMode(mode, true)
	if !enabled {
		t.Error("ColorModeAuto should enable color when terminal")
	}

	// Auto should disable color when not a terminal
	enabled = ResolveColorMode(mode, false)
	if enabled {
		t.Error("ColorModeAuto should disable color when not terminal")
	}
}

func TestParseColorMode_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected ColorMode
	}{
		{"always", ColorModeAlways},
		{"ALWAYS", ColorModeAlways},
		{"Always", ColorModeAlways},
		{"never", ColorModeNever},
		{"NEVER", ColorModeNever},
		{"auto", ColorModeAuto},
		{"AUTO", ColorModeAuto},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := ParseColorMode(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mode != tt.expected {
				t.Errorf("ParseColorMode(%q) = %v, want %v", tt.input, mode, tt.expected)
			}
		})
	}
}

func TestParseColorMode_Invalid(t *testing.T) {
	_, err := ParseColorMode("invalid")
	if err == nil {
		t.Error("expected error for invalid color mode")
	}
}

func TestParseColorMode_Empty(t *testing.T) {
	// Empty string should default to auto
	mode, err := ParseColorMode("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != ColorModeAuto {
		t.Errorf("empty string should default to ColorModeAuto, got %v", mode)
	}
}

func TestColorConfig_New(t *testing.T) {
	cfg := NewColorConfig(ColorModeAuto, false)
	if cfg == nil {
		t.Fatal("NewColorConfig should not return nil")
	}
}

func TestColorConfig_EnableColorForTerminal(t *testing.T) {
	cfg := NewColorConfig(ColorModeAuto, false)
	cfg.SetIsTerminal(true)

	if !cfg.ShouldUseColor() {
		t.Error("ColorConfig with Auto mode and terminal should enable color")
	}
}

func TestColorConfig_DisableColorForNonTerminal(t *testing.T) {
	cfg := NewColorConfig(ColorModeAuto, false)
	cfg.SetIsTerminal(false)

	if cfg.ShouldUseColor() {
		t.Error("ColorConfig with Auto mode and non-terminal should disable color")
	}
}

func TestColorConfig_TrueColor(t *testing.T) {
	cfg := NewColorConfig(ColorModeAlways, true)
	cfg.SetIsTerminal(true)

	if !cfg.ShouldUseTrueColor() {
		t.Error("ColorConfig with truecolor enabled should use true color")
	}
}

func TestColorConfig_TrueColorDisabled(t *testing.T) {
	cfg := NewColorConfig(ColorModeAlways, false)
	cfg.SetIsTerminal(true)

	if cfg.ShouldUseTrueColor() {
		t.Error("ColorConfig without truecolor flag should not use true color")
	}
}

// CI Formatter specific tests (Task 6.3)

func TestBriefFormatter_SummaryGeneration(t *testing.T) {
	f, _ := FormatterByName("brief")
	opts := DefaultFormatOptions()

	tests := []struct {
		name     string
		diffs    []Difference
		expected []string
	}{
		{
			name: "single added",
			diffs: []Difference{
				{Path: DiffPath{"key"}, Type: DiffAdded, To: "value"},
			},
			expected: []string{"1 added"},
		},
		{
			name: "single removed",
			diffs: []Difference{
				{Path: DiffPath{"key"}, Type: DiffRemoved, From: "value"},
			},
			expected: []string{"1 removed"},
		},
		{
			name: "single modified",
			diffs: []Difference{
				{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			},
			expected: []string{"1 modified"},
		},
		{
			name: "mixed changes",
			diffs: []Difference{
				{Path: DiffPath{"a"}, Type: DiffAdded, To: "new"},
				{Path: DiffPath{"b"}, Type: DiffAdded, To: "new2"},
				{Path: DiffPath{"c"}, Type: DiffRemoved, From: "old"},
				{Path: DiffPath{"d"}, Type: DiffModified, From: "old", To: "new"},
			},
			expected: []string{"2 added", "1 removed", "1 modified"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format(tt.diffs, opts)
			for _, exp := range tt.expected {
				if !containsSubstr(output, exp) {
					t.Errorf("expected '%s' in brief output, got: %s", exp, output)
				}
			}
		})
	}
}

func TestBriefFormatter_NoDifferences(t *testing.T) {
	f, _ := FormatterByName("brief")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	if !containsSubstr(output, "no differences") {
		t.Errorf("expected 'no differences' message, got: %s", output)
	}
}

func TestGitHubFormatter_WorkflowCommandFormat(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	// GitHub Actions workflow command format: ::warning title=YAML Modified::{message}
	if !containsSubstr(output, "::warning title=YAML Modified::") {
		t.Errorf("expected GitHub Actions warning with title format, got: %s", output)
	}
	if !containsSubstr(output, "config.timeout") {
		t.Errorf("expected path in output, got: %s", output)
	}
}

func TestGitHubFormatter_AllDiffTypes(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	tests := []struct {
		name     string
		diff     Difference
		expected string
	}{
		{
			name:     "added",
			diff:     Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: "value"},
			expected: "Added:",
		},
		{
			name:     "removed",
			diff:     Difference{Path: DiffPath{"key"}, Type: DiffRemoved, From: "value"},
			expected: "Removed:",
		},
		{
			name:     "modified",
			diff:     Difference{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			expected: "Modified:",
		},
		{
			name:     "order changed",
			diff:     Difference{Path: DiffPath{"list"}, Type: DiffOrderChanged},
			expected: "Order changed:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if !containsSubstr(output, tt.expected) {
				t.Errorf("expected '%s' in GitHub output, got: %s", tt.expected, output)
			}
		})
	}
}

func TestGitHubFormatter_EmptyOutput(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	// GitHub formatter returns empty string for no differences
	if output != "" {
		t.Errorf("expected empty output for no differences, got: %s", output)
	}
}

func TestGitLabFormatter_CodeQualityJSON(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	// GitLab Code Quality format is JSON
	if !containsSubstr(output, "[") || !containsSubstr(output, "]") {
		t.Errorf("expected JSON array format, got: %s", output)
	}
	if !containsSubstr(output, "description") {
		t.Errorf("expected 'description' field in JSON, got: %s", output)
	}
	if !containsSubstr(output, "fingerprint") {
		t.Errorf("expected 'fingerprint' field in JSON, got: %s", output)
	}
	if !containsSubstr(output, "severity") {
		t.Errorf("expected 'severity' field in JSON, got: %s", output)
	}
	if !containsSubstr(output, "location") {
		t.Errorf("expected 'location' field in JSON, got: %s", output)
	}
	if !containsSubstr(output, "check_name") {
		t.Errorf("expected 'check_name' field in JSON, got: %s", output)
	}
	if !containsSubstr(output, `"lines"`) {
		t.Errorf("expected 'lines' field in JSON, got: %s", output)
	}
	if !containsSubstr(output, `"begin"`) {
		t.Errorf("expected 'begin' field in JSON, got: %s", output)
	}
}

func TestGitLabFormatter_EmptyArray(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	// Empty differences should return empty JSON array
	if !containsSubstr(output, "[]") {
		t.Errorf("expected empty JSON array for no differences, got: %s", output)
	}
}

func TestGitLabFormatter_MultipleDiffs(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffAdded, To: "new"},
		{Path: DiffPath{"b"}, Type: DiffRemoved, From: "old"},
	}

	output := f.Format(diffs, opts)
	// Should have proper JSON array with comma separation
	if !containsSubstr(output, ",") {
		t.Errorf("expected comma-separated JSON entries, got: %s", output)
	}
}

func TestGiteaFormatter_GitHubCompatible(t *testing.T) {
	giteaF, _ := FormatterByName("gitea")
	githubF, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "value"}, Type: DiffModified, From: "old", To: "new"},
	}

	giteaOutput := giteaF.Format(diffs, opts)
	githubOutput := githubF.Format(diffs, opts)

	// Gitea should produce GitHub-compatible output
	if giteaOutput != githubOutput {
		t.Errorf("Gitea output should match GitHub output\nGitea: %s\nGitHub: %s", giteaOutput, githubOutput)
	}
}

// CompactFormatter specific tests

func TestCompactFormatter_SingleLineFormat(t *testing.T) {
	f, _ := FormatterByName("compact")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	// Compact format should use single-line-per-change format: indicator path : from → to
	if !containsSubstr(output, "±") {
		t.Errorf("expected '±' indicator for modified, got: %s", output)
	}
	if !containsSubstr(output, "config.timeout") {
		t.Errorf("expected path in output, got: %s", output)
	}
	if !containsSubstr(output, "30") || !containsSubstr(output, "60") {
		t.Errorf("expected both values in output, got: %s", output)
	}
	if !containsSubstr(output, "→") {
		t.Errorf("expected arrow separator in output, got: %s", output)
	}
}

func TestCompactFormatter_ChangeTypeIndicators(t *testing.T) {
	f, _ := FormatterByName("compact")
	opts := DefaultFormatOptions()

	tests := []struct {
		name      string
		diffType  DiffType
		indicator string
	}{
		{"added", DiffAdded, "+"},
		{"removed", DiffRemoved, "-"},
		{"modified", DiffModified, "±"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := []Difference{
				{Path: DiffPath{"test", "path"}, Type: tt.diffType, From: "old", To: "new"},
			}
			output := f.Format(diffs, opts)
			if !containsSubstr(output, tt.indicator) {
				t.Errorf("expected indicator '%s' in output, got: %s", tt.indicator, output)
			}
		})
	}
}

func TestStyleConstants_BoldAndItalic(t *testing.T) {
	// Verify bold constant is correct SGR code
	if styleBold != "\033[1m" {
		t.Errorf("styleBold = %q, want %q", styleBold, "\033[1m")
	}
	// Verify bold-off uses selective reset (SGR 22), not full reset
	if styleBoldOff != "\033[22m" {
		t.Errorf("styleBoldOff = %q, want %q", styleBoldOff, "\033[22m")
	}
	// Verify italic constant is correct SGR code
	if styleItalic != "\033[3m" {
		t.Errorf("styleItalic = %q, want %q", styleItalic, "\033[3m")
	}
	// Verify italic-off uses selective reset (SGR 23), not full reset
	if styleItalicOff != "\033[23m" {
		t.Errorf("styleItalicOff = %q, want %q", styleItalicOff, "\033[23m")
	}
}

func TestStyleConstants_CombiningWithColor(t *testing.T) {
	// Style combining via string concatenation should produce valid ANSI sequences
	boldGreen := styleBold + colorGreen
	expected := "\033[1m\033[32m"
	if boldGreen != expected {
		t.Errorf("styleBold + colorGreen = %q, want %q", boldGreen, expected)
	}

	italicYellow := styleItalic + colorYellow
	expected = "\033[3m\033[33m"
	if italicYellow != expected {
		t.Errorf("styleItalic + colorYellow = %q, want %q", italicYellow, expected)
	}
}

func TestCompactFormatter_ColorCodes(t *testing.T) {
	f, _ := FormatterByName("compact")
	diffs := []Difference{
		{Path: DiffPath{"test"}, Type: DiffAdded, From: nil, To: "new"},
	}
	opts := DefaultFormatOptions()
	opts.Color = true

	output := f.Format(diffs, opts)
	// Green color code for additions
	if !containsSubstr(output, "\033[32m") {
		t.Errorf("expected green color code for additions, got: %s", output)
	}
}

func TestGitHubFormatter_DifferentiatedCommands(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	tests := []struct {
		name            string
		diff            Difference
		expectedCommand string
		expectedTitle   string
	}{
		{
			name:            "added uses notice",
			diff:            Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: "value"},
			expectedCommand: "::notice",
			expectedTitle:   "title=YAML Added",
		},
		{
			name:            "removed uses error",
			diff:            Difference{Path: DiffPath{"key"}, Type: DiffRemoved, From: "value"},
			expectedCommand: "::error",
			expectedTitle:   "title=YAML Removed",
		},
		{
			name:            "modified uses warning",
			diff:            Difference{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			expectedCommand: "::warning",
			expectedTitle:   "title=YAML Modified",
		},
		{
			name:            "order changed uses notice",
			diff:            Difference{Path: DiffPath{"list"}, Type: DiffOrderChanged},
			expectedCommand: "::notice",
			expectedTitle:   "title=YAML Order Changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if !containsSubstr(output, tt.expectedCommand) {
				t.Errorf("expected command %q in output, got: %s", tt.expectedCommand, output)
			}
			if !containsSubstr(output, tt.expectedTitle) {
				t.Errorf("expected title %q in output, got: %s", tt.expectedTitle, output)
			}
		})
	}
}

func TestGitHubFormatter_FileParameter(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	// Must include file=deploy.yaml parameter
	if !containsSubstr(output, "file=deploy.yaml") {
		t.Errorf("expected file=deploy.yaml in output, got: %s", output)
	}
	// Must follow format: ::command file=path,title=title::message
	expected := "::warning file=deploy.yaml,title=YAML Modified::Modified: config.timeout changed from 30 to 60\n"
	if output != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestGitHubFormatter_NoFileParameter(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()
	// FilePath is empty — backward compatible

	diffs := []Difference{
		{Path: DiffPath{"config", "timeout"}, Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	// Must NOT include file= parameter
	if containsSubstr(output, "file=") {
		t.Errorf("expected no file= parameter when FilePath is empty, got: %s", output)
	}
	// Must follow format: ::command title=title::message
	expected := "::warning title=YAML Modified::Modified: config.timeout changed from 30 to 60\n"
	if output != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestGitHubFormatter_FileParameterAllDiffTypes(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()
	opts.FilePath = "service.yaml"

	tests := []struct {
		name     string
		diff     Difference
		expected string
	}{
		{
			name:     "added with file",
			diff:     Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: "value"},
			expected: "::notice file=service.yaml,title=YAML Added::Added: key = value\n",
		},
		{
			name:     "removed with file",
			diff:     Difference{Path: DiffPath{"key"}, Type: DiffRemoved, From: "value"},
			expected: "::error file=service.yaml,title=YAML Removed::Removed: key = value\n",
		},
		{
			name:     "modified with file",
			diff:     Difference{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			expected: "::warning file=service.yaml,title=YAML Modified::Modified: key changed from old to new\n",
		},
		{
			name:     "order changed with file",
			diff:     Difference{Path: DiffPath{"list"}, Type: DiffOrderChanged},
			expected: "::notice file=service.yaml,title=YAML Order Changed::Order changed: list\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if output != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, output)
			}
		})
	}
}

func TestGitHubFormatter_AnnotationLimitTruncation(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	// Generate 13 warning diffs (DiffModified → ::warning)
	var diffs []Difference
	for i := 0; i < 13; i++ {
		diffs = append(diffs, Difference{
			Path: DiffPath{fmt.Sprintf("key%d", i)},
			Type: DiffModified,
			From: "old",
			To:   "new",
		})
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Should have 10 warning commands + 1 summary = 11 lines
	if len(lines) != 11 {
		t.Fatalf("expected 11 lines (10 warnings + 1 summary), got %d:\n%s", len(lines), output)
	}

	// First 10 lines should be ::warning commands
	for i := 0; i < 10; i++ {
		if !strings.HasPrefix(lines[i], "::warning ") {
			t.Errorf("line %d should be ::warning, got: %s", i, lines[i])
		}
	}

	// Last line should be summary
	expectedSummary := "::warning title=diffyml::3 additional warning annotations omitted due to GitHub Actions limit"
	if lines[10] != expectedSummary {
		t.Errorf("expected summary:\n%s\ngot:\n%s", expectedSummary, lines[10])
	}
}

func TestGitHubFormatter_AnnotationLimitNotTriggered(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	// Generate exactly 10 warning diffs — should not trigger summary
	var diffs []Difference
	for i := 0; i < 10; i++ {
		diffs = append(diffs, Difference{
			Path: DiffPath{fmt.Sprintf("key%d", i)},
			Type: DiffModified,
			From: "old",
			To:   "new",
		})
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Should have exactly 10 lines, no summary
	if len(lines) != 10 {
		t.Fatalf("expected 10 lines (no summary), got %d:\n%s", len(lines), output)
	}

	if containsSubstr(output, "omitted due to GitHub Actions limit") {
		t.Errorf("summary should not appear when at or below limit, got: %s", output)
	}
}

func TestGitHubFormatter_AnnotationLimitMixedNotice(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	// DiffAdded and DiffOrderChanged both map to ::notice — they share the budget
	var diffs []Difference
	for i := 0; i < 7; i++ {
		diffs = append(diffs, Difference{
			Path: DiffPath{fmt.Sprintf("added%d", i)},
			Type: DiffAdded,
			To:   "val",
		})
	}
	for i := 0; i < 5; i++ {
		diffs = append(diffs, Difference{
			Path: DiffPath{fmt.Sprintf("order%d", i)},
			Type: DiffOrderChanged,
		})
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// 7 added + 3 order changed (budget exhausted at 10) + 1 summary = 11 lines
	if len(lines) != 11 {
		t.Fatalf("expected 11 lines (10 notices + 1 summary), got %d:\n%s", len(lines), output)
	}

	// Count notice commands (should be exactly 10)
	noticeCount := 0
	for _, line := range lines[:len(lines)-1] {
		if strings.HasPrefix(line, "::notice ") {
			noticeCount++
		}
	}
	if noticeCount != 10 {
		t.Errorf("expected 10 notice annotations, got %d", noticeCount)
	}

	expectedSummary := "::notice title=diffyml::2 additional notice annotations omitted due to GitHub Actions limit"
	lastLine := lines[len(lines)-1]
	if lastLine != expectedSummary {
		t.Errorf("expected summary:\n%s\ngot:\n%s", expectedSummary, lastLine)
	}
}

func TestGitHubFormatter_AnnotationLimitMultipleTypes(t *testing.T) {
	f, _ := FormatterByName("github")
	opts := DefaultFormatOptions()

	// 12 notices (DiffAdded) + 11 warnings (DiffModified) + 3 errors (DiffRemoved)
	var diffs []Difference
	for i := 0; i < 12; i++ {
		diffs = append(diffs, Difference{Path: DiffPath{fmt.Sprintf("a%d", i)}, Type: DiffAdded, To: "v"})
	}
	for i := 0; i < 11; i++ {
		diffs = append(diffs, Difference{Path: DiffPath{fmt.Sprintf("m%d", i)}, Type: DiffModified, From: "o", To: "n"})
	}
	for i := 0; i < 3; i++ {
		diffs = append(diffs, Difference{Path: DiffPath{fmt.Sprintf("r%d", i)}, Type: DiffRemoved, From: "v"})
	}

	output := f.Format(diffs, opts)

	// 10 notices + 10 warnings + 3 errors + 2 summaries (notice, warning) = 25 lines
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 25 {
		t.Fatalf("expected 25 lines, got %d:\n%s", len(lines), output)
	}

	// No summary for error (only 3, under limit)
	if containsSubstr(output, "additional error annotations") {
		t.Errorf("should not have error summary when under limit")
	}

	// Summary for notice and warning
	if !containsSubstr(output, "2 additional notice annotations omitted") {
		t.Errorf("expected notice summary, got:\n%s", output)
	}
	if !containsSubstr(output, "1 additional warning annotations omitted") {
		t.Errorf("expected warning summary, got:\n%s", output)
	}
}

func TestGitLabFormatter_RequiredFields(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "key"}, Type: DiffAdded, To: "value"},
	}

	output := f.Format(diffs, opts)

	requiredFields := []string{"description", "check_name", "fingerprint", "severity", "location", "path", "lines", "begin"}
	for _, field := range requiredFields {
		if !containsSubstr(output, field) {
			t.Errorf("expected required field %q in GitLab output, got: %s", field, output)
		}
	}
}

func TestGitLabFormatter_SeverityMapping(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	tests := []struct {
		name             string
		diff             Difference
		expectedSeverity string
	}{
		{
			name:             "added is info",
			diff:             Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: "val"},
			expectedSeverity: `"severity": "info"`,
		},
		{
			name:             "removed is major",
			diff:             Difference{Path: DiffPath{"key"}, Type: DiffRemoved, From: "val"},
			expectedSeverity: `"severity": "major"`,
		},
		{
			name:             "modified is major",
			diff:             Difference{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			expectedSeverity: `"severity": "major"`,
		},
		{
			name:             "order changed is minor",
			diff:             Difference{Path: DiffPath{"list"}, Type: DiffOrderChanged},
			expectedSeverity: `"severity": "minor"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if !containsSubstr(output, tt.expectedSeverity) {
				t.Errorf("expected severity %q in output, got: %s", tt.expectedSeverity, output)
			}
		})
	}
}

func TestGitLabFormatter_CheckNameMapping(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	tests := []struct {
		name              string
		diff              Difference
		expectedCheckName string
	}{
		{
			name:              "added check name",
			diff:              Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: "val"},
			expectedCheckName: "diffyml/added",
		},
		{
			name:              "removed check name",
			diff:              Difference{Path: DiffPath{"key"}, Type: DiffRemoved, From: "val"},
			expectedCheckName: "diffyml/removed",
		},
		{
			name:              "modified check name",
			diff:              Difference{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			expectedCheckName: "diffyml/modified",
		},
		{
			name:              "order changed check name",
			diff:              Difference{Path: DiffPath{"list"}, Type: DiffOrderChanged},
			expectedCheckName: "diffyml/order-changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if !containsSubstr(output, tt.expectedCheckName) {
				t.Errorf("expected check_name %q in output, got: %s", tt.expectedCheckName, output)
			}
		})
	}
}

func TestGitLabFormatter_UniqueFingerprints(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "key"}, Type: DiffAdded, To: "value1"},
		{Path: DiffPath{"config", "key"}, Type: DiffRemoved, From: "value2"},
	}

	output := f.Format(diffs, opts)

	// Extract fingerprints - count occurrences of "fingerprint" to ensure both are present
	fpCount := strings.Count(output, "fingerprint")
	if fpCount != 2 {
		t.Fatalf("expected 2 fingerprint fields, got %d", fpCount)
	}

	// The two entries have different descriptions so fingerprints must differ.
	// Split by fingerprint field and verify they're different values.
	parts := strings.Split(output, `"fingerprint": "`)
	if len(parts) < 3 {
		t.Fatal("could not extract fingerprints from output")
	}
	fp1 := parts[1][:64] // SHA-256 hex is 64 chars
	fp2 := parts[2][:64]
	if fp1 == fp2 {
		t.Errorf("fingerprints should be unique for different diffs, both got: %s", fp1)
	}
}

func TestGitLabFormatter_FingerprintDeterministic(t *testing.T) {
	f, _ := FormatterByName("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "key"}, Type: DiffModified, From: "old", To: "new"},
	}

	output1 := f.Format(diffs, opts)
	output2 := f.Format(diffs, opts)

	if output1 != output2 {
		t.Errorf("fingerprint should be deterministic, got different outputs:\n%s\nvs\n%s", output1, output2)
	}
}

// Task 2.1 Tests: GitLab formatter with file paths

func TestGitLabFormatter_LocationPathUsesFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	// location.path should be the file path, not the YAML key path
	if !strings.Contains(output, `"path": "deploy.yaml"`) {
		t.Errorf("expected location.path to be file path 'deploy.yaml', got: %s", output)
	}
	// Should NOT contain config.host as the path
	if strings.Contains(output, `"path": "config.host"`) {
		t.Errorf("location.path should not be YAML key path, got: %s", output)
	}
}

func TestGitLabFormatter_LocationPathFallback(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	// FilePath is empty — should fall back to diff.Path

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	// When FilePath is empty, location.path should fall back to diff.Path
	if !strings.Contains(output, `"path": "config.host"`) {
		t.Errorf("expected location.path fallback to YAML key path, got: %s", output)
	}
}

func TestGitLabFormatter_FingerprintIncludesFilePath(t *testing.T) {
	f := &GitLabFormatter{}

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
	}

	opts1 := DefaultFormatOptions()
	opts1.FilePath = "file1.yaml"
	output1 := f.Format(diffs, opts1)

	opts2 := DefaultFormatOptions()
	opts2.FilePath = "file2.yaml"
	output2 := f.Format(diffs, opts2)

	// Extract fingerprints
	fp1 := extractFingerprint(t, output1)
	fp2 := extractFingerprint(t, output2)

	// Same YAML change in different files must have different fingerprints
	if fp1 == fp2 {
		t.Errorf("fingerprints should differ for same change in different files, both got: %s", fp1)
	}
}

func TestGitLabFormatter_FingerprintUnchangedWhenNoFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	// FilePath is empty

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)
	fp := extractFingerprint(t, output)

	// Should match the old fingerprint formula: sha256(description)
	desc := diffDescription(diffs[0])
	expectedFP := gitLabFingerprint("", desc)
	if fp != expectedFP {
		t.Errorf("fingerprint with empty FilePath should match legacy formula\ngot:  %s\nwant: %s", fp, expectedFP)
	}
}

func TestGitLabFormatter_DescriptionContainsYAMLPath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
		{Path: DiffPath{"config", "port"}, Type: DiffAdded, To: 8080},
		{Path: DiffPath{"config", "old"}, Type: DiffRemoved, From: "value"},
		{Path: DiffPath{"items"}, Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)

	// All YAML paths should appear in descriptions
	for _, d := range diffs {
		if !strings.Contains(output, d.Path.String()) {
			t.Errorf("expected YAML path %q in description, got: %s", d.Path.String(), output)
		}
	}
}

func TestGitLabFormatter_ValidJSON_WithFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
		{Path: DiffPath{"config", "port"}, Type: DiffAdded, To: 8080},
	}

	output := f.Format(diffs, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

func TestGitLabFormatter_NoBOM(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	// UTF-8 BOM is 0xEF 0xBB 0xBF
	if len(output) >= 3 && output[0] == 0xEF && output[1] == 0xBB && output[2] == 0xBF {
		t.Error("output should not contain BOM")
	}
}

// Task 2.2 Tests: FormatAll for directory mode

func TestGitLabFormatter_FormatAll_SingleArray(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"service", "port"}, Type: DiffAdded, To: 8080},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatAll output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries in single array, got %d", len(result))
	}
}

func TestGitLabFormatter_FormatAll_EmptyGroups(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	output := f.FormatAll([]DiffGroup{}, opts)

	if output != "[]\n" {
		t.Errorf("expected empty JSON array for no groups, got: %q", output)
	}
}

func TestGitLabFormatter_FormatAll_DescriptionIncludesFilename(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	// Description should include filename prefix
	if !strings.Contains(output, "deploy.yaml") {
		t.Errorf("expected filename 'deploy.yaml' in description, got: %s", output)
	}
	// And still contain YAML path
	if !strings.Contains(output, "config.host") {
		t.Errorf("expected YAML path 'config.host' in description, got: %s", output)
	}
}

func TestGitLabFormatter_FormatAll_LocationPath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	// location.path should be the file path from the group
	if !strings.Contains(output, `"path": "deploy.yaml"`) {
		t.Errorf("expected location.path 'deploy.yaml', got: %s", output)
	}
}

func TestGitLabFormatter_FormatAll_UniqueFingerprintsAcrossFiles(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	// Same YAML change in two different files
	groups := []DiffGroup{
		{
			FilePath: "file1.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
			},
		},
		{
			FilePath: "file2.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatAll output is not valid JSON: %v", err)
	}

	fp1 := result[0]["fingerprint"].(string)
	fp2 := result[1]["fingerprint"].(string)

	if fp1 == fp2 {
		t.Errorf("fingerprints should differ for same change in different files, both got: %s", fp1)
	}
}

func TestGitLabFormatter_FormatAll_ValidJSON(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
				{Path: DiffPath{"config", "port"}, Type: DiffAdded, To: 8080},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"service", "name"}, Type: DiffRemoved, From: "old-svc"},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatAll output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 total entries, got %d", len(result))
	}
}

func TestGitLabFormatter_ImplementsStructuredFormatter(t *testing.T) {
	var f Formatter = &GitLabFormatter{}
	sf, ok := f.(StructuredFormatter)
	if !ok {
		t.Fatal("GitLabFormatter should implement StructuredFormatter")
	}

	// Verify the interface works with empty groups
	output := sf.FormatAll([]DiffGroup{}, DefaultFormatOptions())
	if output != "[]\n" {
		t.Errorf("expected empty array, got: %q", output)
	}
}

// --- Task 4.1: Regression and backward compatibility validation ---

func TestGitLabFormatter_BackwardCompat_EmptyFilePath(t *testing.T) {
	// When FilePath is empty, the formatter should behave identically to
	// pre-feature behavior: location.path falls back to diff.Path,
	// fingerprints are computed from description only.
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	// FilePath is empty (zero value)

	diffs := []Difference{
		{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"},
		{Path: DiffPath{"config", "port"}, Type: DiffAdded, To: 8080},
		{Path: DiffPath{"config", "old"}, Type: DiffRemoved, From: "value"},
		{Path: DiffPath{"items"}, Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)

	// Parse as JSON to verify structural validity
	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 entries, got %d", len(result))
	}

	// Verify location.path falls back to diff.Path (YAML key path)
	for i, entry := range result {
		location := entry["location"].(map[string]any)
		path := location["path"].(string)
		if path != diffs[i].Path.String() {
			t.Errorf("entry %d: expected location.path=%q (fallback to diff.Path), got %q", i, diffs[i].Path.String(), path)
		}
	}

	// Verify fingerprints match the legacy formula: sha256(description only)
	for i, entry := range result {
		fp := entry["fingerprint"].(string)
		desc := diffDescription(diffs[i])
		expectedFP := gitLabFingerprint("", desc)
		if fp != expectedFP {
			t.Errorf("entry %d: fingerprint mismatch with legacy formula\ngot:  %s\nwant: %s", i, fp, expectedFP)
		}
	}
}

func TestGitLabFormatter_BackwardCompat_FingerprintStability(t *testing.T) {
	// Fingerprints with empty FilePath must be deterministic and match
	// the exact pre-change formula: sha256(description).
	// This guards against accidental changes to the hash input format.
	diff := Difference{Path: DiffPath{"config", "host"}, Type: DiffModified, From: "localhost", To: "production"}
	desc := diffDescription(diff)

	// Compute expected fingerprint manually
	expectedFP := gitLabFingerprint("", desc)

	// Verify through the formatter
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	output := f.Format([]Difference{diff}, opts)

	fp := extractFingerprint(t, output)
	if fp != expectedFP {
		t.Errorf("fingerprint should match legacy formula\ngot:  %s\nwant: %s", fp, expectedFP)
	}

	// Run again to verify determinism
	output2 := f.Format([]Difference{diff}, opts)
	fp2 := extractFingerprint(t, output2)
	if fp != fp2 {
		t.Errorf("fingerprint should be deterministic across calls\ncall1: %s\ncall2: %s", fp, fp2)
	}
}

func TestGitLabFormatter_BackwardCompat_AllDiffTypes_ValidJSON(t *testing.T) {
	// Verify that all diff types produce valid JSON with all required fields
	// when FilePath is empty (backward compat mode).
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	allDiffs := []Difference{
		{Path: DiffPath{"added", "key"}, Type: DiffAdded, To: "value"},
		{Path: DiffPath{"removed", "key"}, Type: DiffRemoved, From: "value"},
		{Path: DiffPath{"modified", "key"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"order", "key"}, Type: DiffOrderChanged},
	}

	output := f.Format(allDiffs, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	requiredFields := []string{"description", "check_name", "fingerprint", "severity", "location"}
	for i, entry := range result {
		for _, field := range requiredFields {
			if _, ok := entry[field]; !ok {
				t.Errorf("entry %d: missing required field %q", i, field)
			}
		}
		// Verify location has path and lines.begin
		location := entry["location"].(map[string]any)
		if _, ok := location["path"]; !ok {
			t.Errorf("entry %d: location missing 'path'", i)
		}
		lines := location["lines"].(map[string]any)
		if begin, ok := lines["begin"]; !ok {
			t.Errorf("entry %d: location.lines missing 'begin'", i)
		} else if begin.(float64) != 1 {
			t.Errorf("entry %d: expected lines.begin=1, got %v", i, begin)
		}
	}
}

func TestGitLabFormatter_BackwardCompat_NilOptions(t *testing.T) {
	// Verify formatter handles nil options without panic (backward compat).
	f := &GitLabFormatter{}

	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, nil)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output with nil opts is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result))
	}
}

func TestGitLabFormatter_BackwardCompat_EmptyDiffs(t *testing.T) {
	// Verify empty diffs produce valid empty JSON array.
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)

	var result []any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("empty diffs output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d entries", len(result))
	}
}

func TestGitLabFormatter_BackwardCompat_SpecialCharsInValues(t *testing.T) {
	// Verify JSON escaping works for values with special characters.
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"config", "msg"}, Type: DiffModified, From: `line1\nline2`, To: `"quoted value"`},
		{Path: DiffPath{"config", "tab"}, Type: DiffModified, From: "no\ttab", To: "has\ttab"},
	}

	output := f.Format(diffs, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output with special chars is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

// FormatSingle tests

func TestFormatSingle_AllFormatters(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"key", "added"}, Type: DiffAdded, To: "value"},
		{Path: DiffPath{"key", "removed"}, Type: DiffRemoved, From: "value"},
		{Path: DiffPath{"key", "modified"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"key", "order"}, Type: DiffOrderChanged},
	}
	opts := DefaultFormatOptions()

	type singleFormatter interface {
		FormatSingle(diff Difference, opts *FormatOptions) string
	}

	formatters := map[string]singleFormatter{
		"compact": &CompactFormatter{},
		"brief":   &BriefFormatter{},
		"github":  &GitHubFormatter{},
		"gitlab":  &GitLabFormatter{},
		"gitea":   &GiteaFormatter{},
	}

	for name, f := range formatters {
		for _, diff := range diffs {
			t.Run(name+"/"+diff.Path.String(), func(t *testing.T) {
				output := f.FormatSingle(diff, opts)
				if output == "" {
					t.Errorf("%s.FormatSingle returned empty for %s", name, diff.Path.String())
				}
			})
		}
	}
}

func TestCompactFormatter_FormatSingle_NilOpts(t *testing.T) {
	f := &CompactFormatter{}
	diff := Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: "value"}

	output := f.FormatSingle(diff, nil)
	if output == "" {
		t.Error("FormatSingle with nil opts should produce output")
	}
}

// Inline format with color for all diff types

func TestCompactFormatter_InlineColor(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.Color = true

	diffs := []Difference{
		{Path: DiffPath{"key", "a"}, Type: DiffAdded, To: "new"},
		{Path: DiffPath{"key", "r"}, Type: DiffRemoved, From: "old"},
		{Path: DiffPath{"key", "m"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"key", "o"}, Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, colorGreen) {
		t.Errorf("expected green color in output, got: %s", output)
	}
	if !strings.Contains(output, colorRed) {
		t.Errorf("expected red color in output, got: %s", output)
	}
	if !strings.Contains(output, colorYellow) {
		t.Errorf("expected yellow color in output, got: %s", output)
	}
}

// Inline diff highlighting tests for compact formatter

func TestCompactFormatter_InlineDiffBold(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"image"}, Type: DiffModified, From: "demo:v1.20.1", To: "demo:v1.21.1"},
	}

	output := f.Format(diffs, opts)
	// Changed parts should be bold
	if !strings.Contains(output, styleBold) {
		t.Errorf("expected bold styling for changed inline diff segments, got: %q", output)
	}
	// Arrow separator should still be present
	if !strings.Contains(output, " → ") {
		t.Errorf("expected arrow separator, got: %q", output)
	}
}

func TestCompactFormatter_InlineDiffFallback(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.OmitHeader = true

	// Completely different values — should fall back to full-color
	diffs := []Difference{
		{Path: DiffPath{"name"}, Type: DiffModified, From: "alpha", To: "omega"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "alpha") || !strings.Contains(output, "omega") {
		t.Errorf("expected both values in fallback output, got: %q", output)
	}
	if !strings.Contains(output, " → ") {
		t.Errorf("expected arrow separator, got: %q", output)
	}
}

func TestCompactFormatter_InlineDiffNoColor(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.Color = false
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"image"}, Type: DiffModified, From: "demo:v1.20.1", To: "demo:v1.21.1"},
	}

	output := f.Format(diffs, opts)
	// No ANSI codes when color disabled
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes when color disabled, got: %q", output)
	}
	if !strings.Contains(output, "demo:v1.20.1 → demo:v1.21.1") {
		t.Errorf("expected plain values, got: %q", output)
	}
}

func TestCompactFormatter_TrueColor_InlineDiffDimmed(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.Color = true
	opts.TrueColor = true
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: DiffPath{"image"}, Type: DiffModified, From: "demo:v1.20.1", To: "demo:v1.21.1"},
	}

	output := f.Format(diffs, opts)
	// Should have dimmed colors for unchanged parts
	dimRed := TrueColorCode((DetailedRedR+128)/2, (DetailedRedG+128)/2, (DetailedRedB+128)/2)
	if !strings.Contains(output, dimRed) {
		t.Errorf("expected dimmed red color, got: %q", output)
	}
}

// DiffOrderChanged compact indicator test

func TestCompactFormatter_OrderChangedIndicator(t *testing.T) {
	f, _ := FormatterByName("compact")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: DiffPath{"list", "items"}, Type: DiffOrderChanged},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "⇆") {
		t.Errorf("expected '⇆' indicator for order changed, got: %s", output)
	}
	if !strings.Contains(output, "(order changed)") {
		t.Errorf("expected '(order changed)' in output, got: %s", output)
	}
}

func TestCompactFormatter_GoPatchStylePath(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	diff := Difference{Path: DiffPath{"config", "items", "0", "name"}, Type: DiffModified, From: "old", To: "new"}
	output := f.FormatSingle(diff, opts)
	if !strings.Contains(output, "/config/items/0/name") {
		t.Errorf("expected Go-Patch style path, got: %s", output)
	}
}

func TestFormatValue_Nil(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()

	// Modified diff with nil From value exercises formatValue(nil)
	diff := Difference{Path: DiffPath{"key"}, Type: DiffModified, From: nil, To: "new"}
	output := f.FormatSingle(diff, opts)
	if !strings.Contains(output, "<nil>") {
		t.Errorf("expected <nil> for nil value, got: %s", output)
	}
}

// --- Task 3.1 / 3.2: GitHubFormatter FormatAll tests ---

func TestGitHubFormatter_ImplementsStructuredFormatter(t *testing.T) {
	var f Formatter = &GitHubFormatter{}
	sf, ok := f.(StructuredFormatter)
	if !ok {
		t.Fatal("GitHubFormatter should implement StructuredFormatter")
	}

	// Verify the interface works with empty groups
	output := sf.FormatAll([]DiffGroup{}, DefaultFormatOptions())
	if output != "" {
		t.Errorf("expected empty string for empty groups, got: %q", output)
	}
}

func TestGitHubFormatter_FormatAll(t *testing.T) {
	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"port"}, Type: DiffAdded, To: 8080},
			},
		},
	}

	output := f.FormatAll(groups, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), output)
	}

	expected0 := "::warning file=deploy.yaml,title=YAML Modified::Modified: key changed from old to new"
	expected1 := "::notice file=service.yaml,title=YAML Added::Added: port = 8080"

	if lines[0] != expected0 {
		t.Errorf("line 0:\n  expected: %s\n  got:      %s", expected0, lines[0])
	}
	if lines[1] != expected1 {
		t.Errorf("line 1:\n  expected: %s\n  got:      %s", expected1, lines[1])
	}
}

func TestGitHubFormatter_FormatAllEmpty(t *testing.T) {
	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	// Empty groups slice
	output := f.FormatAll([]DiffGroup{}, opts)
	if output != "" {
		t.Errorf("expected empty string for empty groups, got: %q", output)
	}

	// Groups with zero diffs
	output = f.FormatAll([]DiffGroup{
		{FilePath: "deploy.yaml", Diffs: []Difference{}},
		{FilePath: "service.yaml", Diffs: []Difference{}},
	}, opts)
	if output != "" {
		t.Errorf("expected empty string when all groups have zero diffs, got: %q", output)
	}
}

func TestGitHubFormatter_FormatAll_AnnotationLimitsAcrossGroups(t *testing.T) {
	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	// Spread 13 warnings across 3 files — limit applies across ALL groups
	groups := []DiffGroup{
		{FilePath: "a.yaml", Diffs: makeDiffs(DiffModified, 5)},
		{FilePath: "b.yaml", Diffs: makeDiffs(DiffModified, 5)},
		{FilePath: "c.yaml", Diffs: makeDiffs(DiffModified, 3)},
	}

	output := f.FormatAll(groups, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// 10 warnings + 1 summary = 11 lines
	if len(lines) != 11 {
		t.Fatalf("expected 11 lines (10 warnings + 1 summary), got %d:\n%s", len(lines), output)
	}

	// Summary should NOT include file= parameter
	lastLine := lines[10]
	expectedSummary := "::warning title=diffyml::3 additional warning annotations omitted due to GitHub Actions limit"
	if lastLine != expectedSummary {
		t.Errorf("expected summary:\n%s\ngot:\n%s", expectedSummary, lastLine)
	}
	if containsSubstr(lastLine, "file=") {
		t.Errorf("summary annotation should not include file= parameter, got: %s", lastLine)
	}
}

func TestGiteaFormatter_FormatAll(t *testing.T) {
	giteaF := &GiteaFormatter{}
	githubF := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"key"}, Type: DiffModified, From: "old", To: "new"},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"port"}, Type: DiffAdded, To: 8080},
			},
		},
	}

	giteaOutput := giteaF.FormatAll(groups, opts)
	githubOutput := githubF.FormatAll(groups, opts)

	if giteaOutput != githubOutput {
		t.Errorf("Gitea FormatAll should match GitHub FormatAll\nGitea:  %s\nGitHub: %s", giteaOutput, githubOutput)
	}
}

func TestGiteaFormatter_ImplementsStructuredFormatter(t *testing.T) {
	var f Formatter = &GiteaFormatter{}
	_, ok := f.(StructuredFormatter)
	if !ok {
		t.Fatal("GiteaFormatter should implement StructuredFormatter")
	}
}

// makeDiffs generates n Difference entries of the given type.
func makeDiffs(dt DiffType, n int) []Difference {
	diffs := make([]Difference, n)
	for i := range diffs {
		diffs[i] = Difference{
			Path: DiffPath{fmt.Sprintf("key%d", i)},
			Type: dt,
			From: "old",
			To:   "new",
		}
	}
	return diffs
}

// extractFingerprint extracts the first fingerprint value from GitLab JSON output.
func extractFingerprint(t *testing.T, output string) string {
	t.Helper()
	parts := strings.Split(output, `"fingerprint": "`)
	if len(parts) < 2 {
		t.Fatalf("could not extract fingerprint from output: %s", output)
	}
	end := strings.Index(parts[1], `"`)
	if end < 0 {
		t.Fatalf("could not find end of fingerprint in output: %s", output)
	}
	return parts[1][:end]
}

// --- Mutation testing: formatter.go compact header counts ---

func TestCompactFormatter_HeaderCounts(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffAdded, To: "x"},
		{Path: DiffPath{"b"}, Type: DiffAdded, To: "y"},
		{Path: DiffPath{"c"}, Type: DiffRemoved, From: "z"},
		{Path: DiffPath{"d"}, Type: DiffModified, From: "old", To: "new"},
	}

	f := &CompactFormatter{}
	opts := &FormatOptions{Color: false}
	output := f.Format(diffs, opts)

	// Use exact format to distinguish "N category" from "-N category" (INCREMENT_DECREMENT mutation)
	if !strings.Contains(output, "(1 removed,") {
		t.Errorf("expected '(1 removed,' in header, got: %s", output)
	}
	if !strings.Contains(output, " 2 added,") {
		t.Errorf("expected ' 2 added,' in header, got: %s", output)
	}
	if !strings.Contains(output, " 1 modified)") {
		t.Errorf("expected ' 1 modified)' in header, got: %s", output)
	}
}

// --- Mutation testing: formatter.go brief formatter zero categories ---

func TestBriefFormatter_ZeroCategories(t *testing.T) {
	// Only-added diffs → output should have no "removed" or "modified"
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffAdded, To: "x"},
		{Path: DiffPath{"b"}, Type: DiffAdded, To: "y"},
	}

	f := &BriefFormatter{}
	output := f.Format(diffs, nil)

	// Check exact string to catch INCREMENT_DECREMENT mutations (-2 vs 2)
	if !strings.HasPrefix(output, "2 added") {
		t.Errorf("expected output starting with '2 added', got: %s", output)
	}
	if strings.Contains(output, "removed") {
		t.Errorf("output should not contain 'removed' when there are none, got: %s", output)
	}
	if strings.Contains(output, "modified") {
		t.Errorf("output should not contain 'modified' when there are none, got: %s", output)
	}
}

func TestBriefFormatter_OnlyModified(t *testing.T) {
	// Only-modified diffs → output should have no "added" or "removed"
	diffs := []Difference{
		{Path: DiffPath{"a"}, Type: DiffModified, From: "old", To: "new"},
	}

	f := &BriefFormatter{}
	output := f.Format(diffs, nil)

	if !strings.HasPrefix(output, "1 modified") {
		t.Errorf("expected output starting with '1 modified', got: %s", output)
	}
	if strings.Contains(output, "added") {
		t.Errorf("output should not contain 'added' when there are none, got: %s", output)
	}
	if strings.Contains(output, "removed") {
		t.Errorf("output should not contain 'removed' when there are none, got: %s", output)
	}
}

// Tests for formatValue with structured types

func TestFormatValue_OrderedMap(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "http"
	om.Values["port"] = 8080

	result := formatValue(om)

	if strings.Contains(result, "&{") {
		t.Errorf("formatValue should not produce Go struct repr for *OrderedMap, got: %s", result)
	}
	if !strings.Contains(result, "name: http") {
		t.Errorf("expected 'name: http' in YAML output, got: %s", result)
	}
	if !strings.Contains(result, "port: 8080") {
		t.Errorf("expected 'port: 8080' in YAML output, got: %s", result)
	}
}

func TestFormatValue_ListWithOrderedMaps(t *testing.T) {
	item := NewOrderedMap()
	item.Keys = append(item.Keys, "name", "value")
	item.Values["name"] = "FOO"
	item.Values["value"] = "bar"

	val := []any{item}
	result := formatValue(val)

	if strings.Contains(result, "&{") {
		t.Errorf("formatValue should not produce Go struct repr for []any with *OrderedMap, got: %s", result)
	}
	if strings.Contains(result, "0x") {
		t.Errorf("formatValue should not produce pointer addresses, got: %s", result)
	}
	if !strings.Contains(result, "name: FOO") {
		t.Errorf("expected 'name: FOO' in YAML output, got: %s", result)
	}
}

func TestFormatValue_MapStringInterface(t *testing.T) {
	val := map[string]any{"key": "value", "count": 42}
	result := formatValue(val)

	if !strings.Contains(result, "key: value") {
		t.Errorf("expected 'key: value' in YAML output, got: %s", result)
	}
	if !strings.Contains(result, "count: 42") {
		t.Errorf("expected 'count: 42' in YAML output, got: %s", result)
	}
}

func TestFormatValue_ScalarUnchanged(t *testing.T) {
	if got := formatValue("hello"); got != "hello" {
		t.Errorf("expected 'hello', got: %s", got)
	}
	if got := formatValue(42); got != "42" {
		t.Errorf("expected '42', got: %s", got)
	}
	if got := formatValue(nil); got != "<nil>" {
		t.Errorf("expected '<nil>', got: %s", got)
	}
}

func TestFormatValue_Timestamp(t *testing.T) {
	t.Run("date only", func(t *testing.T) {
		ts := time.Date(2010, 9, 9, 0, 0, 0, 0, time.UTC)
		got := formatValue(ts)
		if got != "2010-09-09" {
			t.Errorf("expected 2010-09-09, got %s", got)
		}
	})

	t.Run("datetime", func(t *testing.T) {
		ts := time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)
		got := formatValue(ts)
		if got != "2023-06-15T14:30:00Z" {
			t.Errorf("expected 2023-06-15T14:30:00Z, got %s", got)
		}
	})
}

// --- JSON Formatter Tests ---

func TestJSONFormatter_Format(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"spec", "replicas"}, Type: DiffModified, From: 3, To: 5, DocumentIndex: 0},
		{Path: DiffPath{"metadata", "labels", "app"}, Type: DiffAdded, From: nil, To: "web", DocumentIndex: 0},
		{Path: DiffPath{"spec", "image"}, Type: DiffRemoved, From: "nginx:1.20", To: nil, DocumentIndex: 0},
		{Path: DiffPath{"spec", "ports"}, Type: DiffOrderChanged, From: nil, To: nil, DocumentIndex: 1},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(result))
	}

	// Verify first entry
	if result[0]["path"] != "spec.replicas" {
		t.Errorf("expected path 'spec.replicas', got %v", result[0]["path"])
	}
	if result[0]["type"] != "modified" {
		t.Errorf("expected type 'modified', got %v", result[0]["type"])
	}
	// JSON numbers are float64
	if result[0]["from"] != float64(3) {
		t.Errorf("expected from 3, got %v (%T)", result[0]["from"], result[0]["from"])
	}
	if result[0]["to"] != float64(5) {
		t.Errorf("expected to 5, got %v (%T)", result[0]["to"], result[0]["to"])
	}

	// Verify added entry
	if result[1]["type"] != "added" {
		t.Errorf("expected type 'added', got %v", result[1]["type"])
	}
	if result[1]["from"] != nil {
		t.Errorf("expected from nil, got %v", result[1]["from"])
	}
	if result[1]["to"] != "web" {
		t.Errorf("expected to 'web', got %v", result[1]["to"])
	}

	// Verify removed entry
	if result[2]["type"] != "removed" {
		t.Errorf("expected type 'removed', got %v", result[2]["type"])
	}

	// Verify order_changed entry
	if result[3]["type"] != "order_changed" {
		t.Errorf("expected type 'order_changed', got %v", result[3]["type"])
	}
	if result[3]["document_index"] != float64(1) {
		t.Errorf("expected document_index 1, got %v", result[3]["document_index"])
	}
}

func TestJSONFormatter_Format_Empty(t *testing.T) {
	f := &JSONFormatter{}
	output := f.Format([]Difference{}, DefaultFormatOptions())

	var result []any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestJSONFormatter_Format_NilOpts(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffAdded, To: "value"},
	}

	output := f.Format(diffs, nil)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON with nil opts: %v\noutput: %s", err, output)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
}

func TestJSONFormatter_TypePreservation(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"int_val"}, Type: DiffModified, From: 42, To: 99},
		{Path: DiffPath{"bool_val"}, Type: DiffModified, From: true, To: false},
		{Path: DiffPath{"float_val"}, Type: DiffModified, From: 3.14, To: 2.72},
		{Path: DiffPath{"str_val"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"null_val"}, Type: DiffAdded, From: nil, To: "something"},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	// int → float64 in JSON
	if result[0]["from"] != float64(42) {
		t.Errorf("int not preserved: got %v (%T)", result[0]["from"], result[0]["from"])
	}

	// bool preserved
	if result[1]["from"] != true {
		t.Errorf("bool not preserved: got %v (%T)", result[1]["from"], result[1]["from"])
	}

	// float preserved
	if result[2]["from"] != 3.14 {
		t.Errorf("float not preserved: got %v (%T)", result[2]["from"], result[2]["from"])
	}

	// string preserved
	if result[3]["from"] != "old" {
		t.Errorf("string not preserved: got %v (%T)", result[3]["from"], result[3]["from"])
	}

	// nil preserved as null
	if result[4]["from"] != nil {
		t.Errorf("nil not preserved: got %v (%T)", result[4]["from"], result[4]["from"])
	}
}

func TestJSONFormatter_GoPatchStyle(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"spec", "containers", "[0]", "image"}, Type: DiffModified, From: "old", To: "new"},
	}
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	output := f.Format(diffs, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	expected := "/spec/containers/0/image"
	if result[0]["path"] != expected {
		t.Errorf("expected Go-Patch path %q, got %v", expected, result[0]["path"])
	}
}

func TestJSONFormatter_OrderedMapValues(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "port")
	om.Values["name"] = "nginx"
	om.Values["port"] = 80

	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"spec"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	toVal, ok := result[0]["to"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'to' to be a map, got %T", result[0]["to"])
	}
	if toVal["name"] != "nginx" {
		t.Errorf("expected name 'nginx', got %v", toVal["name"])
	}
	if toVal["port"] != float64(80) {
		t.Errorf("expected port 80, got %v", toVal["port"])
	}
}

func TestJSONFormatter_StructuredFormatter(t *testing.T) {
	var f Formatter = &JSONFormatter{}
	_, ok := f.(StructuredFormatter)
	if !ok {
		t.Fatal("JSONFormatter should implement StructuredFormatter")
	}
}

func TestJSONFormatter_FormatAll(t *testing.T) {
	f := &JSONFormatter{}
	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"spec", "replicas"}, Type: DiffModified, From: 3, To: 5},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"spec", "type"}, Type: DiffModified, From: "ClusterIP", To: "NodePort"},
			},
		},
	}

	output := f.FormatAll(groups, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatAll output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	// Verify file field is present in directory mode
	if result[0]["file"] != "deploy.yaml" {
		t.Errorf("expected file 'deploy.yaml', got %v", result[0]["file"])
	}
	if result[1]["file"] != "service.yaml" {
		t.Errorf("expected file 'service.yaml', got %v", result[1]["file"])
	}
}

func TestJSONFormatter_FormatAll_Empty(t *testing.T) {
	f := &JSONFormatter{}
	output := f.FormatAll([]DiffGroup{}, DefaultFormatOptions())

	var result []any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestJSONFormatter_FormatAll_NilOpts(t *testing.T) {
	f := &JSONFormatter{}
	groups := []DiffGroup{
		{
			FilePath: "test.yaml",
			Diffs:    []Difference{{Path: DiffPath{"key"}, Type: DiffAdded, To: "val"}},
		},
	}

	output := f.FormatAll(groups, nil)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON with nil opts: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
}

func TestJSONFormatter_SpecialChars(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"msg"}, Type: DiffModified, From: "hello \"world\"", To: "line1\nline2"},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output with special chars is not valid JSON: %v\noutput: %s", err, output)
	}
	if result[0]["from"] != "hello \"world\"" {
		t.Errorf("expected special chars preserved, got %v", result[0]["from"])
	}
}

func TestJSONFormatter_DottedKeyPath(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"metadata", "labels", "helm.sh/chart"}, Type: DiffAdded, To: "myapp-1.0"},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Dotted keys should be bracket-quoted
	expected := "metadata.labels[helm.sh/chart]"
	if result[0]["path"] != expected {
		t.Errorf("expected path %q, got %v", expected, result[0]["path"])
	}
}

func TestJSONFormatter_MapValues(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"config"}, Type: DiffAdded, To: map[string]any{
			"host": "localhost",
			"port": 8080,
		}},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	toVal, ok := result[0]["to"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'to' to be a map, got %T", result[0]["to"])
	}
	if toVal["host"] != "localhost" {
		t.Errorf("expected host 'localhost', got %v", toVal["host"])
	}
	if toVal["port"] != float64(8080) {
		t.Errorf("expected port 8080, got %v", toVal["port"])
	}
}

func TestJSONFormatter_FormatAll_GoPatchStyle(t *testing.T) {
	f := &JSONFormatter{}
	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"spec", "containers", "[0]", "image"}, Type: DiffModified, From: "old", To: "new"},
			},
		},
	}
	opts := DefaultFormatOptions()
	opts.UseGoPatchStyle = true

	output := f.FormatAll(groups, opts)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	expected := "/spec/containers/0/image"
	if result[0]["path"] != expected {
		t.Errorf("expected Go-Patch path %q, got %v", expected, result[0]["path"])
	}
	if result[0]["file"] != "deploy.yaml" {
		t.Errorf("expected file 'deploy.yaml', got %v", result[0]["file"])
	}
}

func TestJSONFormatter_InfNaNValues(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"inf_val"}, Type: DiffModified, From: math.Inf(1), To: math.Inf(-1)},
		{Path: DiffPath{"nan_val"}, Type: DiffModified, From: math.NaN(), To: 0.0},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output with Inf/NaN is not valid JSON: %v\noutput: %s", err, output)
	}

	// Inf/NaN should be serialized as strings
	if result[0]["from"] != "+Inf" {
		t.Errorf("expected +Inf string, got %v (%T)", result[0]["from"], result[0]["from"])
	}
	if result[0]["to"] != "-Inf" {
		t.Errorf("expected -Inf string, got %v (%T)", result[0]["to"], result[0]["to"])
	}
	if result[1]["from"] != "NaN" {
		t.Errorf("expected NaN string, got %v (%T)", result[1]["from"], result[1]["from"])
	}
}

func TestJSONFormatter_FormatSingle(t *testing.T) {
	f := &JSONFormatter{}
	diff := Difference{
		Path: DiffPath{"spec", "replicas"},
		Type: DiffModified,
		From: 3,
		To:   5,
	}

	output := f.FormatSingle(diff, DefaultFormatOptions())

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatSingle output is not valid JSON: %v\noutput: %s", err, output)
	}
	if result["path"] != "spec.replicas" {
		t.Errorf("expected path 'spec.replicas', got %v", result["path"])
	}
	if result["type"] != "modified" {
		t.Errorf("expected type 'modified', got %v", result["type"])
	}
}

func TestJSONFormatter_FormatSingle_NilOpts(t *testing.T) {
	f := &JSONFormatter{}
	diff := Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: "val"}

	output := f.FormatSingle(diff, nil)

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatSingle with nil opts is not valid JSON: %v", err)
	}
}

func TestJSONFormatter_NestedListValues(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"items"}, Type: DiffAdded, To: []any{"a", "b", "c"}},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	toVal, ok := result[0]["to"].([]any)
	if !ok {
		t.Fatalf("expected 'to' to be a list, got %T", result[0]["to"])
	}
	if len(toVal) != 3 {
		t.Errorf("expected 3 items, got %d", len(toVal))
	}
}

func TestJSONFormatter_FormatSingle_MarshalError(t *testing.T) {
	f := &JSONFormatter{}
	// A func value passes through jsonPrepareValue's default case
	// but causes json.Marshal to fail.
	diff := Difference{Path: DiffPath{"key"}, Type: DiffAdded, To: func() {}}

	output := f.FormatSingle(diff, DefaultFormatOptions())
	if output != "{}\n" {
		t.Errorf("expected fallback {}, got %q", output)
	}
}

func TestJSONFormatter_Format_MarshalError(t *testing.T) {
	f := &JSONFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffAdded, To: func() {}},
	}

	output := f.Format(diffs, DefaultFormatOptions())
	if output != "[]\n" {
		t.Errorf("expected fallback [], got %q", output)
	}
}

// --- JSONPatchFormatter tests ---

func TestFormatterByName_JSONPatch(t *testing.T) {
	f, err := FormatterByName("json-patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
	if _, ok := f.(*JSONPatchFormatter); !ok {
		t.Fatalf("expected *JSONPatchFormatter, got %T", f)
	}
}

func TestJSONPatchFormatter_Format(t *testing.T) {
	f := &JSONPatchFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"spec", "replicas"}, Type: DiffModified, From: 3, To: 5},
		{Path: DiffPath{"metadata", "labels", "app"}, Type: DiffAdded, To: "web"},
		{Path: DiffPath{"spec", "image"}, Type: DiffRemoved, From: "nginx:1.20"},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(result))
	}

	// replace
	if result[0]["op"] != "replace" {
		t.Errorf("expected op 'replace', got %v", result[0]["op"])
	}
	if result[0]["path"] != "/spec/replicas" {
		t.Errorf("expected path '/spec/replicas', got %v", result[0]["path"])
	}
	if result[0]["value"] != float64(5) {
		t.Errorf("expected value 5, got %v", result[0]["value"])
	}

	// add
	if result[1]["op"] != "add" {
		t.Errorf("expected op 'add', got %v", result[1]["op"])
	}
	if result[1]["path"] != "/metadata/labels/app" {
		t.Errorf("expected path '/metadata/labels/app', got %v", result[1]["path"])
	}
	if result[1]["value"] != "web" {
		t.Errorf("expected value 'web', got %v", result[1]["value"])
	}

	// remove
	if result[2]["op"] != "remove" {
		t.Errorf("expected op 'remove', got %v", result[2]["op"])
	}
	if result[2]["path"] != "/spec/image" {
		t.Errorf("expected path '/spec/image', got %v", result[2]["path"])
	}
	if _, hasValue := result[2]["value"]; hasValue {
		t.Error("remove op should not have value field")
	}
}

func TestJSONPatchFormatter_OrderChanged_Skipped(t *testing.T) {
	f := &JSONPatchFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"spec", "replicas"}, Type: DiffModified, From: 3, To: 5},
		{Path: DiffPath{"spec", "ports"}, Type: DiffOrderChanged, From: []any{"http", "grpc"}, To: []any{"grpc", "http"}},
		{Path: DiffPath{"spec", "image"}, Type: DiffRemoved, From: "nginx"},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 ops (order_changed skipped), got %d", len(result))
	}

	if result[0]["op"] != "replace" {
		t.Errorf("expected op 'replace', got %v", result[0]["op"])
	}
	if result[1]["op"] != "remove" {
		t.Errorf("expected op 'remove', got %v", result[1]["op"])
	}
}

func TestJSONPatchFormatter_EmptyDiffs(t *testing.T) {
	f := &JSONPatchFormatter{}
	output := f.Format([]Difference{}, DefaultFormatOptions())

	var result []any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestJSONPatchFormatter_MultiDocument(t *testing.T) {
	f := &JSONPatchFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"[1]", "spec", "replicas"}, Type: DiffModified, From: 3, To: 5},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 op, got %d", len(result))
	}

	if result[0]["path"] != "/1/spec/replicas" {
		t.Errorf("expected path '/1/spec/replicas', got %v", result[0]["path"])
	}
}

func TestJSONPatchFormatter_RFC6901_Escaping(t *testing.T) {
	f := &JSONPatchFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"labels", "helm.sh/chart"}, Type: DiffModified, From: "v1", To: "v2"},
		{Path: DiffPath{"annotations", "key~with~tilde"}, Type: DiffAdded, To: "val"},
		{Path: DiffPath{"mixed", "a/b~c"}, Type: DiffRemoved, From: "old"},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(result))
	}

	// / in key → ~1
	if result[0]["path"] != "/labels/helm.sh~1chart" {
		t.Errorf("expected '/labels/helm.sh~1chart', got %v", result[0]["path"])
	}

	// ~ in key → ~0
	if result[1]["path"] != "/annotations/key~0with~0tilde" {
		t.Errorf("expected '/annotations/key~0with~0tilde', got %v", result[1]["path"])
	}

	// both / and ~ in key
	if result[2]["path"] != "/mixed/a~1b~0c" {
		t.Errorf("expected '/mixed/a~1b~0c', got %v", result[2]["path"])
	}
}

func TestJSONPatchFormatter_AddNullValue(t *testing.T) {
	f := &JSONPatchFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffAdded, To: nil},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 op, got %d", len(result))
	}

	// RFC 6902 requires "value" field even when null
	if _, hasValue := result[0]["value"]; !hasValue {
		t.Error("add op must include 'value' field even when null (RFC 6902)")
	}
	if result[0]["value"] != nil {
		t.Errorf("expected null value, got %v", result[0]["value"])
	}
}

func TestJSONPatchFormatter_NilOpts(t *testing.T) {
	f := &JSONPatchFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"key"}, Type: DiffAdded, To: "value"},
	}

	output := f.Format(diffs, nil)

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON with nil opts: %v\noutput: %s", err, output)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 op, got %d", len(result))
	}
}

func TestJSONPatchFormatter_StructuredFormatter(t *testing.T) {
	var f Formatter = &JSONPatchFormatter{}
	sf, ok := f.(StructuredFormatter)
	if !ok {
		t.Fatal("JSONPatchFormatter should implement StructuredFormatter")
	}
	if sf == nil {
		t.Fatal("StructuredFormatter should not be nil")
	}
}

func TestJSONPatchFormatter_FormatAll(t *testing.T) {
	f := &JSONPatchFormatter{}
	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"spec", "replicas"}, Type: DiffModified, From: 3, To: 5},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: DiffPath{"spec", "type"}, Type: DiffAdded, To: "LoadBalancer"},
			},
		},
	}

	output := f.FormatAll(groups, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}

	if result[0]["file"] != "deploy.yaml" {
		t.Errorf("expected file 'deploy.yaml', got %v", result[0]["file"])
	}

	patch0 := result[0]["patch"].([]any)
	if len(patch0) != 1 {
		t.Fatalf("expected 1 op in first group, got %d", len(patch0))
	}
	op0 := patch0[0].(map[string]any)
	if op0["op"] != "replace" {
		t.Errorf("expected op 'replace', got %v", op0["op"])
	}

	if result[1]["file"] != "service.yaml" {
		t.Errorf("expected file 'service.yaml', got %v", result[1]["file"])
	}
}

func TestJSONPatchFormatter_FormatAll_SkipsEmptyGroups(t *testing.T) {
	f := &JSONPatchFormatter{}
	groups := []DiffGroup{
		{FilePath: "empty.yaml", Diffs: nil},
		{FilePath: "deploy.yaml", Diffs: []Difference{
			{Path: DiffPath{"key"}, Type: DiffAdded, To: "val"},
		}},
	}

	output := f.FormatAll(groups, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 group (empty skipped), got %d", len(result))
	}
	if result[0]["file"] != "deploy.yaml" {
		t.Errorf("expected file 'deploy.yaml', got %v", result[0]["file"])
	}
}

func TestJSONPatchFormatter_ValueSerialization(t *testing.T) {
	f := &JSONPatchFormatter{}
	// Use a multi-key OrderedMap to test map serialization.
	// Single-key OrderedMaps are expanded by expandMapKeyDiff.
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "nested", "count")
	om.Values["nested"] = "value"
	om.Values["count"] = 42
	diffs := []Difference{
		{Path: DiffPath{"int_val"}, Type: DiffModified, From: 42, To: 99},
		{Path: DiffPath{"bool_val"}, Type: DiffAdded, To: true},
		{Path: DiffPath{"map_val"}, Type: DiffAdded, To: om},
		{Path: DiffPath{"list_val"}, Type: DiffAdded, To: []any{"a", "b"}},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if result[0]["value"] != float64(99) {
		t.Errorf("expected int value 99, got %v", result[0]["value"])
	}
	if result[1]["value"] != true {
		t.Errorf("expected bool value true, got %v", result[1]["value"])
	}
	mapVal := result[2]["value"].(map[string]any)
	if mapVal["nested"] != "value" {
		t.Errorf("expected nested map value, got %v", mapVal)
	}
	listVal := result[3]["value"].([]any)
	if len(listVal) != 2 {
		t.Errorf("expected list with 2 items, got %d", len(listVal))
	}
}

func TestJSONPatchFormatter_ListIndex(t *testing.T) {
	f := &JSONPatchFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"items", "[0]", "name"}, Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if result[0]["path"] != "/items/0/name" {
		t.Errorf("expected '/items/0/name', got %v", result[0]["path"])
	}
}

// --- expandMapKeyDiff tests ---

func TestJSONPatchFormatter_ExpandMapKeyDiff_Add(t *testing.T) {
	f := &JSONPatchFormatter{}
	// Simulate how the diff engine reports a map key addition:
	// parent path with single-key *OrderedMap wrapping the new key-value.
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "monitoring")
	om.Values["monitoring"] = true
	diffs := []Difference{
		{Path: DiffPath{"app"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 op, got %d", len(result))
	}
	if result[0]["op"] != "add" {
		t.Errorf("expected op 'add', got %v", result[0]["op"])
	}
	if result[0]["path"] != "/app/monitoring" {
		t.Errorf("expected path '/app/monitoring', got %v", result[0]["path"])
	}
	if result[0]["value"] != true {
		t.Errorf("expected value true, got %v", result[0]["value"])
	}
}

func TestJSONPatchFormatter_ExpandMapKeyDiff_Remove(t *testing.T) {
	f := &JSONPatchFormatter{}
	// Simulate how the diff engine reports a map key removal:
	// parent path with single-key *OrderedMap wrapping the removed key-value.
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "debug")
	om.Values["debug"] = true
	diffs := []Difference{
		{Path: DiffPath{"app"}, Type: DiffRemoved, From: om},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 op, got %d", len(result))
	}
	if result[0]["op"] != "remove" {
		t.Errorf("expected op 'remove', got %v", result[0]["op"])
	}
	if result[0]["path"] != "/app/debug" {
		t.Errorf("expected path '/app/debug', got %v", result[0]["path"])
	}
	if _, hasValue := result[0]["value"]; hasValue {
		t.Error("remove op should not have value field")
	}
}

func TestJSONPatchFormatter_ExpandMapKeyDiff_RFC6901Escaping(t *testing.T) {
	f := &JSONPatchFormatter{}
	// Key with characters that need RFC 6901 escaping.
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "helm.sh/chart")
	om.Values["helm.sh/chart"] = "myapp-1.0"
	diffs := []Difference{
		{Path: DiffPath{"metadata", "labels"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if result[0]["path"] != "/metadata/labels/helm.sh~1chart" {
		t.Errorf("expected '/metadata/labels/helm.sh~1chart', got %v", result[0]["path"])
	}
}

func TestJSONPatchFormatter_ExpandMapKeyDiff_MultiKeyNotExpanded(t *testing.T) {
	f := &JSONPatchFormatter{}
	// Multi-key OrderedMap (e.g. a list item) should NOT be expanded.
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "name", "value")
	om.Values["name"] = "item1"
	om.Values["value"] = "data"
	diffs := []Difference{
		{Path: DiffPath{"items"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 op, got %d", len(result))
	}
	// Path stays at parent level (no expansion for multi-key)
	if result[0]["path"] != "/items" {
		t.Errorf("expected path '/items', got %v", result[0]["path"])
	}
	// Value is the full map
	mapVal := result[0]["value"].(map[string]any)
	if mapVal["name"] != "item1" {
		t.Errorf("expected name 'item1', got %v", mapVal["name"])
	}
}

func TestJSONPatchFormatter_ExpandMapKeyDiff_NestedMapValue(t *testing.T) {
	f := &JSONPatchFormatter{}
	// Adding a key whose value is a nested map.
	inner := NewOrderedMap()
	inner.Keys = append(inner.Keys, "timeout", "retries")
	inner.Values["timeout"] = 30
	inner.Values["retries"] = 5
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "config")
	om.Values["config"] = inner
	diffs := []Difference{
		{Path: DiffPath{"app"}, Type: DiffAdded, To: om},
	}

	output := f.Format(diffs, DefaultFormatOptions())

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if result[0]["path"] != "/app/config" {
		t.Errorf("expected path '/app/config', got %v", result[0]["path"])
	}
	mapVal := result[0]["value"].(map[string]any)
	if mapVal["timeout"] != float64(30) {
		t.Errorf("expected timeout 30, got %v", mapVal["timeout"])
	}
}

// --- DiffPath.JSONPointerString tests ---

func TestDiffPath_JSONPointerString(t *testing.T) {
	tests := []struct {
		path     DiffPath
		expected string
	}{
		{DiffPath{"config", "name"}, "/config/name"},
		{DiffPath{"items", "[0]", "value"}, "/items/0/value"},
		{DiffPath{"root"}, "/root"},
		{DiffPath{"a", "b", "c", "d"}, "/a/b/c/d"},
		{DiffPath{"labels", "helm.sh/chart"}, "/labels/helm.sh~1chart"},
		{DiffPath{"key~with~tilde"}, "/key~0with~0tilde"},
		{DiffPath{"mixed/and~key"}, "/mixed~1and~0key"},
		{nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.path.JSONPointerString()
			if result != tt.expected {
				t.Errorf("DiffPath%v.JSONPointerString() = %q, want %q", []string(tt.path), result, tt.expected)
			}
		})
	}
}

func TestCompactFormatter_DocumentName(t *testing.T) {
	f := &CompactFormatter{}
	diff := Difference{
		Path:         DiffPath{"[0]", "spec", "replicas"},
		Type:         DiffModified,
		From:         3,
		To:           5,
		DocumentName: "apps/v1/Deployment/web",
	}
	output := f.FormatSingle(diff, DefaultFormatOptions())
	if !strings.Contains(output, "(apps/v1/Deployment/web)") {
		t.Errorf("expected document name in compact output, got: %q", output)
	}
}

func TestDiffDescription_WithDocumentName(t *testing.T) {
	diff := Difference{
		Path:         DiffPath{"spec", "replicas"},
		Type:         DiffModified,
		From:         3,
		To:           5,
		DocumentName: "apps/v1/Deployment/web",
	}
	desc := diffDescription(diff)
	if !strings.Contains(desc, "(apps/v1/Deployment/web)") {
		t.Errorf("expected document name in description, got: %q", desc)
	}
}
