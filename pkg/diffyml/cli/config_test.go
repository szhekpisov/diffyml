package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// --- FileConfig parsing tests ---

func TestLoadConfigFile_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := `
output: compact
color: always
truecolor: never
ignore-order-changes: true
ignore-whitespace-changes: true
format-strings: true
ignore-value-changes: true
detect-kubernetes: false
detect-renames: false
ignore-api-version: true
no-cert-inspection: true
swap: true
filter:
  - "metadata"
  - "spec"
exclude:
  - "status"
filter-regexp:
  - "^test"
exclude-regexp:
  - "password"
additional-identifier:
  - "id"
omit-header: true
use-go-patch-style: true
multi-line-context-lines: 8
chroot: "data"
chroot-of-from: "from-root"
chroot-of-to: "to-root"
chroot-list-to-documents: true
summary: true
summary-model: "claude-sonnet-4-20250514"
set-exit-code: true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	fc, err := loadConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc == nil {
		t.Fatal("expected non-nil FileConfig")
	}

	// Output options
	if fc.Output == nil || *fc.Output != "compact" {
		t.Errorf("expected Output='compact', got %v", fc.Output)
	}
	if fc.Color == nil || *fc.Color != "always" {
		t.Errorf("expected Color='always', got %v", fc.Color)
	}
	if fc.TrueColor == nil || *fc.TrueColor != "never" {
		t.Errorf("expected TrueColor='never', got %v", fc.TrueColor)
	}

	// Comparison options
	if fc.IgnoreOrderChanges == nil || !*fc.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true")
	}
	if fc.IgnoreWhitespaceChanges == nil || !*fc.IgnoreWhitespaceChanges {
		t.Error("expected IgnoreWhitespaceChanges=true")
	}
	if fc.FormatStrings == nil || !*fc.FormatStrings {
		t.Error("expected FormatStrings=true")
	}
	if fc.IgnoreValueChanges == nil || !*fc.IgnoreValueChanges {
		t.Error("expected IgnoreValueChanges=true")
	}
	if fc.DetectKubernetes == nil || *fc.DetectKubernetes {
		t.Error("expected DetectKubernetes=false")
	}
	if fc.DetectRenames == nil || *fc.DetectRenames {
		t.Error("expected DetectRenames=false")
	}
	if fc.IgnoreApiVersion == nil || !*fc.IgnoreApiVersion {
		t.Error("expected IgnoreApiVersion=true")
	}
	if fc.NoCertInspection == nil || !*fc.NoCertInspection {
		t.Error("expected NoCertInspection=true")
	}
	if fc.Swap == nil || !*fc.Swap {
		t.Error("expected Swap=true")
	}

	// Filtering
	if len(fc.Filter) != 2 || fc.Filter[0] != "metadata" || fc.Filter[1] != "spec" {
		t.Errorf("expected Filter=['metadata','spec'], got %v", fc.Filter)
	}
	if len(fc.Exclude) != 1 || fc.Exclude[0] != "status" {
		t.Errorf("expected Exclude=['status'], got %v", fc.Exclude)
	}
	if len(fc.FilterRegexp) != 1 || fc.FilterRegexp[0] != "^test" {
		t.Errorf("expected FilterRegexp=['^test'], got %v", fc.FilterRegexp)
	}
	if len(fc.ExcludeRegexp) != 1 || fc.ExcludeRegexp[0] != "password" {
		t.Errorf("expected ExcludeRegexp=['password'], got %v", fc.ExcludeRegexp)
	}
	if len(fc.AdditionalIdentifiers) != 1 || fc.AdditionalIdentifiers[0] != "id" {
		t.Errorf("expected AdditionalIdentifiers=['id'], got %v", fc.AdditionalIdentifiers)
	}

	// Display options
	if fc.OmitHeader == nil || !*fc.OmitHeader {
		t.Error("expected OmitHeader=true")
	}
	if fc.UseGoPatchStyle == nil || !*fc.UseGoPatchStyle {
		t.Error("expected UseGoPatchStyle=true")
	}
	if fc.MultiLineContextLines == nil || *fc.MultiLineContextLines != 8 {
		t.Errorf("expected MultiLineContextLines=8, got %v", fc.MultiLineContextLines)
	}

	// Chroot options
	if fc.Chroot == nil || *fc.Chroot != "data" {
		t.Errorf("expected Chroot='data', got %v", fc.Chroot)
	}
	if fc.ChrootFrom == nil || *fc.ChrootFrom != "from-root" {
		t.Errorf("expected ChrootFrom='from-root', got %v", fc.ChrootFrom)
	}
	if fc.ChrootTo == nil || *fc.ChrootTo != "to-root" {
		t.Errorf("expected ChrootTo='to-root', got %v", fc.ChrootTo)
	}
	if fc.ChrootListToDocuments == nil || !*fc.ChrootListToDocuments {
		t.Error("expected ChrootListToDocuments=true")
	}

	// AI Summary options
	if fc.Summary == nil || !*fc.Summary {
		t.Error("expected Summary=true")
	}
	if fc.SummaryModel == nil || *fc.SummaryModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected SummaryModel='claude-sonnet-4-20250514', got %v", fc.SummaryModel)
	}

	// Exit code
	if fc.SetExitCode == nil || !*fc.SetExitCode {
		t.Error("expected SetExitCode=true")
	}
}

func TestLoadConfigFile_PartialFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := `
output: brief
ignore-order-changes: true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	fc, err := loadConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc == nil {
		t.Fatal("expected non-nil FileConfig")
	}

	if fc.Output == nil || *fc.Output != "brief" {
		t.Errorf("expected Output='brief', got %v", fc.Output)
	}
	if fc.IgnoreOrderChanges == nil || !*fc.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true")
	}

	// Unset fields should be nil
	if fc.Color != nil {
		t.Errorf("expected Color=nil, got %v", fc.Color)
	}
	if fc.DetectKubernetes != nil {
		t.Errorf("expected DetectKubernetes=nil, got %v", fc.DetectKubernetes)
	}
	if fc.Filter != nil {
		t.Errorf("expected Filter=nil, got %v", fc.Filter)
	}
}

func TestLoadConfigFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	fc, err := loadConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc != nil {
		t.Errorf("expected nil FileConfig for empty file, got %+v", fc)
	}
}

func TestLoadConfigFile_CommentsOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := `# this is a comment
# another comment
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	fc, err := loadConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc != nil {
		t.Errorf("expected nil FileConfig for comments-only file, got %+v", fc)
	}
}

func TestLoadConfigFile_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := `
output: compact
unknown-key: value
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfigFile(path)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestLoadConfigFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := `
output: [invalid yaml
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfigFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadConfigFile_NonExistentPath(t *testing.T) {
	_, err := loadConfigFile("/nonexistent/path/config.yml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

// --- findConfigFile tests ---

func TestFindConfigFile_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yml")
	if err := os.WriteFile(path, []byte("output: compact\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := findConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != path {
		t.Errorf("expected path %q, got %q", path, result)
	}
}

func TestFindConfigFile_ExplicitPathNotFound(t *testing.T) {
	_, err := findConfigFile("/nonexistent/custom.yml")
	if err == nil {
		t.Fatal("expected error for missing explicit config path")
	}
}

func TestFindConfigFile_DefaultInCwd(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".diffyml.yml")
	if err := os.WriteFile(configPath, []byte("output: compact\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	result, err := findConfigFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != ".diffyml.yml" {
		t.Errorf("expected '.diffyml.yml', got %q", result)
	}
}

func TestFindConfigFile_DefaultYamlExtension(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".diffyml.yaml")
	if err := os.WriteFile(configPath, []byte("output: compact\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	result, err := findConfigFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != ".diffyml.yaml" {
		t.Errorf("expected '.diffyml.yaml', got %q", result)
	}
}

func TestFindConfigFile_YmlTakesPrecedenceOverYaml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".diffyml.yml"), []byte("output: compact\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".diffyml.yaml"), []byte("output: brief\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	result, err := findConfigFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != ".diffyml.yml" {
		t.Errorf("expected '.diffyml.yml' to take precedence, got %q", result)
	}
}

func TestFindConfigFile_NoConfigFile(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	result, err := findConfigFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// --- applyFileConfig tests ---

func TestApplyFileConfig_NilConfig(t *testing.T) {
	cfg := NewCLIConfig()
	originalOutput := cfg.Output
	cfg.applyFileConfig(nil, nil)

	if cfg.Output != originalOutput {
		t.Errorf("expected no change, Output changed to %q", cfg.Output)
	}
}

func TestApplyFileConfig_StringFields(t *testing.T) {
	cfg := NewCLIConfig()
	output := "compact"
	color := "always"
	fc := &FileConfig{
		Output: &output,
		Color:  &color,
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if cfg.Output != "compact" {
		t.Errorf("expected Output='compact', got %q", cfg.Output)
	}
	if cfg.Color != "always" {
		t.Errorf("expected Color='always', got %q", cfg.Color)
	}
}

func TestApplyFileConfig_BoolFields(t *testing.T) {
	cfg := NewCLIConfig()
	ignoreOrder := true
	swap := true
	fc := &FileConfig{
		IgnoreOrderChanges: &ignoreOrder,
		Swap:               &swap,
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if !cfg.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true")
	}
	if !cfg.Swap {
		t.Error("expected Swap=true")
	}
}

func TestApplyFileConfig_BoolDefaultTrue_ConfigSetsFalse(t *testing.T) {
	cfg := NewCLIConfig()
	if !cfg.DetectKubernetes {
		t.Fatal("precondition: DetectKubernetes should default to true")
	}

	detectK8s := false
	detectRenames := false
	fc := &FileConfig{
		DetectKubernetes: &detectK8s,
		DetectRenames:    &detectRenames,
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if cfg.DetectKubernetes {
		t.Error("expected DetectKubernetes=false from config")
	}
	if cfg.DetectRenames {
		t.Error("expected DetectRenames=false from config")
	}
}

func TestApplyFileConfig_CLIOverridesConfig(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "github" // CLI set this

	output := "compact" // config wants this
	fc := &FileConfig{
		Output: &output,
	}
	// "output" was explicitly set on CLI
	cfg.applyFileConfig(fc, map[string]bool{"output": true})

	if cfg.Output != "github" {
		t.Errorf("expected CLI override Output='github', got %q", cfg.Output)
	}
}

func TestApplyFileConfig_CLIShortFlagOverridesConfig(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "github"

	output := "compact"
	fc := &FileConfig{
		Output: &output,
	}
	// Short flag "o" was explicitly set on CLI
	cfg.applyFileConfig(fc, map[string]bool{"o": true})

	if cfg.Output != "github" {
		t.Errorf("expected CLI short flag override Output='github', got %q", cfg.Output)
	}
}

func TestApplyFileConfig_CLIOverridesBoolDefaultTrue(t *testing.T) {
	cfg := NewCLIConfig()
	// CLI explicitly set --detect-kubernetes=false
	cfg.DetectKubernetes = false

	detectK8s := true // config wants true
	fc := &FileConfig{
		DetectKubernetes: &detectK8s,
	}
	cfg.applyFileConfig(fc, map[string]bool{"detect-kubernetes": true})

	if cfg.DetectKubernetes {
		t.Error("expected CLI override DetectKubernetes=false")
	}
}

func TestApplyFileConfig_SliceFields_ConfigOnly(t *testing.T) {
	cfg := NewCLIConfig()
	fc := &FileConfig{
		Filter:  []string{"metadata", "spec"},
		Exclude: []string{"status"},
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if len(cfg.Filter) != 2 || cfg.Filter[0] != "metadata" || cfg.Filter[1] != "spec" {
		t.Errorf("expected Filter=['metadata','spec'], got %v", cfg.Filter)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "status" {
		t.Errorf("expected Exclude=['status'], got %v", cfg.Exclude)
	}
}

func TestApplyFileConfig_SliceFields_CLIOverrides(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Filter = []string{"cli-filter"} // CLI set this

	fc := &FileConfig{
		Filter: []string{"config-filter1", "config-filter2"},
	}
	cfg.applyFileConfig(fc, map[string]bool{"filter": true})

	if len(cfg.Filter) != 1 || cfg.Filter[0] != "cli-filter" {
		t.Errorf("expected CLI override Filter=['cli-filter'], got %v", cfg.Filter)
	}
}

func TestApplyFileConfig_IntField(t *testing.T) {
	cfg := NewCLIConfig()
	lines := 10
	fc := &FileConfig{
		MultiLineContextLines: &lines,
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if cfg.MultiLineContextLines != 10 {
		t.Errorf("expected MultiLineContextLines=10, got %d", cfg.MultiLineContextLines)
	}
}

func TestApplyFileConfig_IntField_CLIOverrides(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.MultiLineContextLines = 2

	lines := 10
	fc := &FileConfig{
		MultiLineContextLines: &lines,
	}
	cfg.applyFileConfig(fc, map[string]bool{"multi-line-context-lines": true})

	if cfg.MultiLineContextLines != 2 {
		t.Errorf("expected CLI override MultiLineContextLines=2, got %d", cfg.MultiLineContextLines)
	}
}

func TestApplyFileConfig_ChrootFields(t *testing.T) {
	cfg := NewCLIConfig()
	chroot := "data"
	chrootFrom := "from-root"
	chrootTo := "to-root"
	chrootList := true
	fc := &FileConfig{
		Chroot:                &chroot,
		ChrootFrom:            &chrootFrom,
		ChrootTo:              &chrootTo,
		ChrootListToDocuments: &chrootList,
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if cfg.Chroot != "data" {
		t.Errorf("expected Chroot='data', got %q", cfg.Chroot)
	}
	if cfg.ChrootFrom != "from-root" {
		t.Errorf("expected ChrootFrom='from-root', got %q", cfg.ChrootFrom)
	}
	if cfg.ChrootTo != "to-root" {
		t.Errorf("expected ChrootTo='to-root', got %q", cfg.ChrootTo)
	}
	if !cfg.ChrootListToDocuments {
		t.Error("expected ChrootListToDocuments=true")
	}
}

func TestApplyFileConfig_SummaryFields(t *testing.T) {
	cfg := NewCLIConfig()
	summary := true
	model := "claude-sonnet-4-20250514"
	fc := &FileConfig{
		Summary:      &summary,
		SummaryModel: &model,
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if !cfg.Summary {
		t.Error("expected Summary=true")
	}
	if cfg.SummaryModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected SummaryModel='claude-sonnet-4-20250514', got %q", cfg.SummaryModel)
	}
}

func TestApplyFileConfig_AllFilterTypes(t *testing.T) {
	cfg := NewCLIConfig()
	fc := &FileConfig{
		Filter:                []string{"f1"},
		Exclude:               []string{"e1"},
		FilterRegexp:          []string{"^fr"},
		ExcludeRegexp:         []string{"^er"},
		AdditionalIdentifiers: []string{"id"},
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if len(cfg.Filter) != 1 || cfg.Filter[0] != "f1" {
		t.Errorf("expected Filter=['f1'], got %v", cfg.Filter)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "e1" {
		t.Errorf("expected Exclude=['e1'], got %v", cfg.Exclude)
	}
	if len(cfg.FilterRegexp) != 1 || cfg.FilterRegexp[0] != "^fr" {
		t.Errorf("expected FilterRegexp=['^fr'], got %v", cfg.FilterRegexp)
	}
	if len(cfg.ExcludeRegexp) != 1 || cfg.ExcludeRegexp[0] != "^er" {
		t.Errorf("expected ExcludeRegexp=['^er'], got %v", cfg.ExcludeRegexp)
	}
	if len(cfg.AdditionalIdentifiers) != 1 || cfg.AdditionalIdentifiers[0] != "id" {
		t.Errorf("expected AdditionalIdentifiers=['id'], got %v", cfg.AdditionalIdentifiers)
	}
}

func TestApplyFileConfig_EmptySlicesDoNotOverride(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Filter = []string{"existing-filter"}
	cfg.Exclude = []string{"existing-exclude"}
	cfg.FilterRegexp = []string{"existing-regexp"}
	cfg.ExcludeRegexp = []string{"existing-exclude-regexp"}
	cfg.AdditionalIdentifiers = []string{"existing-id"}

	// Config has explicit empty slices — should NOT wipe out existing values
	fc := &FileConfig{
		Filter:                []string{},
		Exclude:               []string{},
		FilterRegexp:          []string{},
		ExcludeRegexp:         []string{},
		AdditionalIdentifiers: []string{},
	}
	cfg.applyFileConfig(fc, map[string]bool{})

	if len(cfg.Filter) != 1 || cfg.Filter[0] != "existing-filter" {
		t.Errorf("expected Filter=['existing-filter'], got %v", cfg.Filter)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "existing-exclude" {
		t.Errorf("expected Exclude=['existing-exclude'], got %v", cfg.Exclude)
	}
	if len(cfg.FilterRegexp) != 1 || cfg.FilterRegexp[0] != "existing-regexp" {
		t.Errorf("expected FilterRegexp=['existing-regexp'], got %v", cfg.FilterRegexp)
	}
	if len(cfg.ExcludeRegexp) != 1 || cfg.ExcludeRegexp[0] != "existing-exclude-regexp" {
		t.Errorf("expected ExcludeRegexp=['existing-exclude-regexp'], got %v", cfg.ExcludeRegexp)
	}
	if len(cfg.AdditionalIdentifiers) != 1 || cfg.AdditionalIdentifiers[0] != "existing-id" {
		t.Errorf("expected AdditionalIdentifiers=['existing-id'], got %v", cfg.AdditionalIdentifiers)
	}
}

// --- Integration tests (through ParseArgs) ---

func TestParseArgs_WithConfigFile(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	if err := os.WriteFile(configPath, []byte("output: compact\nignore-order-changes: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "compact" {
		t.Errorf("expected Output='compact' from config, got %q", cfg.Output)
	}
	if !cfg.IgnoreOrderChanges {
		t.Error("expected IgnoreOrderChanges=true from config")
	}
}

func TestParseArgs_CLIOverridesConfigFile(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	if err := os.WriteFile(configPath, []byte("output: compact\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"-o", "brief", "from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "brief" {
		t.Errorf("expected CLI override Output='brief', got %q", cfg.Output)
	}
}

func TestParseArgs_ConfigFlag(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, "custom-config.yml")
	if err := os.WriteFile(configPath, []byte("output: github\nswap: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"--config", configPath, "from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "github" {
		t.Errorf("expected Output='github' from config, got %q", cfg.Output)
	}
	if !cfg.Swap {
		t.Error("expected Swap=true from config")
	}
}

func TestParseArgs_ConfigFlagEqualsForm(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, "custom-config.yml")
	if err := os.WriteFile(configPath, []byte("output: github\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"--config=" + configPath, "from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "github" {
		t.Errorf("expected Output='github' from config, got %q", cfg.Output)
	}
}

func TestParseArgs_ConfigFlagNotFound(t *testing.T) {
	cfg := NewCLIConfig()
	err := cfg.ParseArgs([]string{"--config", "/nonexistent/config.yml", "from.yaml", "to.yaml"})
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestParseArgs_ConfigFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	if err := os.WriteFile(configPath, []byte("output: [invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	err = cfg.ParseArgs([]string{"from.yaml", "to.yaml"})
	if err == nil {
		t.Fatal("expected error for invalid YAML config")
	}
}

func TestParseArgs_ConfigFileUnknownKey(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	if err := os.WriteFile(configPath, []byte("unknown-key: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	err = cfg.ParseArgs([]string{"from.yaml", "to.yaml"})
	if err == nil {
		t.Fatal("expected error for unknown key in config")
	}
}

func TestParseArgs_ConfigFileDetectKubernetesFalse(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	if err := os.WriteFile(configPath, []byte("detect-kubernetes: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DetectKubernetes {
		t.Error("expected DetectKubernetes=false from config")
	}
}

func TestParseArgs_ConfigFileWithSliceFields(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	content := `
filter:
  - "metadata"
  - "spec"
exclude:
  - "status"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Filter) != 2 || cfg.Filter[0] != "metadata" || cfg.Filter[1] != "spec" {
		t.Errorf("expected Filter=['metadata','spec'], got %v", cfg.Filter)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "status" {
		t.Errorf("expected Exclude=['status'], got %v", cfg.Exclude)
	}
}

func TestParseArgs_CLIOverridesConfigSlice(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	content := `
filter:
  - "config-filter1"
  - "config-filter2"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"--filter", "cli-filter", "from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Filter) != 1 || cfg.Filter[0] != "cli-filter" {
		t.Errorf("expected CLI override Filter=['cli-filter'], got %v", cfg.Filter)
	}
}

func TestParseArgs_NoConfigFileNoError(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Defaults should be unchanged
	if cfg.Output != "detailed" {
		t.Errorf("expected default Output='detailed', got %q", cfg.Output)
	}
}

func TestParseArgs_ConfigFileEmptyNoError(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".diffyml.yml")
	if err := os.WriteFile(configPath, []byte("# empty config\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := NewCLIConfig()
	if err := cfg.ParseArgs([]string{"from.yaml", "to.yaml"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Output != "detailed" {
		t.Errorf("expected default Output='detailed', got %q", cfg.Output)
	}
}
