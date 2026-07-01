package diffyml

import (
	"testing"
)

func TestFilterDiffs_IncludePaths_SinglePath(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"config.name"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(result))
	}
	if result[0].Path.String() != "config.name" {
		t.Errorf("expected path 'config.name', got '%s'", result[0].Path)
	}
}

func TestFilterDiffs_IncludePaths_MultiplePaths(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"config", "nested", "deep"}, Type: DiffAdded, From: nil, To: "value"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"config.version"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(result))
	}
	for _, d := range result {
		if d.Path.String() == "config.version" {
			t.Error("config.version should have been excluded")
		}
	}
}

func TestFilterDiffs_ExcludePaths_PrefixMatch(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"config"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff after excluding 'config' prefix, got %d", len(result))
	}
	if result[0].Path.String() != "metadata.label" {
		t.Errorf("expected path 'metadata.label', got '%s'", result[0].Path)
	}
}

func TestFilterDiffs_IncludeBeforeExclude(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"config", "secret"}, Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
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
		if d.Path.String() == "config.secret" {
			t.Error("config.secret should have been excluded")
		}
		if d.Path.String() == "metadata.label" {
			t.Error("metadata.label should not be included")
		}
	}
}

func TestFilterDiffs_NoFilters(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
	}

	opts := &FilterOptions{}

	result := FilterDiffs(diffs, opts)

	if len(result) != 2 {
		t.Fatalf("expected 2 diffs with no filters, got %d", len(result))
	}
	// No-filter shortcut: must return the input slice itself (same backing array),
	// not a freshly-built copy. Pins the early-return at line 205.
	if &result[0] != &diffs[0] {
		t.Error("expected no-filter FilterDiffs to return the input slice itself, got a copy")
	}
}

func TestFilterDiffs_NilOptions(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
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
		{Path: DiffPath{"items[0]", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"items[1]", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"items[2]", "value"}, Type: DiffAdded, From: nil, To: "added"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"items[0]"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff matching items[0], got %d", len(result))
	}
	if result[0].Path.String() != "items[0].name" {
		t.Errorf("expected path 'items[0].name', got '%s'", result[0].Path)
	}
}

func TestFilterDiffs_ExactMatch(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"configuration"}, Type: DiffModified, From: "old", To: "new"},
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
		if d.Path.String() == "configuration" {
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
		{Path: DiffPath{"spec", "containers"}, Type: DiffModified, From: "a", To: "b"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "secret"}, Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
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
		if d.Path.String() == "config.secret" {
			t.Error("config.secret should have been excluded")
		}
	}
}

func TestFilterDiffsRegex_IncludeBeforeExclude(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"config", "secret"}, Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
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
		{Path: DiffPath{"items[0]", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"items[1]", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"items[10]", "name"}, Type: DiffModified, From: "old", To: "new"},
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "version"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"config", "secret"}, Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
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

// --- Nested key path tests (issue #101) ---

func TestNestedKeyPaths_RemovedOrderedMap(t *testing.T) {
	diff := Difference{
		Path: DiffPath{"metadata"},
		Type: DiffRemoved,
		From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
	}
	paths := nestedKeyPaths(diff)
	if len(paths) != 1 || paths[0] != "metadata.namespace" {
		t.Errorf("expected [metadata.namespace], got %v", paths)
	}
}

func TestNestedKeyPaths_AddedOrderedMap(t *testing.T) {
	diff := Difference{
		Path: DiffPath{"metadata"},
		Type: DiffAdded,
		To:   &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "staging"}},
	}
	paths := nestedKeyPaths(diff)
	if len(paths) != 1 || paths[0] != "metadata.namespace" {
		t.Errorf("expected [metadata.namespace], got %v", paths)
	}
}

func TestNestedKeyPaths_UnchangedOrderedMap(t *testing.T) {
	// Inverse mode collapses a fully-equal subtree to a single DiffUnchanged
	// carrying the whole OrderedMap. Its top-level keys must expand so
	// --filter/--exclude on a nested key matches, like added/removed entries.
	diff := Difference{
		Path: DiffPath{"metadata"},
		Type: DiffUnchanged,
		From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
		To:   &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
	}
	paths := nestedKeyPaths(diff)
	if len(paths) != 1 || paths[0] != "metadata.namespace" {
		t.Errorf("expected [metadata.namespace], got %v", paths)
	}
}

