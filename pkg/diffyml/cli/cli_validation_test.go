package cli

import (
	"strings"
	"testing"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

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
	err := diffyml.ValidateFileExists("/nonexistent/path/file.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	if !containsSubstr(err.Error(), "/nonexistent/path/file.yaml") {
		t.Error("error should include the file path")
	}
}

func TestValidateFileExists_Directory(t *testing.T) {
	// "." is a directory, not a file
	err := diffyml.ValidateFileExists(".")
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
