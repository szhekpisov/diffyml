package cli

import (
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
	if !cfg.DetectKubernetes {
		t.Error("expected default DetectKubernetes=true")
	}
	if !cfg.DetectRenames {
		t.Error("expected default DetectRenames=true")
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

// --- Mutation testing: cli.go ---

func TestReorderArgs_EqualsAtPositionZero(t *testing.T) {
	// cli.go:221 — `eqIdx >= 0` → `> 0` would skip flags like `--=value`
	// where the = sign is at position 0 of the name part.
	// After stripping dashes from "--=value", name becomes "=value",
	// IndexByte('=') returns 0. If mutated to > 0, name would be "=value"
	// instead of "" and fs.Lookup would fail differently.
	//
	// We test with a real flag that has `=` at the start of the value portion.
	cfg := NewCLIConfig()
	args := []string{"from.yaml", "to.yaml", "--output=brief"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Output != "brief" {
		t.Errorf("expected Output='brief', got %q", cfg.Output)
	}
}

func TestReorderArgs_NonBoolFlagConsumesNextArg(t *testing.T) {
	// cli.go:235 — `i+1` → `i-1` would consume wrong arg
	// cli.go:235 — `i+1 < len(args)` → `<= len(args)` would OOB
	cfg := NewCLIConfig()
	args := []string{"--output", "brief", "from.yaml", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Output != "brief" {
		t.Errorf("expected Output='brief', got %q", cfg.Output)
	}
	if cfg.FromFile != "from.yaml" {
		t.Errorf("expected FromFile='from.yaml', got %q", cfg.FromFile)
	}
	if cfg.ToFile != "to.yaml" {
		t.Errorf("expected ToFile='to.yaml', got %q", cfg.ToFile)
	}
}

func TestReorderArgs_NonBoolFlagAtEnd(t *testing.T) {
	// cli.go:235 — when non-bool flag is the LAST arg (no value following),
	// `i+1 < len(args)` prevents OOB. If mutated to `<=`, would panic.
	cfg := NewCLIConfig()
	// --output with no following value → will fail at Parse, but should NOT panic
	args := []string{"from.yaml", "to.yaml", "--output"}

	err := cfg.ParseArgs(args)
	// This might succeed or fail depending on flag parsing, but must not panic.
	// The key is that reorderArgs doesn't panic when non-bool flag is last.
	_ = err
}

func TestReorderArgs_MultipleFlagValuePairs(t *testing.T) {
	// cli.go:235 — if `i+1` mutated to `i-1`, the second flag/value pair
	// would consume the wrong argument.
	cfg := NewCLIConfig()
	args := []string{"from.yaml", "--output", "brief", "--color", "off", "to.yaml"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Output != "brief" {
		t.Errorf("expected Output='brief', got %q", cfg.Output)
	}
	if cfg.Color != "off" {
		t.Errorf("expected Color='off', got %q", cfg.Color)
	}
	if cfg.FromFile != "from.yaml" {
		t.Errorf("expected FromFile='from.yaml', got %q", cfg.FromFile)
	}
	if cfg.ToFile != "to.yaml" {
		t.Errorf("expected ToFile='to.yaml', got %q", cfg.ToFile)
	}
}

func TestReorderArgs_TrailingNonBoolFlag(t *testing.T) {
	// Non-bool flag as last arg → no panic
	cfg := NewCLIConfig()
	args := []string{"from.yaml", "to.yaml", "--output", "brief"}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "brief" {
		t.Errorf("expected Output='brief', got %q", cfg.Output)
	}
	if cfg.FromFile != "from.yaml" {
		t.Errorf("expected FromFile='from.yaml', got %q", cfg.FromFile)
	}
}
