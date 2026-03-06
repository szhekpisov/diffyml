package diffyml

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGitLabFormatter_CodeQualityJSON(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	for _, field := range []string{"description", "fingerprint", "severity", "location", "check_name", `"lines"`, `"begin"`} {
		if !strings.Contains(output, field) {
			t.Errorf("expected field %q in GitLab output, got: %s", field, output)
		}
	}
}

func TestGitLabFormatter_EmptyArray(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	if !strings.Contains(output, "[]") {
		t.Errorf("expected empty JSON array for no differences, got: %s", output)
	}
}

func TestGitLabFormatter_MultipleDiffs(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "a", Type: DiffAdded, To: "new"},
		{Path: "b", Type: DiffRemoved, From: "old"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, ",") {
		t.Errorf("expected comma-separated JSON entries, got: %s", output)
	}
}

func TestGitLabFormatter_RequiredFields(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.key", Type: DiffAdded, To: "value"},
	}

	output := f.Format(diffs, opts)

	for _, field := range []string{"description", "check_name", "fingerprint", "severity", "location", "path", "lines", "begin"} {
		if !strings.Contains(output, field) {
			t.Errorf("expected required field %q in GitLab output, got: %s", field, output)
		}
	}
}

func TestGitLabFormatter_SeverityAndCheckName(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	tests := []struct {
		name              string
		diff              Difference
		expectedSeverity  string
		expectedCheckName string
	}{
		{
			name:              "added",
			diff:              Difference{Path: "key", Type: DiffAdded, To: "val"},
			expectedSeverity:  `"severity": "info"`,
			expectedCheckName: "diffyml/added",
		},
		{
			name:              "removed",
			diff:              Difference{Path: "key", Type: DiffRemoved, From: "val"},
			expectedSeverity:  `"severity": "major"`,
			expectedCheckName: "diffyml/removed",
		},
		{
			name:              "modified",
			diff:              Difference{Path: "key", Type: DiffModified, From: "old", To: "new"},
			expectedSeverity:  `"severity": "major"`,
			expectedCheckName: "diffyml/modified",
		},
		{
			name:              "order changed",
			diff:              Difference{Path: "list", Type: DiffOrderChanged},
			expectedSeverity:  `"severity": "minor"`,
			expectedCheckName: "diffyml/order-changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if !strings.Contains(output, tt.expectedSeverity) {
				t.Errorf("expected severity %q in output, got: %s", tt.expectedSeverity, output)
			}
			if !strings.Contains(output, tt.expectedCheckName) {
				t.Errorf("expected check_name %q in output, got: %s", tt.expectedCheckName, output)
			}
		})
	}
}

func TestGitLabFormatter_UniqueFingerprints(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.key", Type: DiffAdded, To: "value1"},
		{Path: "config.key", Type: DiffRemoved, From: "value2"},
	}

	output := f.Format(diffs, opts)

	fpCount := strings.Count(output, "fingerprint")
	if fpCount != 2 {
		t.Fatalf("expected 2 fingerprint fields, got %d", fpCount)
	}

	parts := strings.Split(output, `"fingerprint": "`)
	if len(parts) < 3 {
		t.Fatal("could not extract fingerprints from output")
	}
	fp1 := parts[1][:64]
	fp2 := parts[2][:64]
	if fp1 == fp2 {
		t.Errorf("fingerprints should be unique for different diffs, both got: %s", fp1)
	}
}

func TestGitLabFormatter_FingerprintDeterministic(t *testing.T) {
	f, _ := GetFormatter("gitlab")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.key", Type: DiffModified, From: "old", To: "new"},
	}

	output1 := f.Format(diffs, opts)
	output2 := f.Format(diffs, opts)

	if output1 != output2 {
		t.Errorf("fingerprint should be deterministic, got different outputs:\n%s\nvs\n%s", output1, output2)
	}
}

// File path tests

func TestGitLabFormatter_LocationPathUsesFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	if !strings.Contains(output, `"path": "deploy.yaml"`) {
		t.Errorf("expected location.path to be file path 'deploy.yaml', got: %s", output)
	}
	if strings.Contains(output, `"path": "config.host"`) {
		t.Errorf("location.path should not be YAML key path, got: %s", output)
	}
}

func TestGitLabFormatter_LocationPathFallback(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	if !strings.Contains(output, `"path": "config.host"`) {
		t.Errorf("expected location.path fallback to YAML key path, got: %s", output)
	}
}

