// detailed_formatter_linediff.go - LCS line-diff algorithm for detailed output.
//
// Computes line-level diffs using the Longest Common Subsequence algorithm
// and renders inline diffs with context collapsing for multiline strings.
package diffyml

import (
	"fmt"
	"slices"
	"strings"
)

// editOpType represents a type of edit operation in a line diff.
type editOpType int

const (
	editKeep editOpType = iota
	editInsert
	editDelete
)

// editOp represents a single edit operation in a line diff.
type editOp struct {
	Type editOpType
	Line string
}

// formatMultilineDiff renders an inline line-by-line diff for multiline strings.
func (f *DetailedFormatter) formatMultilineDiff(sb *strings.Builder, from, to string, opts *FormatOptions) {
	fromLines := strings.Split(from, "\n")
	toLines := strings.Split(to, "\n")
	ops := computeLineDiff(fromLines, toLines)

	// Count additions and deletions
	additions := 0
	deletions := 0
	for _, op := range ops {
		switch op.Type {
		case editInsert:
			additions++
		case editDelete:
			deletions++
		}
	}

	descriptor := fmt.Sprintf("  ± value change in multiline text (%s %s, %s %s)",
		formatCount(additions), pluralize(additions, "insert", "inserts"),
		formatCount(deletions), pluralize(deletions, "deletion", "deletions"))
	f.writeDescriptorLine(sb, descriptor, f.colorModified, opts)

	// Apply context collapsing
	contextLines := opts.ContextLines
	if contextLines < 0 {
		contextLines = 4
	}

	// Mark which ops are near a change
	nearChange := make([]bool, len(ops))
	for i, op := range ops {
		if op.Type != editKeep {
			// Mark surrounding context
			for j := max(0, i-contextLines); j <= min(len(ops)-1, i+contextLines); j++ {
				nearChange[j] = true
			}
		}
	}

	// Render with collapsing
	skipUntil := 0
	for i, op := range ops {
		if i < skipUntil {
			continue
		}
		if op.Type != editKeep || nearChange[i] {
			switch op.Type {
			case editKeep:
				f.writeColoredLine(sb, fmt.Sprintf("      %s", op.Line), f.colorContext(opts), opts)
			case editInsert:
				f.writeColoredLine(sb, fmt.Sprintf("    + %s", op.Line), f.colorAdded(opts), opts)
			case editDelete:
				f.writeColoredLine(sb, fmt.Sprintf("    - %s", op.Line), f.colorRemoved(opts), opts)
			}
		} else {
			// Count consecutive non-near-change keep ops
			collapsed := 0
			for _, sub := range ops[i:] {
				if sub.Type != editKeep || nearChange[i+collapsed] {
					break
				}
				collapsed++
			}
			skipUntil = i + collapsed
			f.writeColoredLine(sb, fmt.Sprintf("    [%d %s unchanged]", collapsed, pluralize(collapsed, "line", "lines")), f.colorContext(opts), opts)
		}
	}
	sb.WriteString("\n")
}

// computeLineDiff computes line-level diff using LCS algorithm.
func computeLineDiff(fromLines, toLines []string) []editOp {
	m := len(fromLines)
	n := len(toLines)

	// Build LCS table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
			if fromLines[i-1] == toLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to produce edit operations
	var ops []editOp
	i, j := m, n
	for i > 0 || j > 0 {
		//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
		if i > 0 && j > 0 && fromLines[i-1] == toLines[j-1] {
			ops = append(ops, editOp{Type: editKeep, Line: fromLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, editOp{Type: editInsert, Line: toLines[j-1]})
			j--
		} else {
			ops = append(ops, editOp{Type: editDelete, Line: fromLines[i-1]})
			i--
		}
	}

	// Reverse to get correct order
	slices.Reverse(ops)

	return ops
}
