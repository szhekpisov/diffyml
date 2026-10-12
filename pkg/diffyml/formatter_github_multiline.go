// formatter_github_multiline.go - Bounded multiline values for GitHub Actions annotations.
//
// A GitHub Actions annotation is a single workflow command line, so a raw
// multiline YAML value would both blow up the message and terminate the command
// at its first newline. Modified values are rendered as a collapsed line diff
// (the same context window the detailed formatter uses); added and removed
// values, which have no counterpart to diff against, are truncated instead.
// Both paths end in a hard cap on lines and on the width of each line, so no
// annotation *message* is unbounded in either dimension: collapsing alone
// cannot shrink a value whose lines all changed, and a line cap alone cannot
// shrink a value that is one enormous line. Those two caps bound each dimension
// but not their product, so gitHubMaxMessageRunes bounds the whole message as
// well. The file= property is outside all of this on purpose — see
// gitHubWriteCommand. A
// wholesale rewrite skips the diff entirely rather than pay for one it cannot
// show, and is rendered as each side truncated on its own — see
// gitHubMaxEditDistance and githubRewrittenPair.
package diffyml

import (
	"fmt"
	"strings"
	"unicode/utf8"
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
// diff is abandoned in favor of showing each side on its own (see
// githubRewrittenPair). The rendered body already stops at gitHubMaxDiffLines,
// so an edit script twice that long overflows anything the annotation could
// show; all a full diff still buys past this point is an exact
// insert/deletion count in the header, and computeLineDiff
// pays O(D^2) memory for it — 1.6 GiB for an 8000-line rewrite, to render 40
// lines. Bounding D is what makes that cost a constant. Only wholesale
// rewrites reach the ceiling; a value with a handful of changed lines has a
// small edit distance however long the value is, so the case collapsing exists
// for is never the case that falls back.
const gitHubMaxEditDistance = 2 * gitHubMaxDiffLines

// gitHubMaxLineRunes caps the length of a single rendered line. Every other cap
// here counts lines, so a value that is one enormous line — minified JSON, a
// base64 blob, a last-applied-configuration annotation — passes all of them
// untouched and produces one multi-megabyte annotation. Runes rather than bytes
// so a multi-byte character is never cut in half.
const gitHubMaxLineRunes = 500

// gitHubMaxMessageRunes caps the assembled message, which the per-line and
// per-value caps do not: they bound each dimension separately, and their
// product is 40 lines of 500 runes, or 61 KB once percent-encoding triples a
// message made of "%". It is also the only bound on the parts of a message that
// are not value lines at all — the difference's path is a document key of any
// length, and nothing else caps it. Sized well above what a readable annotation
// reaches, so it only ever fires on the pathological shapes; GitHub truncates
// the rendered message far below this anyway.
const gitHubMaxMessageRunes = 4000

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

// truncateRunes keeps at most maxRunes runes of s, appending a marker counting
// what was dropped. Ranging a string yields rune-boundary byte indices, so the
// cut never lands inside a multi-byte character and the count is in characters
// rather than bytes.
func truncateRunes(s string, maxRunes int) string {
	count := 0
	for i := range s {
		if count == maxRunes {
			dropped := utf8.RuneCountInString(s[i:])
			return s[:i] + fmt.Sprintf("…[%d more %s]", dropped,
				pluralize(dropped, "character", "characters"))
		}
		count++
	}
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
// single summary marker. The body is capped at gitHubMaxDiffLines; the header
// sits outside the cap so the edit counts survive truncation and still describe
// the whole change.
//
// ok is false when the two values differ by more than gitHubMaxEditDistance
// lines, which is where computing a diff costs more than the diff can show.
// githubRewrittenPair renders that case instead.
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

	chunks := collapseLineDiff(ops, markNearChange(ops, contextLines))
	body := githubDiffBody(chunks, gitHubMaxDiffLines, gitHubMaxLineRunes)

	return strings.Join(append([]string{header}, body...), "\n"), true
}

// renderChunkLine renders one chunk the way an annotation shows it: the shapes
// the detailed formatter renders with deeper indentation and color.
//
// Context is the default arm rather than a case of its own so that a chunk type
// added later renders as an unmarked line instead of claiming to be a collapsed
// run of some other length.
func renderChunkLine(chunk lineDiffChunk) string {
	switch chunk.Type {
	case chunkCollapsed:
		return collapsedRunLabel(chunk.Collapsed)
	case chunkInsert:
		return "+ " + chunk.Line
	case chunkDelete:
		return "- " + chunk.Line
	default: // chunkKeep
		return "  " + chunk.Line
	}
}

// githubDiffBody renders chunks into at most maxLines lines, each capped at
// maxRunes, and summarizes the rest by how many lines of the *value* they stand
// for rather than how many chunks: a dropped "[24 lines unchanged]" hides 24
// lines, not one. That keeps the marker in the same unit as the collapse
// markers above it.
func githubDiffBody(chunks []lineDiffChunk, maxLines, maxRunes int) []string {
	var body []string
	for i, chunk := range chunks {
		if i == maxLines {
			hidden := 0
			for _, rest := range chunks[i:] {
				hidden += rest.lineCount()
			}
			return append(body, fmt.Sprintf("[%d more %s]",
				hidden, pluralize(hidden, "line", "lines")))
		}
		body = append(body, truncateRunes(renderChunkLine(chunk), maxRunes))
	}
	return body
}

// githubRewrittenPair renders a modified multiline pair that githubMultilineDiff
// refused as too far apart: a header saying so, then each side truncated on its
// own, the before side marked "-" and the after side "+".
//
// Falling through to the shared "changed from X to Y" wording instead would show
// no more of either value than this does — both are capped at maxLines — while
// dropping the header and every sign that this is a multiline change at all. The
// one thing the ceiling really costs is the alignment between the two sides, and
// the exact insert/deletion counts that computing it would have bought; the
// header states the fact that stands in for them.
func githubRewrittenPair(from, to string, maxLines, maxRunes int) string {
	lines := []string{fmt.Sprintf("multiline text (rewritten, more than %d lines differ)",
		gitHubMaxEditDistance)}
	lines = append(lines, githubMarkedValue(from, "- ", maxLines, maxRunes)...)
	lines = append(lines, githubMarkedValue(to, "+ ", maxLines, maxRunes)...)
	return strings.Join(lines, "\n")
}

// githubMarkedValue truncates one side of a rewritten pair and marks each kept
// line. The "[N more lines]" marker truncateLines appends is left unmarked, so
// it reads as a marker rather than as one more removed or added line.
func githubMarkedValue(val, mark string, maxLines, maxRunes int) []string {
	lines := truncateLines(strings.Split(val, "\n"), maxLines)
	out := make([]string, len(lines))
	for i, line := range lines {
		if i < maxLines {
			line = mark + line
		}
		out[i] = truncateRunes(line, maxRunes)
	}
	return out
}

// githubTruncatedValue renders a value the way diffDescription does, then keeps
// at most maxLines lines, each of at most maxRunes runes, and summarizes what it
// dropped. Truncating the rendered form rather than the source value bounds
// structured additions too: a whole added map serializes to many lines of YAML
// without ever being a Go string. Values within both caps are returned
// unchanged.
func githubTruncatedValue(val any, maxLines, maxRunes int) string {
	lines := truncateLines(strings.Split(formatValue(val), "\n"), maxLines)
	for i, line := range lines {
		lines[i] = truncateRunes(line, maxRunes)
	}
	return strings.Join(lines, "\n")
}

// githubCertValue replaces a single PEM certificate with its one-line summary,
// mirroring what the detailed formatter does to added, removed and unchanged
// values. Anything else comes back untouched.
func githubCertValue(val any, enabled bool) any {
	if !enabled {
		return val
	}
	if s, ok := val.(string); ok && IsPEMCertificate(s) {
		return FormatCertificate(s)
	}
	return val
}

// githubCertPair does the same for a modified pair, and like the detailed
// formatter only when *both* sides are certificates — a certificate replaced by
// something else is still best shown as a plain value change. Callers must run
// this before the multiline check, or a rotated certificate takes the line-diff
// path and renders as base64 instead of a summary.
func githubCertPair(from, to any, enabled bool) (any, any) {
	if !enabled {
		return from, to
	}
	fromStr, fromOk := from.(string)
	toStr, toOk := to.(string)
	if fromOk && toOk && IsPEMCertificate(fromStr) && IsPEMCertificate(toStr) {
		return FormatCertificate(fromStr), FormatCertificate(toStr)
	}
	return from, to
}

// githubDiffDescription describes a difference for a GitHub Actions annotation,
// keeping multiline values bounded. A changed multiline string becomes a
// collapsed line diff; every other value is truncated at gitHubMaxValueLines.
// Differences whose values fit within the caps are described exactly as
// diffDescription describes them.
func githubDiffDescription(diff Difference, opts *FormatOptions) string {
	// Normalizing here rather than reading fields off a possibly-nil opts keeps
	// the defaults in one place: nil means DefaultFormatOptions, so a context
	// of 4 and certificate inspection enabled, matching the detailed formatter.
	if opts == nil {
		opts = DefaultFormatOptions()
	}
	certs := !opts.NoCertInspection
	docSuffix := diffDocSuffix(diff)

	switch diff.Type {
	case DiffAdded:
		return fmt.Sprintf("Added: %s%s = %s", diff.Path, docSuffix,
			githubTruncatedValue(githubCertValue(diff.To, certs), gitHubMaxValueLines, gitHubMaxLineRunes))
	case DiffRemoved:
		return fmt.Sprintf("Removed: %s%s = %s", diff.Path, docSuffix,
			githubTruncatedValue(githubCertValue(diff.From, certs), gitHubMaxValueLines, gitHubMaxLineRunes))
	case DiffUnchanged:
		return fmt.Sprintf("Unchanged: %s%s = %s", diff.Path, docSuffix,
			githubTruncatedValue(githubCertValue(diff.To, certs), gitHubMaxValueLines, gitHubMaxLineRunes))
	case DiffModified:
		from, to := githubCertPair(diff.From, diff.To, certs)
		if fromStr, toStr, ok := multilineStrings(from, to); ok {
			// A multiline pair keeps the multiline framing either way: when it
			// is too far apart to diff, githubRewrittenPair says so rather than
			// letting it fall through to the shared from/to wording below,
			// which is for values that were never diffable in the first place.
			body, ok := githubMultilineDiff(fromStr, toStr, opts.ContextLines)
			if !ok {
				body = githubRewrittenPair(fromStr, toStr, gitHubMaxValueLines, gitHubMaxLineRunes)
			}
			return fmt.Sprintf("Modified: %s%s changed in %s", diff.Path, docSuffix, body)
		}
		return fmt.Sprintf("Modified: %s%s changed from %s to %s", diff.Path, docSuffix,
			githubTruncatedValue(from, gitHubMaxValueLines, gitHubMaxLineRunes),
			githubTruncatedValue(to, gitHubMaxValueLines, gitHubMaxLineRunes))
	default: // DiffOrderChanged carries no value
		return diffDescription(diff)
	}
}
