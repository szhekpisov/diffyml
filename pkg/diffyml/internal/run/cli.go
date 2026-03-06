// cli.go - Command-line interface parsing and validation.
//
// Key types: CLIConfig (all CLI options).
// Key functions: ParseArgs(), Validate(), Usage().
package run

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// CLIConfig holds all command-line configuration options.
type CLIConfig struct {
	// File arguments
	FromFile string
	ToFile   string

	// Output options
	Output    string // compact, brief, github, gitlab, gitea, detailed
	Color     string // always, never, auto
	TrueColor string // always, never, auto

	// Display options
	OmitHeader            bool
	UseGoPatchStyle       bool
	MultiLineContextLines int

	// Comparison options
	IgnoreOrderChanges      bool
	IgnoreWhitespaceChanges bool
	IgnoreValueChanges      bool
	DetectKubernetes        bool
	DetectRenames           bool
	IgnoreApiVersion        bool
	NoCertInspection        bool
	Swap                    bool
	AdditionalIdentifiers   []string

	// Filtering options
	Filter        []string
	Exclude       []string
	FilterRegexp  []string
	ExcludeRegexp []string

	// Chroot options
	Chroot                string
	ChrootFrom            string
	ChrootTo              string
	ChrootListToDocuments bool

	// AI Summary options
	Summary      bool   // --summary / -S: enable AI summary
	SummaryModel string // --summary-model: Anthropic model override

	// Exit code behavior
	SetExitCode bool
	ShowHelp    bool

	// Internal flagset
	fs *flag.FlagSet
}

// NewCLIConfig creates a new CLI configuration with default values.
func NewCLIConfig() *CLIConfig {
	cfg := &CLIConfig{
		Output:                "detailed",
		Color:                 "auto",
		TrueColor:             "auto",
		DetectKubernetes:      true,
		DetectRenames:         true,
		MultiLineContextLines: 4,
	}
	cfg.initFlags()
	return cfg
}

// sliceFlag registers a flag that appends each occurrence to the given slice.
func (c *CLIConfig) sliceFlag(field *[]string, name, usage string) {
	c.fs.Func(name, usage, func(s string) error {
		*field = append(*field, s)
		return nil
	})
}