func TestNestedKeyPaths_DottedKey(t *testing.T) {
	diff := Difference{
		Path: DiffPath{"metadata", "annotations"},
		Type: DiffRemoved,
		From: &OrderedMap{
			Keys:   []string{"helm.sh/chart"},
			Values: map[string]any{"helm.sh/chart": "myapp-1.0"},
		},
	}
	paths := nestedKeyPaths(diff)
	if len(paths) != 1 || paths[0] != "metadata.annotations[helm.sh/chart]" {
		t.Errorf("expected [metadata.annotations[helm.sh/chart]], got %v", paths)
	}
}

func TestNestedKeyPaths_NonOrderedMap(t *testing.T) {
	diff := Difference{
		Path: DiffPath{"metadata"},
		Type: DiffRemoved,
		From: "scalar-value",
	}
	paths := nestedKeyPaths(diff)
	if paths != nil {
		t.Errorf("expected nil for non-OrderedMap, got %v", paths)
	}
}

func TestNestedKeyPaths_DiffModified(t *testing.T) {
	diff := Difference{
		Path: DiffPath{"metadata", "name"},
		Type: DiffModified,
		From: "old",
		To:   "new",
	}
	paths := nestedKeyPaths(diff)
	if paths != nil {
		t.Errorf("expected nil for DiffModified, got %v", paths)
	}
}

func TestNestedKeyPaths_EmptyPath(t *testing.T) {
	diff := Difference{
		Path: DiffPath{},
		Type: DiffRemoved,
		From: &OrderedMap{Keys: []string{"topkey"}, Values: map[string]any{"topkey": "val"}},
	}
	paths := nestedKeyPaths(diff)
	if len(paths) != 1 || paths[0] != "topkey" {
		t.Errorf("expected [topkey], got %v", paths)
	}
}

func TestFilterDiffs_ExcludePaths_NestedKeyInOrderedMap(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffRemoved,
			From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
		},
		{Path: DiffPath{"data", "key1"}, Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"metadata.namespace"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(result))
	}
	if result[0].Path.String() != "data.key1" {
		t.Errorf("expected data.key1, got %s", result[0].Path)
	}
}

func TestFilterDiffs_ExcludePaths_NestedKeyNoMatch(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffRemoved,
			From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
		},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"metadata.labels"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff (no match), got %d", len(result))
	}
}

func TestFilterDiffs_ExcludePaths_ParentStillExcludesNestedKey(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffRemoved,
			From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
		},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"metadata"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 0 {
		t.Fatalf("expected 0 diffs (parent exclude), got %d", len(result))
	}
}

func TestFilterDiffs_IncludePaths_NestedKeyInOrderedMap(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffRemoved,
			From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
		},
		{Path: DiffPath{"data", "key1"}, Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"metadata.namespace"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(result))
	}
	if result[0].Path.String() != "metadata" {
		t.Errorf("expected metadata, got %s", result[0].Path)
	}
}

func TestFilterDiffs_IncludePaths_NestedKeyNoMatch(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffRemoved,
			From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
		},
	}

	opts := &FilterOptions{
		IncludePaths: []string{"config.name"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 0 {
		t.Fatalf("expected 0 diffs, got %d", len(result))
	}
}

func TestFilterDiffsRegex_ExcludePattern_NestedKey(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffRemoved,
			From: &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "production"}},
		},
		{Path: DiffPath{"data", "key1"}, Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		ExcludeRegexp: []string{`namespace`},
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(result))
	}
	if result[0].Path.String() != "data.key1" {
		t.Errorf("expected data.key1, got %s", result[0].Path)
	}
}

func TestFilterDiffsRegex_IncludePattern_NestedKey(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffAdded,
			To:   &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "staging"}},
		},
		{Path: DiffPath{"data", "key1"}, Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		IncludeRegexp: []string{`^metadata\.namespace$`},
	}

	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(result))
	}
	if result[0].Path.String() != "metadata" {
		t.Errorf("expected metadata, got %s", result[0].Path)
	}
}

