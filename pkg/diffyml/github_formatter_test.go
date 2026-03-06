package diffyml

import (
	"fmt"
	"strings"
	"testing"
)

func TestGitHubFormatter_WorkflowCommandFormat(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	if !strings.Contains(output, "::warning title=YAML Modified::") {
		t.Errorf("expected GitHub Actions warning with title format, got: %s", output)
	}
	if !strings.Contains(output, "config.timeout") {
		t.Errorf("expected path in output, got: %s", output)
	}
}

func TestGitHubFormatter_AllDiffTypes(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	tests := []struct {
		name     string
		diff     Difference
		expected string
	}{
		{"added", Difference{Path: "key", Type: DiffAdded, To: "value"}, "Added:"},
		{"removed", Difference{Path: "key", Type: DiffRemoved, From: "value"}, "Removed:"},
		{"modified", Difference{Path: "key", Type: DiffModified, From: "old", To: "new"}, "Modified:"},
		{"order changed", Difference{Path: "list", Type: DiffOrderChanged}, "Order changed:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in GitHub output, got: %s", tt.expected, output)
			}
		})
	}
}

func TestGitHubFormatter_EmptyOutput(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	output := f.Format([]Difference{}, opts)
	if output != "" {
		t.Errorf("expected empty output for no differences, got: %s", output)
	}
}

func TestGitHubFormatter_DifferentiatedCommands(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	tests := []struct {
		name            string
		diff            Difference
		expectedCommand string
		expectedTitle   string
	}{
		{"added uses notice", Difference{Path: "key", Type: DiffAdded, To: "value"}, "::notice", "title=YAML Added"},
		{"removed uses error", Difference{Path: "key", Type: DiffRemoved, From: "value"}, "::error", "title=YAML Removed"},
		{"modified uses warning", Difference{Path: "key", Type: DiffModified, From: "old", To: "new"}, "::warning", "title=YAML Modified"},
		{"order changed uses notice", Difference{Path: "list", Type: DiffOrderChanged}, "::notice", "title=YAML Order Changed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if !strings.Contains(output, tt.expectedCommand) {
				t.Errorf("expected command %q in output, got: %s", tt.expectedCommand, output)
			}
			if !strings.Contains(output, tt.expectedTitle) {
				t.Errorf("expected title %q in output, got: %s", tt.expectedTitle, output)
			}
		})
	}
}

func TestGitHubFormatter_FileParameter(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()
	opts.FilePath = "deploy.yaml"

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	if !strings.Contains(output, "file=deploy.yaml") {
		t.Errorf("expected file=deploy.yaml in output, got: %s", output)
	}
	expected := "::warning file=deploy.yaml,title=YAML Modified::Modified: config.timeout changed from 30 to 60\n"
	if output != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestGitHubFormatter_NoFileParameter(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.timeout", Type: DiffModified, From: "30", To: "60"},
	}

	output := f.Format(diffs, opts)

	if strings.Contains(output, "file=") {
		t.Errorf("expected no file= parameter when FilePath is empty, got: %s", output)
	}
	expected := "::warning title=YAML Modified::Modified: config.timeout changed from 30 to 60\n"
	if output != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestGitHubFormatter_FileParameterAllDiffTypes(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()
	opts.FilePath = "service.yaml"

	tests := []struct {
		name     string
		diff     Difference
		expected string
	}{
		{"added with file", Difference{Path: "key", Type: DiffAdded, To: "value"}, "::notice file=service.yaml,title=YAML Added::Added: key = value\n"},
		{"removed with file", Difference{Path: "key", Type: DiffRemoved, From: "value"}, "::error file=service.yaml,title=YAML Removed::Removed: key = value\n"},
		{"modified with file", Difference{Path: "key", Type: DiffModified, From: "old", To: "new"}, "::warning file=service.yaml,title=YAML Modified::Modified: key changed from old to new\n"},
		{"order changed with file", Difference{Path: "list", Type: DiffOrderChanged}, "::notice file=service.yaml,title=YAML Order Changed::Order changed: list\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.Format([]Difference{tt.diff}, opts)
			if output != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, output)
			}
		})
	}
}

func TestGitHubFormatter_AnnotationLimitTruncation(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	var diffs []Difference
	for i := 0; i < 13; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("key%d", i),
			Type: DiffModified,
			From: "old",
			To:   "new",
		})
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != 11 {
		t.Fatalf("expected 11 lines (10 warnings + 1 summary), got %d:\n%s", len(lines), output)
	}

	for i := 0; i < 10; i++ {
		if !strings.HasPrefix(lines[i], "::warning ") {
			t.Errorf("line %d should be ::warning, got: %s", i, lines[i])
		}
	}

	expectedSummary := "::warning title=diffyml::3 additional warning annotations omitted due to GitHub Actions limit"
	if lines[10] != expectedSummary {
		t.Errorf("expected summary:\n%s\ngot:\n%s", expectedSummary, lines[10])
	}
}

func TestGitHubFormatter_AnnotationLimitNotTriggered(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	var diffs []Difference
	for i := 0; i < 10; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("key%d", i),
			Type: DiffModified,
			From: "old",
			To:   "new",
		})
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != 10 {
		t.Fatalf("expected 10 lines (no summary), got %d:\n%s", len(lines), output)
	}

	if strings.Contains(output, "omitted due to GitHub Actions limit") {
		t.Errorf("summary should not appear when at or below limit, got: %s", output)
	}
}

