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
		pathStr := diff.Path.String()
		nested := nestedKeyPaths(diff)

		// Step 1: Apply include filter (if specified)
		if len(opts.IncludePaths) > 0 {
			if !matchesAnyPathWithNested(pathStr, nested, opts.IncludePaths) {
				continue // Not included, skip
			}
		}

		// Step 2: Apply exclude filter (if specified)
		if matchesAnyPathWithNested(pathStr, nested, opts.ExcludePaths) {
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

// matchesAnyPathWithNested checks if the diff path or any of its nested
// key paths match any filter path.
func matchesAnyPathWithNested(diffPath string, nestedPaths []string, filterPaths []string) bool {
	if matchesAnyPath(diffPath, filterPaths) {
		return true
	}
	for _, np := range nestedPaths {
		if matchesAnyPath(np, filterPaths) {
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

// matchesAnyRegexWithNested checks if the diff path or any of its nested
// key paths match any regex pattern.
func matchesAnyRegexWithNested(diffPath string, nestedPaths []string, patterns []*regexp.Regexp) bool {
	if matchesAnyRegex(diffPath, patterns) {
		return true
	}
	for _, np := range nestedPaths {
		if matchesAnyRegex(np, patterns) {
			return true
		}
	}
	return false
}

// nestedKeyPaths returns extended path strings for diffs that report
// added/removed map entries at the parent path. For such diffs, the
// actual key lives inside the From/To OrderedMap. Returns nil if the
// diff does not contain nested map keys.
func nestedKeyPaths(diff Difference) []string {
	var om *OrderedMap
	switch diff.Type {
	case DiffRemoved:
		om, _ = diff.From.(*OrderedMap)
	case DiffAdded:
		om, _ = diff.To.(*OrderedMap)
	default:
		return nil
	}
	if om == nil || len(om.Keys) == 0 {
		return nil
	}
	paths := make([]string, len(om.Keys))
	for i, key := range om.Keys {
		paths[i] = diff.Path.Append(key).String()
	}
	return paths
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
		pathStr := diff.Path.String()
		nested := nestedKeyPaths(diff)
		included := true

		// Step 1: Apply include filters (path or regex)
		if hasIncludeFilters {
			included = matchesAnyPathWithNested(pathStr, nested, opts.IncludePaths) ||
				matchesAnyRegexWithNested(pathStr, nested, includeRegex)
		}

		if !included {
			continue
		}

		// Step 2: Apply exclude filters (path or regex)
		if matchesAnyPathWithNested(pathStr, nested, opts.ExcludePaths) ||
			matchesAnyRegexWithNested(pathStr, nested, excludeRegex) {
			continue
		}

		result = append(result, diff)
	}

	return result, nil
}
