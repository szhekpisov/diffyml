// detailed_formatter_linediff.go - Myers line-diff algorithm for detailed output.
//
// Computes line-level diffs using the Myers diff algorithm (Eugene Myers, 1986)
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

// countEditOps counts the number of insertions and deletions in a sequence of edit operations.
func countEditOps(ops []editOp) (additions, deletions int) {
	for _, op := range ops {
		switch op.Type {
		case editInsert:
			additions++
		case editDelete:
			deletions++
		}
	}
	return additions, deletions
}

// lineDiffChunkType classifies a chunk of a collapsed line diff.
type lineDiffChunkType int

const (
	chunkKeep lineDiffChunkType = iota
	chunkInsert
	chunkDelete
	chunkCollapsed
)

// lineDiffChunk is one unit of a collapsed line diff: either a single line
// (keep/insert/delete) or a run of unchanged lines summarized by Collapsed.
type lineDiffChunk struct {
	Type      lineDiffChunkType
	Line      string
	Collapsed int
}

// resolveContextLines returns the effective context line count.
// A negative value falls back to the default of 4.
func resolveContextLines(contextLines int) int {
	if contextLines < 0 {
		return 4
	}
	return contextLines
}

// markNearChange marks which ops lie within contextLines of an insert or delete.
// Unmarked keep ops belong to runs that collapse into a summary marker.
func markNearChange(ops []editOp, contextLines int) []bool {
	contextLines = resolveContextLines(contextLines)
	nearChange := make([]bool, len(ops))
	for i, op := range ops {
		if op.Type != editKeep {
			for j := max(0, i-contextLines); j <= min(len(ops)-1, i+contextLines); j++ {
				nearChange[j] = true
			}
		}
	}
	return nearChange
}

// collapseLineDiff turns edit operations into chunks, replacing every run of
// unchanged lines that is not near a change with a single collapsed chunk.
func collapseLineDiff(ops []editOp, nearChange []bool) []lineDiffChunk {
	var chunks []lineDiffChunk
	skipUntil := 0
	for i, op := range ops {
		if i < skipUntil {
			continue
		}
		if op.Type != editKeep || nearChange[i] {
			switch op.Type {
			case editKeep:
				chunks = append(chunks, lineDiffChunk{Type: chunkKeep, Line: op.Line})
			case editInsert:
				chunks = append(chunks, lineDiffChunk{Type: chunkInsert, Line: op.Line})
			case editDelete:
				chunks = append(chunks, lineDiffChunk{Type: chunkDelete, Line: op.Line})
			}
			continue
		}
		end := i
		for end < len(ops) && ops[end].Type == editKeep && !nearChange[end] {
			end++
		}
		skipUntil = end
		chunks = append(chunks, lineDiffChunk{Type: chunkCollapsed, Collapsed: end - i})
	}
	return chunks
}

// collapsedRunLabel renders the summary marker for a collapsed run of lines.
func collapsedRunLabel(collapsed int) string {
	return fmt.Sprintf("[%d %s unchanged]", collapsed, pluralize(collapsed, "line", "lines"))
}

// renderLineDiffOps renders edit operations with context collapsing.
func (f *DetailedFormatter) renderLineDiffOps(sb *strings.Builder, ops []editOp, nearChange []bool, opts *FormatOptions) {
	for _, chunk := range collapseLineDiff(ops, nearChange) {
		switch chunk.Type {
		case chunkKeep:
			f.writeColoredLine(sb, fmt.Sprintf("      %s", chunk.Line), f.colorContext(opts), opts)
		case chunkInsert:
			f.writeColoredLine(sb, fmt.Sprintf("    + %s", chunk.Line), f.colorAdded(opts), opts)
		case chunkDelete:
			f.writeColoredLine(sb, fmt.Sprintf("    - %s", chunk.Line), f.colorRemoved(opts), opts)
		case chunkCollapsed:
			f.writeColoredLine(sb, fmt.Sprintf("    %s", collapsedRunLabel(chunk.Collapsed)), f.colorContext(opts), opts)
		}
	}
}

