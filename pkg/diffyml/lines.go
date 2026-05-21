// lines.go - Source line-number capture for differences.
//
// When Options.CaptureLineNumbers is set, Compare builds a path -> source-line
// index for each input file by walking the retained yaml.Node trees (which carry
// 1-based .Line info that nodeToInterface discards), then annotates each Difference
// with LineFrom/LineTo. Path strings are produced with appendPathSegment so they
// match DiffPath.String() byte-for-byte and the segments the comparator emits.
package diffyml

import (
	"strconv"

	"go.yaml.in/yaml/v3"
)

// lineMapWalker builds a path-string -> source-line map using the same incremental
// byte-buffer push/pop technique as pathWalker, avoiding per-node allocations.
type lineMapWalker struct {
	m    map[string]int
	opts *Options
	buf  []byte
	lens []int
}

func (w *lineMapWalker) push(seg string) {
	w.lens = append(w.lens, len(w.buf))
	w.buf = appendPathSegment(w.buf, seg)
}

func (w *lineMapWalker) pop() {
	n := len(w.lens) - 1
	w.buf = w.buf[:w.lens[n]]
	w.lens = w.lens[:n]
}

// register records the current path at line, first-occurrence-wins (matching
// pathWalker.register). Empty paths (document root) and unknown lines are skipped.
func (w *lineMapWalker) register(line int) {
	if len(w.buf) == 0 || line <= 0 {
		return
	}
	key := string(w.buf)
	if _, ok := w.m[key]; !ok {
		w.m[key] = line
	}
}

// walk registers the current path at the given line, then descends. For mapping
// children the anchor is the KEY node's line; for sequence items it is the item
// node's line. line is the line to register for the path currently held in buf.
func (w *lineMapWalker) walk(node *yaml.Node, line int) {
	if node == nil {
		return
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return
		}
		w.walk(node.Content[0], node.Content[0].Line)
		return
	}

	w.register(line)

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			if keyNode.Value == "<<" {
				// YAML merge key: merged entries have no single source line in v1.
				continue
			}
			w.push(keyNode.Value)
			w.walk(node.Content[i+1], keyNode.Line)
			w.pop()
		}
	case yaml.SequenceNode:
		for i, item := range node.Content {
			var seg string
			if id := getIdentifier(nodeToInterface(item), w.opts); isComparableIdentifier(id) {
				seg = sprintIdentifier(id)
			} else {
				seg = strconv.Itoa(i)
			}
			w.push(seg)
			w.walk(item, item.Line)
			w.pop()
		}
	}
}

// buildLineMap returns a path-string -> 1-based source line map for the given
// document node trees. maxLen is max(len(fromNodes), len(toNodes)); when > 1 a
// "[i]" document prefix is applied, matching compareDocs' multi-document gating.
func buildLineMap(nodes []*yaml.Node, maxLen int, opts *Options) map[string]int {
	w := &lineMapWalker{
		m:    make(map[string]int),
		opts: opts,
		buf:  make([]byte, 0, 256),
		lens: make([]int, 0, 16),
	}
	for i, doc := range nodes {
		if maxLen > 1 {
			w.push("[" + strconv.Itoa(i) + "]")
			w.walk(doc, 0)
			w.pop()
		} else {
			w.walk(doc, 0)
		}
	}
	return w.m
}

// annotateLines populates LineFrom/LineTo on each diff from the per-file line maps.
func annotateLines(diffs []Difference, fromMap, toMap map[string]int, opts *Options) {
	for i := range diffs {
		d := &diffs[i]
		switch d.Type {
		case DiffModified, DiffOrderChanged:
			key := d.Path.String()
			d.LineFrom = fromMap[key]
			d.LineTo = toMap[key]
		case DiffAdded:
			d.LineTo = resolveAddRemoveLine(toMap, d.Path, d.To, opts)
		case DiffRemoved:
			d.LineFrom = resolveAddRemoveLine(fromMap, d.Path, d.From, opts)
		}
	}
}

// resolveAddRemoveLine finds the source line for an add/remove diff. The comparator
// reports these at the parent path with the value wrapped, so the real anchor is a
// child path. Tries identifier-segment (list item add/remove), then single-key
// wrapper (map key add/remove), then the path itself (positional list / whole doc).
func resolveAddRemoveLine(m map[string]int, path DiffPath, val any, opts *Options) int {
	base := path.String()
	if id := getIdentifier(val, opts); isComparableIdentifier(id) {
		if line := m[appendKey(base, sprintIdentifier(id))]; line != 0 {
			return line
		}
	}
	if om, ok := val.(*OrderedMap); ok && len(om.Keys) == 1 {
		if line := m[appendKey(base, om.Keys[0])]; line != 0 {
			return line
		}
	}
	return m[base]
}

// appendKey returns base with seg appended using DiffPath.String() formatting.
func appendKey(base, seg string) string {
	return string(appendPathSegment([]byte(base), seg))
}
