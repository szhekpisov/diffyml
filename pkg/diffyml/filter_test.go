package diffyml

import (
	"testing"
)

func TestFilterDiffs_IncludePaths_SinglePath(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"config.name"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(result))
	}
	if result[0].Path != "config.name" {
		t.Errorf("expected path 'config.name', got '%s'", result[0].Path)
	}
}

func TestFilterDiffs_IncludePaths_MultiplePaths(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"config.name", "metadata.label"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(result))
	}
}

func TestFilterDiffs_IncludePaths_PrefixMatch(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "config.nested.deep", Type: DiffAdded, From: nil, To: "value"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"config"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 3 {
		t.Fatalf("expected 3 diffs matching 'config' prefix, got %d", len(result))
	}
}

func TestFilterDiffs_ExcludePaths_SinglePath(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"config.version"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(result))
	}
	for _, d := range result {
		if d.Path == "config.version" {
			t.Error("config.version should have been excluded")
		}
	}
}

func TestFilterDiffs_ExcludePaths_PrefixMatch(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"config"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff after excluding 'config' prefix, got %d", len(result))
	}
	if result[0].Path != "metadata.label" {
		t.Errorf("expected path 'metadata.label', got '%s'", result[0].Path)
	}
}

func TestFilterDiffs_IncludeBeforeExclude(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "config.secret", Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	// Include all config, then exclude config.secret
	opts := &FilterOptions{
		IncludePaths: []string{"config"},
		ExcludePaths: []string{"config.secret"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs (config.* minus config.secret), got %d", len(result))
	}
	for _, d := range result {
		if d.Path == "config.secret" {
			t.Error("config.secret should have been excluded")
		}
		if d.Path == "metadata.label" {
			t.Error("metadata.label should not be included")
		}
	}
}

func TestFilterDiffs_NoFilters(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
	}

	opts := &FilterOptions{}

	result := FilterDiffs(diffs, opts)

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs with no filters, got %d", len(result))
	}
}

func TestFilterDiffs_NilOptions(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
	}

	result := FilterDiffs(diffs, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff with nil options, got %d", len(result))
	}
}

func TestFilterDiffs_EmptyDiffs(t *testing.T) {
	diffs := []Difference{}

	opts := &FilterOptions{
		IncludePaths: []string{"config"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 0 {
		t.Fatalf("expected 0 diffs, got %d", len(result))
	}
}

func TestFilterDiffs_ArrayIndexPaths(t *testing.T) {
	diffs := []Difference{
		{Path: "items[0].name", Type: DiffModified, From: "old", To: "new"},
		{Path: "items[1].name", Type: DiffModified, From: "old", To: "new"},
		{Path: "items[2].value", Type: DiffAdded, From: nil, To: "added"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"items[0]"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff matching items[0], got %d", len(result))
	}
	if result[0].Path != "items[0].name" {
		t.Errorf("expected path 'items[0].name', got '%s'", result[0].Path)
	}
}

func TestFilterDiffs_ExactMatch(t *testing.T) {
	diffs := []Difference{
		{Path: "config", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "configuration", Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"config"},
	}

	result := FilterDiffs(diffs, opts)

	// Should match "config" and "config.name" but NOT "configuration"
	if len(result) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(result))
	}
	for _, d := range result {
		if d.Path == "configuration" {
			t.Error("configuration should not match config prefix")
		}
	}
}

func TestPathMatches_ExactMatch(t *testing.T) {
	if !pathMatches("config.name", "config.name") {
		t.Error("exact match should return true")
	}
}

func TestPathMatches_PrefixMatch(t *testing.T) {
	if !pathMatches("config.name.deep", "config") {
		t.Error("prefix match should return true")
	}
	if !pathMatches("config.name", "config") {
		t.Error("prefix match should return true")
	}
}

func TestPathMatches_NoMatch(t *testing.T) {
	if pathMatches("metadata.label", "config") {
		t.Error("non-matching path should return false")
	}
}

func TestPathMatches_PartialWordNoMatch(t *testing.T) {
	// "configuration" should not match filter "config"
	if pathMatches("configuration", "config") {
		t.Error("partial word match should return false")
	}
}

func TestPathMatches_ArrayPathMatch(t *testing.T) {
	if !pathMatches("items[0].name", "items[0]") {
		t.Error("array path prefix should match")
	}
	if !pathMatches("items[0]", "items") {
		t.Error("array under parent should match")
	}
}

// Regex filtering tests

func TestFilterDiffsRegex_IncludePattern_SinglePattern(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		IncludeRegexp: []string{`^config\.`},
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs matching '^config\\.', got %d", len(result))
	}
}

func TestFilterDiffsRegex_IncludePattern_MultiplePatterns(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
		{Path: "spec.containers", Type: DiffModified, From: "a", To: "b"},
	}

	opts := &FilterOptions{
		IncludeRegexp: []string{`^config`, `^spec`},
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(result))
	}
}

