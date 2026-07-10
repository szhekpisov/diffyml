// inverse.go - Inverse ("unchanged") diff collection.
//
// Implements the Options.Unchanged mode requested in issue #183: instead of
// reporting how two YAML documents differ, report the keys/values that are
// EQUAL between them. The normal comparator (comparator.go) discards equal
// subtrees, so this is a dedicated parallel walk rather than a flag threaded
// through the existing branches.
//
// Equal nodes collapse to the highest fully-equal node: a wholly-equal map or
// list yields one DiffUnchanged entry; partially-equal maps/lists are descended
// into so only the matching leaves/subtrees are reported. Keys present on only
// one side, kind mismatches, and unequal scalars emit nothing — they are, by
// definition, not "unchanged". Equality honors the same Options flags as the
// normal compare via deepEqual/equalValues.
//
// Document and list pairing reuse the normal comparator's matching so reordered
// or renamed counterparts are recognized: Kubernetes documents are matched by
// resource identifier (and rename detection), and named list items are matched
// by their name/id identifier — both falling back to positional pairing.
package diffyml

import (
	"strconv"

	"go.yaml.in/yaml/v3"
)

// collectUnchangedDocs mirrors compareDocs: Kubernetes documents are paired by
// resource identifier (with rename detection) when detected; otherwise documents
// are paired positionally. opts is non-nil — Compare normalizes it before calling.
func collectUnchangedDocs(from, to []*yaml.Node, opts *Options) []Difference {
	if opts.DetectKubernetes {
		if fromDocs, toDocs, ok := detectK8sDocsCached(from, to); ok {
			return collectUnchangedK8sDocs(from, to, fromDocs, toDocs, opts)
		}
	}

	var diffs []Difference
	maxLen := max(len(from), len(to))
	for i := range maxLen {
		var fromN, toN *yaml.Node
		if i < len(from) {
			fromN = from[i]
		}
		if i < len(to) {
			toN = to[i]
		}

		// Build path prefix for multi-document files (matches compareDocs).
		var pathPrefix DiffPath
		if maxLen > 1 {
			pathPrefix = DiffPath{"[" + strconv.Itoa(i) + "]"}
		}

		// Documents are not sequence elements; doc-level collapses are recognized
		// by isListEntryDiff via IsBareDocIndex, so inList is false here.
		nodeDiffs := collectUnchanged(pathPrefix, fromN, toN, opts, false)
		for j := range nodeDiffs {
			nodeDiffs[j].DocumentIndex = i
		}
		diffs = append(diffs, nodeDiffs...)
	}

	return diffs
}

// collectUnchangedK8sDocs pairs Kubernetes documents by resource identifier
// (and rename detection) and reports the equal values within each matched pair.
// Mirrors compareK8sDocsCached's matching, but emits nothing for order changes
// or for documents present on only one side (those are not "unchanged").
func collectUnchangedK8sDocs(fromNodes, toNodes []*yaml.Node, fromDocs, toDocs []any, opts *Options) []Difference {
	matched, unmatchedFrom, unmatchedTo := matchK8sDocsValues(fromDocs, toDocs, opts)

	var diffs []Difference
	diffs = append(diffs, collectMatchedK8sUnchanged(matched, fromNodes, toNodes, toDocs, opts, false)...)

	renameMatched, _, _ := detectRenames(fromDocs, toDocs, unmatchedFrom, unmatchedTo, opts)
	diffs = append(diffs, collectMatchedK8sUnchanged(renameMatched, fromNodes, toNodes, toDocs, opts, true)...)

	return diffs
}

// collectMatchedK8sUnchanged runs collectUnchanged over each matched document
// pair, stamping DocumentIndex/Name/Kind. Mirrors compareMatchedK8sDocs:
// useToIdx selects the to-side document index for rename-matched pairs.
func collectMatchedK8sUnchanged(matched map[int]int, fromNodes, toNodes []*yaml.Node, toDocs []any, opts *Options, useToIdx bool) []Difference {
	var diffs []Difference
	for fromIdx, toIdx := range matched {
		docIdx := fromIdx
		if useToIdx {
			docIdx = toIdx
		}

		var pathPrefix DiffPath
		if len(fromNodes) > 1 || len(toNodes) > 1 {
			pathPrefix = DiffPath{"[" + strconv.Itoa(docIdx) + "]"}
		}

		// Document root is not a sequence element (inList false).
		nodeDiffs := collectUnchanged(pathPrefix, fromNodes[fromIdx], toNodes[toIdx], opts, false)
		var docName, docKind string
		if f, ok := k8sExtractFields(toDocs[toIdx]); ok {
			docName = f.displayName()
			docKind = f.kind
		}
		for i := range nodeDiffs {
			nodeDiffs[i].DocumentIndex = docIdx
			nodeDiffs[i].DocumentName = docName
			nodeDiffs[i].DocumentKind = docKind
		}
		diffs = append(diffs, nodeDiffs...)
	}

	return diffs
}

