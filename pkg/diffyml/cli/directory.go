// directory.go - Directory-mode comparison orchestration.
//
// Contains the runDirectory function and its helpers for comparing
// all YAML files across two directories.
package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

// summaryEntry pairs a DiffGroup with its file pair type for non-structured summary output.
type summaryEntry struct {
	Group    diffyml.DiffGroup
	PairType diffyml.FilePairType
}

// loadFilePairContent loads the from/to content for a single file pair.
// In testing mode (filePairs != nil), reads from the in-memory map.
// In real mode, reads from the filesystem based on pair type.
func loadFilePairContent(pair diffyml.FilePair, filePairs map[string][2][]byte) ([]byte, []byte, error) {
	if filePairs != nil {
		contents := filePairs[pair.Name]
		from := contents[0]
		if from == nil {
			from = []byte{}
		}
		to := contents[1]
		if to == nil {
			to = []byte{}
		}
		return from, to, nil
	}

	var from, to []byte
	var err error
	switch pair.Type {
	case diffyml.FilePairBothExist:
		from, err = diffyml.LoadContent(pair.FromPath)
		if err != nil {
			return nil, nil, err
		}
		to, err = diffyml.LoadContent(pair.ToPath)
		if err != nil {
			return nil, nil, err
		}
	case diffyml.FilePairOnlyFrom:
		from, err = diffyml.LoadContent(pair.FromPath)
		if err != nil {
			return nil, nil, err
		}
		to = []byte{}
	case diffyml.FilePairOnlyTo:
		from = []byte{}
		to, err = diffyml.LoadContent(pair.ToPath)
		if err != nil {
			return nil, nil, err
		}
	}
	return from, to, nil
}

// compareAndFilterPair compares two YAML contents and filters the results.
func compareAndFilterPair(from, to []byte, compareOpts *diffyml.Options, filterOpts *diffyml.FilterOptions) ([]diffyml.Difference, error) {
	diffs, err := diffyml.Compare(from, to, compareOpts)
	if err != nil {
		return nil, err
	}
	return diffyml.FilterDiffsWithRegexp(diffs, filterOpts)
}

// emitDirectorySummary generates and emits AI summaries for directory mode.
// Handles both structured (aggregated) and non-structured (per-file) formatters.
func emitDirectorySummary(cfg *CLIConfig, rc *RunConfig, groups []diffyml.DiffGroup, entries []summaryEntry,
	formatOpts *diffyml.FormatOptions, formatter diffyml.Formatter, isStructured, isBriefSummary bool) {
	if isStructured && len(groups) > 0 {
		summarizer := NewSummarizer(cfg.SummaryModel)
		if rc.SummaryAPIURL != "" {
			summarizer.apiURL = rc.SummaryAPIURL
		}
		summary, err := summarizer.Summarize(context.Background(), groups)
		if err != nil {
			fmt.Fprintf(rc.Stderr, "Warning: AI summary unavailable: %v\n", err)
		} else {
			fmt.Fprint(rc.Stdout, diffyml.FormatSummaryOutput(summary, formatOpts))
		}
		return
	}

	if !isStructured && len(entries) > 0 {
		summaryGroups := make([]diffyml.DiffGroup, len(entries))
		for i, e := range entries {
			summaryGroups[i] = e.Group
		}
		summarizer := NewSummarizer(cfg.SummaryModel)
		if rc.SummaryAPIURL != "" {
			summarizer.apiURL = rc.SummaryAPIURL
		}
		summary, err := summarizer.Summarize(context.Background(), summaryGroups)
		if err != nil {
			if isBriefSummary {
				for _, e := range entries {
					fmt.Fprint(rc.Stdout, diffyml.FormatFileHeader(e.Group.FilePath, e.PairType, formatOpts))
					fmt.Fprint(rc.Stdout, formatter.Format(e.Group.Diffs, formatOpts))
				}
			}
			fmt.Fprintf(rc.Stderr, "Warning: AI summary unavailable: %v\n", err)
		} else {
			fmt.Fprint(rc.Stdout, diffyml.FormatSummaryOutput(summary, formatOpts))
		}
	}
}

// buildFilePairsFromMap builds a sorted slice of FilePair from an in-memory map
// (used for testing).
func buildFilePairsFromMap(m map[string][2][]byte) []diffyml.FilePair {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	pairs := make([]diffyml.FilePair, 0, len(names))
	for _, name := range names {
		contents := m[name]
		var pair diffyml.FilePair
		pair.Name = name
		//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
		if contents[0] != nil && contents[1] != nil {
			pair.Type = diffyml.FilePairBothExist
		} else if contents[0] != nil {
			pair.Type = diffyml.FilePairOnlyFrom
		} else {
			pair.Type = diffyml.FilePairOnlyTo
		}
		pairs = append(pairs, pair)
	}
	return pairs
}

