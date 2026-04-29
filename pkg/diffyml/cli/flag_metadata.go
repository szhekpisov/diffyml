// flag_metadata.go - Documentation metadata for CLI flags.
// Maintained alongside initFlags(); TestFlagDocsCoverage enforces parity.
package cli

// FlagDoc describes a single CLI flag for documentation generators.
type FlagDoc struct {
	// Long is the long flag name (without leading dashes), e.g. "output".
	Long string
	// Short is the optional short flag name (without dash), e.g. "o". Empty if none.
	Short string
	// Type is one of: bool, string, int, list (repeatable string).
	Type string
	// Default is the default value as a string. Empty for no default.
	Default string
	// Usage is the human-readable description.
	Usage string
	// Category groups flags in the rendered reference (e.g. "Output", "Comparison").
	Category string
}

// FlagDocs returns documentation metadata for every CLI flag registered in initFlags().
// Order matches the layout of CLIConfig.Usage() so the generated reference reads naturally.
func FlagDocs() []FlagDoc {
	return []FlagDoc{
		// Output
		{Long: "output", Short: "o", Type: "string", Default: "detailed", Category: "Output", Usage: "specify output style: compact, brief, github, gitlab, gitea, json, json-patch, detailed"},
		{Long: "color", Short: "c", Type: "string", Default: "auto", Category: "Output", Usage: "specify color usage: always, never, or auto"},
		{Long: "truecolor", Short: "t", Type: "string", Default: "auto", Category: "Output", Usage: "specify true color usage: always, never, or auto"},

		// Comparison
		{Long: "ignore-order-changes", Short: "i", Type: "bool", Category: "Comparison", Usage: "ignore order changes in lists"},
		{Long: "ignore-whitespace-changes", Type: "bool", Category: "Comparison", Usage: "ignore leading or trailing whitespace changes"},
		{Long: "format-strings", Type: "bool", Category: "Comparison", Usage: "canonicalize embedded JSON strings before comparison"},
		{Long: "ignore-value-changes", Short: "v", Type: "bool", Category: "Comparison", Usage: "exclude changes in values"},
		{Long: "detect-kubernetes", Type: "bool", Default: "true", Category: "Comparison", Usage: "detect kubernetes entities"},
		{Long: "detect-renames", Type: "bool", Default: "true", Category: "Comparison", Usage: "enable detection for renames"},
		{Long: "ignore-api-version", Type: "bool", Category: "Comparison", Usage: "ignore apiVersion when matching Kubernetes resources"},
		{Long: "no-cert-inspection", Short: "x", Type: "bool", Category: "Comparison", Usage: "disable x509 certificate inspection"},
		{Long: "swap", Type: "bool", Category: "Comparison", Usage: "swap 'from' and 'to' for comparison"},

		// Filtering
		{Long: "filter", Type: "list", Category: "Filtering", Usage: "filter reports to a subset of differences (repeatable)"},
		{Long: "exclude", Type: "list", Category: "Filtering", Usage: "exclude reports from a set of differences (repeatable)"},
		{Long: "filter-regexp", Type: "list", Category: "Filtering", Usage: "filter reports using regular expressions (repeatable)"},
		{Long: "exclude-regexp", Type: "list", Category: "Filtering", Usage: "exclude reports using regular expressions (repeatable)"},
		{Long: "additional-identifier", Type: "list", Category: "Filtering", Usage: "use additional identifier in named entry lists (repeatable)"},

		// Sensitive value masking
		{Long: "mask-secrets", Type: "bool", Category: "Masking", Usage: "auto-mask data/stringData of Kubernetes Secret resources"},
		{Long: "mask-path", Type: "list", Category: "Masking", Usage: "additional path to mask (dot-notation, prefix match; repeatable)"},
		{Long: "mask-path-regexp", Type: "list", Category: "Masking", Usage: "additional path to mask (regex; repeatable)"},
		{Long: "mask-placeholder", Type: "string", Default: "***", Category: "Masking", Usage: "placeholder for masked values"},

		// Display
		{Long: "omit-header", Short: "b", Type: "bool", Category: "Display", Usage: "omit the diffyml summary header"},
		{Long: "use-go-patch-style", Short: "g", Type: "bool", Category: "Display", Usage: "use Go-Patch style paths in outputs"},
		{Long: "multi-line-context-lines", Type: "int", Default: "4", Category: "Display", Usage: "context lines for multi-line strings"},

		// Chroot
		{Long: "chroot", Type: "string", Category: "Chroot", Usage: "change the root level of the input file"},
		{Long: "chroot-of-from", Type: "string", Category: "Chroot", Usage: "only change the root level of the from input file"},
		{Long: "chroot-of-to", Type: "string", Category: "Chroot", Usage: "only change the root level of the to input file"},
		{Long: "chroot-list-to-documents", Type: "bool", Category: "Chroot", Usage: "treat chroot list as set of documents"},

		// AI Summary
		{Long: "summary", Short: "S", Type: "bool", Category: "AI Summary", Usage: "enable AI-powered summary of differences (requires ANTHROPIC_API_KEY)"},
		{Long: "summary-model", Type: "string", Default: "claude-haiku-4-5-20251001", Category: "AI Summary", Usage: "specify Anthropic model for summary"},

		// Configuration
		{Long: "config", Type: "string", Default: ".diffyml.yml", Category: "Configuration", Usage: "path to config file"},

		// Other
		{Long: "set-exit-code", Short: "s", Type: "bool", Category: "Other", Usage: "set program exit code based on differences"},
		{Long: "help", Short: "h", Type: "bool", Category: "Other", Usage: "show help"},
	}
}