func TestFilterDiffsRegex_ExcludePattern(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.secret", Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		ExcludeRegexp: []string{`secret`},
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs after excluding 'secret', got %d", len(result))
	}
	for _, d := range result {
		if d.Path == "config.secret" {
			t.Error("config.secret should have been excluded")
		}
	}
}

func TestFilterDiffsRegex_IncludeBeforeExclude(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "config.secret", Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	// Include all config, then exclude secrets
	opts := &FilterOptions{
		IncludeRegexp: []string{`^config\.`},
		ExcludeRegexp: []string{`secret`},
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs (config.* minus secrets), got %d", len(result))
	}
}

func TestFilterDiffsRegex_InvalidIncludePattern(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		IncludeRegexp: []string{`[invalid`}, // Invalid regex
	}

	_, err := FilterDiffsWithRegexp(diffs, opts)
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestFilterDiffsRegex_InvalidExcludePattern(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		ExcludeRegexp: []string{`(?P<invalid`}, // Invalid regex
	}

	_, err := FilterDiffsWithRegexp(diffs, opts)
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestFilterDiffsRegex_NoPatterns(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
	}

	opts := &FilterOptions{}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs with no patterns, got %d", len(result))
	}
}

func TestFilterDiffsRegex_NilOptions(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
	}

	result, err := FilterDiffsWithRegexp(diffs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 diff with nil options, got %d", len(result))
	}
}

func TestFilterDiffsRegex_ArrayIndexPattern(t *testing.T) {
	diffs := []Difference{
		{Path: "items[0].name", Type: DiffModified, From: "old", To: "new"},
		{Path: "items[1].name", Type: DiffModified, From: "old", To: "new"},
		{Path: "items[10].name", Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		IncludeRegexp: []string{`items\[\d\]\.`}, // Match single digit index only
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs matching single digit index, got %d", len(result))
	}
}

func TestFilterDiffsRegex_CombinedPathAndRegex(t *testing.T) {
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.version", Type: DiffModified, From: "1", To: "2"},
		{Path: "config.secret", Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
	}

	// Use path filter for include, regex for exclude
	opts := &FilterOptions{
		IncludePaths:  []string{"config"},
		ExcludeRegexp: []string{`secret`},
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs (config minus secret), got %d", len(result))
	}
}

func TestCompileRegexPatterns_Valid(t *testing.T) {
	patterns := []string{`^config\.`, `secret$`, `\d+`}

	compiled, err := compileRegexPatterns(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compiled) != 3 {
		t.Fatalf("expected 3 compiled patterns, got %d", len(compiled))
	}
}

func TestCompileRegexPatterns_Invalid(t *testing.T) {
	patterns := []string{`valid`, `[invalid`, `also-valid`}

	_, err := compileRegexPatterns(patterns)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestCompileRegexPatterns_Empty(t *testing.T) {
	patterns := []string{}

	compiled, err := compileRegexPatterns(patterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compiled) != 0 {
		t.Fatalf("expected 0 compiled patterns, got %d", len(compiled))
	}
}

// --- Mutation testing: filter.go combined include path + regex ---

func TestFilterDiffsRegex_CombinedIncludePathAndRegex(t *testing.T) {
	// Item matches IncludeRegexp but not IncludePaths → still included
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "metadata.label", Type: DiffAdded, From: nil, To: "value"},
		{Path: "spec.replicas", Type: DiffModified, From: 3, To: 5},
	}

	opts := &FilterOptions{
		IncludePaths:  []string{"config"},  // matches config.name
		IncludeRegexp: []string{`^spec\.`}, // matches spec.replicas
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs (config.name via path + spec.replicas via regex), got %d", len(result))
	}

	paths := map[string]bool{}
	for _, d := range result {
		paths[d.Path] = true
	}
	if !paths["config.name"] {
		t.Error("config.name should be included via path filter")
	}
	if !paths["spec.replicas"] {
		t.Error("spec.replicas should be included via regex filter")
	}
}

func TestFilterDiffsRegex_CombinedExcludePathAndRegex(t *testing.T) {
	// Item matches ExcludeRegexp but not ExcludePaths → still excluded
	diffs := []Difference{
		{Path: "config.name", Type: DiffModified, From: "old", To: "new"},
		{Path: "config.secret", Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: "metadata.password", Type: DiffAdded, From: nil, To: "secret"},
	}

	opts := &FilterOptions{
		ExcludePaths:  []string{"config.secret"}, // excludes config.secret via path
		ExcludeRegexp: []string{`password`},      // excludes metadata.password via regex
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 diff (only config.name), got %d", len(result))
	}
	if result[0].Path != "config.name" {
		t.Errorf("expected config.name, got %q", result[0].Path)
	}
}