func TestFilterDiffs_ExcludePaths_AddedNestedKey(t *testing.T) {
	diffs := []Difference{
		{
			Path: DiffPath{"metadata"},
			Type: DiffAdded,
			To:   &OrderedMap{Keys: []string{"namespace"}, Values: map[string]any{"namespace": "staging"}},
		},
		{Path: DiffPath{"data", "key1"}, Type: DiffModified, From: "old", To: "new"},
	}

	opts := &FilterOptions{
		ExcludePaths: []string{"metadata.namespace"},
	}

	result := FilterDiffs(diffs, opts)

	if len(result) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(result))
	}
	if result[0].Path.String() != "data.key1" {
		t.Errorf("expected data.key1, got %s", result[0].Path)
	}
}

// --- Mutation testing: filter.go combined include path + regex ---

func TestFilterDiffsRegex_CombinedIncludePathAndRegex(t *testing.T) {
	// Item matches IncludeRegexp but not IncludePaths → still included
	diffs := []Difference{
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"metadata", "label"}, Type: DiffAdded, From: nil, To: "value"},
		{Path: DiffPath{"spec", "replicas"}, Type: DiffModified, From: 3, To: 5},
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
		paths[d.Path.String()] = true
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
		{Path: DiffPath{"config", "name"}, Type: DiffModified, From: "old", To: "new"},
		{Path: DiffPath{"config", "secret"}, Type: DiffModified, From: "xxx", To: "yyy"},
		{Path: DiffPath{"metadata", "password"}, Type: DiffAdded, From: nil, To: "secret"},
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
	if result[0].Path.String() != "config.name" {
		t.Errorf("expected config.name, got %q", result[0].Path)
	}
}

// k8sNoiseDiffs returns a synthetic set of differences covering each neat
// profile plus paths that should survive (a real spec change).
func k8sNoiseDiffs() []Difference {
	return []Difference{
		// K8s server noise
		{Path: DiffPath{"metadata", "managedFields"}, Type: DiffModified, From: "a", To: "b"},
		{Path: DiffPath{"metadata", "resourceVersion"}, Type: DiffModified, From: "1", To: "2"},
		{Path: DiffPath{"metadata", "generation"}, Type: DiffModified, From: 1, To: 2},
		// kubectl
		{Path: DiffPath{"metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration"}, Type: DiffModified, From: "{}", To: "{...}"},
		// Helm
		{Path: DiffPath{"metadata", "labels", "helm.sh/chart"}, Type: DiffModified, From: "v1.0.0", To: "v1.0.1"},
		{Path: DiffPath{"metadata", "annotations", "meta.helm.sh/release-name"}, Type: DiffModified, From: "old", To: "new"},
		// ArgoCD
		{Path: DiffPath{"metadata", "annotations", "argocd.argoproj.io/tracking-id"}, Type: DiffModified, From: "x", To: "y"},
		{Path: DiffPath{"metadata", "labels", "argocd.argoproj.io/instance"}, Type: DiffModified, From: "app", To: "app2"},
		// Flux
		{Path: DiffPath{"metadata", "annotations", "kustomize.toolkit.fluxcd.io/checksum"}, Type: DiffModified, From: "abc", To: "def"},
		// Status
		{Path: DiffPath{"status", "replicas"}, Type: DiffModified, From: 2, To: 3},
		// REAL changes that must survive
		{Path: DiffPath{"spec", "replicas"}, Type: DiffModified, From: 3, To: 5},
		{Path: DiffPath{"spec", "template", "spec", "containers", "[0]", "image"}, Type: DiffModified, From: "nginx:1.20", To: "nginx:1.21"},
	}
}

// TestFilterDiffs_NeatExcludesNoise wires the curated neat bundle through the
// existing regex filter and asserts only the real diffs survive.
func TestFilterDiffs_NeatExcludesNoise(t *testing.T) {
	diffs := k8sNoiseDiffs()
	opts := &FilterOptions{
		ExcludeRegexp: BuildNeatExcludeRegexp(DefaultNeatOptions()),
	}
	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("FilterDiffsWithRegexp: %v", err)
	}
	survived := make([]string, len(result))
	for i, d := range result {
		survived[i] = d.Path.String()
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 diffs to survive (spec.replicas, spec.template…image), got %d: %v", len(result), survived)
	}
	for _, path := range survived {
		if path != "spec.replicas" && path != "spec.template.spec.containers[0].image" {
			t.Errorf("unexpected surviving diff: %q", path)
		}
	}
}

