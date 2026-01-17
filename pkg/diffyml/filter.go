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
		if len(opts.ExcludePaths) > 0 {
			if matchesAnyPath(diff.Path, opts.ExcludePaths) {
				continue // Excluded, skip
			}
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
		// Check that the prefix ends at a path boundary
		remaining := diffPath[len(filterPath):]
		if len(remaining) > 0 {
			firstChar := remaining[0]
			if firstChar == '.' || firstChar == '[' {
				return true
			}
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

	// Check if any filters are specified
	hasPathFilters := len(opts.IncludePaths) > 0 || len(opts.ExcludePaths) > 0
	hasRegexFilters := len(opts.IncludeRegexp) > 0 || len(opts.ExcludeRegexp) > 0

	if !hasPathFilters && !hasRegexFilters {
		return diffs, nil
	}

	// Compile regex patterns (cached for performance within this call)
	var includeRegex, excludeRegex []*regexp.Regexp
	var err error

	if len(opts.IncludeRegexp) > 0 {
		includeRegex, err = compileRegexPatterns(opts.IncludeRegexp)
		if err != nil {
			return nil, err
		}
	}

	if len(opts.ExcludeRegexp) > 0 {
		excludeRegex, err = compileRegexPatterns(opts.ExcludeRegexp)
		if err != nil {
			return nil, err
		}
	}

	var result []Difference

	for _, diff := range diffs {
		included := true

		// Step 1: Apply include filters (path or regex)
		if len(opts.IncludePaths) > 0 || len(opts.IncludeRegexp) > 0 {
			included = false

			// Check path includes
			if len(opts.IncludePaths) > 0 && matchesAnyPath(diff.Path, opts.IncludePaths) {
				included = true
			}

			// Check regex includes
			if !included && len(includeRegex) > 0 && matchesAnyRegex(diff.Path, includeRegex) {
				included = true
			}
		}

		if !included {
			continue
		}

		// Step 2: Apply exclude filters (path or regex)
		excluded := false

		if len(opts.ExcludePaths) > 0 && matchesAnyPath(diff.Path, opts.ExcludePaths) {
			excluded = true
		}

		if !excluded && len(excludeRegex) > 0 && matchesAnyRegex(diff.Path, excludeRegex) {
			excluded = true
		}

		if excluded {
			continue
		}

		result = append(result, diff)
	}

	return result, nil
}
