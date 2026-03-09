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

// processDirPair processes a single file pair in directory mode.
// Returns the diffs and an error if processing failed.
func processDirPair(pair diffyml.FilePair, filePairs map[string][2][]byte, compareOpts *diffyml.Options, filterOpts *diffyml.FilterOptions) ([]diffyml.Difference, error) {
	fromContent, toContent, err := loadFilePairContent(pair, filePairs)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", pair.Name, err)
	}
	diffs, err := compareAndFilterPair(fromContent, toContent, compareOpts, filterOpts)
	if err != nil {
		return nil, fmt.Errorf("comparing %s: %w", pair.Name, err)
	}
	return diffs, nil
}

// setupDirFormatting creates the formatter and format options for directory mode.
func setupDirFormatting(cfg *CLIConfig) (diffyml.Formatter, *diffyml.FormatOptions, error) {
	formatter, err := diffyml.FormatterByName(cfg.Output)
	if err != nil {
		return nil, nil, err
	}

	formatOpts := cfg.ToFormatOptions()
	colorMode, _ := diffyml.ParseColorMode(cfg.Color)
	trueColorMode, _ := diffyml.ParseColorMode(cfg.TrueColor)
	colorCfg := diffyml.NewColorConfig(colorMode, trueColorMode == diffyml.ColorModeAlways)
	colorCfg.DetectTerminal()
	colorCfg.ToFormatOptions(formatOpts)

	return formatter, formatOpts, nil
}

// buildDirFilePairs builds the file pair plan for directory comparison.
func buildDirFilePairs(rc *RunConfig, fromDir, toDir string) ([]diffyml.FilePair, error) {
	if rc.FilePairs != nil {
		return buildFilePairsFromMap(rc.FilePairs), nil
	}
	return diffyml.BuildFilePairPlan(fromDir, toDir)
}

// dirPairCollector accumulates diff results during directory-mode iteration.
type dirPairCollector struct {
	// Loop-invariant context
	rc             *RunConfig
	formatter      diffyml.Formatter
	formatOpts     *diffyml.FormatOptions
	isStructured   bool
	isBriefSummary bool
	wantSummary    bool

	// Accumulated state
	groups         []diffyml.DiffGroup
	summaryEntries []summaryEntry
	hasDiffs       bool
	hasErrors      bool
}

// collectPairResult records the diff results for a single file pair, emitting output as needed.
func (c *dirPairCollector) collectPairResult(pair diffyml.FilePair, diffs []diffyml.Difference) {
	c.hasDiffs = true

	if c.isStructured {
		filePath := strings.TrimPrefix(pair.Name, "./")
		c.groups = append(c.groups, diffyml.DiffGroup{
			FilePath: filePath,
			Diffs:    diffs,
		})
		return
	}

	if c.wantSummary {
		c.summaryEntries = append(c.summaryEntries, summaryEntry{
			Group:    diffyml.DiffGroup{FilePath: pair.Name, Diffs: diffs},
			PairType: pair.Type,
		})
	}
	if !c.isBriefSummary {
		fmt.Fprint(c.rc.Stdout, diffyml.FormatFileHeader(pair.Name, pair.Type, c.formatOpts))
		fmt.Fprint(c.rc.Stdout, c.formatter.Format(diffs, c.formatOpts))
	}
}

// runDirectory executes directory-mode comparison.
// Unexported; called from Run() when both arguments are directories.
func runDirectory(cfg *CLIConfig, rc *RunConfig, fromDir, toDir string) *ExitResult {
	if cfg.Swap {
		fromDir, toDir = toDir, fromDir
	}

	pairs, err := buildDirFilePairs(rc, fromDir, toDir)
	if err != nil {
		fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
		return NewExitResult(ExitCodeError, err)
	}

	formatter, formatOpts, err := setupDirFormatting(cfg)
	if err != nil {
		fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
		return NewExitResult(ExitCodeError, err)
	}

	compareOpts := cfg.ToCompareOptions()
	if cfg.Swap {
		compareOpts.Swap = false
	}
	filterOpts := cfg.ToFilterOptions()

	sf, isStructured := formatter.(diffyml.StructuredFormatter)

	c := dirPairCollector{
		rc:             rc,
		formatter:      formatter,
		formatOpts:     formatOpts,
		isStructured:   isStructured,
		isBriefSummary: cfg.Output == "brief" && cfg.Summary,
		wantSummary:    cfg.Summary,
	}
	for _, pair := range pairs {
		diffs, diffErr := processDirPair(pair, rc.FilePairs, compareOpts, filterOpts)
		if diffErr != nil {
			fmt.Fprintf(rc.Stderr, "Error: %v\n", diffErr)
			c.hasErrors = true
			continue
		}
		if len(diffs) > 0 {
			c.collectPairResult(pair, diffs)
		}
	}

	if isStructured {
		fmt.Fprint(rc.Stdout, sf.FormatAll(c.groups, formatOpts))
		c.hasDiffs = len(c.groups) > 0
	}

	if cfg.Summary {
		emitDirectorySummary(cfg, rc, c.groups, c.summaryEntries, formatOpts, formatter, isStructured, c.isBriefSummary)
	}

	if cfg.SetExitCode && c.hasDiffs {
		return NewExitResult(ExitCodeDifferences, nil)
	}
	if c.hasErrors {
		return NewExitResult(ExitCodeError, nil)
	}
	return NewExitResult(ExitCodeSuccess, nil)
}
