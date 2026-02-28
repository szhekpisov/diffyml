// cli.go - Command-line interface parsing and execution.
//
// Key types: CLIConfig (all CLI options), RunConfig (runtime IO), ExitResult.
// Key functions: Run() executes the full comparison flow.
// Exit codes: 0=success, 1=differences (with -s), 255=error.
package diffyml

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	FixedWidth            int
	OmitHeader            bool
	NoTableStyle          bool
	UseGoPatchStyle       bool
	MultiLineContextLines int
	MinorChangeThreshold  float64

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
		FixedWidth:            -1,
		DetectKubernetes:      true,
		DetectRenames:         true,
		MultiLineContextLines: 4,
		MinorChangeThreshold:  0.1,
	}
	cfg.initFlags()
	return cfg
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
	c.fs.IntVar(&c.FixedWidth, "w", c.FixedWidth, "")
	c.fs.IntVar(&c.FixedWidth, "fixed-width", c.FixedWidth, "disable terminal width detection and use provided fixed value")
	c.fs.BoolVar(&c.OmitHeader, "b", c.OmitHeader, "")
	c.fs.BoolVar(&c.OmitHeader, "omit-header", c.OmitHeader, "omit the diffyml summary header")
	c.fs.BoolVar(&c.NoTableStyle, "l", c.NoTableStyle, "")
	c.fs.BoolVar(&c.NoTableStyle, "no-table-style", c.NoTableStyle, "do not place blocks next to each other")
	c.fs.BoolVar(&c.UseGoPatchStyle, "g", c.UseGoPatchStyle, "")
	c.fs.BoolVar(&c.UseGoPatchStyle, "use-go-patch-style", c.UseGoPatchStyle, "use Go-Patch style paths in outputs")
	c.fs.IntVar(&c.MultiLineContextLines, "multi-line-context-lines", c.MultiLineContextLines, "multi-line context lines")
	c.fs.Float64Var(&c.MinorChangeThreshold, "minor-change-threshold", c.MinorChangeThreshold, "minor change threshold")

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
	c.fs.Func("filter", "filter reports to a subset of differences", func(s string) error {
		c.Filter = append(c.Filter, s)
		return nil
	})
	c.fs.Func("exclude", "exclude reports from a set of differences", func(s string) error {
		c.Exclude = append(c.Exclude, s)
		return nil
	})
	c.fs.Func("filter-regexp", "filter reports using regular expressions", func(s string) error {
		c.FilterRegexp = append(c.FilterRegexp, s)
		return nil
	})
	c.fs.Func("exclude-regexp", "exclude reports using regular expressions", func(s string) error {
		c.ExcludeRegexp = append(c.ExcludeRegexp, s)
		return nil
	})
	c.fs.Func("additional-identifier", "use additional identifier in named entry lists", func(s string) error {
		c.AdditionalIdentifiers = append(c.AdditionalIdentifiers, s)
		return nil
	})

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

	skip := false
	for i, arg := range args {
		if skip {
			skip = false
			continue
		}

		if arg == "--" {
			positional = append(positional, args[i:]...)
			break
		}

		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		// Extract flag name: strip leading dashes, remove =value.
		name := strings.TrimLeft(arg, "-")
		if eqIdx := strings.IndexByte(name, '='); eqIdx >= 0 {
			name = name[:eqIdx]
		}

		f := fs.Lookup(name)
		if f == nil {
			// Unknown flag — keep as positional so fs.Parse reports the error.
			positional = append(positional, arg)
			continue
		}

		flags = append(flags, arg)

		// If this is a non-bool flag without "=" form, consume next arg as value.
		if !strings.Contains(arg, "=") && !isBoolFlag(f) && i+1 < len(args) {
			flags = append(flags, args[i+1])
			skip = true
		}
	}

	return append(flags, positional...)
}

