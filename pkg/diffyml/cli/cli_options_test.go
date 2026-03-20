package cli

import (
	"testing"
)

func TestCLIConfig_ToCompareOptions(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.IgnoreOrderChanges = true
	cfg.IgnoreWhitespaceChanges = true
	cfg.FormatStrings = true
	cfg.Swap = true
	cfg.Chroot = "data"

	opts := cfg.ToCompareOptions()

	if !opts.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true in Options")
	}
	if !opts.IgnoreWhitespaceChanges {
		t.Error("expected IgnoreWhitespaceChanges=true in Options")
	}
	if !opts.FormatStrings {
		t.Error("expected FormatStrings=true in Options")
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
	cfg.UseGoPatchStyle = true
	cfg.MultiLineContextLines = 10

	opts := cfg.ToFormatOptions()

	if !opts.OmitHeader {
		t.Error("expected OmitHeader=true")
	}
	if !opts.UseGoPatchStyle {
		t.Error("expected UseGoPatchStyle=true")
	}
	if opts.ContextLines != 10 {
		t.Errorf("expected ContextLines=10, got %d", opts.ContextLines)
	}
}

func TestCLIConfig_ToFormatOptions_NoCertInspection(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.NoCertInspection = true

	opts := cfg.ToFormatOptions()

	if !opts.NoCertInspection {
		t.Error("expected NoCertInspection=true in FormatOptions when set in CLI config")
	}
}

func TestCLIConfig_ToFormatOptions_NoCertInspection_Default(t *testing.T) {
	cfg := NewCLIConfig()

	opts := cfg.ToFormatOptions()

	if opts.NoCertInspection {
		t.Error("expected NoCertInspection=false by default in FormatOptions")
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