// initFlags sets up the flag definitions.
func (c *CLIConfig) initFlags() {
	c.fs = flag.NewFlagSet("diffyml", flag.ContinueOnError)

	// Output options
	c.fs.StringVar(&c.Output, "o", c.Output, "")
	c.fs.StringVar(&c.Output, "output", c.Output, "specify the output style: compact, brief, github, gitlab, gitea, detailed")
	c.fs.StringVar(&c.Color, "c", c.Color, "")
	c.fs.StringVar(&c.Color, "color", c.Color, "specify color usage: always, never, or auto")
	c.fs.StringVar(&c.TrueColor, "t", c.TrueColor, "")
	c.fs.StringVar(&c.TrueColor, "truecolor", c.TrueColor, "specify true color usage: always, never, or auto")

	// Display options
	c.fs.BoolVar(&c.OmitHeader, "b", c.OmitHeader, "")
	c.fs.BoolVar(&c.OmitHeader, "omit-header", c.OmitHeader, "omit the diffyml summary header")
	c.fs.BoolVar(&c.UseGoPatchStyle, "g", c.UseGoPatchStyle, "")
	c.fs.BoolVar(&c.UseGoPatchStyle, "use-go-patch-style", c.UseGoPatchStyle, "use Go-Patch style paths in outputs")
	c.fs.IntVar(&c.MultiLineContextLines, "multi-line-context-lines", c.MultiLineContextLines, "multi-line context lines")

	// Comparison options
	c.fs.BoolVar(&c.IgnoreOrderChanges, "i", c.IgnoreOrderChanges, "")
	c.fs.BoolVar(&c.IgnoreOrderChanges, "ignore-order-changes", c.IgnoreOrderChanges, "ignore order changes in lists")
	c.fs.BoolVar(&c.IgnoreWhitespaceChanges, "ignore-whitespace-changes", c.IgnoreWhitespaceChanges, "ignore leading or trailing whitespace changes")
	c.fs.BoolVar(&c.IgnoreValueChanges, "v", c.IgnoreValueChanges, "")
	c.fs.BoolVar(&c.IgnoreValueChanges, "ignore-value-changes", c.IgnoreValueChanges, "exclude changes in values")
	c.fs.BoolVar(&c.DetectKubernetes, "detect-kubernetes", c.DetectKubernetes, "detect kubernetes entities")
	c.fs.BoolVar(&c.DetectRenames, "detect-renames", c.DetectRenames, "enable detection for renames")
	c.fs.BoolVar(&c.IgnoreApiVersion, "ignore-api-version", c.IgnoreApiVersion, "ignore apiVersion when matching Kubernetes resources")
	c.fs.BoolVar(&c.NoCertInspection, "x", c.NoCertInspection, "")
	c.fs.BoolVar(&c.NoCertInspection, "no-cert-inspection", c.NoCertInspection, "disable x509 certificate inspection")
	c.fs.BoolVar(&c.Swap, "swap", c.Swap, "swap 'from' and 'to' for comparison")

	// Filter options - using custom slice vars
	c.sliceFlag(&c.Filter, "filter", "filter reports to a subset of differences")
	c.sliceFlag(&c.Exclude, "exclude", "exclude reports from a set of differences")
	c.sliceFlag(&c.FilterRegexp, "filter-regexp", "filter reports using regular expressions")
	c.sliceFlag(&c.ExcludeRegexp, "exclude-regexp", "exclude reports using regular expressions")
	c.sliceFlag(&c.AdditionalIdentifiers, "additional-identifier", "use additional identifier in named entry lists")

	// Chroot options
	c.fs.StringVar(&c.Chroot, "chroot", c.Chroot, "change the root level of the input file")
	c.fs.StringVar(&c.ChrootFrom, "chroot-of-from", c.ChrootFrom, "only change the root level of the from input file")
	c.fs.StringVar(&c.ChrootTo, "chroot-of-to", c.ChrootTo, "only change the root level of the to input file")
	c.fs.BoolVar(&c.ChrootListToDocuments, "chroot-list-to-documents", c.ChrootListToDocuments, "treat chroot list as set of documents")

	// AI Summary options
	c.fs.BoolVar(&c.Summary, "S", c.Summary, "")
	c.fs.BoolVar(&c.Summary, "summary", c.Summary, "enable AI-powered summary of differences")
	c.fs.StringVar(&c.SummaryModel, "summary-model", c.SummaryModel, "specify Anthropic model for summary")

	// Exit code behavior
	c.fs.BoolVar(&c.SetExitCode, "s", c.SetExitCode, "")
	c.fs.BoolVar(&c.SetExitCode, "set-exit-code", c.SetExitCode, "set program exit code based on differences")
	c.fs.BoolVar(&c.ShowHelp, "h", c.ShowHelp, "")
	c.fs.BoolVar(&c.ShowHelp, "help", c.ShowHelp, "show help")
}

// ParseArgs parses command-line arguments.
// Expects at least two non-flag arguments: <from> and <to> files.
//
// Supports interspersed flags and positional arguments (e.g.
// "dir1 dir2 --set-exit-code") because kubectl places
// KUBECTL_EXTERNAL_DIFF flags after the directory paths.
func (c *CLIConfig) ParseArgs(args []string) error {
	reordered := reorderArgs(args, c.fs)

	if err := c.fs.Parse(reordered); err != nil {
		return err
	}

	// Get remaining non-flag arguments (file paths)
	remaining := c.fs.Args()

	if len(remaining) < 2 {
		return fmt.Errorf("requires two file arguments: <from> <to>")
	}

	c.FromFile = remaining[0]
	c.ToFile = remaining[1]

	return nil
}

