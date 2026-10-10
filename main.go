package main

import (
	"io"
	"os"

	"github.com/szhekpisov/diffyml/pkg/diffyml/cli"
)

// Version information - can be overridden at build time using ldflags
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// formatVersion returns the version information string
func formatVersion() string {
	return "diffyml version " + version + " (commit: " + commit + ", built: " + buildDate + ")\n"
}

// run parses args and executes the CLI, returning the process exit code.
// Split from main so it is testable without os.Exit; stdout/stderr are
// injected rather than referenced globally.
func run(args []string, stdout, stderr io.Writer) int {
	cfg := cli.NewCLIConfig()

	if err := cfg.ParseArgs(args); err != nil {
		// A bad flag no longer falls through to the help text the way the old
		// os.Args pre-scan allowed, so point at it instead of leaving the user
		// with only the error.
		_, _ = io.WriteString(stderr, "Error: "+err.Error()+"\nRun 'diffyml --help' for usage.\n")
		return cli.ExitCodeError
	}

	// Version takes precedence over help, matching the previous ordering.
	if cfg.ShowVersion {
		_, _ = io.WriteString(stdout, formatVersion())
		return cli.ExitCodeSuccess
	}

	// Built from NewRunConfig rather than a bare literal so any future
	// RunConfig default applies here too; only the writers are overridden.
	rc := cli.NewRunConfig()
	rc.Stdout = stdout
	rc.Stderr = stderr

	// cli.Run handles cfg.ShowHelp by writing usage to stdout.
	result := cli.Run(cfg, rc)
	return result.Code
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