func TestGitHubFormatter_AnnotationLimitMixedNotice(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	var diffs []Difference
	for i := 0; i < 7; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("added%d", i),
			Type: DiffAdded,
			To:   "val",
		})
	}
	for i := 0; i < 5; i++ {
		diffs = append(diffs, Difference{
			Path: fmt.Sprintf("order%d", i),
			Type: DiffOrderChanged,
		})
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != 11 {
		t.Fatalf("expected 11 lines (10 notices + 1 summary), got %d:\n%s", len(lines), output)
	}

	noticeCount := 0
	for _, line := range lines[:len(lines)-1] {
		if strings.HasPrefix(line, "::notice ") {
			noticeCount++
		}
	}
	if noticeCount != 10 {
		t.Errorf("expected 10 notice annotations, got %d", noticeCount)
	}

	expectedSummary := "::notice title=diffyml::2 additional notice annotations omitted due to GitHub Actions limit"
	if lines[len(lines)-1] != expectedSummary {
		t.Errorf("expected summary:\n%s\ngot:\n%s", expectedSummary, lines[len(lines)-1])
	}
}

func TestGitHubFormatter_AnnotationLimitMultipleTypes(t *testing.T) {
	f, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	var diffs []Difference
	for i := 0; i < 12; i++ {
		diffs = append(diffs, Difference{Path: fmt.Sprintf("a%d", i), Type: DiffAdded, To: "v"})
	}
	for i := 0; i < 11; i++ {
		diffs = append(diffs, Difference{Path: fmt.Sprintf("m%d", i), Type: DiffModified, From: "o", To: "n"})
	}
	for i := 0; i < 3; i++ {
		diffs = append(diffs, Difference{Path: fmt.Sprintf("r%d", i), Type: DiffRemoved, From: "v"})
	}

	output := f.Format(diffs, opts)

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 25 {
		t.Fatalf("expected 25 lines, got %d:\n%s", len(lines), output)
	}

	if strings.Contains(output, "additional error annotations") {
		t.Errorf("should not have error summary when under limit")
	}
	if !strings.Contains(output, "2 additional notice annotations omitted") {
		t.Errorf("expected notice summary, got:\n%s", output)
	}
	if !strings.Contains(output, "1 additional warning annotations omitted") {
		t.Errorf("expected warning summary, got:\n%s", output)
	}
}

// FormatAll tests

func TestGitHubFormatter_FormatAll(t *testing.T) {
	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs:    []Difference{{Path: "key", Type: DiffModified, From: "old", To: "new"}},
		},
		{
			FilePath: "service.yaml",
			Diffs:    []Difference{{Path: "port", Type: DiffAdded, To: 8080}},
		},
	}

	output := f.FormatAll(groups, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), output)
	}

	expected0 := "::warning file=deploy.yaml,title=YAML Modified::Modified: key changed from old to new"
	expected1 := "::notice file=service.yaml,title=YAML Added::Added: port = 8080"

	if lines[0] != expected0 {
		t.Errorf("line 0:\n  expected: %s\n  got:      %s", expected0, lines[0])
	}
	if lines[1] != expected1 {
		t.Errorf("line 1:\n  expected: %s\n  got:      %s", expected1, lines[1])
	}
}

func TestGitHubFormatter_FormatAllEmpty(t *testing.T) {
	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	output := f.FormatAll([]DiffGroup{}, opts)
	if output != "" {
		t.Errorf("expected empty string for empty groups, got: %q", output)
	}

	output = f.FormatAll([]DiffGroup{
		{FilePath: "deploy.yaml", Diffs: []Difference{}},
		{FilePath: "service.yaml", Diffs: []Difference{}},
	}, opts)
	if output != "" {
		t.Errorf("expected empty string when all groups have zero diffs, got: %q", output)
	}
}

func TestGitHubFormatter_FormatAll_AnnotationLimitsAcrossGroups(t *testing.T) {
	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{FilePath: "a.yaml", Diffs: makeDiffs(DiffModified, 5)},
		{FilePath: "b.yaml", Diffs: makeDiffs(DiffModified, 5)},
		{FilePath: "c.yaml", Diffs: makeDiffs(DiffModified, 3)},
	}

	output := f.FormatAll(groups, opts)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != 11 {
		t.Fatalf("expected 11 lines (10 warnings + 1 summary), got %d:\n%s", len(lines), output)
	}

	lastLine := lines[10]
	expectedSummary := "::warning title=diffyml::3 additional warning annotations omitted due to GitHub Actions limit"
	if lastLine != expectedSummary {
		t.Errorf("expected summary:\n%s\ngot:\n%s", expectedSummary, lastLine)
	}
	if strings.Contains(lastLine, "file=") {
		t.Errorf("summary annotation should not include file= parameter, got: %s", lastLine)
	}
}

// Gitea tests — Gitea delegates to GitHub

func TestGiteaFormatter_GitHubCompatible(t *testing.T) {
	giteaF, _ := GetFormatter("gitea")
	githubF, _ := GetFormatter("github")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "config.value", Type: DiffModified, From: "old", To: "new"},
	}

	giteaOutput := giteaF.Format(diffs, opts)
	githubOutput := githubF.Format(diffs, opts)

	if giteaOutput != githubOutput {
		t.Errorf("Gitea output should match GitHub output\nGitea: %s\nGitHub: %s", giteaOutput, githubOutput)
	}
}

func TestGiteaFormatter_FormatAll(t *testing.T) {
	giteaF := &GiteaFormatter{}
	githubF := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs:    []Difference{{Path: "key", Type: DiffModified, From: "old", To: "new"}},
		},
		{
			FilePath: "service.yaml",
			Diffs:    []Difference{{Path: "port", Type: DiffAdded, To: 8080}},
		},
	}

	giteaOutput := giteaF.FormatAll(groups, opts)
	githubOutput := githubF.FormatAll(groups, opts)

	if giteaOutput != githubOutput {
		t.Errorf("Gitea FormatAll should match GitHub FormatAll\nGitea:  %s\nGitHub: %s", giteaOutput, githubOutput)
	}
}
