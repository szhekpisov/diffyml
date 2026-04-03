// directory.go - Directory-level comparison support for kubectl compatibility.
//
// Enables diffyml to serve as KUBECTL_EXTERNAL_DIFF provider by accepting
// two directory paths and comparing YAML files within them.
package diffyml

import (
	"fmt"
	"io/fs"
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
	info, err := os.Stat(filepath.Clean(path)) //nolint:gosec // CLI tool intentionally accepts user-supplied paths
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DiscoverFiles returns sorted relative paths of all regular files
// in the given directory, recursing into subdirectories.
// Paths use forward slashes regardless of OS (e.g. "ns-a/deploy.yaml").
// Symlinks are skipped, including symlinks to directories, to avoid
// infinite loops from circular symlinks.
// All regular files are included regardless of extension, so that
// kubectl temp files (e.g. "apps.v1.Deployment.default.nginx") are
// discovered when diffyml is used as KUBECTL_EXTERNAL_DIFF.
func DiscoverFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		// filepath.Rel cannot fail here: WalkDir always yields paths under dir.
		rel, _ := filepath.Rel(dir, path)
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
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
	Name     string       // Relative path from root dir (e.g., "deployment.yaml" or "ns/deployment.yaml")
	Type     FilePairType // Relationship between from and to
	FromPath string       // Full path in from-directory (empty if OnlyTo)
	ToPath   string       // Full path in to-directory (empty if OnlyFrom)
}

// BuildFilePairPlan creates an alphabetically sorted plan of file
// pairs from two directories, matching files by relative path.
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

	// Compute union of filenames
	nameSet := make(map[string]bool)
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

// FormatRenameFileHeader returns a file header for renamed files, showing
// the old name on the "---" line and the new name on the "+++" line.
func FormatRenameFileHeader(fromName, toName string, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
	}
	prefix := colorStart(opts, styleBold+colorWhite)
	suffix := colorEnd(opts)
	return fmt.Sprintf("%s--- a/%s%s\n%s+++ b/%s%s\n", prefix, fromName, suffix, prefix, toName, suffix)
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

	if opts == nil {
		opts = DefaultFormatOptions()
	}
	prefix := colorStart(opts, styleBold+colorWhite)
	suffix := colorEnd(opts)
	return fmt.Sprintf("%s%s%s\n%s%s%s\n", prefix, fromLine, suffix, prefix, toLine, suffix)
}
