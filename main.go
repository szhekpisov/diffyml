package main

import (
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

func main() {
	cfg := cli.NewCLIConfig()

	// Check for version flag first
	for _, arg := range os.Args[1:] {
		if arg == "-V" || arg == "--version" || arg == "-version" {
			_, _ = os.Stdout.WriteString(formatVersion())
			os.Exit(0)
		}
	}

	// Check for help flag
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" || arg == "-help" {
			_, _ = os.Stdout.WriteString(cfg.Usage())
			os.Exit(0)
		}
	}

	if err := cfg.ParseArgs(os.Args[1:]); err != nil {
		_, _ = os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(cli.ExitCodeError)
	}

	result := cli.Run(cfg, nil)
	os.Exit(result.Code)
}
