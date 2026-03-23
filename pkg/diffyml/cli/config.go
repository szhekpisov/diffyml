// config.go - YAML configuration file support.
//
// Loads .diffyml.yml from the current directory (or --config path).
// Config file values are applied as defaults; CLI flags override them.
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
	"gopkg.in/yaml.v3"
)

// FileConfig represents the YAML configuration file structure.
// All scalar fields are pointers to distinguish "not set" (nil) from zero values.
type FileConfig struct {
	// Output options
	Output    *string `yaml:"output"`
	Color     *string `yaml:"color"`
	TrueColor *string `yaml:"truecolor"`

	// Comparison options
	IgnoreOrderChanges      *bool `yaml:"ignore-order-changes"`
	IgnoreWhitespaceChanges *bool `yaml:"ignore-whitespace-changes"`
	FormatStrings           *bool `yaml:"format-strings"`
	IgnoreValueChanges      *bool `yaml:"ignore-value-changes"`
	DetectKubernetes        *bool `yaml:"detect-kubernetes"`
	DetectRenames           *bool `yaml:"detect-renames"`
	IgnoreApiVersion        *bool `yaml:"ignore-api-version"`
	NoCertInspection        *bool `yaml:"no-cert-inspection"`
	Swap                    *bool `yaml:"swap"`

	// Filtering options
	Filter                []string `yaml:"filter"`
	Exclude               []string `yaml:"exclude"`
	FilterRegexp          []string `yaml:"filter-regexp"`
	ExcludeRegexp         []string `yaml:"exclude-regexp"`
	AdditionalIdentifiers []string `yaml:"additional-identifier"`

	// Display options
	OmitHeader            *bool `yaml:"omit-header"`
	UseGoPatchStyle       *bool `yaml:"use-go-patch-style"`
	MultiLineContextLines *int  `yaml:"multi-line-context-lines"`

	// Chroot options
	Chroot                *string `yaml:"chroot"`
	ChrootFrom            *string `yaml:"chroot-of-from"`
	ChrootTo              *string `yaml:"chroot-of-to"`
	ChrootListToDocuments *bool   `yaml:"chroot-list-to-documents"`

	// AI Summary options
	Summary      *bool   `yaml:"summary"`
	SummaryModel *string `yaml:"summary-model"`

	// Exit code behavior
	SetExitCode *bool `yaml:"set-exit-code"`

	// Custom color options
	Colors *ColorOverrides `yaml:"colors"`
}

// ColorOverrides holds custom color specifications for config file.
type ColorOverrides struct {
	Added    *string `yaml:"added"`
	Removed  *string `yaml:"removed"`
	Modified *string `yaml:"modified"`
	Context  *string `yaml:"context"`
	DocName  *string `yaml:"doc-name"`
}

// findConfigFile returns the config path: --config flag if given (must exist),
// else .diffyml.yml or .diffyml.yaml in cwd (empty string if absent).
func findConfigFile(configFlag string) (string, error) {
	if configFlag != "" {
		if _, err := os.Stat(configFlag); err != nil {
			return "", fmt.Errorf("config file %s: %w", configFlag, err)
		}
		return configFlag, nil
	}

	for _, name := range []string{".diffyml.yml", ".diffyml.yaml"} {
		if _, err := os.Stat(name); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		return name, nil
	}
	return "", nil
}

// loadConfigFile reads and parses a YAML config file.
// Returns nil for empty files. Rejects unknown keys.
func loadConfigFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is from --config flag or hardcoded default
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg FileConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil // empty file
		}
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return &cfg, nil
}

