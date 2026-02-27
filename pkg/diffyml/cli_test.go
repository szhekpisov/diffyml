package diffyml

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIConfig_Defaults(t *testing.T) {
	cfg := NewCLIConfig()

	// Check all default values match spec
	if cfg.Output != "detailed" {
		t.Errorf("expected default Output='detailed', got %q", cfg.Output)
	}
	if cfg.Color != "auto" {
		t.Errorf("expected default Color='auto', got %q", cfg.Color)
	}
	if cfg.TrueColor != "auto" {
		t.Errorf("expected default TrueColor='auto', got %q", cfg.TrueColor)
	}
	if cfg.FixedWidth != -1 {
		t.Errorf("expected default FixedWidth=-1, got %d", cfg.FixedWidth)
	}
	if !cfg.DetectKubernetes {
		t.Error("expected default DetectKubernetes=true")
	}
	if !cfg.DetectRenames {
		t.Error("expected default DetectRenames=true")
	}
	if cfg.MinorChangeThreshold != 0.1 {
		t.Errorf("expected default MinorChangeThreshold=0.1, got %f", cfg.MinorChangeThreshold)
	}
	if cfg.MultiLineContextLines != 4 {
		t.Errorf("expected default MultiLineContextLines=4, got %d", cfg.MultiLineContextLines)
	}
	if cfg.IgnoreApiVersion {
		t.Error("expected default IgnoreApiVersion=false")
	}
}

func TestCLIConfig_ParseArgs_TwoFiles(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.FromFile != "from.yaml" {
		t.Errorf("expected FromFile='from.yaml', got %q", cfg.FromFile)
	}
	if cfg.ToFile != "to.yaml" {
		t.Errorf("expected ToFile='to.yaml', got %q", cfg.ToFile)
	}
}

func TestCLIConfig_ParseArgs_MissingFiles(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{}

	err := cfg.ParseArgs(args)
	if err == nil {
		t.Error("expected error for missing file arguments")
	}
}

func TestCLIConfig_ParseArgs_OnlyOneFile(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"only.yaml"}

	err := cfg.ParseArgs(args)
	if err == nil {
		t.Error("expected error for only one file argument")
	}
}

func TestCLIConfig_ParseArgs_WithFlags(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"-o", "brief", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "brief" {
		t.Errorf("expected Output='brief', got %q", cfg.Output)
	}
}

func TestCLIConfig_ParseArgs_IgnoreOrderChanges(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"-i", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true with -i flag")
	}
}

func TestCLIConfig_ParseArgs_SetExitCode(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"-s", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.SetExitCode {
		t.Error("expected SetExitCode=true with -s flag")
	}
}

func TestCLIConfig_ParseArgs_ColorAlways(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"-c", "always", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Color != "always" {
		t.Errorf("expected Color='always', got %q", cfg.Color)
	}
}

func TestCLIConfig_ParseArgs_LongFlags(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--output", "github", "--ignore-order-changes", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "github" {
		t.Errorf("expected Output='github', got %q", cfg.Output)
	}
	if !cfg.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true")
	}
}

func TestCLIConfig_ParseArgs_FilterAndExclude(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--filter", "config", "--exclude", "secret", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Filter) != 1 || cfg.Filter[0] != "config" {
		t.Errorf("expected Filter=['config'], got %v", cfg.Filter)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "secret" {
		t.Errorf("expected Exclude=['secret'], got %v", cfg.Exclude)
	}
}

func TestCLIConfig_ParseArgs_ChrootOptions(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--chroot", "data.items", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Chroot != "data.items" {
		t.Errorf("expected Chroot='data.items', got %q", cfg.Chroot)
	}
}

func TestCLIConfig_ParseArgs_Swap(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--swap", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Swap {
		t.Error("expected Swap=true")
	}
}

