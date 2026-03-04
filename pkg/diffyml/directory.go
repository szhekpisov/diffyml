// directory.go - Directory-level comparison support for kubectl compatibility.
//
// Enables diffyml to serve as KUBECTL_EXTERNAL_DIFF provider by accepting
// two directory paths and comparing YAML files within them.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/run"
)

// IsDirectory reports whether path is an existing directory.
func IsDirectory(path string) bool { return run.IsDirectory(path) }

// DiscoverFiles returns sorted filenames of all regular files in the given directory.
func DiscoverFiles(dir string) ([]string, error) { return run.DiscoverFiles(dir) }

// FilePairType describes the relationship between source and target files.
type FilePairType = format.FilePairType

const (
	// FilePairBothExist means the file exists in both directories.
	FilePairBothExist = format.FilePairBothExist
	// FilePairOnlyFrom means the file exists only in the source directory.
	FilePairOnlyFrom = format.FilePairOnlyFrom
	// FilePairOnlyTo means the file exists only in the target directory.
	FilePairOnlyTo = format.FilePairOnlyTo
)

// FilePair represents a matched pair of files for comparison.
type FilePair = run.FilePair

// BuildFilePairPlan creates an alphabetically sorted plan of file pairs from two directories.
func BuildFilePairPlan(fromDir, toDir string) ([]FilePair, error) {
	return run.BuildFilePairPlan(fromDir, toDir)
}

// FormatFileHeader returns a unified-diff-style file header for directory mode.
func FormatFileHeader(filename string, pairType FilePairType, opts *FormatOptions) string {
	return format.FormatFileHeader(filename, pairType, opts)
}

// buildFilePairsFromMap builds a sorted slice of FilePair from an in-memory map.
// Unexported; required by tests in package diffyml.
func buildFilePairsFromMap(m map[string][2][]byte) []FilePair {
	return run.BuildFilePairsFromMap(m)
}

// runDirectory executes directory-mode comparison using a 3-phase pipeline.
// Unexported; required by tests in package diffyml.
func runDirectory(runOpts *RunOptions, rc *RunConfig, fromDir, toDir string) *ExitResult {
	return run.RunDirectory(runOpts, rc, fromDir, toDir)
}
