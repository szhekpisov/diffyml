package diffyml

import (
	"strconv"
	"strings"
)

// DiffPath represents a structured path through a YAML document as a sequence of key segments.
// Each element is a map key, list index, or document index at that level.
type DiffPath []string

// String returns the display format of the path.
// Segments containing dots are bracket-quoted (e.g., [helm.sh/chart]).
// Document index segments like [0] are written as-is without a preceding dot.
func (p DiffPath) String() string {
	var sb strings.Builder
	for _, seg := range p {
		switch {
		case strings.Contains(seg, "."):
			sb.WriteByte('[')
			sb.WriteString(seg)
			sb.WriteByte(']')
		case sb.Len() > 0 && !strings.HasPrefix(seg, "["):
			sb.WriteByte('.')
			sb.WriteString(seg)
		default:
			sb.WriteString(seg)
		}
	}
	return sb.String()
}

// Append returns a new DiffPath with the given key appended.
func (p DiffPath) Append(key string) DiffPath {
	return append(p[:len(p):len(p)], key)
}

// Last returns the last segment, or "" if the path is empty.
func (p DiffPath) Last() string {
	if len(p) == 0 {
		return ""
	}
	return p[len(p)-1]
}

// Root returns the first segment, or "" if the path is empty.
func (p DiffPath) Root() string {
	if len(p) == 0 {
		return ""
	}
	return p[0]
}

// Depth returns the path depth (number of separators, i.e. len - 1).
func (p DiffPath) Depth() int {
	if len(p) == 0 {
		return 0
	}
	return len(p) - 1
}

// Parent returns all but the last segment, or nil if empty or single-element.
func (p DiffPath) Parent() DiffPath {
	if len(p) <= 1 {
		return nil
	}
	return p[:len(p)-1]
}

// IsEmpty returns true if the path has no segments.
func (p DiffPath) IsEmpty() bool {
	return len(p) == 0
}

// GoPatchString returns the go-patch style path (/a/b/c).
// Document index brackets [N] are stripped to produce /N.
// Dots inside keys are preserved.
func (p DiffPath) GoPatchString() string {
	var sb strings.Builder
	for _, seg := range p {
		sb.WriteByte('/')
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			sb.WriteString(seg[1 : len(seg)-1])
		} else {
			sb.WriteString(seg)
		}
	}
	if sb.Len() == 0 {
		return "/"
	}
	return sb.String()
}

// HasNumericLast returns true if the last segment is a non-negative integer.
func (p DiffPath) HasNumericLast() bool {
	if len(p) == 0 {
		return false
	}
	last := p[len(p)-1]
	if last == "" {
		return false
	}
	for _, c := range last {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// IsBareDocIndex returns true if the path is a single document index segment like [0].
func (p DiffPath) IsBareDocIndex() bool {
	if len(p) != 1 {
		return false
	}
	seg := p[0]
	if !strings.HasPrefix(seg, "[") || !strings.HasSuffix(seg, "]") {
		return false
	}
	_, err := strconv.Atoi(seg[1 : len(seg)-1])
	return err == nil
}

// DocIndex returns the document index from the first segment if it's a [N] segment.
func (p DiffPath) DocIndex() (int, bool) {
	if len(p) == 0 {
		return 0, false
	}
	seg := p[0]
	if !strings.HasPrefix(seg, "[") || !strings.HasSuffix(seg, "]") {
		return 0, false
	}
	idx, err := strconv.Atoi(seg[1 : len(seg)-1])
	if err != nil {
		return 0, false
	}
	return idx, true
}

// JSONPointerString returns an RFC 6901 JSON Pointer representation of the path.
// Each segment is escaped per RFC 6901: ~ → ~0, / → ~1, then prefixed with /.
// Document index brackets [N] are stripped to bare N (same as GoPatchString).
// Empty path → "" (RFC 6901: empty string = whole document).
func (p DiffPath) JSONPointerString() string {
	var sb strings.Builder
	for _, seg := range p {
		sb.WriteByte('/')
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			sb.WriteString(seg[1 : len(seg)-1])
		} else {
			// RFC 6901 escaping: ~ must be escaped first, then /
			escaped := strings.ReplaceAll(seg, "~", "~0")
			escaped = strings.ReplaceAll(escaped, "/", "~1")
			sb.WriteString(escaped)
		}
	}
	return sb.String()
}

// DocIndexPrefix returns (index, remaining path, true) if path starts with [N] followed by more segments.
func (p DiffPath) DocIndexPrefix() (int, DiffPath, bool) {
	if len(p) < 2 {
		return 0, p, false
	}
	idx, ok := p.DocIndex()
	if !ok {
		return 0, p, false
	}
	return idx, p[1:], true
}
