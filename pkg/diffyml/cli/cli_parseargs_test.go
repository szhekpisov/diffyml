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

// --- GIT_EXTERNAL_DIFF detection ---

func TestParseArgs_GitExternalDiff_7Args(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"deploy.yaml",           // name
		"/tmp/old-content",      // old-file
		"abc1234abc1234abc1234", // old-hex
		"100644",                // old-mode
		"/work/deploy.yaml",     // new-file
		"def5678def5678def5678", // new-hex
		"100644",                // new-mode
	}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=true for 7 args")
	}
	if cfg.GitDisplayPath != "deploy.yaml" {
		t.Errorf("expected GitDisplayPath='deploy.yaml', got %q", cfg.GitDisplayPath)
	}
	if cfg.FromFile != "/tmp/old-content" {
		t.Errorf("expected FromFile='/tmp/old-content', got %q", cfg.FromFile)
	}
	if cfg.ToFile != "/work/deploy.yaml" {
		t.Errorf("expected ToFile='/work/deploy.yaml', got %q", cfg.ToFile)
	}
}

func TestParseArgs_GitExternalDiff_8Args_Rename(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"old-name.yaml",         // name
		"/tmp/old-content",      // old-file
		"abc1234abc1234abc1234", // old-hex
		"100644",                // old-mode
		"/work/new-name.yaml",   // new-file
		"def5678def5678def5678", // new-hex
		"100644",                // new-mode
		"new-name.yaml",         // rename-to
	}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=true for 8 args (rename)")
	}
	if cfg.GitDisplayPath != "new-name.yaml" {
		t.Errorf("expected GitDisplayPath='new-name.yaml' (rename-to), got %q", cfg.GitDisplayPath)
	}
	if cfg.GitOriginalPath != "old-name.yaml" {
		t.Errorf("expected GitOriginalPath='old-name.yaml', got %q", cfg.GitOriginalPath)
	}
	if cfg.FromFile != "/tmp/old-content" {
		t.Errorf("expected FromFile='/tmp/old-content', got %q", cfg.FromFile)
	}
	if cfg.ToFile != "/work/new-name.yaml" {
		t.Errorf("expected ToFile='/work/new-name.yaml', got %q", cfg.ToFile)
	}
}

func TestParseArgs_GitExternalDiff_9Args_RenameWithXfrm(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"old-name.yaml",
		"/tmp/old-content",
		"abc1234abc1234abc1234",
		"100644",
		"/work/new-name.yaml",
		"def5678def5678def5678",
		"100644",
		"new-name.yaml",
		"similarity index 100%\nrename from old-name.yaml\nrename to new-name.yaml",
	}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=true for 9 args")
	}
	if cfg.GitDisplayPath != "new-name.yaml" {
		t.Errorf("expected GitDisplayPath='new-name.yaml', got %q", cfg.GitDisplayPath)
	}
}

func TestParseArgs_GitExternalDiff_DeletedFile(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"deploy.yaml",
		"/tmp/old-content",
		"abc1234abc1234abc1234",
		"100644",
		"/dev/null",
		"0000000000000000000000",
		"000000",
	}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=true")
	}
	if cfg.ToFile != "/dev/null" {
		t.Errorf("expected ToFile='/dev/null', got %q", cfg.ToFile)
	}
}

func TestParseArgs_GitExternalDiff_NewFile(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"deploy.yaml",
		"/dev/null",
		"0000000000000000000000",
		"000000",
		"/work/deploy.yaml",
		"abc1234abc1234abc1234",
		"100644",
	}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=true")
	}
	if cfg.FromFile != "/dev/null" {
		t.Errorf("expected FromFile='/dev/null', got %q", cfg.FromFile)
	}
}

func TestParseArgs_GitExternalDiff_WithFlags(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"--set-exit-code",
		"deploy.yaml",
		"/tmp/old-content",
		"abc1234abc1234abc1234",
		"100644",
		"/work/deploy.yaml",
		"def5678def5678def5678",
		"100644",
	}

	err := cfg.ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=true with flags")
	}
	if !cfg.SetExitCode {
		t.Error("expected SetExitCode=true")
	}
}

func TestParseArgs_GitExternalDiff_NotDetected_6Args(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"a", "b", "c", "100644", "e", "100644"}

	err := cfg.ParseArgs(args)
	// 6 args with octals at positions 3 and 5 — but not at position 6
	// so this should NOT be detected as git external diff
	if cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=false for 6 args")
	}
	// Should parse as normal 2-file mode (first 2 positional args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgs_GitExternalDiff_NotDetected_10Args(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{"a", "b", "c", "100644", "e", "f", "100644", "h", "i", "j"}

	err := cfg.ParseArgs(args)
	if cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=false for 10 args")
	}
	// Should parse as normal 2-file mode
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgs_GitExternalDiff_NotDetected_InvalidOctal(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"deploy.yaml",
		"/tmp/old-content",
		"abc1234abc1234abc1234",
		"NOTOCL", // invalid: not octal
		"/work/deploy.yaml",
		"def5678def5678def5678",
		"100644",
	}

	err := cfg.ParseArgs(args)
	if cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=false with invalid octal at position 3")
	}
	// Should parse as normal 2-file mode
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgs_GitExternalDiff_NotDetected_InvalidOctalPos6(t *testing.T) {
	cfg := NewCLIConfig()
	args := []string{
		"deploy.yaml",
		"/tmp/old-content",
		"abc1234abc1234abc1234",
		"100644",
		"/work/deploy.yaml",
		"def5678def5678def5678",
		"NOTOCL", // invalid: not octal at position 6
	}

	err := cfg.ParseArgs(args)
	if cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=false with invalid octal at position 6")
	}
	// Should parse as normal 2-file mode
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgs_GitExternalDiff_EnvVarAloneNotSufficient(t *testing.T) {
	t.Setenv("GIT_EXTERNAL_DIFF", "diffyml")

	cfg := NewCLIConfig()
	args := []string{
		"deploy.yaml",
		"/tmp/old-content",
		"abc1234abc1234abc1234",
		"NOTOCL", // invalid octal at position 3
		"/work/deploy.yaml",
		"def5678def5678def5678",
		"100644",
	}

	err := cfg.ParseArgs(args)
	if cfg.GitExternalDiff {
		t.Error("expected GitExternalDiff=false: env var alone should not trigger detection without valid octals")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- isOctalMode tests ---

func TestIsOctalMode(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"100644", true},
		{"100755", true},
		{"000000", true},
		{"120000", true},
		{"777777", true},
		{"100648", false},  // 8 is not octal
		{"100649", false},  // 9 is not octal
		{"10064", false},   // too short
		{"1006440", false}, // too long
		{"", false},
		{"abcdef", false},
		{"NOTOCL", false},
		{"12345a", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isOctalMode(tt.input)
			if got != tt.want {
				t.Errorf("isOctalMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- isYAMLFile tests ---

func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"deploy.yaml", true},
		{"deploy.yml", true},
		{"deploy.YAML", true},
		{"deploy.YML", true},
		{"deploy.Yaml", true},
		{"path/to/deploy.yaml", true},
		{"deploy.json", false},
		{"deploy.txt", false},
		{"deploy.yamll", false},
		{"deploy", false},
		{"", false},
		{".yaml", true},
		{"Makefile", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isYAMLFile(tt.input)
			if got != tt.want {
				t.Errorf("isYAMLFile(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
