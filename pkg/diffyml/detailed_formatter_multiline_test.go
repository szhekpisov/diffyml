package diffyml

import (
	"strings"
	"testing"
)

// Task 3.2: Multiline text diff with context and collapse

func TestDetailedFormatter_MultilineDescriptor(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "line1\nline2\nline3"
	to := "line1\nchanged\nline3"

	diffs := []Difference{
		{Path: "config.data", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± value change in multiline text") {
		t.Errorf("expected multiline descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineAdditionDeletionCount(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "line1\nline2\nline3"
	to := "line1\nchanged\nline3\nline4"

	diffs := []Difference{
		{Path: "data", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	// Should mention inserts and deletions count
	if !strings.Contains(output, "insert") || !strings.Contains(output, "deletion") {
		t.Errorf("expected insert/deletion counts in multiline descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineDiffMarkers(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "aaa\nbbb\nccc"
	to := "aaa\nBBB\nccc"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	lines := strings.Split(output, "\n")

	hasAdded := false
	hasRemoved := false
	hasContext := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "+ ") {
			hasAdded = true
		}
		if strings.HasPrefix(trimmed, "- ") {
			hasRemoved = true
		}
		// Context lines start with space
		if len(trimmed) > 0 && strings.HasPrefix(line, "    ") && !strings.HasPrefix(trimmed, "+") && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "±") {
			hasContext = true
		}
	}

	if !hasAdded {
		t.Errorf("expected '+' prefixed added lines in multiline diff, got: %q", output)
	}
	if !hasRemoved {
		t.Errorf("expected '-' prefixed removed lines in multiline diff, got: %q", output)
	}
	if !hasContext {
		t.Errorf("expected context lines (space-prefixed) in multiline diff, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineCollapseUnchanged(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.ContextLines = 1

	// Many unchanged lines between changes should be collapsed
	from := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	to := "CHANGED\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nALSO_CHANGED"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "lines unchanged") {
		t.Errorf("expected collapse marker '[N lines unchanged]', got: %q", output)
	}
}

func TestDetailedFormatter_MultilineContextLinesOption(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.ContextLines = 1

	// With 8 unchanged lines between two changes and context=1, most should collapse
	from := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj"
	to := "CHANGED\nb\nc\nd\ne\nf\ng\nh\ni\nCHANGED"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	// With context=1, many middle lines should be collapsed
	if !strings.Contains(output, "lines unchanged") {
		t.Errorf("expected collapse with context=1, got: %q", output)
	}
}

func TestDetailedFormatter_SingleLineNotMultiline(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	// Single-line strings should NOT use multiline diff path
	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "old value", To: "new value"},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "multiline") {
		t.Errorf("single-line change should not use multiline diff, got: %q", output)
	}
	if !strings.Contains(output, "± value change") {
		t.Errorf("expected '± value change' for single-line change, got: %q", output)
	}
}

func TestComputeLineDiff(t *testing.T) {
	from := []string{"a", "b", "c"}
	to := []string{"a", "B", "c"}

	ops := computeLineDiff(from, to)

	hasKeep := false
	hasInsert := false
	hasDelete := false
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			hasKeep = true
		case editInsert:
			hasInsert = true
		case editDelete:
			hasDelete = true
		}
	}

	if !hasKeep {
		t.Error("expected keep operations in line diff")
	}
	if !hasInsert {
		t.Error("expected insert operations in line diff")
	}
	if !hasDelete {
		t.Error("expected delete operations in line diff")
	}
}

func TestComputeLineDiff_AllNew(t *testing.T) {
	ops := computeLineDiff([]string{}, []string{"a", "b"})

	for _, op := range ops {
		if op.Type != editInsert {
			t.Errorf("expected all insert ops for new content, got type %d", op.Type)
		}
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 insert ops, got %d", len(ops))
	}
}

func TestComputeLineDiff_AllRemoved(t *testing.T) {
	ops := computeLineDiff([]string{"a", "b"}, []string{})

	for _, op := range ops {
		if op.Type != editDelete {
			t.Errorf("expected all delete ops for removed content, got type %d", op.Type)
		}
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 delete ops, got %d", len(ops))
	}
}

func TestComputeLineDiff_Identical(t *testing.T) {
	ops := computeLineDiff([]string{"a", "b"}, []string{"a", "b"})

	for _, op := range ops {
		if op.Type != editKeep {
			t.Errorf("expected all keep ops for identical content, got type %d", op.Type)
		}
	}
}

// Task 3.3: Whitespace-only change detection and visualization

func TestDetailedFormatter_WhitespaceOnlyChange(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "hello world", To: "hello  world"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± whitespace only change") {
		t.Errorf("expected '± whitespace only change' descriptor, got: %q", output)
	}
}

func TestDetailedFormatter_WhitespaceVisualization(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "a b", To: "a  b"},
	}

	output := f.Format(diffs, opts)
	// Spaces should be visualized as middle dots
	if !strings.Contains(output, "·") {
		t.Errorf("expected middle dot '·' for whitespace visualization, got: %q", output)
	}
}

func TestDetailedFormatter_WhitespaceNewlineVisualization(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	diffs := []Difference{
		{Path: "key", Type: DiffModified, From: "hello\n", To: "hello"},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "± whitespace only change") {
		t.Errorf("expected whitespace-only descriptor for trailing newline change, got: %q", output)
	}
	if !strings.Contains(output, "↵") {
		t.Errorf("expected return symbol '↵' for newline visualization, got: %q", output)
	}
}

func TestIsWhitespaceOnlyChange(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected bool
	}{
		{"hello world", "hello  world", true},
		{"hello", "hello\n", true},
		{" a ", "a", true},
		{"hello", "world", false},
		{"abc", "abc", false}, // no change at all
		{"a b", "a c", false},
	}

	for _, tt := range tests {
		result := isWhitespaceOnlyChange(tt.from, tt.to)
		if result != tt.expected {
			t.Errorf("isWhitespaceOnlyChange(%q, %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
		}
	}
}

func TestVisualizeWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a b", "a·b"},
		{"hello\n", "hello↵"},
		{"a  b\n", "a··b↵"},
		{"no spaces", "no·spaces"},
	}

	for _, tt := range tests {
		result := visualizeWhitespace(tt.input)
		if result != tt.expected {
			t.Errorf("visualizeWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDetailedFormatter_MultilineDiffNoCollapse(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()
	opts.ContextLines = 100 // Very high, nothing should collapse

	from := "a\nb\nc\nd\ne"
	to := "a\nB\nc\nd\ne"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if strings.Contains(output, "unchanged") {
		t.Errorf("expected no collapse with large context, got: %q", output)
	}
}

func TestDetailedFormatter_MultilineDiffSingleAddition(t *testing.T) {
	f, _ := GetFormatter("detailed")
	opts := DefaultFormatOptions()

	from := "line1\nline2"
	to := "line1\nline2\nline3"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}

	output := f.Format(diffs, opts)
	if !strings.Contains(output, "one insert") {
		t.Errorf("expected 'one insert' (singular), got: %q", output)
	}
	if !strings.Contains(output, "zero deletions") {
		t.Errorf("expected 'zero deletions', got: %q", output)
	}
}