// applyFileConfig applies config file values to CLIConfig,
// skipping fields explicitly set via CLI flags (tracked in cliSet).
func (c *CLIConfig) applyFileConfig(fc *FileConfig, cliSet map[string]bool) {
	if fc == nil {
		return
	}

	// notSet returns true if none of the given flag names were explicitly set on CLI.
	notSet := func(names ...string) bool {
		for _, n := range names {
			if cliSet[n] {
				return false
			}
		}
		return true
	}

	// Output options
	if fc.Output != nil && notSet("output", "o") {
		c.Output = *fc.Output
	}
	if fc.Color != nil && notSet("color", "c") {
		c.Color = *fc.Color
	}
	if fc.TrueColor != nil && notSet("truecolor", "t") {
		c.TrueColor = *fc.TrueColor
	}

	// Comparison options
	if fc.IgnoreOrderChanges != nil && notSet("ignore-order-changes", "i") {
		c.IgnoreOrderChanges = *fc.IgnoreOrderChanges
	}
	if fc.IgnoreWhitespaceChanges != nil && notSet("ignore-whitespace-changes") {
		c.IgnoreWhitespaceChanges = *fc.IgnoreWhitespaceChanges
	}
	if fc.FormatStrings != nil && notSet("format-strings") {
		c.FormatStrings = *fc.FormatStrings
	}
	if fc.IgnoreValueChanges != nil && notSet("ignore-value-changes", "v") {
		c.IgnoreValueChanges = *fc.IgnoreValueChanges
	}
	if fc.DetectKubernetes != nil && notSet("detect-kubernetes") {
		c.DetectKubernetes = *fc.DetectKubernetes
	}
	if fc.DetectRenames != nil && notSet("detect-renames") {
		c.DetectRenames = *fc.DetectRenames
	}
	if fc.IgnoreApiVersion != nil && notSet("ignore-api-version") {
		c.IgnoreApiVersion = *fc.IgnoreApiVersion
	}
	if fc.NoCertInspection != nil && notSet("no-cert-inspection", "x") {
		c.NoCertInspection = *fc.NoCertInspection
	}
	if fc.Swap != nil && notSet("swap") {
		c.Swap = *fc.Swap
	}

	// Filtering options (replace semantics: CLI replaces config entirely)
	if len(fc.Filter) > 0 && notSet("filter") {
		c.Filter = fc.Filter
	}
	if len(fc.Exclude) > 0 && notSet("exclude") {
		c.Exclude = fc.Exclude
	}
	if len(fc.FilterRegexp) > 0 && notSet("filter-regexp") {
		c.FilterRegexp = fc.FilterRegexp
	}
	if len(fc.ExcludeRegexp) > 0 && notSet("exclude-regexp") {
		c.ExcludeRegexp = fc.ExcludeRegexp
	}
	if len(fc.AdditionalIdentifiers) > 0 && notSet("additional-identifier") {
		c.AdditionalIdentifiers = fc.AdditionalIdentifiers
	}

	// Display options
	if fc.OmitHeader != nil && notSet("omit-header", "b") {
		c.OmitHeader = *fc.OmitHeader
	}
	if fc.UseGoPatchStyle != nil && notSet("use-go-patch-style", "g") {
		c.UseGoPatchStyle = *fc.UseGoPatchStyle
	}
	if fc.MultiLineContextLines != nil && notSet("multi-line-context-lines") {
		c.MultiLineContextLines = *fc.MultiLineContextLines
	}

	// Chroot options
	if fc.Chroot != nil && notSet("chroot") {
		c.Chroot = *fc.Chroot
	}
	if fc.ChrootFrom != nil && notSet("chroot-of-from") {
		c.ChrootFrom = *fc.ChrootFrom
	}
	if fc.ChrootTo != nil && notSet("chroot-of-to") {
		c.ChrootTo = *fc.ChrootTo
	}
	if fc.ChrootListToDocuments != nil && notSet("chroot-list-to-documents") {
		c.ChrootListToDocuments = *fc.ChrootListToDocuments
	}

	// AI Summary options
	if fc.Summary != nil && notSet("summary", "S") {
		c.Summary = *fc.Summary
	}
	if fc.SummaryModel != nil && notSet("summary-model") {
		c.SummaryModel = *fc.SummaryModel
	}

	// Exit code behavior
	if fc.SetExitCode != nil && notSet("set-exit-code", "s") {
		c.SetExitCode = *fc.SetExitCode
	}
}

// loadColorPalette builds a custom color palette from config file and environment variables.
// Environment variables override config file values. Returns nil when no custom colors are set.
func loadColorPalette(fc *FileConfig) (*diffyml.CustomColorPalette, error) {
	palette := diffyml.DefaultCustomColorPalette()
	anyCustom := false

	applyColor := func(name, envVar string, cfgVal *string, target **diffyml.CustomColor) error {
		spec := os.Getenv(envVar)
		if spec == "" && cfgVal != nil {
			spec = *cfgVal
		}
		if spec == "" {
			return nil
		}
		c, err := diffyml.ParseColor(spec)
		if err != nil {
			return fmt.Errorf("invalid %s color: %w", name, err)
		}
		*target = c
		anyCustom = true
		return nil
	}

	var added, removed, modified, context, docName *string
	if fc != nil && fc.Colors != nil {
		added = fc.Colors.Added
		removed = fc.Colors.Removed
		modified = fc.Colors.Modified
		context = fc.Colors.Context
		docName = fc.Colors.DocName
	}

	if err := applyColor("added", "DIFFYML_COLOR_ADDED", added, &palette.Added); err != nil {
		return nil, err
	}
	if err := applyColor("removed", "DIFFYML_COLOR_REMOVED", removed, &palette.Removed); err != nil {
		return nil, err
	}
	if err := applyColor("modified", "DIFFYML_COLOR_MODIFIED", modified, &palette.Modified); err != nil {
		return nil, err
	}
	if err := applyColor("context", "DIFFYML_COLOR_CONTEXT", context, &palette.Context); err != nil {
		return nil, err
	}
	if err := applyColor("doc-name", "DIFFYML_COLOR_DOC_NAME", docName, &palette.DocName); err != nil {
		return nil, err
	}

	if !anyCustom {
		return nil, nil
	}
	return palette, nil
}
