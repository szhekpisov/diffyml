// directory.go - Directory-level comparison support for kubectl compatibility.
//
// Enables diffyml to serve as KUBECTL_EXTERNAL_DIFF provider by accepting
// two directory paths and comparing YAML files within them.
package diffyml

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

// IsDirectory reports whether path is an existing directory.
// Returns false for files, URLs, non-existent paths, or errors.
func IsDirectory(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(filepath.Clean(path)) //nolint:gosec // CLI tool intentionally accepts user-supplied paths
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DiscoverFiles returns sorted filenames of all regular files
// in the given directory (non-recursive).
// Returns base names only (not full paths), sorted alphabetically.
// Skips subdirectories and symlinks silently.
// All regular files are included regardless of extension, so that
// kubectl temp files (e.g. "apps.v1.Deployment.default.nginx") are
// discovered when diffyml is used as KUBECTL_EXTERNAL_DIFF.
func DiscoverFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		files = append(files, entry.Name())
	}

	sort.Strings(files)
	return files, nil
}

// FilePairType describes the relationship between source and target files.
type FilePairType int

const (
	// FilePairBothExist means the file exists in both directories.
	FilePairBothExist FilePairType = iota
	// FilePairOnlyFrom means the file exists only in the source directory.
	FilePairOnlyFrom
	// FilePairOnlyTo means the file exists only in the target directory.
	FilePairOnlyTo
)

// FilePair represents a matched pair of files for comparison.
type FilePair struct {
	Name     string       // Base filename (e.g., "deployment.yaml")
	Type     FilePairType // Relationship between from and to
	FromPath string       // Full path in from-directory (empty if OnlyTo)
	ToPath   string       // Full path in to-directory (empty if OnlyFrom)
}