func TestCLIConfig_ParseArgs_FlagsAfterPositionalArgs(t *testing.T) {
	cfg := NewCLIConfig()
	// Simulates kubectl's KUBECTL_EXTERNAL_DIFF arg order: dirs first, flags after
	args := []string{"from.yaml", "to.yaml", "--set-exit-code", "--omit-header", "--color", "never"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.FromFile != "from.yaml" {
		t.Errorf("expected FromFile='from.yaml', got %q", cfg.FromFile)
	}
	if cfg.ToFile != "to.yaml" {
		t.Errorf("expected ToFile='to.yaml', got %q", cfg.ToFile)
	}
	if !cfg.SetExitCode {
		t.Error("expected SetExitCode=true")
	}
	if !cfg.OmitHeader {
		t.Error("expected OmitHeader=true")
	}
	if cfg.Color != "never" {
		t.Errorf("expected Color='never', got %q", cfg.Color)
	}
}

func TestCLIConfig_ParseArgs_FlagsMixedWithPositionalArgs(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--omit-header", "from.yaml", "to.yaml", "--set-exit-code"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.FromFile != "from.yaml" {
		t.Errorf("expected FromFile='from.yaml', got %q", cfg.FromFile)
	}
	if !cfg.OmitHeader {
		t.Error("expected OmitHeader=true")
	}
	if !cfg.SetExitCode {
		t.Error("expected SetExitCode=true")
	}
}

func TestCLIConfig_ParseArgs_DoubleDashTerminator(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--set-exit-code", "--", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.FromFile != "from.yaml" {
		t.Errorf("expected FromFile='from.yaml', got %q", cfg.FromFile)
	}
	if !cfg.SetExitCode {
		t.Error("expected SetExitCode=true")
	}
}

func TestCLIConfig_ParseArgs_EqualsForm(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"from.yaml", "to.yaml", "--color=never", "--output=compact"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Color != "never" {
		t.Errorf("expected Color='never', got %q", cfg.Color)
	}
	if cfg.Output != "compact" {
		t.Errorf("expected Output='compact', got %q", cfg.Output)
	}
}

func TestCLIConfig_ParseArgs_UnknownFlagAfterPositional(t *testing.T) {
	cfg := NewCLIConfig()
	// Unknown flags are passed through as positional args by reorderArgs,
	// then fs.Parse reports the error.
	args := []string{"--unknown-flag", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestCLIConfig_ToCompareOptions(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.IgnoreOrderChanges = true
	cfg.IgnoreWhitespaceChanges = true
	cfg.Swap = true
	cfg.Chroot = "data"

	opts := cfg.ToCompareOptions()

	if !opts.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true in Options")
	}
	if !opts.IgnoreWhitespaceChanges {
		t.Error("expected IgnoreWhitespaceChanges=true in Options")
	}
	if !opts.Swap {
		t.Error("expected Swap=true in Options")
	}
	if opts.Chroot != "data" {
		t.Errorf("expected Chroot='data', got %q", opts.Chroot)
	}
}

func TestCLIConfig_ToFilterOptions(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Filter = []string{"config"}
	cfg.Exclude = []string{"secret"}
	cfg.FilterRegexp = []string{`^test\.`}
	cfg.ExcludeRegexp = []string{`password`}

	opts := cfg.ToFilterOptions()

	if len(opts.IncludePaths) != 1 || opts.IncludePaths[0] != "config" {
		t.Errorf("expected IncludePaths=['config'], got %v", opts.IncludePaths)
	}
	if len(opts.ExcludePaths) != 1 || opts.ExcludePaths[0] != "secret" {
		t.Errorf("expected ExcludePaths=['secret'], got %v", opts.ExcludePaths)
	}
	if len(opts.IncludeRegexp) != 1 {
		t.Errorf("expected IncludeRegexp length 1, got %d", len(opts.IncludeRegexp))
	}
	if len(opts.ExcludeRegexp) != 1 {
		t.Errorf("expected ExcludeRegexp length 1, got %d", len(opts.ExcludeRegexp))
	}
}

func TestCLIConfig_ToFormatOptions(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.OmitHeader = true
	cfg.NoTableStyle = true
	cfg.UseGoPatchStyle = true
	cfg.MultiLineContextLines = 10
	cfg.MinorChangeThreshold = 0.2

	opts := cfg.ToFormatOptions()

	if !opts.OmitHeader {
		t.Error("expected OmitHeader=true")
	}
	if !opts.NoTableStyle {
		t.Error("expected NoTableStyle=true")
	}
	if !opts.UseGoPatchStyle {
		t.Error("expected UseGoPatchStyle=true")
	}
	if opts.ContextLines != 10 {
		t.Errorf("expected ContextLines=10, got %d", opts.ContextLines)
	}
	if opts.MinorChangeThreshold != 0.2 {
		t.Errorf("expected MinorChangeThreshold=0.2, got %f", opts.MinorChangeThreshold)
	}
}

func TestCLIConfig_UsageContainsFlags(t *testing.T) {
	cfg := NewCLIConfig()
	usage := cfg.Usage()

	expectedFlags := []string{
		"-o", "--output",
		"-c", "--color",
		"-i", "--ignore-order-changes",
		"-s", "--set-exit-code",
		"-h", "--help",
		"--filter",
		"--exclude",
		"--chroot",
		"--swap",
	}

	for _, flag := range expectedFlags {
		if !containsSubstr(usage, flag) {
			t.Errorf("usage should contain flag %q", flag)
		}
	}
}

func TestCLIConfig_UsageAlignment(t *testing.T) {
	cfg := NewCLIConfig()
	usage := cfg.Usage()

	const descColumn = 38

	for _, line := range strings.Split(usage, "\n") {
		// Skip non-flag lines (header, section breaks, empty lines)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "-") {
			continue
		}

		// Find where the description starts: first lowercase letter after
		// the flag/type tokens, preceded by at least two spaces.
		descIdx := strings.Index(line, "  ")
		if descIdx == -1 {
			continue
		}
		// Walk past all the padding spaces to find the description start
		for descIdx < len(line) && line[descIdx] == ' ' {
			descIdx++
		}
		// Skip the flag token itself — we need the *second* run of 2+ spaces
		// (the first run is the leading indent).
		// Strategy: find two-or-more spaces that appear after position 6
		// (past the short-flag column).
		pos := 6
		for pos < len(line)-1 {
			if line[pos] == ' ' && line[pos+1] == ' ' {
				// Found a gap — skip all spaces to reach the description
				start := pos
				for start < len(line) && line[start] == ' ' {
					start++
				}
				if start < len(line) {
					if start != descColumn {
						t.Errorf("description not at column %d (got %d) in line: %s", descColumn, start, line)
					}
					break
				}
			}
			pos++
		}
	}
}

