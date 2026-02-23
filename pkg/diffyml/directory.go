// directory.go - Directory-level comparison support for kubectl compatibility.
//
// Enables diffyml to serve as KUBECTL_EXTERNAL_DIFF provider by accepting
// two directory paths and comparing YAML files within them.
package diffyml

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// IsDirectory reports whether path is an existing directory.
// Returns false for files, URLs, non-existent paths, or errors.
func IsDirectory(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DiscoverYAMLFiles returns sorted filenames of .yaml/.yml files
// in the given directory (non-recursive).
// Returns base names only (not full paths), sorted alphabetically.
// Skips subdirectories, symlinks, and non-YAML files silently.
func DiscoverYAMLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, entry.Name())
		}
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
// Every YAML file from both directories appears exactly once.
// Returns an error if either directory cannot be read.
func BuildFilePairPlan(fromDir, toDir string) ([]FilePair, error) {
	fromFiles, err := DiscoverYAMLFiles(fromDir)
	if err != nil {
		return nil, err
	}
	toFiles, err := DiscoverYAMLFiles(toDir)
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

	// Compute union of filenames
	nameSet := make(map[string]bool, len(fromFiles)+len(toFiles))
	for _, f := range fromFiles {
		nameSet[f] = true
	}
	for _, f := range toFiles {
		nameSet[f] = true
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
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
		switch {
		case contents[0] != nil && contents[1] != nil:
			pair.Type = FilePairBothExist
		case contents[0] != nil:
			pair.Type = FilePairOnlyFrom
		default:
			pair.Type = FilePairOnlyTo
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
	var pairs []FilePair
	if rc.FilePairs != nil {
		// Testing mode: use in-memory file pairs
		pairs = buildFilePairsFromMap(rc.FilePairs)
	} else {
		var err error
		pairs, err = BuildFilePairPlan(fromDir, toDir)
		if err != nil {
			fmt.Fprintf(rc.Stderr, "Error: %v\n", err)
			return NewExitResult(ExitCodeError, err)
		}
	}

	// Get the formatter
	formatter, err := GetFormatter(cfg.Output)
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
	colorMode, _ := ParseColorMode(cfg.Color)
	trueColorMode, _ := ParseColorMode(cfg.TrueColor)
	colorCfg := NewColorConfig(colorMode, trueColorMode == ColorModeOn, cfg.FixedWidth)
	colorCfg.DetectTerminal()
	colorCfg.ToFormatOptions(formatOpts)

	hasDiffs := false
	hasErrors := false

	for _, pair := range pairs {
		var fromContent, toContent []byte

		if rc.FilePairs != nil {
			// Testing mode: read from in-memory map
			contents := rc.FilePairs[pair.Name]
			if contents[0] != nil {
				fromContent = contents[0]
			} else {
				fromContent = []byte{}
			}
			if contents[1] != nil {
				toContent = contents[1]
			} else {
				toContent = []byte{}
			}
		} else {
			// Real mode: load content from filesystem
			switch pair.Type {
			case FilePairBothExist:
				fromContent, err = LoadContent(pair.FromPath)
				if err != nil {
					fmt.Fprintf(rc.Stderr, "Error reading %s: %v\n", pair.Name, err)
					hasErrors = true
					continue
				}
				toContent, err = LoadContent(pair.ToPath)
				if err != nil {
					fmt.Fprintf(rc.Stderr, "Error reading %s: %v\n", pair.Name, err)
					hasErrors = true
					continue
				}
			case FilePairOnlyFrom:
				fromContent, err = LoadContent(pair.FromPath)
				if err != nil {
					fmt.Fprintf(rc.Stderr, "Error reading %s: %v\n", pair.Name, err)
					hasErrors = true
					continue
				}
				toContent = []byte{}
			case FilePairOnlyTo:
				fromContent = []byte{}
				toContent, err = LoadContent(pair.ToPath)
				if err != nil {
					fmt.Fprintf(rc.Stderr, "Error reading %s: %v\n", pair.Name, err)
					hasErrors = true
					continue
				}
			}
		}

		// Compare
		diffs, err := Compare(fromContent, toContent, compareOpts)
		if err != nil {
			fmt.Fprintf(rc.Stderr, "Error comparing %s: %v\n", pair.Name, err)
			hasErrors = true
			continue
		}

		// Filter
		diffs, err = FilterDiffsWithRegexp(diffs, filterOpts)
		if err != nil {
			fmt.Fprintf(rc.Stderr, "Error filtering %s: %v\n", pair.Name, err)
			hasErrors = true
			continue
		}

		if len(diffs) == 0 {
			continue
		}

		hasDiffs = true

		// Format file header (always shown, even with --omit-header)
		header := FormatFileHeader(pair.Name, pair.Type, formatOpts)
		fmt.Fprint(rc.Stdout, header)

		// Format and output diffs
		output := formatter.Format(diffs, formatOpts)
		fmt.Fprint(rc.Stdout, output)
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