// collectUnchanged recursively reports values equal between fromN and toN.
// A side that is null/absent, a kind mismatch, or an unequal scalar yields
// nothing; a fully-equal node yields a single collapsed entry; partially-equal
// maps and sequences are descended into.
//
// inList reports whether the node being collected is a direct sequence element,
// so a collapsed entry is tagged for isListEntryDiff to render it with the "- "
// list prefix. Map children and document roots pass false.
func collectUnchanged(path DiffPath, fromN, toN *yaml.Node, opts *Options, inList bool) []Difference {
	// Only-one-side (or both-null) means "different" for inverse purposes.
	if isNullNode(fromN) || isNullNode(toN) {
		return nil
	}
	fromN = resolveNode(fromN)
	toN = resolveNode(toN)

	// Different kinds can never be equal.
	if fromN.Kind != toN.Kind {
		return nil
	}

	// Highest-equal-node collapse: if the whole subtree matches, emit one entry
	// and stop descending; otherwise descend into common children only. The
	// equality check is dispatched per kind (rather than through the generic
	// deepEqualNodes) so the mapping case can build its value indices once and
	// share them with the descent instead of paying for a second index pass. The
	// subtree is materialized (nodeToInterface) only on the collapse, never on
	// the partial-match path below.
	switch fromN.Kind {
	case yaml.MappingNode:
		fromIdx := indexMappingValues(fromN)
		toIdx := indexMappingValues(toN)
		if mappingNodesEqualIdx(fromN, toN, opts, fromIdx, toIdx) {
			return []Difference{unchangedEntry(path, fromN, toN, inList)}
		}
		return descendUnchangedMapping(path, fromN, toN, opts, fromIdx, toIdx)
	case yaml.SequenceNode:
		if deepEqualSequenceNodes(fromN, toN, opts) {
			return []Difference{unchangedEntry(path, fromN, toN, inList)}
		}
		return collectUnchangedSequence(path, fromN, toN, opts)
	default:
		// Scalar: isNullNode already excluded !!null on both sides, so this
		// mirrors deepEqualNodes' scalar fall-through to equalValues.
		if equalValues(resolveScalar(fromN), resolveScalar(toN), opts) {
			return []Difference{unchangedEntry(path, fromN, toN, inList)}
		}
		// Unequal scalar: nothing is unchanged here.
		return nil
	}
}

// unchangedEntry builds a collapsed DiffUnchanged entry for a fully-equal node,
// materializing both sides once. inList tags a collapse whose container is a
// sequence so isListEntryDiff renders it with the "- " list prefix.
func unchangedEntry(path DiffPath, fromN, toN *yaml.Node, inList bool) Difference {
	return Difference{Path: path, Type: DiffUnchanged, From: nodeToInterface(fromN), To: nodeToInterface(toN), listEntry: inList}
}

// collectUnchangedMapping recurses on keys present in BOTH mappings, preserving
// the from-side source order. Mirrors compareMappingNodes' last-write-wins index
// lookup so duplicate keys resolve identically.
func collectUnchangedMapping(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	return descendUnchangedMapping(path, fromN, toN, opts, indexMappingValues(fromN), indexMappingValues(toN))
}

// descendUnchangedMapping is collectUnchangedMapping with the value indices
// supplied by the caller, letting collectUnchanged reuse the indices it built
// for the whole-map equality check.
func descendUnchangedMapping(path DiffPath, fromN, toN *yaml.Node, opts *Options, fromIdx, toIdx map[string]int) []Difference {
	var diffs []Difference
	for i := 0; i+1 < len(fromN.Content); i += 2 {
		key := fromN.Content[i].Value
		toPos, inTo := toIdx[key]
		if !inTo {
			continue
		}
		fromVal := fromN.Content[fromIdx[key]+1]
		toVal := toN.Content[toPos+1]
		// Map child: a collapse here is a map entry, not a list item.
		diffs = append(diffs, collectUnchanged(path.Append(key), fromVal, toVal, opts, false)...)
	}

	return diffs
}

// collectUnchangedSequence mirrors compareSequenceNodes' full dispatch so the
// inverse walk recognizes the same equalities the normal comparator does:
// identifier-matched (reordered named lists), order-independent (under
// --ignore-order-changes or for heterogeneous single-key-map lists), otherwise
// positional.
func collectUnchangedSequence(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	if canMatchByIdentifierNodes(fromN.Content, opts) && canMatchByIdentifierNodes(toN.Content, opts) {
		return collectUnchangedSequenceByIdentifier(path, fromN, toN, opts)
	}

	if opts.IgnoreOrderChanges || areSequenceItemsHeterogeneous(fromN, toN) {
		return collectUnchangedSequenceUnordered(path, fromN, toN, opts)
	}

	from := fromN.Content
	to := toN.Content
	minLen := min(len(from), len(to))

	var diffs []Difference
	for i := range minLen {
		// Sequence element: a collapse here is a list item.
		diffs = append(diffs, collectUnchanged(path.Append(strconv.Itoa(i)), from[i], to[i], opts, true)...)
	}

	return diffs
}