// Tests for input validation (Task 5.2)

func TestCLIConfig_Validate_ValidOutput(t *testing.T) {
	validFormats := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}
	for _, format := range validFormats {
		cfg := NewCLIConfig()
		cfg.Output = format
		cfg.FromFile = "from.yaml"
		cfg.ToFile = "to.yaml"

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error for output format %q, got: %v", format, err)
		}
	}
}

func TestCLIConfig_Validate_InvalidOutput(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "invalid"
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for invalid output format")
	}
	// Should list valid options
	if !containsSubstr(err.Error(), "compact") {
		t.Error("error should list valid options including 'compact'")
	}
}

func TestCLIConfig_Validate_InvalidColor(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Color = "invalid"
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for invalid color mode")
	}
	if !containsSubstr(err.Error(), "color") {
		t.Error("error should mention color mode")
	}
}

func TestCLIConfig_Validate_InvalidTrueColor(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.TrueColor = "invalid"
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for invalid truecolor mode")
	}
}

func TestCLIConfig_Validate_ValidRegexPatterns(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.FilterRegexp = []string{`^test\.`, `config\.\d+`}
	cfg.ExcludeRegexp = []string{`password`}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error for valid regex patterns, got: %v", err)
	}
}

func TestCLIConfig_Validate_InvalidFilterRegexp(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.FilterRegexp = []string{`[invalid`} // Invalid regex

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for invalid filter regex")
	}
	if !containsSubstr(err.Error(), "filter-regexp") {
		t.Error("error should mention filter-regexp")
	}
}

func TestCLIConfig_Validate_InvalidExcludeRegexp(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.ExcludeRegexp = []string{`(unclosed`} // Invalid regex

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for invalid exclude regex")
	}
	if !containsSubstr(err.Error(), "exclude-regexp") {
		t.Error("error should mention exclude-regexp")
	}
}

func TestCLIConfig_Validate_MissingFromFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.ToFile = "to.yaml"
	// FromFile is empty

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing from file")
	}
}

func TestCLIConfig_Validate_MissingToFile(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	// ToFile is empty

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing to file")
	}
}

func TestValidateFileExists_NonExistent(t *testing.T) {
	err := ValidateFileExists("/nonexistent/path/file.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	if !containsSubstr(err.Error(), "/nonexistent/path/file.yaml") {
		t.Error("error should include the file path")
	}
}

func TestValidateFileExists_Directory(t *testing.T) {
	// "." is a directory, not a file
	err := ValidateFileExists(".")
	if err == nil {
		t.Error("expected error when path is a directory")
	}
	if !containsSubstr(err.Error(), "directory") {
		t.Error("error should mention that path is a directory")
	}
}

