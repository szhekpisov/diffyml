// runner.go - Execution orchestration for the comparison flow.
//
// Key types: RunConfig (runtime IO), ExitResult.
// Key functions: Run() executes the full comparison flow.
// Exit codes: 0=success, 1=differences (with -s), 255=error.
package run

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Run executes the main comparison flow with the given configuration.
// Returns an ExitResult with the appropriate exit code and any error.
func Run(cfg *CLIConfig, rc *types.RunConfig) *types.ExitResult {
	if rc == nil {
		rc = types.NewRunConfig()
	}

	// Handle help flag
	if cfg.ShowHelp {
		fmt.Fprint(rc.Stdout, cfg.Usage())
		return &types.ExitResult{Code: types.ExitCodeSuccess, Err: nil}
	}

	// Validate configuration (skip in test mode)
	if rc.IsRealMode() {
		if err := cfg.Validate(); err != nil {
			return types.ExitError(rc, err)
		}

		if cfg.Summary && os.Getenv("ANTHROPIC_API_KEY") == "" {
			return types.ExitError(rc, fmt.Errorf("--summary requires ANTHROPIC_API_KEY environment variable to be set"))
		}
	}

	runOpts := cfg.ToRunOptions()

	// Directory detection (real mode only)
	if rc.IsRealMode() {
		fromIsDir := IsDirectory(runOpts.FromFile)
		toIsDir := IsDirectory(runOpts.ToFile)

		if fromIsDir && toIsDir {
			return RunDirectory(runOpts, rc, runOpts.FromFile, runOpts.ToFile)
		}
		if fromIsDir != toIsDir {
			return types.ExitError(rc, fmt.Errorf("both arguments must be the same type (both files or both directories)"))
		}
	}

	return RunWithOpts(runOpts, rc)
}

// RunWithOpts executes the core comparison flow using resolved RunOptions.
func RunWithOpts(opts *types.RunOptions, rc *types.RunConfig) *types.ExitResult {
	// Build formatter and shared options
	ro, err := BuildRunOpts(opts)
	if err != nil {
		return types.ExitError(rc, err)
	}

	// Load file contents
	fromContent, err := LoadOrUse(rc.FromContent, opts.FromFile)
	if err != nil {
		return types.ExitError(rc, err)
	}
	toContent, err := LoadOrUse(rc.ToContent, opts.ToFile)
	if err != nil {
		return types.ExitError(rc, err)
	}

	// Compare and filter
	diffs, err := CompareAndFilter(fromContent, toContent, ro.compare, ro.filter)
	if err != nil {
		return types.ExitError(rc, err)
	}

	// Set file path for formatters that use it (e.g., GitLab)
	filePath := NormalizeFilePath(opts.ToFile, rc.Stderr)
	ro.format.FilePath = filePath

	// Format output (defer printing for brief+summary mode)
	formatted := ro.formatter.Format(diffs, ro.format)
	if !opts.IsBriefSummary() {
		fmt.Fprint(rc.Stdout, formatted)
	}

	// AI summary
	if opts.Summary && len(diffs) > 0 {
		groups := []types.DiffGroup{{FilePath: filePath, Diffs: diffs}}
		summaryOutput, err := InvokeSummary(opts.SummaryModel, rc, groups, ro.format)
		if err != nil {
			if opts.IsBriefSummary() {
				fmt.Fprint(rc.Stdout, formatted)
			}
			fmt.Fprintf(rc.Stderr, "Warning: AI summary unavailable: %v\n", err)
		} else {
			fmt.Fprint(rc.Stdout, summaryOutput)
		}
	}

	// Determine exit code
	exitCode := types.DetermineExitCode(opts.SetExitCode, len(diffs), nil)
	return &types.ExitResult{Code: exitCode, Err: nil}
}

// ApplyColorConfig applies color settings to format options.
func ApplyColorConfig(cfg *CLIConfig, formatOpts *types.FormatOptions) {
	colorMode, _ := types.ParseColorMode(cfg.Color)
	trueColorMode, _ := types.ParseColorMode(cfg.TrueColor)
	colorCfg := types.NewColorConfig(colorMode, trueColorMode == types.ColorModeAlways)
	colorCfg.DetectTerminal()
	colorCfg.ToFormatOptions(formatOpts)
}

// runOpts holds the shared formatter and options built from RunOptions.
type runOpts struct {
	formatter types.Formatter
	compare   *types.Options
	filter    *types.FilterOptions
	format    *types.FormatOptions
}

// BuildRunOpts creates the internal runOpts from RunOptions.
func BuildRunOpts(o *types.RunOptions) (*runOpts, error) {
	formatter, err := format.GetFormatter(o.Output)
	if err != nil {
		return nil, err
	}
	return &runOpts{
		formatter: formatter,
		compare:   o.CompareOpts,
		filter:    o.FilterOpts,
		format:    o.FormatOpts,
	}, nil
}

// LoadOrUse returns preloaded content if non-nil, otherwise loads from path.
func LoadOrUse(preloaded []byte, path string) ([]byte, error) {
	if preloaded != nil {
		return preloaded, nil
	}
	return LoadContent(path)
}

// CompareAndFilter runs Compare followed by FilterDiffsWithRegexp.
func CompareAndFilter(from, to []byte, compareOpts *types.Options, filterOpts *types.FilterOptions) ([]types.Difference, error) {
	diffs, err := compare.Compare(from, to, compareOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to compare files: %w", err)
	}
	diffs, err = compare.FilterDiffsWithRegexp(diffs, filterOpts)
	if err != nil {
		return nil, fmt.Errorf("filter error: %w", err)
	}
	return diffs, nil
}

// InvokeSummary runs the AI summarizer and returns the formatted summary string.
// Callers must check Summary flag and non-empty diffs before calling.
func InvokeSummary(model string, rc *types.RunConfig, groups []types.DiffGroup, formatOpts *types.FormatOptions) (string, error) {
	summarizer := NewSummarizer(model)
	if rc.SummaryAPIURL != "" {
		summarizer.apiURL = rc.SummaryAPIURL
	}
	summary, err := summarizer.Summarize(context.Background(), groups)
	if err != nil {
		return "", err
	}
	return formatSummaryOutput(summary, formatOpts), nil
}

// NormalizeFilePath converts a file path to a clean relative path.
// Strips "./" prefix, converts absolute paths to relative from CWD.
// Falls back to the original path if relative conversion fails or
// produces a parent-traversing path (".."). Emits a warning to stderr
// if the fallback results in an absolute path.
func NormalizeFilePath(path string, stderr io.Writer) string {
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
