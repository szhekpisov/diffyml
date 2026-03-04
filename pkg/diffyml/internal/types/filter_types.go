package types

// FilterOptions configures how differences are filtered.
type FilterOptions struct {
	IncludePaths  []string
	ExcludePaths  []string
	IncludeRegexp []string
	ExcludeRegexp []string
}