func TestValidateOutputFormat_Valid(t *testing.T) {
	validFormats := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed", "COMPACT", "Compact", "BRIEF", "Brief", "DETAILED", "Detailed"}
	for _, format := range validFormats {
		err := ValidateOutputFormat(format)
		if err != nil {
			t.Errorf("expected no error for format %q, got: %v", format, err)
		}
	}
}

func TestValidateOutputFormat_Invalid(t *testing.T) {
	err := ValidateOutputFormat("unknown")
	if err == nil {
		t.Error("expected error for invalid format")
	}
	// Should list valid options
	if !containsSubstr(err.Error(), "compact") || !containsSubstr(err.Error(), "brief") {
		t.Error("error should list valid format options")
	}
}

func TestValidateRegexPatterns_Valid(t *testing.T) {
	patterns := []string{`^test`, `\d+`, `foo\.bar`}
	err := ValidateRegexPatterns(patterns, "test-flag")
	if err != nil {
		t.Errorf("expected no error for valid patterns, got: %v", err)
	}
}

func TestValidateRegexPatterns_Invalid(t *testing.T) {
	patterns := []string{`valid`, `[invalid`}
	err := ValidateRegexPatterns(patterns, "test-flag")
	if err == nil {
		t.Error("expected error for invalid pattern")
	}
	if !containsSubstr(err.Error(), "[invalid") {
		t.Error("error should include the invalid pattern")
	}
	if !containsSubstr(err.Error(), "test-flag") {
		t.Error("error should include the flag name")
	}
}

func TestValidateRegexPatterns_Empty(t *testing.T) {
	err := ValidateRegexPatterns(nil, "test-flag")
	if err != nil {
		t.Errorf("expected no error for empty patterns, got: %v", err)
	}
	err = ValidateRegexPatterns([]string{}, "test-flag")
	if err != nil {
		t.Errorf("expected no error for empty slice, got: %v", err)
	}
}

// Tests for exit code handling (Task 5.3)

func TestExitCodes_Constants(t *testing.T) {
	// Verify exit code constants match spec
	if ExitCodeSuccess != 0 {
		t.Errorf("expected ExitCodeSuccess=0, got %d", ExitCodeSuccess)
	}
	if ExitCodeDifferences != 1 {
		t.Errorf("expected ExitCodeDifferences=1, got %d", ExitCodeDifferences)
	}
	if ExitCodeError != 255 {
		t.Errorf("expected ExitCodeError=255, got %d", ExitCodeError)
	}
}

func TestDetermineExitCode_WithSetExitCode_NoDifferences(t *testing.T) {
	code := DetermineExitCode(true, 0, nil)
	if code != ExitCodeSuccess {
		t.Errorf("expected exit code %d with -s and no differences, got %d", ExitCodeSuccess, code)
	}
}

func TestDetermineExitCode_WithSetExitCode_HasDifferences(t *testing.T) {
	code := DetermineExitCode(true, 5, nil)
	if code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with -s and differences, got %d", ExitCodeDifferences, code)
	}
}

func TestDetermineExitCode_WithSetExitCode_HasError(t *testing.T) {
	code := DetermineExitCode(true, 0, fmt.Errorf("some error"))
	if code != ExitCodeError {
		t.Errorf("expected exit code %d with -s and error, got %d", ExitCodeError, code)
	}
}

func TestDetermineExitCode_WithSetExitCode_ErrorTakesPrecedence(t *testing.T) {
	// Error should take precedence over differences
	code := DetermineExitCode(true, 5, fmt.Errorf("some error"))
	if code != ExitCodeError {
		t.Errorf("expected exit code %d when error present, got %d", ExitCodeError, code)
	}
}

func TestDetermineExitCode_WithoutSetExitCode_NoDifferences(t *testing.T) {
	code := DetermineExitCode(false, 0, nil)
	if code != ExitCodeSuccess {
		t.Errorf("expected exit code %d without -s and no differences, got %d", ExitCodeSuccess, code)
	}
}

func TestDetermineExitCode_WithoutSetExitCode_HasDifferences(t *testing.T) {
	// Without -s flag, should still return 0 even with differences
	code := DetermineExitCode(false, 5, nil)
	if code != ExitCodeSuccess {
		t.Errorf("expected exit code %d without -s flag (regardless of differences), got %d", ExitCodeSuccess, code)
	}
}

