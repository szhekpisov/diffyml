// inline_diff.go - Word-level inline diff highlighting for scalar value changes.
//
// Tokenizes strings on \w+ boundaries and diffs the token lists using the
// existing Myers algorithm (computeLineDiff). Changed tokens are rendered
// with bold styling; unchanged tokens are dimmed for visual contrast.
package diffyml

import (
	"regexp"
	"strings"
)

// inlineSegment represents a contiguous run of tokens classified as changed or unchanged.
type inlineSegment struct {
	Text    string
	Changed bool
}

// wordPattern matches runs of word characters (\w+). Non-word characters
// between matches become individual single-character tokens.
var wordPattern = regexp.MustCompile(`\w+`)

// tokenize splits a string into tokens: word-character runs are single tokens,
// everything between them becomes individual single-character tokens.
//
// Example: "192.168.10.1" → ["192", ".", "168", ".", "10", ".", "1"]
func tokenize(s string) []string {
	matches := wordPattern.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		// No word characters — split into individual runes.
		tokens := make([]string, 0, len(s))
		for _, r := range s {
			tokens = append(tokens, string(r))
		}
		return tokens
	}

	var tokens []string
	pos := 0
	for _, m := range matches {
		// Characters before this word match → individual tokens.
		for _, r := range s[pos:m[0]] {
			tokens = append(tokens, string(r))
		}
		// The word itself → single token.
		tokens = append(tokens, s[m[0]:m[1]])
		pos = m[1]
	}
	// Trailing non-word characters.
	for _, r := range s[pos:] {
		tokens = append(tokens, string(r))
	}
	return tokens
}

// computeInlineDiff computes word-level diff segments between two strings.
// Returns (fromSegments, toSegments) where each segment is classified as
// changed or unchanged. Returns nil, nil when inline diff would not be
// useful (values too short, too different, identical, or multiline).
func computeInlineDiff(from, to string) ([]inlineSegment, []inlineSegment) {
	if from == to || from == "" || to == "" {
		return nil, nil
	}
	if strings.Contains(from, "\n") || strings.Contains(to, "\n") {
		return nil, nil
	}

	fromTokens := tokenize(from)
	toTokens := tokenize(to)

	if len(fromTokens) <= 2 && len(toTokens) <= 2 {
		return nil, nil
	}

	ops := computeLineDiff(fromTokens, toTokens)

	// Similarity threshold: if less than 30% of characters (by length) are
	// kept, inline highlighting adds noise rather than clarity. We weight by
	// character count so that shared punctuation (e.g. dots between otherwise
	// different words) doesn't inflate the similarity score.
	keepChars := 0
	for _, op := range ops {
		if op.Type == editKeep {
			keepChars += len(op.Line)
		}
	}
	longer := max(len(from), len(to))
	if keepChars*10 < longer*3 { // keepChars/longer < 0.3, avoiding float
		return nil, nil
	}

	// Build segments by coalescing consecutive same-type operations.
	var fromSegs, toSegs []inlineSegment
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			fromSegs = appendSegment(fromSegs, op.Line, false)
			toSegs = appendSegment(toSegs, op.Line, false)
		case editDelete:
			fromSegs = appendSegment(fromSegs, op.Line, true)
		case editInsert:
			toSegs = appendSegment(toSegs, op.Line, true)
		}
	}

	return fromSegs, toSegs
}

// appendSegment appends text to the last segment if it has the same Changed
// classification, otherwise starts a new segment.
func appendSegment(segs []inlineSegment, text string, changed bool) []inlineSegment {
	if len(segs) > 0 && segs[len(segs)-1].Changed == changed {
		segs[len(segs)-1].Text += text
		return segs
	}
	return append(segs, inlineSegment{Text: text, Changed: changed})
}

// renderInlineSegments writes segments to a builder with inline diff styling.
// Changed segments are rendered in bold with the base color; unchanged segments
// use the dim color.
func renderInlineSegments(sb *strings.Builder, segments []inlineSegment, baseColor, dimColor string, opts *FormatOptions) {
	if !opts.Color {
		for _, seg := range segments {
			sb.WriteString(seg.Text)
		}
		return
	}
	for _, seg := range segments {
		if seg.Changed {
			sb.WriteString(styleBold)
			sb.WriteString(baseColor)
			sb.WriteString(seg.Text)
			sb.WriteString(styleBoldOff)
		} else {
			sb.WriteString(dimColor)
			sb.WriteString(seg.Text)
		}
	}
}

// dimColorCode returns a dimmed color code for unchanged inline diff segments.
// In TrueColor mode, blends the role's RGB 50% toward neutral gray.
// In 8-color mode, returns the base ANSI code (bold/normal contrast suffices).
func dimColorCode(role ColorRole, opts *FormatOptions) string {
	p := resolvedPalette(opts)
	c := p.colorForRole(role)

	if opts.TrueColor {
		r, g, b := c.R, c.G, c.B
		return TrueColorCode((r+128)/2, (g+128)/2, (b+128)/2)
	}
	return c.ANSICode
}