// collectUnchangedSequenceUnordered reports the unchanged values of a sequence
// order-independently, mirroring compareSequenceNodesUnordered's matching. All
// item indices participate; pairing is delegated to collectUnchangedUnorderedItems.
func collectUnchangedSequenceUnordered(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	fromIdxs := make([]int, len(fromN.Content))
	for i := range fromIdxs {
		fromIdxs[i] = i
	}
	toIdxs := make([]int, len(toN.Content))
	for i := range toIdxs {
		toIdxs[i] = i
	}
	return collectUnchangedUnorderedItems(path, fromN.Content, toN.Content, fromIdxs, toIdxs, opts)
}

// collectUnchangedUnorderedItems pairs the given from/to item indices order-
// independently and reports the unchanged values within each pair. Mirrors the
// two-phase matching of compareSequenceNodesUnordered / compareUnidentifiedItems:
// exact (deepEqualNodes) matches across positions drop out first — here emitted
// as collapsed unchanged list entries keyed by their from-side index — then the
// remaining unmatched items are paired positionally and descended into so their
// equal leaves are still reported. Items unmatched on only one side emit nothing
// (they are, by definition, not unchanged).
func collectUnchangedUnorderedItems(path DiffPath, from, to []*yaml.Node, fromIdxs, toIdxs []int, opts *Options) []Difference {
	toMatched := make([]bool, len(toIdxs))

	var diffs []Difference
	var remFrom []int
	for _, fromIdx := range fromIdxs {
		matched := false
		for b, toIdx := range toIdxs {
			if toMatched[b] {
				continue
			}
			if deepEqualNodes(from[fromIdx], to[toIdx], opts) {
				toMatched[b] = true
				// Whole item is equal: collapse as a list entry at the from index.
				diffs = append(diffs, Difference{
					Path:      path.Append(strconv.Itoa(fromIdx)),
					Type:      DiffUnchanged,
					From:      nodeToInterface(from[fromIdx]),
					To:        nodeToInterface(to[toIdx]),
					listEntry: true,
				})
				matched = true
				break
			}
		}
		if !matched {
			remFrom = append(remFrom, fromIdx)
		}
	}

	// Collect the unmatched to-side indices in source order.
	var remTo []int
	for b, toIdx := range toIdxs {
		if !toMatched[b] {
			remTo = append(remTo, toIdx)
		}
	}

	// Pair the leftovers positionally and descend so partial equality within an
	// unmatched pair is still reported, keyed by the from-side index (matching
	// compareUnidentifiedItems' path convention).
	n := min(len(remFrom), len(remTo))
	for k := range n {
		diffs = append(diffs, collectUnchanged(path.Append(strconv.Itoa(remFrom[k])), from[remFrom[k]], to[remTo[k]], opts, true)...)
	}

	return diffs
}

// collectUnchangedSequenceByIdentifier pairs items sharing an identifier
// (mirroring compareSequenceNodesByIdentifier's indexing) and reports the equal
// values within each pair at the identifier-keyed child path. Items whose
// identifier exists on only one side emit nothing; items lacking a usable
// identifier fall back to order-independent pairing among themselves.
func collectUnchangedSequenceByIdentifier(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	from := fromN.Content
	to := toN.Content

	fromIndex := make(map[any]int, len(from))
	var fromNoID []int
	for i, item := range from {
		id := getIdentifierNode(item, opts)
		if isComparableIdentifier(id) {
			fromIndex[id] = i // last-write-wins, matching the normal path
			continue
		}
		fromNoID = append(fromNoID, i)
	}

	toIndex := make(map[any]int, len(to))
	var toNoID []int
	for i, item := range to {
		id := getIdentifierNode(item, opts)
		if isComparableIdentifier(id) {
			toIndex[id] = i // last-write-wins, matching the normal path
			continue
		}
		toNoID = append(toNoID, i)
	}

	var diffs []Difference
	for i, item := range from {
		id := getIdentifierNode(item, opts)
		// gomutants:disable-next-line BRANCH_IF reason="defensive; non-comparable identifiers are absent from both indexes, so the following index guards also continue"
		if !isComparableIdentifier(id) {
			continue
		}
		if fromIndex[id] != i {
			continue
		}
		toIdx, ok := toIndex[id]
		if !ok {
			continue
		}
		// Identifier-matched sequence element: a collapse here is a list item.
		diffs = append(diffs, collectUnchanged(path.Append(sprintIdentifier(id)), from[i], to[toIdx], opts, true)...)
	}

	// Order-independent fallback for items without a usable identifier, mirroring
	// compareUnidentifiedItems (exact matches first, then positional remainder).
	diffs = append(diffs, collectUnchangedUnorderedItems(path, from, to, fromNoID, toNoID, opts)...)

	return diffs
}
