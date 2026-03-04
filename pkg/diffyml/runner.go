// runner.go - Execution orchestration for the comparison flow.
//
// Key types: RunConfig (runtime IO), ExitResult.
// Key functions: Run() executes the full comparison flow.
// Exit codes: 0=success, 1=differences (with -s), 255=error.
package diffyml

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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

// exitError logs an error to stderr and returns an ExitResult with ExitCodeError.
func exitError(rc *RunConfig, err error) *ExitResult {
	fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
	return &ExitResult{ExitCodeError, err}
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

// isRealMode reports whether the RunConfig has no pre-loaded test content.
func (rc *RunConfig) isRealMode() bool {
	return rc.FromContent == nil && rc.ToContent == nil && rc.FilePairs == nil
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
		return &ExitResult{ExitCodeSuccess, nil}
	}

	// Validate configuration and detect directories (skip in test mode)
	if rc.isRealMode() {
		if err := cfg.Validate(); err != nil {
			return exitError(rc, err)
		}

		if cfg.Summary && os.Getenv("ANTHROPIC_API_KEY") == "" {
			return exitError(rc, fmt.Errorf("--summary requires ANTHROPIC_API_KEY environment variable to be set"))
		}

		// Directory detection
		fromIsDir := IsDirectory(cfg.FromFile)
		toIsDir := IsDirectory(cfg.ToFile)

		if fromIsDir && toIsDir {
			return runDirectory(cfg, rc, cfg.FromFile, cfg.ToFile)
		}
		if fromIsDir != toIsDir {
			return exitError(rc, fmt.Errorf("both arguments must be the same type (both files or both directories)"))
		}
	}

	// Build shared options
	opts, err := cfg.buildRunOpts()
	if err != nil {
		return exitError(rc, err)
	}

	// Load file contents
	fromContent, err := loadOrUse(rc.FromContent, cfg.FromFile)
	if err != nil {
		return exitError(rc, err)
	}
	toContent, err := loadOrUse(rc.ToContent, cfg.ToFile)
	if err != nil {
		return exitError(rc, err)
	}

	// Compare and filter
	diffs, err := compareAndFilter(fromContent, toContent, opts.compare, opts.filter)
	if err != nil {
		return exitError(rc, err)
	}

	// Set file path for formatters that use it (e.g., GitLab)
	filePath := normalizeFilePath(cfg.ToFile, rc.Stderr)
	opts.format.FilePath = filePath

	// Format output (defer printing for brief+summary mode)
	formatted := opts.formatter.Format(diffs, opts.format)
	if !cfg.isBriefSummary() {
		fmt.Fprint(rc.Stdout, formatted)
	}

	// AI summary
	if cfg.Summary && len(diffs) > 0 {
		groups := []DiffGroup{{FilePath: filePath, Diffs: diffs}}
		summaryOutput, err := invokeSummary(cfg, rc, groups, opts.format)
		if err != nil {
			if cfg.isBriefSummary() {
				fmt.Fprint(rc.Stdout, formatted)
			}
			fmt.Fprintf(rc.Stderr, "Warning: AI summary unavailable: %v\n", err)
		} else {
			fmt.Fprint(rc.Stdout, summaryOutput)
		}
	}

	// Determine exit code
	exitCode := DetermineExitCode(cfg.SetExitCode, len(diffs), nil)
	return &ExitResult{exitCode, nil}
}

// applyColorConfig applies color settings to format options.
func applyColorConfig(cfg *CLIConfig, formatOpts *FormatOptions) {
	colorMode, _ := ParseColorMode(cfg.Color)
	trueColorMode, _ := ParseColorMode(cfg.TrueColor)
	colorCfg := NewColorConfig(colorMode, trueColorMode == ColorModeAlways)
	colorCfg.DetectTerminal()
	colorCfg.ToFormatOptions(formatOpts)
}

// runOpts holds the shared formatter and options built from CLIConfig.
type runOpts struct {
	formatter Formatter
	compare   *Options
	filter    *FilterOptions
	format    *FormatOptions
}

// buildRunOpts creates shared formatter and options from the CLI config.
func (cfg *CLIConfig) buildRunOpts() (*runOpts, error) {
	formatter, err := GetFormatter(cfg.Output)
	if err != nil {
		return nil, err
	}
	formatOpts := cfg.ToFormatOptions()
	applyColorConfig(cfg, formatOpts)
	return &runOpts{
		formatter: formatter,
		compare:   cfg.ToCompareOptions(),
		filter:    cfg.ToFilterOptions(),
		format:    formatOpts,
	}, nil
}

// loadOrUse returns preloaded content if non-nil, otherwise loads from path.
func loadOrUse(preloaded []byte, path string) ([]byte, error) {
	if preloaded != nil {
		return preloaded, nil
	}
	return LoadContent(path)
}

// compareAndFilter runs Compare followed by FilterDiffsWithRegexp.
func compareAndFilter(from, to []byte, compareOpts *Options, filterOpts *FilterOptions) ([]Difference, error) {
	diffs, err := Compare(from, to, compareOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to compare files: %w", err)
	}
	diffs, err = FilterDiffsWithRegexp(diffs, filterOpts)
	if err != nil {
		return nil, fmt.Errorf("filter error: %w", err)
	}
	return diffs, nil
}

// invokeSummary runs the AI summarizer and returns the formatted summary string.
// Callers must check cfg.Summary and non-empty diffs before calling.
func invokeSummary(cfg *CLIConfig, rc *RunConfig, groups []DiffGroup, formatOpts *FormatOptions) (string, error) {
	summarizer := NewSummarizer(cfg.SummaryModel)
	if rc.SummaryAPIURL != "" {
		summarizer.apiURL = rc.SummaryAPIURL
	}
	summary, err := summarizer.Summarize(context.Background(), groups)
	if err != nil {
		return "", err
	}
	return formatSummaryOutput(summary, formatOpts), nil
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
