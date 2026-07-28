// formatter_github_multiline.go - Bounded multiline values for GitHub Actions annotations.
//
// A GitHub Actions annotation is a single workflow command line, so a raw
// multiline YAML value would both blow up the message and terminate the command
// at its first newline. Modified values are rendered as a collapsed line diff
// (the same context window the detailed formatter uses); added and removed
// values, which have no counterpart to diff against, are truncated instead.
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
// single summary marker.
func githubMultilineDiff(from, to string, contextLines int) string {
	ops := computeLineDiff(strings.Split(from, "\n"), strings.Split(to, "\n"))
	additions, deletions := countEditOps(ops)

	lines := []string{fmt.Sprintf("multiline text (%s %s, %s %s)",
		formatCount(additions), pluralize(additions, "insert", "inserts"),
		formatCount(deletions), pluralize(deletions, "deletion", "deletions"))}

	for _, chunk := range collapseLineDiff(ops, markNearChange(ops, contextLines)) {
		switch chunk.Type {
		case chunkKeep:
			lines = append(lines, "  "+chunk.Line)
		case chunkInsert:
			lines = append(lines, "+ "+chunk.Line)
		case chunkDelete:
			lines = append(lines, "- "+chunk.Line)
		case chunkCollapsed:
			lines = append(lines, collapsedRunLabel(chunk.Collapsed))
		}
	}

	return strings.Join(lines, "\n")
}

// githubTruncatedValue renders a value the way diffDescription does, then keeps
// at most maxLines lines and summarizes the remainder. Truncating the rendered
// form rather than the source value bounds structured additions too: a whole
// added map serializes to many lines of YAML without ever being a Go string.
// Values within the cap are returned unchanged.
func githubTruncatedValue(val any, maxLines int) string {
	s := formatValue(val)

	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}

	remaining := len(lines) - maxLines
	return strings.Join(lines[:maxLines], "\n") +
		fmt.Sprintf("\n[%d more %s]", remaining, pluralize(remaining, "line", "lines"))
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
			return fmt.Sprintf("Modified: %s%s changed in %s", diff.Path, docSuffix,
				githubMultilineDiff(fromStr, toStr, contextLines))
		}
		return fmt.Sprintf("Modified: %s%s changed from %s to %s", diff.Path, docSuffix,
			githubTruncatedValue(diff.From, gitHubMaxValueLines),
			githubTruncatedValue(diff.To, gitHubMaxValueLines))
	default: // DiffOrderChanged carries no value
		return diffDescription(diff)
	}
}