func TestDetermineExitCode_WithoutSetExitCode_HasError(t *testing.T) {
	// Error still returns error code even without -s flag
	code := DetermineExitCode(false, 0, fmt.Errorf("some error"))
	if code != ExitCodeError {
		t.Errorf("expected exit code %d on error, got %d", ExitCodeError, code)
	}
}

func TestExitResult_Success(t *testing.T) {
	result := NewExitResult(0, nil)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected code %d, got %d", ExitCodeSuccess, result.Code)
	}
	if result.Err != nil {
		t.Errorf("expected nil error, got %v", result.Err)
	}
	if !result.IsSuccess() {
		t.Error("expected IsSuccess() to return true")
	}
}

func TestExitResult_WithError(t *testing.T) {
	err := fmt.Errorf("test error")
	result := NewExitResult(ExitCodeError, err)
	if result.Code != ExitCodeError {
		t.Errorf("expected code %d, got %d", ExitCodeError, result.Code)
	}
	if result.Err != err {
		t.Errorf("expected error %v, got %v", err, result.Err)
	}
	if result.IsSuccess() {
		t.Error("expected IsSuccess() to return false")
	}
}

func TestExitResult_HasDifferences(t *testing.T) {
	result := NewExitResult(ExitCodeDifferences, nil)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected code %d, got %d", ExitCodeDifferences, result.Code)
	}
	if result.HasDifferences() != true {
		t.Error("expected HasDifferences() to return true")
	}
}

func TestExitResult_String(t *testing.T) {
	tests := []struct {
		code     int
		err      error
		contains string
	}{
		{ExitCodeSuccess, nil, "success"},
		{ExitCodeDifferences, nil, "differences"},
		{ExitCodeError, fmt.Errorf("parse failed"), "parse failed"},
	}

	for _, tc := range tests {
		result := NewExitResult(tc.code, tc.err)
		str := result.String()
		if !containsSubstr(str, tc.contains) {
			t.Errorf("expected String() to contain %q, got %q", tc.contains, str)
		}
	}
}

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

// Task 7.3 Tests - CLI and help text for new format names

func TestCLI_Usage_ListsAllFiveFormats(t *testing.T) {
	cfg := NewCLIConfig()
	usage := cfg.Usage()

	// All five formats should be listed in the usage text
	formats := []string{"compact", "brief", "github", "gitlab", "gitea"}
	for _, format := range formats {
		if !containsSubstr(usage, format) {
			t.Errorf("Usage() should list format %q", format)
		}
	}
}

func TestCLI_Usage_OutputFlagDescriptionIncludesCompact(t *testing.T) {
	cfg := NewCLIConfig()
	usage := cfg.Usage()

	// The --output flag description should include "compact"
	// Look for the line that describes output styles
	if !containsSubstr(usage, "compact") {
		t.Error("Usage() --output flag description should include 'compact'")
	}
}

func TestCLI_OutputFormat_CompactIsValid(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected 'compact' to be a valid output format, got error: %v", err)
	}
}

func TestValidateOutputFormat_CompactValid(t *testing.T) {
	// Test that "compact" is recognized as a valid format
	err := ValidateOutputFormat("compact")
	if err != nil {
		t.Errorf("expected 'compact' to be valid, got error: %v", err)
	}

	// Case insensitive
	err = ValidateOutputFormat("COMPACT")
	if err != nil {
		t.Errorf("expected 'COMPACT' to be valid (case-insensitive), got error: %v", err)
	}
}

func TestValidateOutputFormat_InvalidListsCompact(t *testing.T) {
	err := ValidateOutputFormat("unknown")
	if err == nil {
		t.Error("expected error for invalid format")
	}
	// Error should list "compact" among valid options
	if !containsSubstr(err.Error(), "compact") {
		t.Error("error message should list 'compact' among valid formats")
	}
}

func TestCLI_DefaultFormatIsDetailed(t *testing.T) {
	cfg := NewCLIConfig()
	if cfg.Output != "detailed" {
		t.Errorf("default output format should be 'detailed', got %q", cfg.Output)
	}
}

