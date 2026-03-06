// runner.go - Execution orchestration for the comparison flow.
//
// Key types: RunConfig (runtime IO), ExitResult.
// Key functions: Run() executes the full comparison flow.
// Exit codes: 0=success, 1=differences (with -s), 255=error.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/run"
)

// Exit code constants for program termination.
const (
	// ExitCodeSuccess indicates successful execution with no differences.
	ExitCodeSuccess = run.ExitCodeSuccess
	// ExitCodeDifferences indicates differences were found (with -s flag).
	ExitCodeDifferences = run.ExitCodeDifferences
	// ExitCodeError indicates a program error occurred.
	ExitCodeError = run.ExitCodeError
)

// DetermineExitCode returns the appropriate exit code based on execution results.
func DetermineExitCode(setExitCode bool, diffCount int, err error) int {
	return run.DetermineExitCode(setExitCode, diffCount, err)
}

// ExitResult encapsulates the result of program execution.
type ExitResult = run.ExitResult

// RunConfig holds runtime configuration for the Run function.
type RunConfig = run.RunConfig

// RunOptions holds the library-level options for running a comparison.
type RunOptions = run.RunOptions

// NewRunConfig creates a new RunConfig with default values.
func NewRunConfig() *RunConfig { return run.NewRunConfig() }

// Run executes the main comparison flow with the given configuration.
func Run(cfg *CLIConfig, rc *RunConfig) *ExitResult { return run.Run(cfg, rc) }

// exitError logs an error to stderr and returns an ExitResult with ExitCodeError.
// Unexported; required by tests in package diffyml.
func exitError(rc *RunConfig, err error) *ExitResult { return run.ExitError(rc, err) }