// ToCompareOptions converts CLI config to comparison Options.
func (c *CLIConfig) ToCompareOptions() *Options {
	return &Options{
		IgnoreOrderChanges:      c.IgnoreOrderChanges,
		IgnoreWhitespaceChanges: c.IgnoreWhitespaceChanges,
		IgnoreValueChanges:      c.IgnoreValueChanges,
		DetectKubernetes:        c.DetectKubernetes,
		DetectRenames:           c.DetectRenames,
		IgnoreApiVersion:        c.IgnoreApiVersion,
		AdditionalIdentifiers:   c.AdditionalIdentifiers,
		NoCertInspection:        c.NoCertInspection,
		Swap:                    c.Swap,
		Chroot:                  c.Chroot,
		ChrootFrom:              c.ChrootFrom,
		ChrootTo:                c.ChrootTo,
		ChrootListToDocuments:   c.ChrootListToDocuments,
	}
}

// ToFilterOptions converts CLI config to FilterOptions.
func (c *CLIConfig) ToFilterOptions() *FilterOptions {
	return &FilterOptions{
		IncludePaths:  c.Filter,
		ExcludePaths:  c.Exclude,
		IncludeRegexp: c.FilterRegexp,
		ExcludeRegexp: c.ExcludeRegexp,
	}
}

// ToFormatOptions converts CLI config to FormatOptions.
func (c *CLIConfig) ToFormatOptions() *FormatOptions {
	return &FormatOptions{
		OmitHeader:           c.OmitHeader,
		NoTableStyle:         c.NoTableStyle,
		UseGoPatchStyle:      c.UseGoPatchStyle,
		ContextLines:         c.MultiLineContextLines,
		MinorChangeThreshold: c.MinorChangeThreshold,
		Width:                c.FixedWidth,
	}
}

// Usage returns the usage help text.
func (c *CLIConfig) Usage() string {
	var sb strings.Builder

	sb.WriteString("diffyml - A diff tool for YAML files\n\n")
	sb.WriteString("Usage:\n")
	sb.WriteString("  diffyml [flags] <from> <to>\n\n")
	sb.WriteString("Flags:\n")

	// Output options
	sb.WriteString("  -o, --output string                 specify output style: compact, brief, github, gitlab, gitea, detailed (default \"detailed\")\n")
	sb.WriteString("  -c, --color string                  specify color usage: always, never, or auto (default \"auto\")\n")
	sb.WriteString("  -t, --truecolor string              specify true color usage: always, never, or auto (default \"auto\")\n")
	sb.WriteString("  -w, --fixed-width int               disable terminal width detection and use provided fixed value (default -1)\n")
	sb.WriteString("\n")

	// Comparison options
	sb.WriteString("  -i, --ignore-order-changes          ignore order changes in lists\n")
	sb.WriteString("      --ignore-whitespace-changes     ignore leading or trailing whitespace changes\n")
	sb.WriteString("  -v, --ignore-value-changes          exclude changes in values\n")
	sb.WriteString("      --detect-kubernetes             detect kubernetes entities (default true)\n")
	sb.WriteString("      --detect-renames                enable detection for renames (default true)\n")
	sb.WriteString("      --ignore-api-version            ignore apiVersion when matching Kubernetes resources\n")
	sb.WriteString("  -x, --no-cert-inspection            disable x509 certificate inspection\n")
	sb.WriteString("      --swap                          swap 'from' and 'to' for comparison\n")
	sb.WriteString("\n")

	// Filter options
	sb.WriteString("      --filter strings                filter reports to a subset of differences\n")
	sb.WriteString("      --exclude strings               exclude reports from a set of differences\n")
	sb.WriteString("      --filter-regexp strings         filter reports using regular expressions\n")
	sb.WriteString("      --exclude-regexp strings        exclude reports using regular expressions\n")
	sb.WriteString("      --additional-identifier string  use additional identifier in named entry lists\n")
	sb.WriteString("\n")

	// Display options
	sb.WriteString("  -b, --omit-header                   omit the diffyml summary header\n")
	sb.WriteString("  -l, --no-table-style                do not place blocks next to each other\n")
	sb.WriteString("  -g, --use-go-patch-style            use Go-Patch style paths in outputs\n")
	sb.WriteString("      --multi-line-context-lines int  multi-line context lines (default 4)\n")
	sb.WriteString("      --minor-change-threshold float  minor change threshold (default 0.1)\n")
	sb.WriteString("\n")

	// Chroot options
	sb.WriteString("      --chroot string                 change the root level of the input file\n")
	sb.WriteString("      --chroot-of-from string         only change the root level of the from input file\n")
	sb.WriteString("      --chroot-of-to string           only change the root level of the to input file\n")
	sb.WriteString("      --chroot-list-to-documents      treat chroot list as set of documents\n")
	sb.WriteString("\n")

	// AI Summary options
	sb.WriteString("  -S, --summary                       enable AI-powered summary of differences\n")
	sb.WriteString("      --summary-model string          specify Anthropic model for summary\n")
	sb.WriteString("\n")

	// Other options
	sb.WriteString("  -s, --set-exit-code                 set program exit code based on differences\n")
	sb.WriteString("  -h, --help                          show this help\n")
	sb.WriteString("  -V, --version                       show version information\n")

	return sb.String()
}

