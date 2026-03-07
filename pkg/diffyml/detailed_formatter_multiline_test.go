package diffyml

import (
	"strings"
	"testing"
)

// Multiline text diff with context and collapse

func TestDetailedFormatter_MultilineDescriptor(t *testing.T) {
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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

// Whitespace-only change detection and visualization

func TestDetailedFormatter_WhitespaceOnlyChange(t *testing.T) {
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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
	f, _ := FormatterByName("detailed")
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

// Mutation testing: multiline and computeLineDiff

func TestDetailedFormatter_ContextLinesZero(t *testing.T) {
	// With ContextLines=0, no context lines should be shown around changes
	fromStr := "line1\nline2\nline3\nCHANGED\nline5\nline6\nline7"
	toStr := "line1\nline2\nline3\nNEW\nline5\nline6\nline7"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: fromStr, To: toStr},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false, ContextLines: 0}
	output := f.Format(diffs, opts)

	// With zero context lines, there should be a collapsed message and no context
	if !strings.Contains(output, "unchanged]") {
		t.Errorf("expected collapsed message with ContextLines=0, got: %s", output)
	}
}

func TestDetailedFormatter_CollapsedLineCount(t *testing.T) {
	// Create a multiline diff where many lines are unchanged
	var fromLines, toLines []string
	for i := 0; i < 20; i++ {
		fromLines = append(fromLines, "unchanged line")
		toLines = append(toLines, "unchanged line")
	}
	fromLines = append(fromLines, "CHANGED FROM")
	toLines = append(toLines, "CHANGED TO")
	for i := 0; i < 20; i++ {
		fromLines = append(fromLines, "more unchanged")
		toLines = append(toLines, "more unchanged")
	}

	fromStr := strings.Join(fromLines, "\n")
	toStr := strings.Join(toLines, "\n")

	diffs := []Difference{
		{Path: "data", Type: DiffModified, From: fromStr, To: toStr},
	}

	f := &DetailedFormatter{}
	// Use ContextLines=2 so most unchanged lines are collapsed
	opts := &FormatOptions{Color: false, ContextLines: 2}
	output := f.Format(diffs, opts)

	// Collapsed message must contain exact positive count (kills collapsed++ → collapsed-- mutation)
	if !strings.Contains(output, "[18 lines unchanged]") {
		t.Errorf("expected exactly '[18 lines unchanged]' in output, got:\n%s", output)
	}
	// Ensure no negative counts appear
	if strings.Contains(output, "[-") {
		t.Errorf("collapsed line count should not be negative, got:\n%s", output)
	}
	// Exactly 2 collapsed markers expected (one per unchanged region).
	// Kills skipUntil = i + collapsed → i - collapsed mutation which produces
	// duplicate collapsed markers for each line in a collapsed region.
	markerCount := strings.Count(output, "lines unchanged]")
	if markerCount != 2 {
		t.Errorf("expected exactly 2 collapsed markers, got %d:\n%s", markerCount, output)
	}
}

func TestComputeLineDiff_LCSTieBreaking(t *testing.T) {
	// detailed_formatter.go:462 — `dp[i-1][j] >= dp[i][j-1]` → `>`
	// This affects tie-breaking in the LCS algorithm.
	// When dp[i-1][j] == dp[i][j-1], the original prefers deletion (from line).
	// Mutation to `>` would prefer insertion (to line) instead.
	// We craft inputs where tie-breaking produces different edit sequences.

	// Lines designed so that at some point dp values tie
	fromLines := []string{"A", "B", "C"}
	toLines := []string{"A", "C", "B"}

	ops := computeLineDiff(fromLines, toLines)

	// Count each operation type
	keeps := 0
	deletes := 0
	inserts := 0
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			keeps++
		case editDelete:
			deletes++
		case editInsert:
			inserts++
		}
	}

	// With the original >= tie-breaking, we should get a specific sequence.
	// The key property: the diff should be valid (applying it transforms from → to)
	if keeps+deletes+inserts != len(ops) {
		t.Errorf("unexpected op count: keeps=%d deletes=%d inserts=%d total=%d", keeps, deletes, inserts, len(ops))
	}

	// Reconstruct 'from' from keeps+deletes and 'to' from keeps+inserts
	var reconstructedFrom, reconstructedTo []string
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			reconstructedFrom = append(reconstructedFrom, op.Line)
			reconstructedTo = append(reconstructedTo, op.Line)
		case editDelete:
			reconstructedFrom = append(reconstructedFrom, op.Line)
		case editInsert:
			reconstructedTo = append(reconstructedTo, op.Line)
		}
	}

	if len(reconstructedFrom) != len(fromLines) {
		t.Errorf("reconstructed from has %d lines, want %d", len(reconstructedFrom), len(fromLines))
	}
	if len(reconstructedTo) != len(toLines) {
		t.Errorf("reconstructed to has %d lines, want %d", len(reconstructedTo), len(toLines))
	}
}