// isBoolFlag returns true if the flag is a boolean flag.
func isBoolFlag(f *flag.Flag) bool {
	// BoolVar flags have zero value "false".
	// This matches how Go's flag package internally identifies bool flags.
	bf, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && bf.IsBoolFlag()
}

// reorderArgs moves flag arguments before positional arguments so that
// Go's flag package (which stops at the first non-flag arg) can parse
// all flags. Positional arguments preserve their relative order.
func reorderArgs(args []string, fs *flag.FlagSet) []string {
	var flags, positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		// Extract flag name: strip leading dashes, remove =value.
		name := strings.TrimLeft(arg, "-")
		name, _, _ = strings.Cut(name, "=")

		f := fs.Lookup(name)
		if f == nil {
			// Unknown flag — keep as positional so fs.Parse reports the error.
			positional = append(positional, arg)
			continue
		}

		flags = append(flags, arg)

		// If this is a non-bool flag without "=" form, consume next arg as value.
		if !strings.Contains(arg, "=") && !isBoolFlag(f) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}

	return append(flags, positional...)
}

// ToCompareOptions converts CLI config to comparison Options.
func (c *CLIConfig) ToCompareOptions() *types.Options {
	return &types.Options{
		IgnoreOrderChanges:      c.IgnoreOrderChanges,
		IgnoreWhitespaceChanges: c.IgnoreWhitespaceChanges,
		IgnoreValueChanges:      c.IgnoreValueChanges,
		DetectKubernetes:        c.DetectKubernetes,
		DetectRenames:           c.DetectRenames,
		IgnoreApiVersion:        c.IgnoreApiVersion,
		AdditionalIdentifiers:   c.AdditionalIdentifiers,
		Swap:                    c.Swap,
		Chroot:                  c.Chroot,
		ChrootFrom:              c.ChrootFrom,
		ChrootTo:                c.ChrootTo,
		ChrootListToDocuments:   c.ChrootListToDocuments,
	}
}

// ToFilterOptions converts CLI config to FilterOptions.
func (c *CLIConfig) ToFilterOptions() *types.FilterOptions {
	return &types.FilterOptions{
		IncludePaths:  c.Filter,
		ExcludePaths:  c.Exclude,
		IncludeRegexp: c.FilterRegexp,
		ExcludeRegexp: c.ExcludeRegexp,
	}
}

// ToFormatOptions converts CLI config to FormatOptions.
func (c *CLIConfig) ToFormatOptions() *types.FormatOptions {
	return &types.FormatOptions{
		OmitHeader:       c.OmitHeader,
		UseGoPatchStyle:  c.UseGoPatchStyle,
		ContextLines:     c.MultiLineContextLines,
		NoCertInspection: c.NoCertInspection,
	}
}

// ToRunOptions converts CLIConfig to RunOptions with all options resolved.
func (c *CLIConfig) ToRunOptions() *RunOptions {
	formatOpts := c.ToFormatOptions()
	ApplyColorConfig(c, formatOpts)
	return &RunOptions{
		FromFile:     c.FromFile,
		ToFile:       c.ToFile,
		SetExitCode:  c.SetExitCode,
		Swap:         c.Swap,
		Output:       c.Output,
		Summary:      c.Summary,
		SummaryModel: c.SummaryModel,
		CompareOpts:  c.ToCompareOptions(),
		FilterOpts:   c.ToFilterOptions(),
		FormatOpts:   formatOpts,
	}
}