// Validate validates the CLI configuration.
// Returns an error if any configuration is invalid.
func (c *CLIConfig) Validate() error {
	// Validate file arguments
	if c.FromFile == "" {
		return fmt.Errorf("missing 'from' file argument")
	}
	if c.ToFile == "" {
		return fmt.Errorf("missing 'to' file argument")
	}

	// Validate output format
	if err := ValidateOutputFormat(c.Output); err != nil {
		return err
	}

	// Validate color mode
	if _, err := ParseColorMode(c.Color); err != nil {
		return fmt.Errorf("invalid color mode %q, valid modes: always, never, auto", c.Color)
	}

	// Validate truecolor mode
	if _, err := ParseColorMode(c.TrueColor); err != nil {
		return fmt.Errorf("invalid truecolor mode %q, valid modes: always, never, auto", c.TrueColor)
	}

	// Validate regex patterns
	if err := ValidateRegexPatterns(c.FilterRegexp, "filter-regexp"); err != nil {
		return err
	}
	if err := ValidateRegexPatterns(c.ExcludeRegexp, "exclude-regexp"); err != nil {
		return err
	}

	// Validate AI summary configuration
	if c.Summary && os.Getenv("ANTHROPIC_API_KEY") == "" {
		return fmt.Errorf("--summary requires ANTHROPIC_API_KEY environment variable to be set")
	}

	return nil
}

// ValidateFileExists checks if a file exists and is not a directory.
// Returns an error with the file path if validation fails.
func ValidateFileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("cannot access file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}
	return nil
}

// validOutputFormats lists all valid output format names.
var validOutputFormats = []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}

// ValidateOutputFormat checks if the output format name is valid.
// Returns an error listing valid options if the format is invalid.
func ValidateOutputFormat(format string) error {
	lower := strings.ToLower(format)
	for _, valid := range validOutputFormats {
		if lower == valid {
			return nil
		}
	}
	return fmt.Errorf("unknown output format %q, valid formats: %s",
		format, strings.Join(validOutputFormats, ", "))
}

// ValidateRegexPatterns validates a list of regex patterns.
// Returns an error with the invalid pattern and flag name if any pattern is invalid.
func ValidateRegexPatterns(patterns []string, flagName string) error {
	for _, pattern := range patterns {
		_, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern %q in --%s: %w", pattern, flagName, err)
		}
	}
	return nil
}

// Exit code constants for program termination.
const (
	// ExitCodeSuccess indicates successful execution with no differences.
	ExitCodeSuccess = 0
	// ExitCodeDifferences indicates differences were found (with -s flag).
	ExitCodeDifferences = 1
	// ExitCodeError indicates a program error occurred.
	ExitCodeError = 255
)