// formatMultilineDiff renders an inline line-by-line diff for multiline strings.
func (f *DetailedFormatter) formatMultilineDiff(sb *strings.Builder, from, to string, opts *FormatOptions) {
	fromLines := strings.Split(from, "\n")
	toLines := strings.Split(to, "\n")
	ops := computeLineDiff(fromLines, toLines)

	additions, deletions := countEditOps(ops)

	descriptor := fmt.Sprintf("  ± value change in multiline text (%s %s, %s %s)",
		formatCount(additions), pluralize(additions, "insert", "inserts"),
		formatCount(deletions), pluralize(deletions, "deletion", "deletions"))
	f.writeDescriptorLine(sb, descriptor, f.colorModified, opts)

	f.renderLineDiffOps(sb, ops, markNearChange(ops, opts.ContextLines), opts)
	sb.WriteString("\n")
}

// unboundedEditDistance lets computeLineDiffBounded search the whole edit
// script, however long it turns out to be.
const unboundedEditDistance = -1

// computeLineDiff computes line-level diff using the Myers diff algorithm.
// It finds the shortest edit script (SES) in O(ND) time where N=m+n and D=edit distance.
func computeLineDiff(fromLines, toLines []string) []editOp {
	ops, _ := computeLineDiffBounded(fromLines, toLines, unboundedEditDistance)
	return ops
}

// computeLineDiffBounded is computeLineDiff with a ceiling on how far it
// searches. The forward pass keeps one (2*(m+n)+1)-int row per step of the edit
// script, so memory is O(D*(m+n)) — quadratic for two values that share no
// lines. Capping D caps that at O(maxD*(m+n)), linear in the input, at the cost
// of producing no diff at all for values that differ by more than maxD lines:
// ok is false in that case and the caller is expected to fall back to something
// cheaper. Pass unboundedEditDistance to search without a ceiling, which always
// succeeds since D never exceeds m+n.
func computeLineDiffBounded(fromLines, toLines []string, maxD int) (ops []editOp, ok bool) {
	m := len(fromLines)
	n := len(toLines)

	// Forward pass: find shortest edit script.
	// V[k+offset] stores the furthest-reaching x-coordinate on diagonal k.
	offset := m + n
	vSize := 2*(m+n) + 1
	v := make([]int, vSize)
	var trace [][]int
	finalD := 0
	found := false

	limit := m + n
	if maxD >= 0 && maxD < limit {
		limit = maxD
	}

	for d := range limit + 1 {
		snapshot := make([]int, vSize)
		copy(snapshot, v)
		trace = append(trace, snapshot)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+offset] < v[k+1+offset]) {
				x = v[k+1+offset]
			} else {
				x = v[k-1+offset] + 1
			}
			y := x - k

			for x < m && y < n && fromLines[x] == toLines[y] {
				x++
				y++
			}

			v[k+offset] = x

			if x == m && y == n {
				finalD = d
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	// The search ran out of budget before reaching the end of both inputs.
	if !found {
		return nil, false
	}

	// Backtrack through trace to produce edit operations.
	x, y := m, n

	for d := finalD; d > 0; d-- {
		prev := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && prev[k-1+offset] < prev[k+1+offset]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := prev[prevK+offset]
		prevY := prevX - prevK

		// Record diagonal matches (snake) in reverse
		for x > prevX && y > prevY {
			x--
			y--
			ops = append(ops, editOp{Type: editKeep, Line: fromLines[x]})
		}

		// Record the non-diagonal move
		if prevK == k-1 {
			x--
			ops = append(ops, editOp{Type: editDelete, Line: fromLines[x]})
		} else {
			y--
			ops = append(ops, editOp{Type: editInsert, Line: toLines[y]})
		}
	}

	// Record any remaining diagonal matches from the d=0 snake
	for x > 0 {
		x--
		ops = append(ops, editOp{Type: editKeep, Line: fromLines[x]})
	}

	slices.Reverse(ops)

	return ops, true
}
