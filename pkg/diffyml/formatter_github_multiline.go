// formatter_github_multiline.go - Bounded multiline values for GitHub Actions annotations.
//
// A GitHub Actions annotation is a single workflow command line, so a raw
// multiline YAML value would both blow up the message and terminate the command
// at its first newline. Modified values are rendered as a collapsed line diff
// (the same context window the detailed formatter uses); added and removed
// values, which have no counterpart to diff against, are truncated instead.
// Both paths end in a hard line cap, so no single annotation runs to an
// unbounded number of lines: collapsing alone cannot shrink a value whose lines
// all changed. A wholesale rewrite also skips the diff entirely rather than pay
// for one it cannot show — see gitHubMaxEditDistance.
package diffyml

import (
	"fmt"
	"strings"
)

// gitHubMaxValueLines caps how many lines of an added or removed multiline
// value appear in an annotation before the remainder is summarized. Unlike a
// modified value there is nothing to diff against, so the context window does
// not apply and a fixed cap keeps the annotation readable.
const gitHubMaxValueLines = 20

// gitHubMaxDiffLines caps the body of a rendered line diff. The cap is higher
// than gitHubMaxValueLines because the body is already collapsed: every line
// that survives is either a change or context the caller asked for. Without it
// a wholly rewritten value still emits every insert and delete, since context
// collapsing only ever removes *unchanged* lines.
const gitHubMaxDiffLines = 40

// gitHubMaxEditDistance caps how different two values may be before the line
// diff is abandoned in favor of plain truncation. The rendered body already
// stops at gitHubMaxDiffLines, so an edit script twice that long overflows
// anything the annotation could show; all a full diff still buys past this
// point is an exact insert/deletion count in the header, and computeLineDiff
// pays O(D*(m+n)) memory for it — 63 MiB to render 40 lines of a 1000-line
// rewrite, 1.6 GiB for 8000. Bounding D keeps the cost linear in the value
// size. Only wholesale rewrites reach the ceiling; a value with a handful of
// changed lines has a small edit distance however long the value is, so the
// case collapsing exists for is never the case that falls back.
const gitHubMaxEditDistance = 2 * gitHubMaxDiffLines

// escapeGitHubData percent-encodes data for a GitHub Actions workflow command.
// Without this a message containing a newline would terminate the command
// early, dropping everything after the first line into the raw build log.
// Order matters: "%" must be encoded before the escapes that introduce it.
func escapeGitHubData(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// escapeGitHubProperty percent-encodes a workflow command property value such
// as file= or title=. Beyond the data escapes, ":" and "," delimit the property
// list itself: an unescaped comma in a file path ends the value early, so
// "file=a,b.yaml,title=X" makes GitHub read "b.yaml" as a property name and
// drop the title.
func escapeGitHubProperty(s string) string {
	// escapeGitHubData encodes "%" first, so the escapes introduced below are
	// not themselves re-encoded.
	s = escapeGitHubData(s)
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, ",", "%2C")
	return s
}

// truncateLines keeps at most maxLines entries, replacing the remainder with a
// single summary marker. The full slice expression keeps the caller's backing
// array intact, so appending the marker cannot clobber a dropped line.
func truncateLines(lines []string, maxLines int) []string {
	if len(lines) <= maxLines {
		return lines
	}
	remaining := len(lines) - maxLines
	return append(lines[:maxLines:maxLines],
		fmt.Sprintf("[%d more %s]", remaining, pluralize(remaining, "line", "lines")))
}

// isMultiline reports whether s spans more than one line.
func isMultiline(s string) bool {
	return strings.Contains(s, "\n")
}

// multilineStrings reports whether both values are strings and at least one of
// them is multiline, mirroring the detection the detailed formatter uses to
// switch to a line-by-line diff.
func multilineStrings(from, to any) (fromStr, toStr string, ok bool) {
	fromStr, fromOk := from.(string)
	toStr, toOk := to.(string)
	if !fromOk || !toOk {
		return "", "", false
	}
	return fromStr, toStr, isMultiline(fromStr) || isMultiline(toStr)
}