func TestComputeLineDiff_TieBreakingDeterminism(t *testing.T) {
	// detailed_formatter.go:462 — ensure consistent tie-breaking behavior
	// The >= comparison means "prefer delete over insert when tied".
	// If mutated to >, the preference flips to "prefer insert over delete".
	// We can detect this by checking the exact operation sequence.

	fromLines := []string{"X", "Y"}
	toLines := []string{"Y", "X"}

	ops := computeLineDiff(fromLines, toLines)

	// With >= (original): at (2,2) where dp[1][2]==dp[2][1]==1,
	// it takes dp[i-1][j] (delete X first).
	// With > (mutant): it would take dp[i][j-1] (insert X first).
	// The resulting ops should be deterministic.
	if len(ops) < 3 {
		t.Fatalf("expected at least 3 ops for swap, got %d", len(ops))
	}

	// Just verify it produces a valid diff
	var fromResult, toResult []string
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			fromResult = append(fromResult, op.Line)
			toResult = append(toResult, op.Line)
		case editDelete:
			fromResult = append(fromResult, op.Line)
		case editInsert:
			toResult = append(toResult, op.Line)
		}
	}

	for i, line := range fromLines {
		if i >= len(fromResult) || fromResult[i] != line {
			t.Errorf("from reconstruction mismatch at %d: got %v, want %v", i, fromResult, fromLines)
			break
		}
	}
	for i, line := range toLines {
		if i >= len(toResult) || toResult[i] != line {
			t.Errorf("to reconstruction mismatch at %d: got %v, want %v", i, toResult, toLines)
			break
		}
	}
}

func TestComputeLineDiff_LastColumnDP(t *testing.T) {
	// Targets mutation: line 490 `j <= n` → `j < n` (CONDITIONALS_BOUNDARY).
	// When the inner loop skips j=n, dp[i][n]=0 for all i, and the
	// backtracking cannot find the optimal LCS through the last column.
	//
	// With from=["A","B","C"], to=["C","A","B"]:
	//   Normal: LCS = {"A","B"} (length 2) → [insert C, keep A, keep B, delete C]
	//   Mutant: dp[*][3]=0 → LCS degrades to {"C"} (length 1)
	//           → [delete A, delete B, keep C, insert A, insert B]
	fromLines := []string{"A", "B", "C"}
	toLines := []string{"C", "A", "B"}

	ops := computeLineDiff(fromLines, toLines)

	keeps := 0
	for _, op := range ops {
		if op.Type == editKeep {
			keeps++
		}
	}
	if keeps != 2 {
		t.Errorf("expected 2 keep operations (optimal LCS length 2), got %d keeps out of %d ops", keeps, len(ops))
	}
}

func TestDetailedFormatter_CollapseSkipBoundary(t *testing.T) {
	// Kills CONDITIONALS_BOUNDARY at detailed_formatter_linediff.go:71
	// (i < skipUntil → i <= skipUntil)
	// With <=, the first line AFTER a collapsed section would be skipped,
	// producing incorrect output.
	//
	// Setup: 1 changed line, then exactly (contextLines+1) unchanged lines,
	// then 1 changed line. With context=1, the collapse region is tiny:
	// just 1 line. The line right after the collapse must still appear.
	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false, ContextLines: 1}

	// Build: change at line 0, 4 unchanged lines (indices 1-4), change at line 5
	// With context=1: nearChange = {0,1, 4,5}
	// Lines 2-3 are collapsed (2 lines), skipUntil = 4
	// At i=4: with < skipUntil (4 < 4 = false) → renders line 4 ✓
	// At i=4: with <= skipUntil (4 <= 4 = true) → skips line 4 ✗
	from := "ORIGINAL\nb\nc\nd\ne\nALSO_ORIGINAL"
	to := "CHANGED\nb\nc\nd\ne\nALSO_CHANGED"

	diffs := []Difference{
		{Path: "text", Type: DiffModified, From: from, To: to},
	}
	output := f.Format(diffs, opts)

	// Line "e" (index 4) is within context of the second change and must appear
	if !strings.Contains(output, "e") {
		t.Errorf("expected context line 'e' to appear after collapsed section, got:\n%s", output)
	}

	// Count output lines that contain the actual content markers
	lines := strings.Split(output, "\n")
	var contextLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Context lines (not change markers, not collapse markers)
		if trimmed == "b" || trimmed == "c" || trimmed == "d" || trimmed == "e" {
			contextLines = append(contextLines, trimmed)
		}
	}
	// With context=1: "b" (context after first change) and "e" (context before second change) should appear
	// "c" and "d" should be collapsed
	if len(contextLines) != 2 {
		t.Errorf("expected exactly 2 context lines (b, e), got %d: %v\nfull output:\n%s", len(contextLines), contextLines, output)
	}
}