// TestFilterDiffs_NeatNoHelmKeepsHelmDiffs verifies that disabling the Helm
// profile lets Helm-injected diffs survive while everything else is filtered.
func TestFilterDiffs_NeatNoHelmKeepsHelmDiffs(t *testing.T) {
	diffs := k8sNoiseDiffs()
	opts := &FilterOptions{
		ExcludeRegexp: BuildNeatExcludeRegexp(NeatOptions{
			K8s: true, Status: true, ArgoCD: true, Flux: true, // Helm: false
		}),
	}
	result, err := FilterDiffsWithRegexp(diffs, opts)
	if err != nil {
		t.Fatalf("FilterDiffsWithRegexp: %v", err)
	}
	helmFound := 0
	survived := make([]string, len(result))
	for i, d := range result {
		survived[i] = d.Path.String()
		if survived[i] == "metadata.labels[helm.sh/chart]" ||
			survived[i] == "metadata.annotations[meta.helm.sh/release-name]" {
			helmFound++
		}
	}
	if helmFound != 2 {
		t.Errorf("expected 2 Helm diffs to survive --no-neat-helm, got %d (full: %v)", helmFound, survived)
	}
}

// TestFilterDiffsWithRegexpReport_Counts verifies the per-pattern hit counter
// is populated correctly. Counts attribute each diff to the FIRST regex
// matching it (scan order).
func TestFilterDiffsWithRegexpReport_Counts(t *testing.T) {
	diffs := k8sNoiseDiffs()
	patterns := BuildNeatExcludeRegexp(DefaultNeatOptions())
	opts := &FilterOptions{ExcludeRegexp: patterns}
	report := &FilterReport{}
	result, err := FilterDiffsWithRegexpReport(diffs, opts, report)
	if err != nil {
		t.Fatalf("FilterDiffsWithRegexpReport: %v", err)
	}
	if len(report.ExcludeHits) != len(patterns) {
		t.Fatalf("ExcludeHits length: got %d, want %d", len(report.ExcludeHits), len(patterns))
	}
	totalHits := 0
	for _, h := range report.ExcludeHits {
		totalHits += h
	}
	excluded := len(diffs) - len(result)
	if totalHits != excluded {
		t.Errorf("sum of hits %d should equal excluded count %d", totalHits, excluded)
	}
}

// TestFilterDiffsWithRegexpReport_NilReportSafe verifies passing a nil report
// behaves identically to FilterDiffsWithRegexp.
func TestFilterDiffsWithRegexpReport_NilReportSafe(t *testing.T) {
	diffs := k8sNoiseDiffs()
	opts := &FilterOptions{ExcludeRegexp: BuildNeatExcludeRegexp(DefaultNeatOptions())}
	a, errA := FilterDiffsWithRegexp(diffs, opts)
	b, errB := FilterDiffsWithRegexpReport(diffs, opts, nil)
	if errA != nil || errB != nil {
		t.Fatalf("errors: %v, %v", errA, errB)
	}
	if len(a) != len(b) {
		t.Errorf("results differ: %d vs %d", len(a), len(b))
	}
}

// TestFilterDiffsWithRegexpReport_PathExclusionNotCounted verifies that a
// diff excluded by an ExcludePaths entry does NOT increment any regex
// hit counter (path exclusions are not regex-attributed).
func TestFilterDiffsWithRegexpReport_PathExclusionNotCounted(t *testing.T) {
	diffs := []Difference{
		{Path: DiffPath{"a", "b"}, Type: DiffModified, From: 1, To: 2},
	}
	opts := &FilterOptions{
		ExcludePaths:  []string{"a.b"},
		ExcludeRegexp: []string{`^a\.b$`}, // would also match
	}
	report := &FilterReport{}
	_, err := FilterDiffsWithRegexpReport(diffs, opts, report)
	if err != nil {
		t.Fatalf("FilterDiffsWithRegexpReport: %v", err)
	}
	for i, h := range report.ExcludeHits {
		if h != 0 {
			t.Errorf("hit count[%d] should be 0 (path filter ran first), got %d", i, h)
		}
	}
}
