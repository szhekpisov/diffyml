// filter.go - Difference filtering by path patterns.
//
// Supports include/exclude filtering using exact path matching and regex.
// Key types: FilterOptions.
// Key functions: FilterDiffs(), FilterDiffsWithRegexp().
package diffyml

import (
	"fmt"
	"regexp"
	"strings"
)

// FilterOptions configures how differences are filtered.
type FilterOptions struct {
	// IncludePaths filters differences to include only those matching specified paths.
	// Uses dot-notation path matching with prefix support.
	IncludePaths []string
	// ExcludePaths filters differences to exclude those matching specified paths.
	// Uses dot-notation path matching with prefix support.
	ExcludePaths []string
	// IncludeRegexp filters differences to include only those matching specified regex patterns.
	IncludeRegexp []string
	// ExcludeRegexp filters differences to exclude those matching specified regex patterns.
	ExcludeRegexp []string
}

// FilterDiffs filters the list of differences based on the provided options.
// If opts is nil or has no filters, returns the original diffs unchanged.
// Include filters are applied before exclude filters.
func FilterDiffs(diffs []Difference, opts *FilterOptions) []Difference {
	if opts == nil {
		return diffs
	}

	// If no filters specified, return all diffs
	if len(opts.IncludePaths) == 0 && len(opts.ExcludePaths) == 0 {
		return diffs
	}

	var result []Difference

	for _, diff := range diffs {
		// Step 1: Apply include filter (if specified)
		if len(opts.IncludePaths) > 0 {
			if !matchesAnyPath(diff.Path, opts.IncludePaths) {
				continue // Not included, skip
			}
		}

		// Step 2: Apply exclude filter (if specified)
		if matchesAnyPath(diff.Path, opts.ExcludePaths) {
			continue // Excluded, skip
		}

		result = append(result, diff)
	}

	return result
}

// matchesAnyPath checks if the diff path matches any of the filter paths.
func matchesAnyPath(diffPath string, filterPaths []string) bool {
	for _, filterPath := range filterPaths {
		if pathMatches(diffPath, filterPath) {
			return true
		}
	}
	return false
}

// pathMatches checks if a diff path matches a filter path.
// Supports exact match and prefix match (filter is prefix of diff path).
// Prefix match requires the path to have a proper boundary (dot or array bracket).
func pathMatches(diffPath, filterPath string) bool {
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

// compileRegexPatterns compiles a list of regex pattern strings.
// Returns an error with meaningful message if any pattern is invalid.
func compileRegexPatterns(patterns []string) ([]*regexp.Regexp, error) {
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

// matchesAnyRegex checks if the diff path matches any of the compiled regex patterns.
func matchesAnyRegex(diffPath string, patterns []*regexp.Regexp) bool {
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
func FilterDiffsWithRegexp(diffs []Difference, opts *FilterOptions) ([]Difference, error) {
	if opts == nil {
		return diffs, nil
	}

	// Compile regex patterns
	includeRegex, err := compileRegexPatterns(opts.IncludeRegexp)
	if err != nil {
		return nil, err
	}
	excludeRegex, err := compileRegexPatterns(opts.ExcludeRegexp)
	if err != nil {
		return nil, err
	}

	// Check if any filters are specified
	hasIncludeFilters := len(opts.IncludePaths) > 0 || len(includeRegex) > 0

	if !hasIncludeFilters && len(opts.ExcludePaths) == 0 && len(excludeRegex) == 0 {
		return diffs, nil
	}

	var result []Difference

	for _, diff := range diffs {
		included := true

		// Step 1: Apply include filters (path or regex)
		if hasIncludeFilters {
			included = matchesAnyPath(diff.Path, opts.IncludePaths) ||
				matchesAnyRegex(diff.Path, includeRegex)
		}

		if !included {
			continue
		}

		// Step 2: Apply exclude filters (path or regex)
		if matchesAnyPath(diff.Path, opts.ExcludePaths) ||
			matchesAnyRegex(diff.Path, excludeRegex) {
			continue
		}

		result = append(result, diff)
	}

	return result, nil
}