func TestRun_RemoteFromFile(t *testing.T) {
	fromYAML := "key: remote_value\n"
	toYAML := "key: local_value\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = server.URL + "/from.yaml"
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.ToContent = []byte(toYAML)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d; stderr: %s", ExitCodeSuccess, result.Code, stderr.String())
	}
	// Output should contain the diff showing value change
	output := stdout.String()
	if !strings.Contains(output, "remote_value") || !strings.Contains(output, "local_value") {
		t.Errorf("expected output to contain diff values, got: %s", output)
	}
}

func TestRun_BothRemote(t *testing.T) {
	fromYAML := "name: alice\nage: 30\n"
	toYAML := "name: alice\nage: 31\n"

	fromServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer fromServer.Close()

	toServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, toYAML)
	}))
	defer toServer.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = fromServer.URL + "/from.yaml"
	cfg.ToFile = toServer.URL + "/to.yaml"
	cfg.Output = "compact"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d; stderr: %s", ExitCodeSuccess, result.Code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "age") {
		t.Errorf("expected output to contain 'age' diff, got: %s", output)
	}
}

func TestRun_RemoteWithSwap(t *testing.T) {
	fromYAML := "key: from_value\n"
	toYAML := "key: to_value\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = server.URL + "/from.yaml"
	cfg.Swap = true
	cfg.Output = "compact"
	cfg.SetExitCode = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.ToContent = []byte(toYAML)

	result := Run(cfg, rc)
	// With swap and differences, and -s flag, should return ExitCodeDifferences
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --swap and -s, got %d; stderr: %s",
			ExitCodeDifferences, result.Code, stderr.String())
	}
	// The swap means from and to are reversed, so from_value should appear as the "to" in the diff
	output := stdout.String()
	if !strings.Contains(output, "from_value") || !strings.Contains(output, "to_value") {
		t.Errorf("expected output to contain swapped diff values, got: %s", output)
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

func TestRun_RemoteWithFilters(t *testing.T) {
	fromYAML := "app:\n  name: myapp\n  version: \"1.0\"\ndb:\n  host: localhost\n"
	toYAML := "app:\n  name: myapp\n  version: \"2.0\"\ndb:\n  host: remotehost\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fromYAML)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.FromFile = server.URL + "/config.yaml"
	cfg.Output = "compact"
	cfg.Filter = []string{"app"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.ToContent = []byte(toYAML)

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d; stderr: %s", ExitCodeSuccess, result.Code, stderr.String())
	}
	output := stdout.String()
	// Should include app.version change but NOT db.host change (filtered out)
	if !strings.Contains(output, "version") {
		t.Errorf("expected output to contain 'version' diff, got: %s", output)
	}
	if strings.Contains(output, "db.host") {
		t.Errorf("expected output to NOT contain 'db.host' (filtered), got: %s", output)
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
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}
	location := result[0]["location"].(map[string]interface{})
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

// --- Task 2.1: Summary CLI flags and API key validation ---

func TestCLIConfig_ParseArgs_SummaryFlag(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--summary", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Summary {
		t.Error("expected Summary=true with --summary flag")
	}
}

func TestCLIConfig_ParseArgs_SummaryShortFlag(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"-S", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Summary {
		t.Error("expected Summary=true with -S flag")
	}
}

func TestCLIConfig_ParseArgs_SummaryModelFlag(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--summary-model", "claude-sonnet-4-20250514", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.SummaryModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected SummaryModel='claude-sonnet-4-20250514', got %q", cfg.SummaryModel)
	}
}

func TestCLIConfig_ParseArgs_SummaryDefaultOff(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Summary {
		t.Error("expected Summary=false by default")
	}
	if cfg.SummaryModel != "" {
		t.Errorf("expected SummaryModel='' by default, got %q", cfg.SummaryModel)
	}
}

func TestCLIConfig_Validate_SummaryWithoutAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.Summary = true

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error when --summary is set but ANTHROPIC_API_KEY is missing")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("error should mention ANTHROPIC_API_KEY, got: %v", err)
	}
}

func TestCLIConfig_Validate_SummaryWithAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-123")

	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.Summary = true

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error when --summary with API key set, got: %v", err)
	}
}

func TestCLIConfig_Validate_NoSummaryNoAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.Summary = false

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error when --summary is not set, got: %v", err)
	}
}