// DetermineExitCode returns the appropriate exit code based on execution results.
// When setExitCode is true (-s flag):
//   - Returns 0 when no differences found
//   - Returns 1 when differences detected
//   - Returns 255 on program errors
//
// When setExitCode is false:
//   - Returns 0 on success regardless of differences
//   - Returns 255 on program errors
func DetermineExitCode(setExitCode bool, diffCount int, err error) int {
	// Error always takes precedence
	if err != nil {
		return ExitCodeError
	}

	// Without -s flag, always return success (even with differences)
	if !setExitCode {
		return ExitCodeSuccess
	}

	// With -s flag, return 1 if differences found
	if diffCount > 0 {
		return ExitCodeDifferences
	}

	return ExitCodeSuccess
}

// ExitResult encapsulates the result of program execution.
type ExitResult struct {
	Code int
	Err  error
}

// NewExitResult creates a new ExitResult.
func NewExitResult(code int, err error) *ExitResult {
	return &ExitResult{
		Code: code,
		Err:  err,
	}
}

// IsSuccess returns true if the execution was successful (code 0).
func (r *ExitResult) IsSuccess() bool {
	return r.Code == ExitCodeSuccess
}

// HasDifferences returns true if differences were detected (code 1).
func (r *ExitResult) HasDifferences() bool {
	return r.Code == ExitCodeDifferences
}

// String returns a human-readable description of the result.
func (r *ExitResult) String() string {
	switch r.Code {
	case ExitCodeSuccess:
		return "success: no differences found"
	case ExitCodeDifferences:
		return "differences detected"
	case ExitCodeError:
		if r.Err != nil {
			return fmt.Sprintf("error: %v", r.Err)
		}
		return "error: unknown error"
	default:
		return fmt.Sprintf("unknown exit code: %d", r.Code)
	}
}

// RunConfig holds runtime configuration for the Run function.
type RunConfig struct {
	// Stdout is the writer for standard output.
	Stdout io.Writer
	// Stderr is the writer for error output.
	Stderr io.Writer
	// FromContent is optional pre-loaded content for 'from' file (for testing).
	FromContent []byte
	// ToContent is optional pre-loaded content for 'to' file (for testing).
	ToContent []byte
	// FilePairs is optional in-memory file pairs for directory-mode testing.
	// Key: filename; Value: [0]=from content, [1]=to content.
	// nil content at [0] means file absent in from-dir.
	// nil content at [1] means file absent in to-dir.
	FilePairs map[string][2][]byte
	// SummaryAPIURL overrides the Anthropic API URL (for testing).
	SummaryAPIURL string
}

