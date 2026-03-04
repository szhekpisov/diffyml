// filter.go - Difference filtering by path patterns.
package diffyml

import (
	"regexp"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/compare"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

type FilterOptions = types.FilterOptions

func FilterDiffs(diffs []Difference, opts *FilterOptions) []Difference {
	return compare.FilterDiffs(diffs, opts)
}

func FilterDiffsWithRegexp(diffs []Difference, opts *FilterOptions) ([]Difference, error) {
	return compare.FilterDiffsWithRegexp(diffs, opts)
}

func matchesAnyPath(diffPath string, filterPaths []string) bool {
	return compare.MatchesAnyPath(diffPath, filterPaths)
}

func pathMatches(diffPath, filterPath string) bool {
	return compare.PathMatches(diffPath, filterPath)
}

func compileRegexPatterns(patterns []string) ([]*regexp.Regexp, error) {
	return compare.CompileRegexPatterns(patterns)
}

func matchesAnyRegex(diffPath string, patterns []*regexp.Regexp) bool {
	return compare.MatchesAnyRegex(diffPath, patterns)
}
