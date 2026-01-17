package diffyml

import (
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

	if opts.Width != 0 {
		t.Errorf("expected default Width 0 (auto-detect), got %d", opts.Width)
	}
	if opts.OmitHeader {
		t.Error("expected default OmitHeader to be false")
	}
	if opts.NoTableStyle {
		t.Error("expected default NoTableStyle to be false")
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

func TestColorMode_On(t *testing.T) {
	mode := ColorModeOn
	// ColorModeOn should always enable color
	enabled := ResolveColorMode(mode, false)
	if !enabled {
		t.Error("ColorModeOn should enable color even when not a terminal")
	}

	enabled = ResolveColorMode(mode, true)
	if !enabled {
		t.Error("ColorModeOn should enable color when terminal")
	}
}

func TestColorMode_Off(t *testing.T) {
	mode := ColorModeOff
	// ColorModeOff should never enable color
	enabled := ResolveColorMode(mode, true)
	if enabled {
		t.Error("ColorModeOff should disable color even when terminal")
	}

	enabled = ResolveColorMode(mode, false)
	if enabled {
		t.Error("ColorModeOff should disable color when not a terminal")
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
		{"on", ColorModeOn},
		{"ON", ColorModeOn},
		{"On", ColorModeOn},
		{"off", ColorModeOff},
		{"OFF", ColorModeOff},
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

func TestTerminalWidth_Default(t *testing.T) {
	// When width is 0, should return default
	width := GetTerminalWidth(0)
	if width <= 0 {
		t.Error("GetTerminalWidth(0) should return a positive default")
	}
}

func TestTerminalWidth_Override(t *testing.T) {
	// When width is set, should return that value
	width := GetTerminalWidth(120)
	if width != 120 {
		t.Errorf("GetTerminalWidth(120) = %d, want 120", width)
	}
}

func TestTerminalWidth_MinimumBound(t *testing.T) {
	// Should enforce minimum width
	width := GetTerminalWidth(10)
	if width < 40 {
		t.Errorf("GetTerminalWidth should enforce minimum width, got %d", width)
	}
}

func TestColorConfig_New(t *testing.T) {
	cfg := NewColorConfig(ColorModeAuto, false, 0)
	if cfg == nil {
		t.Fatal("NewColorConfig should not return nil")
	}
}

func TestColorConfig_EnableColorForTerminal(t *testing.T) {
	cfg := NewColorConfig(ColorModeAuto, false, 0)
	cfg.SetIsTerminal(true)

	if !cfg.ShouldUseColor() {
		t.Error("ColorConfig with Auto mode and terminal should enable color")
	}
}

func TestColorConfig_DisableColorForNonTerminal(t *testing.T) {
	cfg := NewColorConfig(ColorModeAuto, false, 0)
	cfg.SetIsTerminal(false)

	if cfg.ShouldUseColor() {
		t.Error("ColorConfig with Auto mode and non-terminal should disable color")
	}
}

func TestColorConfig_TrueColor(t *testing.T) {
	cfg := NewColorConfig(ColorModeOn, true, 0)
	cfg.SetIsTerminal(true)

	if !cfg.ShouldUseTrueColor() {
		t.Error("ColorConfig with truecolor enabled should use true color")
	}
}

func TestColorConfig_TrueColorDisabled(t *testing.T) {
	cfg := NewColorConfig(ColorModeOn, false, 0)
	cfg.SetIsTerminal(true)

	if cfg.ShouldUseTrueColor() {
		t.Error("ColorConfig without truecolor flag should not use true color")
	}
}

func TestColorConfig_Width(t *testing.T) {
	cfg := NewColorConfig(ColorModeOn, false, 100)

	width := cfg.GetWidth()
	if width != 100 {
		t.Errorf("ColorConfig.GetWidth() = %d, want 100", width)
	}
}

func TestColorConfig_DefaultWidth(t *testing.T) {
	cfg := NewColorConfig(ColorModeOn, false, 0)

	width := cfg.GetWidth()
	if width <= 0 {
		t.Error("ColorConfig.GetWidth() should return positive default when not set")
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

	// GitHub Actions workflow command format: ::warning ::{message}
	if !containsSubstr(output, "::warning ::") {
		t.Errorf("expected GitHub Actions warning format, got: %s", output)
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
