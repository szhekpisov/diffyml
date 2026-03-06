package diffyml

import (
	"strings"
	"testing"
)

func TestBriefFormatter_SummaryGeneration(t *testing.T) {
	f, _ := GetFormatter("brief")
	opts := DefaultFormatOptions()

	tests := []struct {
		name     string
		diffs    []Difference
		expected []string
	}{
		{
			name:     "single added",
			diffs:    []Difference{{Path: "key", Type: DiffAdded, To: "value"}},
			expected: []string{"1 added"},
		},
		{
			name:     "single removed",
			diffs:    []Difference{{Path: "key", Type: DiffRemoved, From: "value"}},
			expected: []string{"1 removed"},
		},
		{
			name:     "single modified",
			diffs:    []Difference{{Path: "key", Type: DiffModified, From: "old", To: "new"}},
			expected: []string{"1 modified"},
		},
		{
			name: "mixed changes",
			diffs: []Difference{
				{Path: "a", Type: DiffAdded, To: "new"},
				{Path: "b", Type: DiffAdded, To: "new2"},
				{Path: "c", Type: DiffRemoved, From: "old"},
				{Path: "d", Type: DiffModified, From: "old", To: "new"},
			},
			expected: []string{"2 added", "1 removed", "1 modified"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format(tt.diffs, opts)
			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in brief output, got: %s", exp, output)
				}
			}
		})
	}
}

func TestBriefFormatter_NoDifferences(t *testing.T) {
	f, _ := GetFormatter("brief")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	if !strings.Contains(output, "no differences") {
		t.Errorf("expected 'no differences' message, got: %s", output)
	}
}

func TestBriefFormatter_ZeroCategories(t *testing.T) {
	diffs := []Difference{
		{Path: "a", Type: DiffAdded, To: "x"},
		{Path: "b", Type: DiffAdded, To: "y"},
	}

	f := &BriefFormatter{}
	output := f.Format(diffs, nil)

	if !strings.HasPrefix(output, "2 added") {
		t.Errorf("expected output starting with '2 added', got: %s", output)
	}
	for _, absent := range []string{"removed", "modified"} {
		if strings.Contains(output, absent) {
			t.Errorf("output should not contain %q when there are none, got: %s", absent, output)
		}
	}
}

func TestBriefFormatter_OnlyModified(t *testing.T) {
	diffs := []Difference{
		{Path: "a", Type: DiffModified, From: "old", To: "new"},
	}

	f := &BriefFormatter{}
	output := f.Format(diffs, nil)

	if !strings.HasPrefix(output, "1 modified") {
		t.Errorf("expected output starting with '1 modified', got: %s", output)
	}
	for _, absent := range []string{"added", "removed"} {
		if strings.Contains(output, absent) {
			t.Errorf("output should not contain %q when there are none, got: %s", absent, output)
		}
	}
}
