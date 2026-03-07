package cli

import (
	"strings"
	"testing"
)

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

func TestCLI_DefaultFormatIsDetailed(t *testing.T) {
	cfg := NewCLIConfig()
	if cfg.Output != "detailed" {
		t.Errorf("default output format should be 'detailed', got %q", cfg.Output)
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

func TestCLIConfig_Usage_IncludesIgnoreApiVersion(t *testing.T) {
	cfg := NewCLIConfig()
	usage := cfg.Usage()

	if !strings.Contains(usage, "--ignore-api-version") {
		t.Error("expected usage output to contain --ignore-api-version flag")
	}
}