// BuildFilePairPlan creates an alphabetically sorted plan of file
// pairs from two directories, matching files by filename.
// Every file from both directories appears exactly once.
// Returns an error if either directory cannot be read.
func BuildFilePairPlan(fromDir, toDir string) ([]FilePair, error) {
	fromFiles, err := DiscoverFiles(fromDir)
	if err != nil {
		return nil, err
	}
	toFiles, err := DiscoverFiles(toDir)
	if err != nil {
		return nil, err
	}

	fromSet := make(map[string]bool, len(fromFiles))
	for _, f := range fromFiles {
		fromSet[f] = true
	}
	toSet := make(map[string]bool, len(toFiles))
	for _, f := range toFiles {
		toSet[f] = true
	}

	// Build sorted union of filenames from both sets
	names := make([]string, 0, len(fromFiles)+len(toFiles))
	for name := range fromSet {
		names = append(names, name)
	}
	for name := range toSet {
		if !fromSet[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	pairs := make([]FilePair, 0, len(names))
	for _, name := range names {
		inFrom := fromSet[name]
		inTo := toSet[name]

		var pair FilePair
		pair.Name = name
		switch {
		case inFrom && inTo:
			pair.Type = FilePairBothExist
			pair.FromPath = filepath.Join(fromDir, name)
			pair.ToPath = filepath.Join(toDir, name)
		case inFrom:
			pair.Type = FilePairOnlyFrom
			pair.FromPath = filepath.Join(fromDir, name)
		default:
			pair.Type = FilePairOnlyTo
			pair.ToPath = filepath.Join(toDir, name)
		}
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// FormatFileHeader returns a unified-diff-style file header for directory mode.
// Uses "--- a/<filename>" / "+++ b/<filename>" for BothExist,
// "/dev/null" for the absent side on OnlyFrom/OnlyTo.
// Applies yellow/bold color when opts.Color is true.
func FormatFileHeader(filename string, pairType FilePairType, opts *FormatOptions) string {
	var fromLine, toLine string

	switch pairType {
	case FilePairBothExist:
		fromLine = "--- a/" + filename
		toLine = "+++ b/" + filename
	case FilePairOnlyFrom:
		fromLine = "--- a/" + filename
		toLine = "+++ /dev/null"
	case FilePairOnlyTo:
		fromLine = "--- /dev/null"
		toLine = "+++ b/" + filename
	}

	if opts != nil && opts.Color {
		return fmt.Sprintf("%s%s%s\n%s%s%s\n",
			styleBold+colorWhite, fromLine, colorReset,
			styleBold+colorWhite, toLine, colorReset)
	}
	return fmt.Sprintf("%s\n%s\n", fromLine, toLine)
}

// buildFilePairsFromMap builds a sorted slice of FilePair from an in-memory map
// (used for testing).
func buildFilePairsFromMap(m map[string][2][]byte) []FilePair {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	pairs := make([]FilePair, 0, len(names))
	for _, name := range names {
		contents := m[name]
		var pair FilePair
		pair.Name = name
		//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
		if contents[0] != nil && contents[1] != nil {
			pair.Type = FilePairBothExist
		} else if contents[0] != nil {
			pair.Type = FilePairOnlyFrom
		} else {
			pair.Type = FilePairOnlyTo
		}
		pairs = append(pairs, pair)
	}
	return pairs
}

// loadPairContent loads the from/to content for a file pair.
// In testing mode (rc.FilePairs != nil), reads from the in-memory map.
// In real mode, loads from the filesystem based on the pair type.
func loadPairContent(pair FilePair, rc *RunConfig) (from, to []byte, err error) {
	if rc.FilePairs != nil {
		contents := rc.FilePairs[pair.Name]
		if contents[0] != nil {
			from = contents[0]
		} else {
			from = []byte{}
		}
		if contents[1] != nil {
			to = contents[1]
		} else {
			to = []byte{}
		}
		return from, to, nil
	}

	switch pair.Type {
	case FilePairBothExist:
		from, err = LoadContent(pair.FromPath)
		if err != nil {
			return nil, nil, err
		}
		to, err = LoadContent(pair.ToPath)
		if err != nil {
			return nil, nil, err
		}
	case FilePairOnlyFrom:
		from, err = LoadContent(pair.FromPath)
		if err != nil {
			return nil, nil, err
		}
		to = []byte{}
	case FilePairOnlyTo:
		from = []byte{}
		to, err = LoadContent(pair.ToPath)
		if err != nil {
			return nil, nil, err
		}
	}
	return from, to, nil
}

// pairResult holds the per-file result of loading and comparing a file pair.
type pairResult struct {
	pair     FilePair
	from, to []byte
	diffs    []Difference
	loadErr  error
	cmpErr   error
}

// runDirectory executes directory-mode comparison using a 3-phase pipeline:
// sequential load, parallel compare, sequential format.
// Unexported; called from Run() when both arguments are directories.
func runDirectory(runOpts *RunOptions, rc *RunConfig, fromDir, toDir string) *ExitResult {
	// Handle swap: swap directories before planning to avoid double-swap
	if runOpts.Swap {
		fromDir, toDir = toDir, fromDir
	}

	// Build file pair plan
	var pairs []FilePair
	if rc.FilePairs != nil {
		// Testing mode: use in-memory file pairs
		pairs = buildFilePairsFromMap(rc.FilePairs)
	} else {
		var err error
		pairs, err = BuildFilePairPlan(fromDir, toDir)
		if err != nil {
			return exitError(rc, err)
		}
	}

	// Build shared options
	ro, err := runOpts.buildRunOpts()
	if err != nil {
		return exitError(rc, err)
	}
	// Disable swap in compareOpts since we already swapped dirs
	if runOpts.Swap {
		ro.compare.Swap = false
	}

	// Phase 1: Load content sequentially (preserves disk order)
	results := make([]pairResult, len(pairs))
	for i, pair := range pairs {
		results[i].pair = pair
		from, to, err := loadPairContent(pair, rc)
		if err != nil {
			results[i].loadErr = err
			continue
		}
		results[i].from = from
		results[i].to = to
	}

	// Phase 2: Compare in parallel (stateless, safe for concurrent use)
	if len(pairs) > 1 {
		sem := make(chan struct{}, runtime.NumCPU())
		var wg sync.WaitGroup
		for i := range results {
			if results[i].loadErr != nil {
				continue
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int) {
				defer wg.Done()
				defer func() { <-sem }()
				diffs, err := compareAndFilter(results[idx].from, results[idx].to, ro.compare, ro.filter)
				results[idx].diffs = diffs
				results[idx].cmpErr = err
			}(i)
		}
		wg.Wait()
	} else {
		for i := range results {
			if results[i].loadErr != nil {
				continue
			}
			diffs, err := compareAndFilter(results[i].from, results[i].to, ro.compare, ro.filter)
			results[i].diffs = diffs
			results[i].cmpErr = err
		}
	}

	// Phase 3: Format sequentially (preserves output order)
	sf, isStructured := ro.formatter.(StructuredFormatter)

	hasDiffs := false
	hasErrors := false
	var groups []DiffGroup
	var pairTypes []FilePairType

	for i := range results {
		r := &results[i]
		if r.loadErr != nil {
			fmt.Fprintf(rc.Stderr, "Error reading %s: %v\n", r.pair.Name, r.loadErr)
			hasErrors = true
			continue
		}
		if r.cmpErr != nil {
			fmt.Fprintf(rc.Stderr, "Error processing %s: %v\n", r.pair.Name, r.cmpErr)
			hasErrors = true
			continue
		}

		if len(r.diffs) == 0 {
			continue
		}

		hasDiffs = true
		filePath := normalizeFilePath(r.pair.Name, nil)
		groups = append(groups, DiffGroup{FilePath: filePath, Diffs: r.diffs})
		pairTypes = append(pairTypes, r.pair.Type)

		// Non-structured: per-file header + format (skip for brief+summary)
		if !isStructured && !runOpts.isBriefSummary() {
			fmt.Fprint(rc.Stdout, FormatFileHeader(filePath, r.pair.Type, ro.format))
			fmt.Fprint(rc.Stdout, ro.formatter.Format(r.diffs, ro.format))
		}
	}

	// For structured formatters, always write output (even when empty)
	if isStructured {
		fmt.Fprint(rc.Stdout, sf.FormatAll(groups, ro.format))
	}

	// AI Summary (unified for both structured and non-structured)
	if runOpts.Summary && len(groups) > 0 {
		summaryOutput, summaryErr := invokeSummary(runOpts.SummaryModel, rc, groups, ro.format)
		if summaryErr != nil {
			if runOpts.isBriefSummary() {
				for i, g := range groups {
					fmt.Fprint(rc.Stdout, FormatFileHeader(g.FilePath, pairTypes[i], ro.format))
					fmt.Fprint(rc.Stdout, ro.formatter.Format(g.Diffs, ro.format))
				}
			}
			fmt.Fprintf(rc.Stderr, "Warning: AI summary unavailable: %v\n", summaryErr)
		} else {
			fmt.Fprint(rc.Stdout, summaryOutput)
		}
	}

	// Compute aggregated exit code (diffs take precedence over errors)
	diffCount := 0
	if hasDiffs {
		diffCount = 1
	}
	if code := DetermineExitCode(runOpts.SetExitCode, diffCount, nil); code != ExitCodeSuccess {
		return &ExitResult{code, nil}
	}
	if hasErrors {
		return &ExitResult{ExitCodeError, nil}
	}
	return &ExitResult{ExitCodeSuccess, nil}
}
