package diffyml

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestGetFormatter_Compact(t *testing.T) {
	f, err := GetFormatter("compact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestGetFormatter_Brief(t *testing.T) {
	f, err := GetFormatter("brief")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestGetFormatter_GitHub(t *testing.T) {
	f, err := GetFormatter("github")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestGetFormatter_GitLab(t *testing.T) {
	f, err := GetFormatter("gitlab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestGetFormatter_Gitea(t *testing.T) {
	f, err := GetFormatter("gitea")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestGetFormatter_Invalid(t *testing.T) {
	_, err := GetFormatter("invalid")
	if err == nil {
		t.Error("expected error for invalid formatter name")
	}
}

func TestGetFormatter_EmptyName(t *testing.T) {
	_, err := GetFormatter("")
	if err == nil {
		t.Error("expected error for empty formatter name")
	}
}

func TestGetFormatter_CaseInsensitive(t *testing.T) {
	// Formatter names should be case-insensitive for convenience
	f, err := GetFormatter("COMPACT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected formatter, got nil")
	}
}

func TestGetFormatter_ListsValidFormats(t *testing.T) {
	_, err := GetFormatter("badname")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	// Error message should list valid formats
	errStr := err.Error()
	expectedFormats := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}
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
	if opts.MinorChangeThreshold != 0.1 {
		t.Errorf("expected default MinorChangeThreshold 0.1, got %f", opts.MinorChangeThreshold)
	}
	if opts.FilePath != "" {
		t.Errorf("expected default FilePath to be empty, got %q", opts.FilePath)
	}
}

func TestDiffGroup_Construction(t *testing.T) {
	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "old", To: "new"},
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
	formatters := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}

	diffs := []Difference{
		{Path: "test.path", Type: DiffModified, From: "old", To: "new"},
	}
	opts := DefaultFormatOptions()

	for _, name := range formatters {
		t.Run(name, func(t *testing.T) {
			f, err := GetFormatter(name)
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
	formatters := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}

	diffs := []Difference{}
	opts := DefaultFormatOptions()

	for _, name := range formatters {
		t.Run(name, func(t *testing.T) {
			f, err := GetFormatter(name)
			if err != nil {
				t.Fatalf("failed to get formatter: %v", err)
			}

			// Should handle empty diffs gracefully (no panic)
			_ = f.Format(diffs, opts)
		})
	}
}

func TestFormatter_NilOptions(t *testing.T) {
	f, _ := GetFormatter("compact")

	diffs := []Difference{
		{Path: "test", Type: DiffAdded, From: nil, To: "value"},
	}

	// Should handle nil options gracefully (no panic)
	output := f.Format(diffs, nil)
	if output == "" {
		t.Error("formatter should produce output even with nil options")
	}
}

func TestConvertToGoPatchPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"config.name", "/config/name"},
		{"items[0].value", "/items/0/value"},
		{"root", "/root"},
		{"a.b.c.d", "/a/b/c/d"},
		{"list[0][1].nested", "/list/0/1/nested"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertToGoPatchPath(tt.input)
			if result != tt.expected {
				t.Errorf("convertToGoPatchPath(%q) = %q, want %q", tt.input, result, tt.expected)
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
	f, _ := GetFormatter("brief")
	opts := DefaultFormatOptions()

	tests := []struct {
		name     string
		diffs    []Difference
		expected []string
	}{
		{
			name: "single added",
			diffs: []Difference{
				{Path: "key", Type: DiffAdded, To: "value"},
			},
			expected: []string{"1 added"},
		},
		{
			name: "single removed",
			diffs: []Difference{
				{Path: "key", Type: DiffRemoved, From: "value"},
			},
			expected: []string{"1 removed"},
		},
		{
			name: "single modified",
			diffs: []Difference{
				{Path: "key", Type: DiffModified, From: "old", To: "new"},
			},
			expected: []string{"1 modified"},
		},
		{
			name: "mixed changes",
			diffs: []Difference{
				{Path: "a", Type: DiffAdded, To: "new"},
				{Path: "b", Type: DiffAdded, To: "new2"},
				{Path: "c", Type: DiffRemoved, From: "old"},
				{Path: "d", Type: DiffModified, From: "old", To: "new"},
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
	f, _ := GetFormatter("brief")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	if !containsSubstr(output, "no differences") {
		t.Errorf("expected 'no differences' message, got: %s", output)
	}
}

func TestGitHubFormatter_WorkflowCommandFormat(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	tests := []struct {
		name     string
		diff     Difference
		expected string
	}{
		{
			name:     "added",
			diff:     Difference{Path: "key", Type: DiffAdded, To: "value"},
			expected: "Added:",
		},
		{
			name:     "removed",
			diff:     Difference{Path: "key", Type: DiffRemoved, From: "value"},
			expected: "Removed:",
		},
		{
			name:     "modified",
			diff:     Difference{Path: "key", Type: DiffModified, From: "old", To: "new"},
			expected: "Modified:",
		},
		{
			name:     "order changed",
			diff:     Difference{Path: "list", Type: DiffOrderChanged},
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	// GitHub formatter returns empty string for no differences
	if output != "" {
		t.Errorf("expected empty output for no differences, got: %s", output)
	}
}

func TestGitLabFormatter_CodeQualityJSON(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
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
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	// Empty differences should return empty JSON array
	if !containsSubstr(output, "[]") {
		t.Errorf("expected empty JSON array for no differences, got: %s", output)
	}
}

func TestGitLabFormatter_MultipleDiffs(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "a", Type: DiffAdded, To: "new"},
		{Path: "b", Type: DiffRemoved, From: "old"},
	}

	output := f.Format(diffs, opts)
	// Should have proper JSON array with comma separation
	if !containsSubstr(output, ",") {
		t.Errorf("expected comma-separated JSON entries, got: %s", output)
	}
}

func TestGiteaFormatter_GitHubCompatible(t *testing.T) {
	giteaF, _ := GetFormatter("gitea")
	githubF, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.value", Type: DiffModified, From: "old", To: "new"},
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
	f, _ := GetFormatter("compact")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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
	f, _ := GetFormatter("compact")
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
				{Path: "test.path", Type: tt.diffType, From: "old", To: "new"},
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
	f, _ := GetFormatter("compact")
	diffs := []Difference{
		{Path: "test", Type: DiffAdded, From: nil, To: "new"},
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	tests := []struct {
		name            string
		diff            Difference
		expectedCommand string
		expectedTitle   string
	}{
		{
			name:            "added uses notice",
			diff:            Difference{Path: "key", Type: DiffAdded, To: "value"},
			expectedCommand: "::notice",
			expectedTitle:   "title=YAML Added",
		},
		{
			name:            "removed uses error",
			diff:            Difference{Path: "key", Type: DiffRemoved, From: "value"},
			expectedCommand: "::error",
			expectedTitle:   "title=YAML Removed",
		},
		{
			name:            "modified uses warning",
			diff:            Difference{Path: "key", Type: DiffModified, From: "old", To: "new"},
			expectedCommand: "::warning",
			expectedTitle:   "title=YAML Modified",
		},
		{
			name:            "order changed uses notice",
			diff:            Difference{Path: "list", Type: DiffOrderChanged},
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()
	// FilePath is empty — backward compatible

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()
	opts.FilePath = "service.yaml"

	tests := []struct {
		name     string
		diff     Difference
		expected string
	}{
		{
			name:     "added with file",
			diff:     Difference{Path: "key", Type: DiffAdded, To: "value"},
			expected: "::notice file=service.yaml,title=YAML Added::Added: key = value\n",
		},
		{
			name:     "removed with file",
			diff:     Difference{Path: "key", Type: DiffRemoved, From: "value"},
			expected: "::error file=service.yaml,title=YAML Removed::Removed: key = value\n",
		},
		{
			name:     "modified with file",
			diff:     Difference{Path: "key", Type: DiffModified, From: "old", To: "new"},
			expected: "::warning file=service.yaml,title=YAML Modified::Modified: key changed from old to new\n",
		},
		{
			name:     "order changed with file",
			diff:     Difference{Path: "list", Type: DiffOrderChanged},
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	// Generate 13 warning diffs (DiffModified → ::warning)
	var diffs []Difference
	for i := 0; i < 13; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("key%d", i),
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	// Generate exactly 10 warning diffs — should not trigger summary
	var diffs []Difference
	for i := 0; i < 10; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("key%d", i),
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	// DiffAdded and DiffOrderChanged both map to ::notice — they share the budget
	var diffs []Difference
	for i := 0; i < 7; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("added%d", i),
			Type: DiffAdded,
			To:   "val",
		})
	}
	for i := 0; i < 5; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("order%d", i),
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
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	// 12 notices (DiffAdded) + 11 warnings (DiffModified) + 3 errors (DiffRemoved)
	var diffs []Difference
	for i := 0; i < 12; i++ {
		diffs = append(diffs, Difference{Path: fmt.Sprintf("a%d", i), Type: DiffAdded, To: "v"})
	}
	for i := 0; i < 11; i++ {
		diffs = append(diffs, Difference{Path: fmt.Sprintf("m%d", i), Type: DiffModified, From: "o", To: "n"})
	}
	for i := 0; i < 3; i++ {
		diffs = append(diffs, Difference{Path: fmt.Sprintf("r%d", i), Type: DiffRemoved, From: "v"})
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
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.key", Type: DiffAdded, To: "value"},
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
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	tests := []struct {
		name             string
		diff             Difference
		expectedSeverity string
	}{
		{
			name:             "added is info",
			diff:             Difference{Path: "key", Type: DiffAdded, To: "val"},
			expectedSeverity: `"severity": "info"`,
		},
		{
			name:             "removed is major",
			diff:             Difference{Path: "key", Type: DiffRemoved, From: "val"},
			expectedSeverity: `"severity": "major"`,
		},
		{
			name:             "modified is major",
			diff:             Difference{Path: "key", Type: DiffModified, From: "old", To: "new"},
			expectedSeverity: `"severity": "major"`,
		},
		{
			name:             "order changed is minor",
			diff:             Difference{Path: "list", Type: DiffOrderChanged},
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
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	tests := []struct {
		name              string
		diff              Difference
		expectedCheckName string
	}{
		{
			name:              "added check name",
			diff:              Difference{Path: "key", Type: DiffAdded, To: "val"},
			expectedCheckName: "diffyml/added",
		},
		{
			name:              "removed check name",
			diff:              Difference{Path: "key", Type: DiffRemoved, From: "val"},
			expectedCheckName: "diffyml/removed",
		},
		{
			name:              "modified check name",
			diff:              Difference{Path: "key", Type: DiffModified, From: "old", To: "new"},
			expectedCheckName: "diffyml/modified",
		},
		{
			name:              "order changed check name",
			diff:              Difference{Path: "list", Type: DiffOrderChanged},
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
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.key", Type: DiffAdded, To: "value1"},
		{Path: "config.key", Type: DiffRemoved, From: "value2"},
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
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.key", Type: DiffModified, From: "old", To: "new"},
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
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
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
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
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
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
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
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)
	fp := extractFingerprint(t, output)

	// Should match the old fingerprint formula: sha256(description)
	desc := gitLabDescription(diffs[0])
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
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: DiffAdded, To: 8080},
		{Path: "config.old", Type: DiffRemoved, From: "value"},
		{Path: "items", Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)

	// All YAML paths should appear in descriptions
	for _, d := range diffs {
		if !strings.Contains(output, d.Path) {
			t.Errorf("expected YAML path %q in description, got: %s", d.Path, output)
		}
	}
}

func TestGitLabFormatter_ValidJSON_WithFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: DiffAdded, To: 8080},
	}

	output := f.Format(diffs, opts)

	var result []map[string]interface{}
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
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
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
				{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: "service.port", Type: DiffAdded, To: 8080},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]interface{}
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
				{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
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
				{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
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
				{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
			},
		},
		{
			FilePath: "file2.yaml",
			Diffs: []Difference{
				{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]interface{}
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
				{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
				{Path: "config.port", Type: DiffAdded, To: 8080},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: "service.name", Type: DiffRemoved, From: "old-svc"},
			},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]interface{}
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
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: DiffAdded, To: 8080},
		{Path: "config.old", Type: DiffRemoved, From: "value"},
		{Path: "items", Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)

	// Parse as JSON to verify structural validity
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 entries, got %d", len(result))
	}

	// Verify location.path falls back to diff.Path (YAML key path)
	for i, entry := range result {
		location := entry["location"].(map[string]interface{})
		path := location["path"].(string)
		if path != diffs[i].Path {
			t.Errorf("entry %d: expected location.path=%q (fallback to diff.Path), got %q", i, diffs[i].Path, path)
		}
	}

	// Verify fingerprints match the legacy formula: sha256(description only)
	for i, entry := range result {
		fp := entry["fingerprint"].(string)
		desc := gitLabDescription(diffs[i])
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
	diff := Difference{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"}
	desc := gitLabDescription(diff)

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
		{Path: "added.key", Type: DiffAdded, To: "value"},
		{Path: "removed.key", Type: DiffRemoved, From: "value"},
		{Path: "modified.key", Type: DiffModified, From: "old", To: "new"},
		{Path: "order.key", Type: DiffOrderChanged},
	}

	output := f.Format(allDiffs, opts)

	var result []map[string]interface{}
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
		location := entry["location"].(map[string]interface{})
		if _, ok := location["path"]; !ok {
			t.Errorf("entry %d: location missing 'path'", i)
		}
		lines := location["lines"].(map[string]interface{})
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
		{Path: "key", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, nil)

	var result []map[string]interface{}
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

	var result []interface{}
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
		{Path: "config.msg", Type: DiffModified, From: `line1\nline2`, To: `"quoted value"`},
		{Path: "config.tab", Type: DiffModified, From: "no\ttab", To: "has\ttab"},
	}

	output := f.Format(diffs, opts)

	var result []map[string]interface{}
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
		{Path: "key.added", Type: DiffAdded, To: "value"},
		{Path: "key.removed", Type: DiffRemoved, From: "value"},
		{Path: "key.modified", Type: DiffModified, From: "old", To: "new"},
		{Path: "key.order", Type: DiffOrderChanged},
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
			t.Run(name+"/"+diff.Path, func(t *testing.T) {
				output := f.FormatSingle(diff, opts)
				if output == "" {
					t.Errorf("%s.FormatSingle returned empty for %s", name, diff.Path)
				}
			})
		}
	}
}

func TestCompactFormatter_FormatSingle_NilOpts(t *testing.T) {
	f := &CompactFormatter{}
	diff := Difference{Path: "key", Type: DiffAdded, To: "value"}

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
		{Path: "key.a", Type: DiffAdded, To: "new"},
		{Path: "key.r", Type: DiffRemoved, From: "old"},
		{Path: "key.m", Type: DiffModified, From: "old", To: "new"},
		{Path: "key.o", Type: DiffOrderChanged},
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

// DiffOrderChanged compact indicator test

func TestCompactFormatter_OrderChangedIndicator(t *testing.T) {
	f, _ := GetFormatter("compact")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "list.items", Type: DiffOrderChanged},
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

	diff := Difference{Path: "config.items[0].name", Type: DiffModified, From: "old", To: "new"}
	output := f.FormatSingle(diff, opts)
	if !strings.Contains(output, "/config/items/0/name") {
		t.Errorf("expected Go-Patch style path, got: %s", output)
	}
}

func TestFormatValue_Nil(t *testing.T) {
	f := &CompactFormatter{}
	opts := DefaultFormatOptions()

	// Modified diff with nil From value exercises formatValue(nil, opts)
	diff := Difference{Path: "key", Type: DiffModified, From: nil, To: "new"}
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
				{Path: "key", Type: DiffModified, From: "old", To: "new"},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: "port", Type: DiffAdded, To: 8080},
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
				{Path: "key", Type: DiffModified, From: "old", To: "new"},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs: []Difference{
				{Path: "port", Type: DiffAdded, To: 8080},
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
			Path: fmt.Sprintf("key%d", i),
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
		{Path: "a", Type: DiffAdded, To: "x"},
		{Path: "b", Type: DiffAdded, To: "y"},
		{Path: "c", Type: DiffRemoved, From: "z"},
		{Path: "d", Type: DiffModified, From: "old", To: "new"},
	}

	f := &CompactFormatter{}
	opts := &FormatOptions{Color: false}
	output := f.Format(diffs, opts)

	// Use exact format to distinguish "N category" from "-N category" (INCREMENT_DECREMENT mutation)
	if !strings.Contains(output, "(2 added,") {
		t.Errorf("expected '(2 added,' in header, got: %s", output)
	}
	if !strings.Contains(output, " 1 removed,") {
		t.Errorf("expected ' 1 removed,' in header, got: %s", output)
	}
	if !strings.Contains(output, " 1 modified)") {
		t.Errorf("expected ' 1 modified)' in header, got: %s", output)
	}
}

// --- Mutation testing: formatter.go brief formatter zero categories ---

func TestBriefFormatter_ZeroCategories(t *testing.T) {
	// Only-added diffs → output should have no "removed" or "modified"
	diffs := []Difference{
		{Path: "a", Type: DiffAdded, To: "x"},
		{Path: "b", Type: DiffAdded, To: "y"},
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
		{Path: "a", Type: DiffModified, From: "old", To: "new"},
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