func TestCLI_Usage_ContainsSummaryFlags(t *testing.T) {
	cfg := NewCLIConfig()
	usage := cfg.Usage()

	if !strings.Contains(usage, "--summary") {
		t.Error("Usage() should contain --summary flag")
	}
	if !strings.Contains(usage, "-S") {
		t.Error("Usage() should contain -S short flag")
	}
	if !strings.Contains(usage, "--summary-model") {
		t.Error("Usage() should contain --summary-model flag")
	}
}

// --- Task 3.1: Wire summarizer into single-file comparison mode ---

func TestRun_WithSummary_AppendsSummaryToOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	// Start a mock Anthropic API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("expected x-api-key header")
		}

		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"The key value was changed from value1 to value2."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should contain standard diff output
	if !strings.Contains(output, "key") {
		t.Error("expected standard diff output containing 'key'")
	}
	// Should contain AI summary header and text
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected 'AI Summary:' header in output, got: %s", output)
	}
	if !strings.Contains(output, "value was changed") {
		t.Errorf("expected summary text in output, got: %s", output)
	}
}

func TestRun_WithSummary_NoDiffs_NoAPICall(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml := "key: value\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if apiCalled {
		t.Error("API should not be called when there are no differences")
	}
}

func TestRun_WithSummary_APIFailure_WarningOnStderr(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"internal error"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)

	// Exit code should not be affected by summary failure
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d", ExitCodeSuccess, result.Code)
	}
	// Standard diff output should still be present
	if !strings.Contains(stdout.String(), "key") {
		t.Error("expected standard diff output despite API failure")
	}
	// Warning on stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
	// No AI Summary in stdout
	if strings.Contains(stdout.String(), "AI Summary:") {
		t.Error("expected no AI Summary header on API failure")
	}
}

func TestRun_WithSummary_PreservesExitCode(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = true
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	// Exit code 1 should be preserved even with summary
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --set-exit-code, got %d", ExitCodeDifferences, result.Code)
	}
}

func TestRun_WithSummary_APIFailure_PreservesExitCode(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"fail"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = true
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	// Exit code should still be 1 (differences) even on API failure
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --set-exit-code and API failure, got %d",
			ExitCodeDifferences, result.Code)
	}
}

func TestRun_WithoutSummary_NoAPICall(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = false
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if apiCalled {
		t.Error("API should not be called when --summary is not set")
	}
}

// --- Task 3.3: Brief format special case ---

func TestRun_BriefSummary_ReplacesOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"The key was updated."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "brief"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should contain AI summary
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header, got: %s", output)
	}
	if !strings.Contains(output, "The key was updated.") {
		t.Errorf("expected summary text, got: %s", output)
	}
	// Should NOT contain brief format markers (± or "modified")
	if strings.Contains(output, "±") {
		t.Errorf("expected brief output to be suppressed, but found '±' in: %s", output)
	}
}

func TestRun_BriefSummary_FallbackOnAPIFailure(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"fail"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "brief"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should fall back to brief output
	if !strings.Contains(output, "±") && !strings.Contains(output, "modified") {
		t.Errorf("expected brief fallback output on API failure, got: %s", output)
	}
	// Warning on stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
	// No AI Summary header
	if strings.Contains(output, "AI Summary:") {
		t.Error("expected no AI Summary header on API failure")
	}
}

func TestRun_BriefSummary_NoDiffs_ShowsStandardOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml := "key: value\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "brief"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if apiCalled {
		t.Error("API should not be called when there are no differences")
	}
}

// --- Task 4.1: End-to-end CLI flag and validation tests ---

func TestRun_SummaryValidation_NoAPIKey_ExitCode255(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.Summary = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code %d (255) when --summary without API key, got %d", ExitCodeError, result.Code)
	}
	if !strings.Contains(stderr.String(), "ANTHROPIC_API_KEY") {
		t.Errorf("expected error mentioning ANTHROPIC_API_KEY, got: %s", stderr.String())
	}
}

func TestRun_SummaryValidation_NoAPIKey_ParseAndRun(t *testing.T) {
	// End-to-end: parse args then run
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := NewCLIConfig()
	args := []string{"--summary", "from.yaml", "to.yaml"}
	if err := cfg.ParseArgs(args); err != nil {
		t.Fatalf("failed to parse args: %v", err)
	}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code 255, got %d", result.Code)
	}
}

