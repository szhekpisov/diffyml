package diffyml

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestGetFormatter(t *testing.T) {
	valid := []struct {
		name string
	}{
		{"compact"}, {"brief"}, {"github"}, {"gitlab"}, {"gitea"}, {"COMPACT"},
	}
	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			f, err := GetFormatter(tt.name)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f == nil {
				t.Fatal("expected formatter, got nil")
			}
		})
	}

	invalid := []string{"invalid", ""}
	for _, name := range invalid {
		t.Run("error/"+name, func(t *testing.T) {
			_, err := GetFormatter(name)
			if err == nil {
				t.Error("expected error for invalid formatter name")
			}
		})
	}
}

func TestGetFormatter_ListsValidFormats(t *testing.T) {
	_, err := GetFormatter("badname")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	errStr := err.Error()
	for _, format := range []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"} {
		if !strings.Contains(errStr, format) {
			t.Errorf("error message should list valid format %q, got: %s", format, errStr)
		}
	}
}

func TestFormatOptions_Defaults(t *testing.T) {
	opts := DefaultFormatOptions()

	if opts.OmitHeader {
		t.Error("expected default OmitHeader to be false")
	}
	if opts.UseGoPatchStyle {
		t.Error("expected default UseGoPatchStyle to be false")
	}
	if opts.ContextLines != 4 {
		t.Errorf("expected default ContextLines 4, got %d", opts.ContextLines)
	}
	if opts.FilePath != "" {
		t.Errorf("expected default FilePath to be empty, got %q", opts.FilePath)
	}
}

func TestDiffGroup_Construction(t *testing.T) {
	diffs := []Difference{
		{Path: "config.host", Type: DiffModified, From: "old", To: "new"},
	}
	group := DiffGroup{
		FilePath: "deploy.yaml",
		Diffs:    diffs,
	}
	if group.FilePath != "deploy.yaml" {
		t.Errorf("expected FilePath %q, got %q", "deploy.yaml", group.FilePath)
	}
	if len(group.Diffs) != 1 {
		t.Errorf("expected 1 diff, got %d", len(group.Diffs))
	}
}

func TestStructuredFormatter_Interface(t *testing.T) {
	types := []struct {
		name string
		f    Formatter
	}{
		{"GitLab", &GitLabFormatter{}},
		{"GitHub", &GitHubFormatter{}},
		{"Gitea", &GiteaFormatter{}},
	}
	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := tt.f.(StructuredFormatter); !ok {
				t.Fatalf("%s should implement StructuredFormatter", tt.name)
			}
		})
	}
}

func TestFormatter_Interface(t *testing.T) {
	formatters := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}

	diffs := []Difference{
		{Path: "test.path", Type: DiffModified, From: "old", To: "new"},
	}
	opts := DefaultFormatOptions()

	for _, name := range formatters {
		t.Run(name, func(t *testing.T) {
			f, err := GetFormatter(name)
			if err != nil {
				t.Fatalf("failed to get formatter: %v", err)
			}
			output := f.Format(diffs, opts)
			if output == "" {
				t.Errorf("formatter %s returned empty output", name)
			}
		})
	}
}

func TestFormatter_EmptyDiffs(t *testing.T) {
	formatters := []string{"compact", "brief", "github", "gitlab", "gitea", "detailed"}
	opts := DefaultFormatOptions()

	for _, name := range formatters {
		t.Run(name, func(t *testing.T) {
			f, err := GetFormatter(name)
			if err != nil {
				t.Fatalf("failed to get formatter: %v", err)
			}
			_ = f.Format([]Difference{}, opts)
		})
	}
}

func TestFormatter_NilOptions(t *testing.T) {
	f, _ := GetFormatter("compact")

	diffs := []Difference{
		{Path: "test", Type: DiffAdded, From: nil, To: "value"},
	}
	output := f.Format(diffs, nil)
	if output == "" {
		t.Error("formatter should produce output even with nil options")
	}
}

func TestConvertToGoPatchPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"config.name", "/config/name"},
		{"items[0].value", "/items/0/value"},
		{"root", "/root"},
		{"a.b.c.d", "/a/b/c/d"},
		{"list[0][1].nested", "/list/0/1/nested"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertToGoPatchPath(tt.input)
			if result != tt.expected {
				t.Errorf("convertToGoPatchPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatSingle_AllFormatters(t *testing.T) {
	diffs := []Difference{
		{Path: "key.added", Type: DiffAdded, To: "value"},
		{Path: "key.removed", Type: DiffRemoved, From: "value"},
		{Path: "key.modified", Type: DiffModified, From: "old", To: "new"},
		{Path: "key.order", Type: DiffOrderChanged},
	}
	opts := DefaultFormatOptions()

	type singleFormatter interface {
		FormatSingle(diff Difference, opts *FormatOptions) string
	}

	formatters := map[string]singleFormatter{
		"compact": &CompactFormatter{},
		"brief":   &BriefFormatter{},
		"github":  &GitHubFormatter{},
		"gitlab":  &GitLabFormatter{},
		"gitea":   &GiteaFormatter{},
	}

	for name, f := range formatters {
		for _, diff := range diffs {
			t.Run(name+"/"+diff.Path, func(t *testing.T) {
				output := f.FormatSingle(diff, opts)
				if output == "" {
					t.Errorf("%s.FormatSingle returned empty for %s", name, diff.Path)
				}
			})
		}
	}
}

// Tests for formatValue with structured types

func TestFormatValue(t *testing.T) {
	t.Run("scalar", func(t *testing.T) {
		cases := []struct {
			input    interface{}
			expected string
		}{
			{"hello", "hello"},
			{42, "42"},
			{nil, "<nil>"},
		}
		for _, tt := range cases {
			got := formatValue(tt.input)
			if got != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		}
	})

	t.Run("OrderedMap", func(t *testing.T) {
		om := NewOrderedMap()
		om.Keys = append(om.Keys, "name", "port")
		om.Values["name"] = "http"
		om.Values["port"] = 8080

		result := formatValue(om)

		if strings.Contains(result, "&{") {
			t.Errorf("formatValue should not produce Go struct repr for *OrderedMap, got: %s", result)
		}
		if !strings.Contains(result, "name: http") {
			t.Errorf("expected 'name: http' in YAML output, got: %s", result)
		}
		if !strings.Contains(result, "port: 8080") {
			t.Errorf("expected 'port: 8080' in YAML output, got: %s", result)
		}
	})

	t.Run("ListWithOrderedMaps", func(t *testing.T) {
		item := NewOrderedMap()
		item.Keys = append(item.Keys, "name", "value")
		item.Values["name"] = "FOO"
		item.Values["value"] = "bar"

		val := []interface{}{item}
		result := formatValue(val)

		if strings.Contains(result, "&{") {
			t.Errorf("formatValue should not produce Go struct repr for []interface{} with *OrderedMap, got: %s", result)
		}
		if strings.Contains(result, "0x") {
			t.Errorf("formatValue should not produce pointer addresses, got: %s", result)
		}
		if !strings.Contains(result, "name: FOO") {
			t.Errorf("expected 'name: FOO' in YAML output, got: %s", result)
		}
	})

	t.Run("MapStringInterface", func(t *testing.T) {
		val := map[string]interface{}{"key": "value", "count": 42}
		result := formatValue(val)

		if !strings.Contains(result, "key: value") {
			t.Errorf("expected 'key: value' in YAML output, got: %s", result)
		}
		if !strings.Contains(result, "count: 42") {
			t.Errorf("expected 'count: 42' in YAML output, got: %s", result)
		}
	})

	t.Run("Timestamp/date_only", func(t *testing.T) {
		ts := time.Date(2010, 9, 9, 0, 0, 0, 0, time.UTC)
		if got := formatValue(ts); got != "2010-09-09" {
			t.Errorf("expected 2010-09-09, got %s", got)
		}
	})

	t.Run("Timestamp/datetime", func(t *testing.T) {
		ts := time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC)
		if got := formatValue(ts); got != "2023-06-15T14:30:00Z" {
			t.Errorf("expected 2023-06-15T14:30:00Z, got %s", got)
		}
	})
}

// makeDiffs generates n Difference entries of the given type.
func makeDiffs(dt DiffType, n int) []Difference {
	diffs := make([]Difference, n)
	for i := range diffs {
		diffs[i] = Difference{
			Path: fmt.Sprintf("key%d", i),
			Type: dt,
			From: "old",
			To:   "new",
		}
	}
	return diffs
}

// extractFingerprint extracts the first fingerprint value from GitLab JSON output.
func extractFingerprint(t *testing.T, output string) string {
	t.Helper()
	parts := strings.Split(output, `"fingerprint": "`)
	if len(parts) < 2 {
		t.Fatalf("could not extract fingerprint from output: %s", output)
	}
	end := strings.Index(parts[1], `"`)
	if end < 0 {
		t.Fatalf("could not find end of fingerprint in output: %s", output)
	}
	return parts[1][:end]
}