// runDirectory executes directory-mode comparison.
// Unexported; called from Run() when both arguments are directories.
func runDirectory(cfg *CLIConfig, rc *RunConfig, fromDir, toDir string) *ExitResult {
	// Handle swap: swap directories before planning to avoid double-swap
	if cfg.Swap {
		fromDir, toDir = toDir, fromDir
	}

	// Build file pair plan
	var pairs []diffyml.FilePair
	if rc.FilePairs != nil {
		// Testing mode: use in-memory file pairs
		pairs = buildFilePairsFromMap(rc.FilePairs)
	} else {
		var err error
		pairs, err = diffyml.BuildFilePairPlan(fromDir, toDir)
		if err != nil {
			fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
			return NewExitResult(ExitCodeError, err)
		}
	}

	// Get the formatter
	formatter, err := diffyml.FormatterByName(cfg.Output)
	if err != nil {
		fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
		return NewExitResult(ExitCodeError, err)
	}

	// Build options once
	compareOpts := cfg.ToCompareOptions()
	// Disable swap in compareOpts since we already swapped dirs
	if cfg.Swap {
		compareOpts.Swap = false
	}
	filterOpts := cfg.ToFilterOptions()
	formatOpts := cfg.ToFormatOptions()

	// Apply color configuration
	colorMode, _ := diffyml.ParseColorMode(cfg.Color)
	trueColorMode, _ := diffyml.ParseColorMode(cfg.TrueColor)
	colorCfg := diffyml.NewColorConfig(colorMode, trueColorMode == diffyml.ColorModeAlways)
	colorCfg.DetectTerminal()
	colorCfg.ToFormatOptions(formatOpts)

	// Check if formatter supports structured (aggregated) output
	sf, isStructured := formatter.(diffyml.StructuredFormatter)

	isBriefSummary := cfg.Output == "brief" && cfg.Summary

	hasDiffs := false
	hasErrors := false

	// For structured formatters, collect all diff groups
	var groups []diffyml.DiffGroup

	// For non-structured summary: collect entries to invoke summarizer after all files
	var summaryEntries []summaryEntry

	for _, pair := range pairs {
		fromContent, toContent, loadErr := loadFilePairContent(pair, rc.FilePairs)
		if loadErr != nil {
			fmt.Fprintf(rc.Stderr, "Error reading %s: %v\n", pair.Name, loadErr)
			hasErrors = true
			continue
		}

		diffs, diffErr := compareAndFilterPair(fromContent, toContent, compareOpts, filterOpts)
		if diffErr != nil {
			fmt.Fprintf(rc.Stderr, "Error comparing %s: %v\n", pair.Name, diffErr)
			hasErrors = true
			continue
		}

		if len(diffs) == 0 {
			continue
		}

		hasDiffs = true

		if isStructured {
			filePath := strings.TrimPrefix(pair.Name, "./")
			groups = append(groups, diffyml.DiffGroup{
				FilePath: filePath,
				Diffs:    diffs,
			})
		} else {
			if cfg.Summary {
				summaryEntries = append(summaryEntries, summaryEntry{
					Group:    diffyml.DiffGroup{FilePath: pair.Name, Diffs: diffs},
					PairType: pair.Type,
				})
			}

			if !isBriefSummary {
				fmt.Fprint(rc.Stdout, diffyml.FormatFileHeader(pair.Name, pair.Type, formatOpts))
				fmt.Fprint(rc.Stdout, formatter.Format(diffs, formatOpts))
			}
		}
	}

	// For structured formatters, always write output (even when empty)
	if isStructured {
		output := sf.FormatAll(groups, formatOpts)
		fmt.Fprint(rc.Stdout, output)
		hasDiffs = len(groups) > 0
	}

	// AI Summary
	if cfg.Summary {
		emitDirectorySummary(cfg, rc, groups, summaryEntries, formatOpts, formatter, isStructured, isBriefSummary)
	}

	// Compute aggregated exit code (directory-specific precedence)
	if cfg.SetExitCode && hasDiffs {
		return NewExitResult(ExitCodeDifferences, nil)
	}
	if hasErrors {
		return NewExitResult(ExitCodeError, nil)
	}
	return NewExitResult(ExitCodeSuccess, nil)
}