// githubMultilineDiff renders a modified multiline value as a collapsed line
// diff: a header counting the edits, then the changed lines with contextLines
// of surrounding context, with every other run of unchanged lines replaced by a
// single summary marker. The body is capped at gitHubMaxDiffLines; the header
// sits outside the cap so the edit counts survive truncation and still describe
// the whole change.
//
// ok is false when the two values differ by more than gitHubMaxEditDistance
// lines, which is where computing a diff costs more than the diff can show. The
// caller falls back to truncating both values.
func githubMultilineDiff(from, to string, contextLines int) (string, bool) {
	ops, ok := computeLineDiffBounded(
		strings.Split(from, "\n"), strings.Split(to, "\n"), gitHubMaxEditDistance)
	if !ok {
		return "", false
	}
	additions, deletions := countEditOps(ops)

	header := fmt.Sprintf("multiline text (%s %s, %s %s)",
		formatCount(additions), pluralize(additions, "insert", "inserts"),
		formatCount(deletions), pluralize(deletions, "deletion", "deletions"))

	var body []string
	for _, chunk := range collapseLineDiff(ops, markNearChange(ops, contextLines)) {
		switch chunk.Type {
		case chunkKeep:
			body = append(body, "  "+chunk.Line)
		case chunkInsert:
			body = append(body, "+ "+chunk.Line)
		case chunkDelete:
			body = append(body, "- "+chunk.Line)
		case chunkCollapsed:
			body = append(body, collapsedRunLabel(chunk.Collapsed))
		}
	}

	return strings.Join(append([]string{header},
		truncateLines(body, gitHubMaxDiffLines)...), "\n"), true
}

// githubTruncatedValue renders a value the way diffDescription does, then keeps
// at most maxLines lines and summarizes the remainder. Truncating the rendered
// form rather than the source value bounds structured additions too: a whole
// added map serializes to many lines of YAML without ever being a Go string.
// Values within the cap are returned unchanged.
func githubTruncatedValue(val any, maxLines int) string {
	return strings.Join(
		truncateLines(strings.Split(formatValue(val), "\n"), maxLines), "\n")
}

// githubDiffDescription describes a difference for a GitHub Actions annotation,
// keeping multiline values bounded. A changed multiline string becomes a
// collapsed line diff; every other value is truncated at gitHubMaxValueLines.
// Differences whose values fit within the cap are described exactly as
// diffDescription describes them.
func githubDiffDescription(diff Difference, opts *FormatOptions) string {
	contextLines := 4
	if opts != nil {
		contextLines = opts.ContextLines
	}
	docSuffix := diffDocSuffix(diff)

	switch diff.Type {
	case DiffAdded:
		return fmt.Sprintf("Added: %s%s = %s", diff.Path, docSuffix,
			githubTruncatedValue(diff.To, gitHubMaxValueLines))
	case DiffRemoved:
		return fmt.Sprintf("Removed: %s%s = %s", diff.Path, docSuffix,
			githubTruncatedValue(diff.From, gitHubMaxValueLines))
	case DiffUnchanged:
		return fmt.Sprintf("Unchanged: %s%s = %s", diff.Path, docSuffix,
			githubTruncatedValue(diff.To, gitHubMaxValueLines))
	case DiffModified:
		if fromStr, toStr, ok := multilineStrings(diff.From, diff.To); ok {
			if body, ok := githubMultilineDiff(fromStr, toStr, contextLines); ok {
				return fmt.Sprintf("Modified: %s%s changed in %s", diff.Path, docSuffix, body)
			}
		}
		return fmt.Sprintf("Modified: %s%s changed from %s to %s", diff.Path, docSuffix,
			githubTruncatedValue(diff.From, gitHubMaxValueLines),
			githubTruncatedValue(diff.To, gitHubMaxValueLines))
	default: // DiffOrderChanged carries no value
		return diffDescription(diff)
	}
}