func TestGitLabFormatter_FingerprintIncludesFilePath(t *testing.T) {
	f := &GitLabFormatter{}

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
	}

	opts1 := DefaultFormatOptions()
	opts1.FilePath = "file1.yaml"
	output1 := f.Format(diffs, opts1)

	opts2 := DefaultFormatOptions()
	opts2.FilePath = "file2.yaml"
	output2 := f.Format(diffs, opts2)

	fp1 := extractFingerprint(t, output1)
	fp2 := extractFingerprint(t, output2)

	if fp1 == fp2 {
		t.Errorf("fingerprints should differ for same change in different files, both got: %s", fp1)
	}
}

func TestGitLabFormatter_FingerprintUnchangedWhenNoFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)
	fp := extractFingerprint(t, output)

	desc := diffDescription(diffs[0])
	expectedFP := gitLabFingerprint("", desc)
	if fp != expectedFP {
		t.Errorf("fingerprint with empty FilePath should match legacy formula\ngot:  %s\nwant: %s", fp, expectedFP)
	}
}

func TestGitLabFormatter_DescriptionContainsYAMLPath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: DiffAdded, To: 8080},
		{Path: "config.old", Type: DiffRemoved, From: "value"},
		{Path: "items", Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)

	for _, d := range diffs {
		if !strings.Contains(output, d.Path) {
			t.Errorf("expected YAML path %q in description, got: %s", d.Path, output)
		}
	}
}

func TestGitLabFormatter_ValidJSON_WithFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: DiffAdded, To: 8080},
	}

	output := f.Format(diffs, opts)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

func TestGitLabFormatter_NoBOM(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
	}

	output := f.Format(diffs, opts)

	if len(output) >= 3 && output[0] == 0xEF && output[1] == 0xBB && output[2] == 0xBF {
		t.Error("output should not contain BOM")
	}
}

// FormatAll tests

func TestGitLabFormatter_FormatAll_SingleArray(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs:    []Difference{{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"}},
		},
		{
			FilePath: "service.yaml",
			Diffs:    []Difference{{Path: "service.port", Type: DiffAdded, To: 8080}},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatAll output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries in single array, got %d", len(result))
	}
}

func TestGitLabFormatter_FormatAll_EmptyGroups(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	output := f.FormatAll([]DiffGroup{}, opts)

	if output != "[]\n" {
		t.Errorf("expected empty JSON array for no groups, got: %q", output)
	}
}

func TestGitLabFormatter_FormatAll_DescriptionIncludesFilename(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs:    []Difference{{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"}},
		},
	}

	output := f.FormatAll(groups, opts)

	if !strings.Contains(output, "deploy.yaml") {
		t.Errorf("expected filename 'deploy.yaml' in description, got: %s", output)
	}
	if !strings.Contains(output, "config.host") {
		t.Errorf("expected YAML path 'config.host' in description, got: %s", output)
	}
}

func TestGitLabFormatter_FormatAll_LocationPath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs:    []Difference{{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"}},
		},
	}

	output := f.FormatAll(groups, opts)

	if !strings.Contains(output, `"path": "deploy.yaml"`) {
		t.Errorf("expected location.path 'deploy.yaml', got: %s", output)
	}
}

func TestGitLabFormatter_FormatAll_UniqueFingerprintsAcrossFiles(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "file1.yaml",
			Diffs:    []Difference{{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"}},
		},
		{
			FilePath: "file2.yaml",
			Diffs:    []Difference{{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"}},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatAll output is not valid JSON: %v", err)
	}

	fp1 := result[0]["fingerprint"].(string)
	fp2 := result[1]["fingerprint"].(string)

	if fp1 == fp2 {
		t.Errorf("fingerprints should differ for same change in different files, both got: %s", fp1)
	}
}

func TestGitLabFormatter_FormatAll_ValidJSON(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
				{Path: "config.port", Type: DiffAdded, To: 8080},
			},
		},
		{
			FilePath: "service.yaml",
			Diffs:    []Difference{{Path: "service.name", Type: DiffRemoved, From: "old-svc"}},
		},
	}

	output := f.FormatAll(groups, opts)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("FormatAll output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 total entries, got %d", len(result))
	}
}