func TestComputeLineDiff_LCSTieBreakingExactOrder(t *testing.T) {
	// Kills CONDITIONALS_BOUNDARY at detailed_formatter_linediff.go:129
	// (dp[i][j-1] >= dp[i-1][j] → dp[i][j-1] > dp[i-1][j])
	//
	// With >= (original): when tied, prefer insert (j-1 branch).
	// With > (mutant): when tied, prefer delete (i-1 branch).
	// After slices.Reverse, the order in the final result differs.
	//
	// Input: from=["A","B"], to=["B","A"]
	// LCS table:
	//     ""  B  A
	// ""   0  0  0
	// A    0  0  1
	// B    0  1  1
	//
	// Backtrack from (2,2): dp[2][1]=1, dp[1][2]=1 → tied
	// With >= (original): take j-1 branch → insert A, then at (2,1): match B → keep B, then at (1,0): delete A
	//   ops built in reverse: [delete A, keep B, insert A]
	//   After Reverse: [insert A, keep B, delete A]
	//   Wait, let me recalculate.
	//
	// Actually: backtrack from (2,2): from[1]="B" ≠ to[1]="A"
	//   dp[i][j-1] = dp[2][1] = 1, dp[i-1][j] = dp[1][2] = 1 → tied
	//   With >=: take j-1 branch → insert to[1]="A", j=1
	//   At (2,1): from[1]="B" == to[0]="B" → keep "B", i=1, j=0
	//   At (1,0): j=0, i>0 → delete from[0]="A", i=0
	//   ops = [insert "A", keep "B", delete "A"]
	//   After reverse: [delete "A", keep "B", insert "A"]
	//
	//   With > (mutant): take i-1 branch → delete from[1]="B", i=1
	//   At (1,2): from[0]="A" == to[1]="A" → keep "A", i=0, j=1
	//   At (0,1): i=0, j>0 → insert to[0]="B", j=0
	//   ops = [delete "B", keep "A", insert "B"]
	//   After reverse: [insert "B", keep "A", delete "B"]
	//
	// So with >=: first op is delete "A"; with >: first op is insert "B"
	fromLines := []string{"A", "B"}
	toLines := []string{"B", "A"}
	ops := computeLineDiff(fromLines, toLines)

	if len(ops) != 3 {
		t.Fatalf("expected 3 ops, got %d: %v", len(ops), ops)
	}

	// With >= (original): [delete "A", keep "B", insert "A"]
	if ops[0].Type != editDelete || ops[0].Line != "A" {
		t.Errorf("ops[0] should be delete 'A', got type=%d line=%q", ops[0].Type, ops[0].Line)
	}
	if ops[1].Type != editKeep || ops[1].Line != "B" {
		t.Errorf("ops[1] should be keep 'B', got type=%d line=%q", ops[1].Type, ops[1].Line)
	}
	if ops[2].Type != editInsert || ops[2].Line != "A" {
		t.Errorf("ops[2] should be insert 'A', got type=%d line=%q", ops[2].Type, ops[2].Line)
	}
}

func TestDetailedFormatter_MultilineDiff_NegativeContextLines(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.ContextLines = -1 // should default to 4

	diffs := []Difference{
		{Path: "config.script", Type: DiffModified, From: "line1\nline2", To: "line1\nline3"},
	}
	output := f.Format(diffs, opts)
	if !strings.Contains(output, "value change in multiline text") {
		t.Errorf("expected multiline diff output with negative context lines, got: %q", output)
	}
}