// NewRunConfig creates a new RunConfig with default values.
func NewRunConfig() *RunConfig {
	return &RunConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run executes the main comparison flow with the given configuration.
// Returns an ExitResult with the appropriate exit code and any error.
func Run(cfg *CLIConfig, rc *RunConfig) *ExitResult {
	if rc == nil {
		rc = NewRunConfig()
	}

	// Handle help flag
	if cfg.ShowHelp {
		fmt.Fprint(rc.Stdout, cfg.Usage())
		return NewExitResult(ExitCodeSuccess, nil)
	}

	// Validate configuration (unless using pre-loaded content for testing)
	if rc.FromContent == nil && rc.ToContent == nil {
		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
			return NewExitResult(ExitCodeError, err)
		}
	}

	// Directory detection (skip when test content is pre-loaded)
	if rc.FromContent == nil && rc.ToContent == nil && rc.FilePairs == nil {
		fromIsDir := IsDirectory(cfg.FromFile)
		toIsDir := IsDirectory(cfg.ToFile)

		if fromIsDir && toIsDir {
			return runDirectory(cfg, rc, cfg.FromFile, cfg.ToFile)
		}
		if fromIsDir != toIsDir {
			err := fmt.Errorf("both arguments must be the same type (both files or both directories)")
			fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
			return NewExitResult(ExitCodeError, err)
		}
	}

	// Get the formatter
	formatter, err := GetFormatter(cfg.Output)
	if err != nil {
		fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
		return NewExitResult(ExitCodeError, err)
	}

	// Load file contents
	var fromContent, toContent []byte

	if rc.FromContent != nil {
		fromContent = rc.FromContent
	} else {
		fromContent, err = LoadContent(cfg.FromFile)
		if err != nil {
			fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
			return NewExitResult(ExitCodeError, err)
		}
	}

	if rc.ToContent != nil {
		toContent = rc.ToContent
	} else {
		toContent, err = LoadContent(cfg.ToFile)
		if err != nil {
			fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
			return NewExitResult(ExitCodeError, err)
		}
	}

	// Setup options
	compareOpts := cfg.ToCompareOptions()
	filterOpts := cfg.ToFilterOptions()
	formatOpts := cfg.ToFormatOptions()

	// Apply color configuration
	colorMode, _ := ParseColorMode(cfg.Color)
	trueColorMode, _ := ParseColorMode(cfg.TrueColor)
	colorCfg := NewColorConfig(colorMode, trueColorMode == ColorModeAlways, cfg.FixedWidth)
	colorCfg.DetectTerminal()
	colorCfg.ToFormatOptions(formatOpts)

	// Compare files
	diffs, err := Compare(fromContent, toContent, compareOpts)
	if err != nil {
		err = fmt.Errorf("failed to compare files: %w", err)
		fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
		return NewExitResult(ExitCodeError, err)
	}

	// Apply filters
	diffs, err = FilterDiffsWithRegexp(diffs, filterOpts)
	if err != nil {
		err = fmt.Errorf("filter error: %w", err)
		fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
		return NewExitResult(ExitCodeError, err)
	}

	// Set file path for formatters that use it (e.g., GitLab)
	formatOpts.FilePath = normalizeFilePath(cfg.ToFile, rc.Stderr)

	// For brief + summary: defer output until we know if the API call succeeds
	isBriefSummary := cfg.Output == "brief" && cfg.Summary

	// Format and output
	if !isBriefSummary {
		output := formatter.Format(diffs, formatOpts)
		fmt.Fprint(rc.Stdout, output)
	}

	// AI Summary
	if cfg.Summary && len(diffs) > 0 {
		summarizer := NewSummarizer(cfg.SummaryModel)
		if rc.SummaryAPIURL != "" {
			summarizer.apiURL = rc.SummaryAPIURL
		}
		groups := []DiffGroup{{FilePath: normalizeFilePath(cfg.ToFile, rc.Stderr), Diffs: diffs}}
		summary, err := summarizer.Summarize(context.Background(), groups)
		if err != nil {
			if isBriefSummary {
				// Fallback: show brief output since AI summary failed
				fmt.Fprint(rc.Stdout, formatter.Format(diffs, formatOpts))
			}
			fmt.Fprintf(rc.Stderr, "Warning: AI summary unavailable: %v\n", err)
		} else {
			fmt.Fprint(rc.Stdout, formatSummaryOutput(summary, formatOpts))
		}
	} else if isBriefSummary {
		// No diffs but brief+summary: write standard brief output
		fmt.Fprint(rc.Stdout, formatter.Format(diffs, formatOpts))
	}

	// Determine exit code
	exitCode := DetermineExitCode(cfg.SetExitCode, len(diffs), nil)
	return NewExitResult(exitCode, nil)
}

// normalizeFilePath converts a file path to a clean relative path.
// Strips "./" prefix, converts absolute paths to relative from CWD.
// Falls back to the original path if relative conversion fails or
// produces a parent-traversing path (".."). Emits a warning to stderr
// if the fallback results in an absolute path.
func normalizeFilePath(path string, stderr io.Writer) string {
	if path == "" {
		return ""
	}

	if filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err == nil {
			rel, err := filepath.Rel(cwd, path)
			if err == nil && !strings.HasPrefix(rel, "..") {
				return strings.TrimPrefix(rel, "./")
			}
		}
		// Fallback: absolute path couldn't be made relative
		if stderr != nil {
			fmt.Fprintf(stderr, "Warning: could not determine relative path for %s, using absolute path\n", path)
		}
		return path
	}

	return strings.TrimPrefix(path, "./")
}
