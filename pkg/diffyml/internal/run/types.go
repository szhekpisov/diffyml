// types.go - Runner-specific types for execution orchestration.
//
// RunConfig, RunOptions, ExitResult, and exit code constants.
package run

import (
	"fmt"
	"io"
	"os"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Exit code constants for program termination.
const (
	ExitCodeSuccess     = 0
	ExitCodeDifferences = 1
	ExitCodeError       = 255
)

// DetermineExitCode returns the appropriate exit code.
func DetermineExitCode(setExitCode bool, diffCount int, err error) int {
	if err != nil {
		return ExitCodeError
	}
	if !setExitCode {
		return ExitCodeSuccess
	}
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

// ExitError logs an error to stderr and returns an ExitResult with ExitCodeError.
func ExitError(rc *RunConfig, err error) *ExitResult {
	fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
	return &ExitResult{ExitCodeError, err}
}

// RunConfig holds runtime configuration for the Run function.
type RunConfig struct {
	Stdout        io.Writer
	Stderr        io.Writer
	FromContent   []byte
	ToContent     []byte
	FilePairs     map[string][2][]byte
	SummaryAPIURL string
}

// IsRealMode reports whether the RunConfig has no pre-loaded test content.
func (rc *RunConfig) IsRealMode() bool {
	return rc.FromContent == nil && rc.ToContent == nil && rc.FilePairs == nil
}

// NewRunConfig creates a new RunConfig with default values.
func NewRunConfig() *RunConfig {
	return &RunConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// RunOptions holds the library-level options for running a comparison.
type RunOptions struct {
	FromFile     string
	ToFile       string
	SetExitCode  bool
	Swap         bool
	Output       string
	Summary      bool
	SummaryModel string
	CompareOpts  *types.Options
	FilterOpts   *types.FilterOptions
	FormatOpts   *types.FormatOptions
}

// IsBriefSummary reports whether the options request brief output with AI summary.
func (o *RunOptions) IsBriefSummary() bool {
	return o.Output == "brief" && o.Summary
}