func TestRun_SummaryModelFlag_ParseAndRun(t *testing.T) {
	// End-to-end: parse --summary-model flag then verify it's used in API call
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	var receivedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if model, ok := req["model"].(string); ok {
			receivedModel = model
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	args := []string{"--summary", "--summary-model", "claude-sonnet-4-20250514", "from.yaml", "to.yaml"}
	if err := cfg.ParseArgs(args); err != nil {
		t.Fatalf("failed to parse args: %v", err)
	}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte("key: value1\n")
	rc.ToContent = []byte("key: value2\n")
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if receivedModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-20250514' in API request, got %q", receivedModel)
	}
}

func TestRun_WithSummary_AllFormats(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary text."}]}`)
	}))
	defer server.Close()

	formats := []string{"compact", "github", "gitlab", "gitea", "detailed"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := NewCLIConfig()
			cfg.Output = format
			cfg.Summary = true
			cfg.Color = "never"

			rc := NewRunConfig()
			var stdout, stderr strings.Builder
			rc.Stdout = &stdout
			rc.Stderr = &stderr
			rc.FromContent = []byte(yaml1)
			rc.ToContent = []byte(yaml2)
			rc.SummaryAPIURL = server.URL

			result := Run(cfg, rc)
			if result.Err != nil {
				t.Fatalf("unexpected error for format %s: %v", format, result.Err)
			}

			output := stdout.String()
			if !strings.Contains(output, "AI Summary:") {
				t.Errorf("expected AI Summary header for format %s, got: %s", format, output)
			}
		})
	}
}

// --- Task 4.2: End-to-end integration tests for summary flows ---

func TestRun_WithSummary_ColorEnabled(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Colored summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "always"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should contain colored AI Summary header
	if !strings.Contains(output, colorCyan) {
		t.Errorf("expected cyan color in AI Summary header with color=always, got: %s", output)
	}
	if !strings.Contains(output, styleBold) {
		t.Errorf("expected bold style in AI Summary header with color=always, got: %s", output)
	}
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header, got: %s", output)
	}
}

func TestRun_WithSummary_WithFilter_OnlyFilteredDiffsSent(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "config:\n  a: 1\n  b: 2\n"
	yaml2 := "config:\n  a: 10\n  b: 20\n"

	var receivedPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 0 {
			receivedPrompt = req.Messages[0].Content
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Filtered summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"
	cfg.Filter = []string{"config.a"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	// Should contain config.a in prompt
	if !strings.Contains(receivedPrompt, "config.a") {
		t.Errorf("expected config.a in API prompt, got: %s", receivedPrompt)
	}
	// Should NOT contain config.b in prompt (filtered out)
	if strings.Contains(receivedPrompt, "config.b") {
		t.Errorf("expected config.b NOT in API prompt (filtered), got: %s", receivedPrompt)
	}
	// Output should contain the summary
	if !strings.Contains(stdout.String(), "AI Summary:") {
		t.Errorf("expected AI Summary in output, got: %s", stdout.String())
	}
}

func TestRun_WithSummary_BriefNoDiffs_StandardOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml := "key: value\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "brief"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code 0, got %d", result.Code)
	}
	if apiCalled {
		t.Error("API should not be called when there are no differences (brief+summary)")
	}
	// Standard brief output should be shown (no diffs, so formatter handles it)
	if strings.Contains(stdout.String(), "AI Summary:") {
		t.Error("should not show AI Summary when there are no diffs")
	}
}

func TestCLIConfig_ParseArgs_IgnoreApiVersion(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"--ignore-api-version", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IgnoreApiVersion {
		t.Error("expected IgnoreApiVersion=true after parsing --ignore-api-version")
	}
}

func TestCLIConfig_ToCompareOptions_IgnoreApiVersion(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.IgnoreApiVersion = true

	opts := cfg.ToCompareOptions()
	if !opts.IgnoreApiVersion {
		t.Error("expected Options.IgnoreApiVersion=true when CLIConfig.IgnoreApiVersion=true")
	}
}

func TestCLIConfig_ToCompareOptions_IgnoreApiVersion_Default(t *testing.T) {
	cfg := NewCLIConfig()

	opts := cfg.ToCompareOptions()
	if opts.IgnoreApiVersion {
		t.Error("expected Options.IgnoreApiVersion=false by default")
	}
}

func TestCLIConfig_Usage_IncludesIgnoreApiVersion(t *testing.T) {
	cfg := NewCLIConfig()
	usage := cfg.Usage()

	if !strings.Contains(usage, "--ignore-api-version") {
		t.Error("expected usage output to contain --ignore-api-version flag")
	}
}