func TestGitLabFormatter_ImplementsStructuredFormatter(t *testing.T) {
	var f Formatter = &GitLabFormatter{}
	sf, ok := f.(StructuredFormatter)
	if !ok {
		t.Fatal("GitLabFormatter should implement StructuredFormatter")
	}

	output := sf.FormatAll([]DiffGroup{}, DefaultFormatOptions())
	if output != "[]\n" {
		t.Errorf("expected empty array, got: %q", output)
	}
}

// Backward compatibility tests

func TestGitLabFormatter_BackwardCompat_EmptyFilePath(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"},
		{Path: "config.port", Type: DiffAdded, To: 8080},
		{Path: "config.old", Type: DiffRemoved, From: "value"},
		{Path: "items", Type: DiffOrderChanged},
	}

	output := f.Format(diffs, opts)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 entries, got %d", len(result))
	}

	for i, entry := range result {
		location := entry["location"].(map[string]interface{})
		path := location["path"].(string)
		if path != diffs[i].Path {
			t.Errorf("entry %d: expected location.path=%q (fallback to diff.Path), got %q", i, diffs[i].Path, path)
		}
	}

	for i, entry := range result {
		fp := entry["fingerprint"].(string)
		desc := diffDescription(diffs[i])
		expectedFP := gitLabFingerprint("", desc)
		if fp != expectedFP {
			t.Errorf("entry %d: fingerprint mismatch with legacy formula\ngot:  %s\nwant: %s", i, fp, expectedFP)
		}
	}
}

func TestGitLabFormatter_BackwardCompat_FingerprintStability(t *testing.T) {
	diff := Difference{Path: "config.host", Type: DiffModified, From: "localhost", To: "production"}
	desc := diffDescription(diff)

	expectedFP := gitLabFingerprint("", desc)

	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()
	output := f.Format([]Difference{diff}, opts)

	fp := extractFingerprint(t, output)
	if fp != expectedFP {
		t.Errorf("fingerprint should match legacy formula\ngot:  %s\nwant: %s", fp, expectedFP)
	}

	output2 := f.Format([]Difference{diff}, opts)
	fp2 := extractFingerprint(t, output2)
	if fp != fp2 {
		t.Errorf("fingerprint should be deterministic across calls\ncall1: %s\ncall2: %s", fp, fp2)
	}
}

func TestGitLabFormatter_BackwardCompat_AllDiffTypes_ValidJSON(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	allDiffs := []Difference{
		{Path: "added.key", Type: DiffAdded, To: "value"},
		{Path: "removed.key", Type: DiffRemoved, From: "value"},
		{Path: "modified.key", Type: DiffModified, From: "old", To: "new"},
		{Path: "order.key", Type: DiffOrderChanged},
	}

	output := f.Format(allDiffs, opts)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	requiredFields := []string{"description", "check_name", "fingerprint", "severity", "location"}
	for i, entry := range result {
		for _, field := range requiredFields {
			if _, ok := entry[field]; !ok {
				t.Errorf("entry %d: missing required field %q", i, field)
			}
		}
		location := entry["location"].(map[string]interface{})
		if _, ok := location["path"]; !ok {
			t.Errorf("entry %d: location missing 'path'", i)
		}
		lines := location["lines"].(map[string]interface{})
		if begin, ok := lines["begin"]; !ok {
			t.Errorf("entry %d: location.lines missing 'begin'", i)
		} else if begin.(float64) != 1 {
			t.Errorf("entry %d: expected lines.begin=1, got %v", i, begin)
		}
	}
}

func TestGitLabFormatter_BackwardCompat_NilOptions(t *testing.T) {
	f := &GitLabFormatter{}

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "old", To: "new"},
	}

	output := f.Format(diffs, nil)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output with nil opts is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result))
	}
}

func TestGitLabFormatter_BackwardCompat_EmptyDiffs(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)

	var result []interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("empty diffs output is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d entries", len(result))
	}
}

func TestGitLabFormatter_BackwardCompat_SpecialCharsInValues(t *testing.T) {
	f := &GitLabFormatter{}
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.msg", Type: DiffModified, From: `line1\nline2`, To: `"quoted value"`},
		{Path: "config.tab", Type: DiffModified, From: "no\ttab", To: "has\ttab"},
	}

	output := f.Format(diffs, opts)

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output with special chars is not valid JSON: %v\noutput: %s", err, output)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}
