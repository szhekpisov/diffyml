// mask.go - Sensitive value masking for diff output.
//
// Redacts values in [Difference.From] and [Difference.To] before they reach any
// formatter (including JSON, JSON-Patch, and the AI summarizer). When MaskSecrets
// is enabled, values under "data" and "stringData" of Kubernetes Secret resources
// are auto-masked. Users can declare additional paths via MaskPaths and
// MaskPathRegexp.
//
// Masking runs after Compare and before filtering, so a redacted diff still
// appears in the report — only the value is replaced with the placeholder.
package diffyml

import "regexp"

// DefaultMaskPlaceholder is the value substituted into masked diffs when
// MaskOptions.Placeholder is empty.
const DefaultMaskPlaceholder = "***"

// MaskOptions configures sensitive value masking.
type MaskOptions struct {
	// MaskSecrets enables auto-masking of "data" and "stringData" fields on
	// documents whose [Difference.DocumentKind] equals "Secret".
	MaskSecrets bool
	// MaskPaths is a list of dot-notation paths whose matching diffs are masked.
	// Paths are matched against the diff path with any leading document-index
	// prefix (e.g., "[0]") stripped. Prefix matches are honored ("data" matches
	// "data.password").
	MaskPaths []string
	// MaskPathRegexp is a list of regex patterns matched against the same
	// stripped path used by MaskPaths.
	MaskPathRegexp []string
	// Placeholder is the value substituted for masked scalars.
	// Defaults to [DefaultMaskPlaceholder] when empty.
	Placeholder string
}

// MaskDifferences redacts sensitive values in the given diffs in-place and
// returns the same slice. Order-change diffs ([DiffOrderChanged]) are never
// masked (their values are identifier lists, not secrets).
//
// Returns an error only if a regex pattern in opts.MaskPathRegexp fails to
// compile.
func MaskDifferences(diffs []Difference, opts MaskOptions) ([]Difference, error) {
	if !opts.MaskSecrets && len(opts.MaskPaths) == 0 && len(opts.MaskPathRegexp) == 0 {
		return diffs, nil
	}

	placeholder := opts.Placeholder
	if placeholder == "" {
		placeholder = DefaultMaskPlaceholder
	}

	regex, err := compileRegexPatterns(opts.MaskPathRegexp)
	if err != nil {
		return nil, err
	}

	for i := range diffs {
		if diffs[i].Type == DiffOrderChanged {
			continue
		}
		switch maskScopeFor(diffs[i], opts, regex) {
		case maskScopeAll:
			diffs[i].From = maskValueRecursive(diffs[i].From, placeholder)
			diffs[i].To = maskValueRecursive(diffs[i].To, placeholder)
		case maskScopeSecretFields:
			diffs[i].From = maskSecretSubtrees(diffs[i].From, placeholder)
			diffs[i].To = maskSecretSubtrees(diffs[i].To, placeholder)
		}
	}
	return diffs, nil
}

type maskScope int

const (
	maskScopeNone maskScope = iota
	// maskScopeAll redacts every leaf in From/To.
	maskScopeAll
	// maskScopeSecretFields redacts only the "data" and "stringData" subtrees
	// within From/To. Used for whole-document Secret add/remove diffs so other
	// fields (apiVersion, metadata) remain visible.
	maskScopeSecretFields
)

// secretMaskedKeys are the top-level Secret fields whose values are auto-masked.
var secretMaskedKeys = map[string]bool{"data": true, "stringData": true}

func maskScopeFor(d Difference, opts MaskOptions, regex []*regexp.Regexp) maskScope {
	pathStr := pathWithoutDocIndex(d.Path)
	if matchesAnyPath(pathStr, opts.MaskPaths) || matchesAnyRegex(pathStr, regex) {
		return maskScopeAll
	}
	if !opts.MaskSecrets || d.DocumentKind != "Secret" {
		return maskScopeNone
	}
	first, hasField := firstFieldAfterDocIndex(d.Path)
	if hasField && secretMaskedKeys[first] {
		return maskScopeAll
	}
	if isWholeDocDiff(d.Path) {
		return maskScopeSecretFields
	}
	return maskScopeNone
}

// maskValueRecursive returns a copy of v with every scalar leaf replaced by
// placeholder. Maps and lists are reconstructed; map keys and list lengths are
// preserved. nil values pass through unchanged.
func maskValueRecursive(v any, placeholder string) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case *OrderedMap:
		out := &OrderedMap{
			Keys:   append([]string(nil), val.Keys...),
			Values: make(map[string]any, len(val.Values)),
		}
		for k, sub := range val.Values {
			out.Values[k] = maskValueRecursive(sub, placeholder)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, sub := range val {
			out[k] = maskValueRecursive(sub, placeholder)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, sub := range val {
			out[i] = maskValueRecursive(sub, placeholder)
		}
		return out
	default:
		return placeholder
	}
}

// maskSecretSubtrees returns a shallow copy of v in which only the values under
// keys in [secretMaskedKeys] are recursively masked. Other branches are
// preserved by reference. Used when an entire Secret document is added or
// removed.
func maskSecretSubtrees(v any, placeholder string) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case *OrderedMap:
		out := &OrderedMap{
			Keys:   append([]string(nil), val.Keys...),
			Values: make(map[string]any, len(val.Values)),
		}
		for k, sub := range val.Values {
			if secretMaskedKeys[k] {
				out.Values[k] = maskValueRecursive(sub, placeholder)
			} else {
				out.Values[k] = sub
			}
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, sub := range val {
			if secretMaskedKeys[k] {
				out[k] = maskValueRecursive(sub, placeholder)
			} else {
				out[k] = sub
			}
		}
		return out
	default:
		return v
	}
}

// pathWithoutDocIndex renders the diff path without a leading document-index
// segment like "[0]". The result is what users specify with --mask-path.
func pathWithoutDocIndex(p DiffPath) string {
	if len(p) > 0 && len(p[0]) > 0 && p[0][0] == '[' {
		return p[1:].String()
	}
	return p.String()
}

// firstFieldAfterDocIndex returns the first non-document-index segment of the
// path, e.g. "data" for both "data.password" and "[0].data.password". Returns
// ("", false) if no field segment exists.
func firstFieldAfterDocIndex(p DiffPath) (string, bool) {
	if len(p) == 0 {
		return "", false
	}
	if len(p[0]) > 0 && p[0][0] == '[' {
		if len(p) < 2 {
			return "", false
		}
		return p[1], true
	}
	return p[0], true
}

// isWholeDocDiff reports whether the path refers to a whole document
// (e.g., "[0]") rather than a field within one.
func isWholeDocDiff(p DiffPath) bool {
	return len(p) == 1 && len(p[0]) > 0 && p[0][0] == '['
}