// Usage returns the usage help text.
func (c *CLIConfig) Usage() string {
	return `diffyml - A diff tool for YAML files

Usage:
  diffyml [flags] <from> <to>

Flags:
  -o, --output string                 specify output style: compact, brief, github, gitlab, gitea, detailed (default "detailed")
  -c, --color string                  specify color usage: always, never, or auto (default "auto")
  -t, --truecolor string              specify true color usage: always, never, or auto (default "auto")

  -i, --ignore-order-changes          ignore order changes in lists
      --ignore-whitespace-changes     ignore leading or trailing whitespace changes
  -v, --ignore-value-changes          exclude changes in values
      --detect-kubernetes             detect kubernetes entities (default true)
      --detect-renames                enable detection for renames (default true)
      --ignore-api-version            ignore apiVersion when matching Kubernetes resources
  -x, --no-cert-inspection            disable x509 certificate inspection
      --swap                          swap 'from' and 'to' for comparison

      --filter strings                filter reports to a subset of differences
      --exclude strings               exclude reports from a set of differences
      --filter-regexp strings         filter reports using regular expressions
      --exclude-regexp strings        exclude reports using regular expressions
      --additional-identifier string  use additional identifier in named entry lists

  -b, --omit-header                   omit the diffyml summary header
  -g, --use-go-patch-style            use Go-Patch style paths in outputs
      --multi-line-context-lines int  multi-line context lines (default 4)

      --chroot string                 change the root level of the input file
      --chroot-of-from string         only change the root level of the from input file
      --chroot-of-to string           only change the root level of the to input file
      --chroot-list-to-documents      treat chroot list as set of documents

  -S, --summary                       enable AI-powered summary of differences
      --summary-model string          specify Anthropic model for summary

  -s, --set-exit-code                 set program exit code based on differences
  -h, --help                          show this help
  -V, --version                       show version information
`
}

// Validate validates the CLI configuration.
// Returns an error if any configuration is invalid.
func (c *CLIConfig) Validate() error {
	// Validate file arguments
	if c.FromFile == "" {
		return &types.ValidationError{Field: "from", Message: "missing 'from' file argument"}
	}
	if c.ToFile == "" {
		return &types.ValidationError{Field: "to", Message: "missing 'to' file argument"}
	}

	// Validate output format
	if err := ValidateOutputFormat(c.Output); err != nil {
		return err
	}

	// Validate color mode
	if _, err := types.ParseColorMode(c.Color); err != nil {
		return &types.ValidationError{
			Field:   "color",
			Value:   c.Color,
			Message: fmt.Sprintf("invalid color mode %q, valid modes: always, never, auto", c.Color),
		}
	}

	// Validate truecolor mode
	if _, err := types.ParseColorMode(c.TrueColor); err != nil {
		return &types.ValidationError{
			Field:   "truecolor",
			Value:   c.TrueColor,
			Message: fmt.Sprintf("invalid truecolor mode %q, valid modes: always, never, auto", c.TrueColor),
		}
	}

	// Validate regex patterns
	if err := ValidateRegexPatterns(c.FilterRegexp, "filter-regexp"); err != nil {
		return err
	}
	if err := ValidateRegexPatterns(c.ExcludeRegexp, "exclude-regexp"); err != nil {
		return err
	}

	return nil
}

// ValidateOutputFormat checks if the output format name is valid.
// Returns an error listing valid options if the format is invalid.
func ValidateOutputFormat(name string) error {
	lower := strings.ToLower(name)
	for _, valid := range format.ValidFormatterNames {
		if lower == valid {
			return nil
		}
	}
	return &types.ValidationError{
		Field:   "output",
		Value:   name,
		Message: fmt.Sprintf("unknown output format %q, valid formats: %s", name, strings.Join(format.ValidFormatterNames, ", ")),
	}
}

// ValidateRegexPatterns validates a list of regex patterns.
// Returns an error with the invalid pattern and flag name if any pattern is invalid.
func ValidateRegexPatterns(patterns []string, flagName string) error {
	for _, pattern := range patterns {
		_, err := regexp.Compile(pattern)
		if err != nil {
			return &types.ValidationError{
				Field:   flagName,
				Value:   pattern,
				Message: fmt.Sprintf("invalid regex pattern %q in --%s: %v", pattern, flagName, err),
			}
		}
	}
	return nil
}
