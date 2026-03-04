// filter.go - Difference filtering by path patterns.
//
// Supports include/exclude filtering using exact path matching and regex.
// Key types: FilterOptions.
// Key functions: FilterDiffs(), FilterDiffsWithRegexp().
package compare

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// FilterDiffs filters the list of differences based on the provided options.
// If opts is nil or has no filters, returns the original diffs unchanged.
// Include filters are applied before exclude filters.
func FilterDiffs(diffs []types.Difference, opts *types.FilterOptions) []types.Difference {
	if opts == nil {
		return diffs
	}

	// If no filters specified, return all diffs
	if len(opts.IncludePaths) == 0 && len(opts.ExcludePaths) == 0 {
		return diffs
	}

	var result []types.Difference

	for _, diff := range diffs {
		// Step 1: Apply include filter (if specified)
		if len(opts.IncludePaths) > 0 {
			if !MatchesAnyPath(diff.Path, opts.IncludePaths) {
				continue // Not included, skip
			}
		}

		// Step 2: Apply exclude filter (if specified)
		if MatchesAnyPath(diff.Path, opts.ExcludePaths) {
			continue // Excluded, skip
		}

		result = append(result, diff)
	}

	return result
}

// MatchesAnyPath checks if the diff path matches any of the filter paths.
func MatchesAnyPath(diffPath string, filterPaths []string) bool {
	for _, filterPath := range filterPaths {
		if PathMatches(diffPath, filterPath) {
			return true
		}
	}
	return false
}

// PathMatches checks if a diff path matches a filter path.
// Supports exact match and prefix match (filter is prefix of diff path).
// Prefix match requires the path to have a proper boundary (dot or array bracket).
func PathMatches(diffPath, filterPath string) bool {
	// Exact match
	if diffPath == filterPath {
		return true
	}

	// Prefix match: filterPath must be a proper prefix of diffPath
	// The character after the prefix must be '.' or '[' to ensure we match
	// at path boundaries (not partial word matches)
	if strings.HasPrefix(diffPath, filterPath) {
		firstChar := diffPath[len(filterPath)]
		if firstChar == '.' || firstChar == '[' {
			return true
		}
	}

	return false
}

// CompileRegexPatterns compiles a list of regex pattern strings.
// Returns an error with meaningful message if any pattern is invalid.
func CompileRegexPatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

// MatchesAnyRegex checks if the diff path matches any of the compiled regex patterns.
func MatchesAnyRegex(diffPath string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(diffPath) {
			return true
		}
	}
	return false
}

// FilterDiffsWithRegexp filters differences with support for regex patterns.
// This function returns an error if any regex pattern is invalid.
// Include filters (paths and regex) are applied before exclude filters.
// Path filters are evaluated first, then regex filters.
func FilterDiffsWithRegexp(diffs []types.Difference, opts *types.FilterOptions) ([]types.Difference, error) {
	if opts == nil {
		return diffs, nil
	}

	// Compile regex patterns
	includeRegex, err := CompileRegexPatterns(opts.IncludeRegexp)
	if err != nil {
		return nil, err
	}
	excludeRegex, err := CompileRegexPatterns(opts.ExcludeRegexp)
	if err != nil {
		return nil, err
	}

	// Check if any filters are specified
	hasIncludeFilters := len(opts.IncludePaths) > 0 || len(includeRegex) > 0

	if !hasIncludeFilters && len(opts.ExcludePaths) == 0 && len(excludeRegex) == 0 {
		return diffs, nil
	}

	var result []types.Difference

	for _, diff := range diffs {
		included := true

		// Step 1: Apply include filters (path or regex)
		if hasIncludeFilters {
			included = MatchesAnyPath(diff.Path, opts.IncludePaths) ||
				MatchesAnyRegex(diff.Path, includeRegex)
		}

		if !included {
			continue
		}

		// Step 2: Apply exclude filters (path or regex)
		if MatchesAnyPath(diff.Path, opts.ExcludePaths) ||
			MatchesAnyRegex(diff.Path, excludeRegex) {
			continue
		}

		result = append(result, diff)
	}

	return result, nil
}
