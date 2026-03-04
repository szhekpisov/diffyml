// cli.go - Command-line interface parsing and validation.
//
// Key types: CLIConfig (all CLI options).
// Key functions: ParseArgs(), Validate(), Usage().
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/run"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// ValidationError represents a CLI configuration validation error.
type ValidationError = types.ValidationError

// CLIConfig holds all command-line configuration options.
type CLIConfig = run.CLIConfig

// NewCLIConfig creates a new CLI configuration with default values.
func NewCLIConfig() *CLIConfig { return run.NewCLIConfig() }

// ValidateOutputFormat checks if the output format name is valid.
func ValidateOutputFormat(format string) error { return run.ValidateOutputFormat(format) }

// ValidateRegexPatterns validates a list of regex patterns.
func ValidateRegexPatterns(patterns []string, flagName string) error {
	return run.ValidateRegexPatterns(patterns, flagName)
}
