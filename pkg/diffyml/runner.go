// runner.go - Execution orchestration for the comparison flow.
//
// Key types: RunConfig (runtime IO), ExitResult.
// Key functions: Run() executes the full comparison flow.
// Exit codes: 0=success, 1=differences (with -s), 255=error.
package diffyml

import (
	"io"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/run"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// Exit code constants for program termination.
const (
	// ExitCodeSuccess indicates successful execution with no differences.
	ExitCodeSuccess = types.ExitCodeSuccess
	// ExitCodeDifferences indicates differences were found (with -s flag).
	ExitCodeDifferences = types.ExitCodeDifferences
	// ExitCodeError indicates a program error occurred.
	ExitCodeError = types.ExitCodeError
)

// DetermineExitCode returns the appropriate exit code based on execution results.
func DetermineExitCode(setExitCode bool, diffCount int, err error) int {
	return types.DetermineExitCode(setExitCode, diffCount, err)
}

// ExitResult encapsulates the result of program execution.
type ExitResult = types.ExitResult

// RunConfig holds runtime configuration for the Run function.
type RunConfig = types.RunConfig

// RunOptions holds the library-level options for running a comparison.
type RunOptions = types.RunOptions

// NewRunConfig creates a new RunConfig with default values.
func NewRunConfig() *RunConfig { return types.NewRunConfig() }

// Run executes the main comparison flow with the given configuration.
func Run(cfg *CLIConfig, rc *RunConfig) *ExitResult { return run.Run(cfg, rc) }

// exitError logs an error to stderr and returns an ExitResult with ExitCodeError.
// Unexported; required by tests in package diffyml.
func exitError(rc *RunConfig, err error) *ExitResult { return types.ExitError(rc, err) }

// normalizeFilePath converts a file path to a clean relative path.
// Unexported; used internally and delegated to the run package.
func normalizeFilePath(path string, stderr io.Writer) string {
	return run.NormalizeFilePath(path, stderr)
}
