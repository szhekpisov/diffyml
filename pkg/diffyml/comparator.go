// comparator.go - Core YAML comparison logic.
//
// Compares YAML documents at the AST level (as *yaml.Node trees) to detect
// semantic differences. Handles mappings, sequences (ordered, unordered, and
// identifier-matched), scalars, and multi-document files. Materializes node
// subtrees to Go values via nodeToInterface only when emitting Difference.From/
// Difference.To, never to drive recursion. Integrates with kubernetes.go for
// K8s resource matching.
package diffyml

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// compareDocs compares two slices of YAML document node trees and returns
// differences. opts is non-nil — Compare normalizes it before calling.
func compareDocs(from, to []*yaml.Node, opts *Options) []Difference {
	if opts.DetectKubernetes {
		if fromDocs, toDocs, ok := detectK8sDocsCached(from, to); ok {
			return compareK8sDocsCached(from, to, fromDocs, toDocs, opts)
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

		// Build path prefix for multi-document files.
		var pathPrefix DiffPath
		if maxLen > 1 {
			pathPrefix = DiffPath{"[" + strconv.Itoa(i) + "]"}
		}

		nodeDiffs := compareNodes(pathPrefix, fromN, toN, opts)
		for j := range nodeDiffs {
			nodeDiffs[j].DocumentIndex = i
		}
		diffs = append(diffs, nodeDiffs...)
	}

	return diffs
}

// detectK8sDocsCached materializes every from/to node to its any view exactly
// once and reports whether at least one document looks like a K8s resource.
// On a hit, the cached any slices are returned so compareK8sDocs can reuse
// them without re-walking the trees. Cost trade-off: when DetectKubernetes is
// enabled but no K8s docs are present, we still materialize the full slice;
// the previous "early-bail on first match" version only saved work in the
// rare "first doc is K8s" path because compareK8sDocs immediately
// re-materialized everything anyway.
func detectK8sDocsCached(from, to []*yaml.Node) (fromDocs, toDocs []any, ok bool) {
	fromDocs = materializeK8sDocs(from)
	toDocs = materializeK8sDocs(to)
	for _, d := range fromDocs {
		if IsKubernetesResource(d) {
			return fromDocs, toDocs, true
		}
	}
	for _, d := range toDocs {
		if IsKubernetesResource(d) {
			return fromDocs, toDocs, true
		}
	}
	return nil, nil, false
}

// resolveNode unwraps DocumentNode wrappers and dereferences AliasNodes. Stage-2
// resolveMergeKeys removes "<<" so the comparator never sees merge keys.
// resolveAlias safely no-ops on nil / non-alias inputs, so the post-DocumentNode
// hand-off is unconditional.
func resolveNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.DocumentNode {
		if len(n.Content) == 0 {
			return nil
		}
		n = n.Content[0]
	}
	return resolveAlias(n)
}

// isNullNode reports whether a node represents a YAML null (either a true nil
// pointer, an empty DocumentNode, or a scalar with the !!null tag).
func isNullNode(n *yaml.Node) bool {
	n = resolveNode(n)
	if n == nil {
		return true
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return true
	}
	return false
}

// mapEntryWrapper builds the single-key *OrderedMap wrapper used for added /
// removed map entries (matches the legacy emission shape byte-for-byte).
// nodeToInterface handles nil val (returns nil), so no separate guard is needed.
func mapEntryWrapper(key string, val *yaml.Node) *OrderedMap {
	return &OrderedMap{
		Keys:   []string{key},
		Values: map[string]any{key: nodeToInterface(val)},
	}
}

// compareNodeNils centralises every null/nil case for compareNodes and reports
// (diffs, true) when it produces the final answer. The four short-circuits:
// both null → nil; only from-side null → DiffAdded; only to-side null →
// DiffModified (or nil under IgnoreValueChanges); neither null → fall through
// to Kind dispatch in the caller. Handling the to-only-null case here keeps
// the dispatch in compareNodes free of nil-toN checks.
func compareNodeNils(path DiffPath, fromN, toN *yaml.Node, opts *Options) ([]Difference, bool) {
	fromIsNull := isNullNode(fromN)
	toIsNull := isNullNode(toN)
	if fromIsNull && toIsNull {
		return nil, true
	}
	if fromIsNull {
		return []Difference{{
			Path: path,
			Type: DiffAdded,
			From: nil,
			To:   nodeToInterface(toN),
		}}, true
	}
	if toIsNull {
		if opts.IgnoreValueChanges {
			return nil, true
		}
		return []Difference{{
			Path: path,
			Type: DiffModified,
			From: nodeToInterface(fromN),
			To:   nil,
		}}, true
	}
	return nil, false
}

// compareNodes recursively compares two YAML node trees. opts is non-nil.
// compareNodeNils handles every null/nil case (both-null, from-only-null,
// to-only-null); on fall-through both fromN and toN resolve to non-nil
// non-null nodes, so the Kind dispatch below can dereference safely.
func compareNodes(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	if diffs, done := compareNodeNils(path, fromN, toN, opts); done {
		return diffs
	}
	fromN = resolveNode(fromN)
	toN = resolveNode(toN)

	if fromN.Kind != toN.Kind {
		if opts.IgnoreValueChanges {
			return nil
		}
		return []Difference{{
			Path: path,
			Type: DiffModified,
			From: nodeToInterface(fromN),
			To:   nodeToInterface(toN),
		}}
	}

	switch fromN.Kind {
	case yaml.MappingNode:
		return compareMappingNodes(path, fromN, toN, opts)
	case yaml.SequenceNode:
		return compareSequenceNodes(path, fromN, toN, opts)
	}
	// ScalarNode: dispatch through compareScalarNodes (the only remaining
	// Kind reachable from a resolved tree post-merge).
	return compareScalarNodes(path, fromN, toN, opts)
}

// compareScalarNodes compares two ScalarNodes, materializing only the typed
// values and delegating equality logic to equalValues. equalValues correctly
// reports unequal for values of different dynamic types (Go's == on `any`
// requires both type and value match), so a type-mismatch fast path would be
// behaviorally indistinguishable from this single equality check.
func compareScalarNodes(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	fromVal := resolveScalar(fromN)
	toVal := resolveScalar(toN)

	if equalValues(fromVal, toVal, opts) {
		return nil
	}
	if opts.IgnoreValueChanges {
		return nil
	}
	return []Difference{{Path: path, Type: DiffModified, From: fromVal, To: toVal}}
}

// compareMappingNodes compares two MappingNodes preserving the from-side's
// source order. Stage-2 resolveMergeKeys removed "<<" entries, so this is a
// straight pair iteration. Duplicate keys (possible via the legacy explicit-
// key-after-merge quirk that resolveMergeKeys preserves) are handled with
// last-write-wins value lookup, matching the legacy *OrderedMap behavior
// exactly: each occurrence in fromN.Content triggers one recursion using the
// LAST value bound to that key in fromN.
//
// Both mappings are indexed once up front (last source-order position per
// key) so the per-iteration value lookups are O(1) and overall cost stays
// O(n+m) instead of the O(n*m) a naive double linear scan would produce.
func compareMappingNodes(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	fromIdx := indexMappingValues(fromN)
	toIdx := indexMappingValues(toN)

	var diffs []Difference

	for i := 0; i+1 < len(fromN.Content); i += 2 {
		key := fromN.Content[i].Value
		fromVal := fromN.Content[fromIdx[key]+1]
		toPos, inTo := toIdx[key]

		if !inTo {
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffRemoved,
				From: mapEntryWrapper(key, fromVal),
				To:   nil,
			})
			continue
		}

		toVal := toN.Content[toPos+1]
		childPath := path.Append(key)
		diffs = append(diffs, compareNodes(childPath, fromVal, toVal, opts)...)
	}

	// Additions: iterate toN.Content in source order and report keys absent
	// from fromN. Matches the legacy behavior of using to.Values map lookup.
	for i := 0; i+1 < len(toN.Content); i += 2 {
		key := toN.Content[i].Value
		if _, inFrom := fromIdx[key]; inFrom {
			continue
		}
		// last-write-wins on duplicate to-side keys: pull the value from the
		// recorded last position rather than the current i+1.
		toVal := toN.Content[toIdx[key]+1]
		diffs = append(diffs, Difference{
			Path: path,
			Type: DiffAdded,
			From: nil,
			To:   mapEntryWrapper(key, toVal),
		})
	}

	return diffs
}

// indexMappingValues records the index of the key node for each unique key in
// a MappingNode's Content slice, keeping the LAST source-order occurrence so
// last-write-wins lookup matches nodeToInterface / lookupMappingValueNode.
// The value node sits at index+1; callers add 1 to dereference. Callers must
// pass a non-nil node whose Content has even length — compareMappingNodes is
// the sole live caller and reaches here only after the Kind dispatch in
// compareNodes filters nil / non-mapping inputs.
//
// No capacity hint on the map: precise sizing would require a `len(n.Content)/2`
// expression whose ARITHMETIC_BASE mutation (`/` → `*`) is observationally
// equivalent (capacity is invisible to behaviour), so the hint was excised in
// favour of letting the map auto-grow. Mappings in this codebase are small
// enough that the missing pre-allocation does not register in benchmarks.
func indexMappingValues(n *yaml.Node) map[string]int {
	idx := make(map[string]int)
	for i := 0; i+1 < len(n.Content); i += 2 {
		idx[n.Content[i].Value] = i
	}
	return idx
}

// compareSequenceNodes compares two SequenceNodes. Dispatches to identifier-
// matched, unordered, positional, or heterogeneous-unordered strategy with
// the same semantics as the legacy compareLists.
func compareSequenceNodes(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	if canMatchByIdentifierNodes(fromN.Content, opts) && canMatchByIdentifierNodes(toN.Content, opts) {
		return compareSequenceNodesByIdentifier(path, fromN, toN, opts)
	}

	if opts.IgnoreOrderChanges {
		return compareSequenceNodesUnordered(path, fromN, toN, opts)
	}

	if areSequenceItemsHeterogeneous(fromN, toN) {
		return compareSequenceNodesUnordered(path, fromN, toN, opts)
	}

	return compareSequenceNodesPositional(path, fromN, toN, opts)
}

// areSequenceItemsHeterogeneous mirrors areListItemsHeterogeneous on nodes:
// single-key map items with distinct keys across the two lists indicate a
// heterogeneous shape (e.g. {namespaceSelector: ...} vs {ipBlock: ...}).
// Both lists must consist entirely of single-key MappingNodes; the union of
// their first keys must exceed one entry for the shape to qualify as
// heterogeneous.
func areSequenceItemsHeterogeneous(fromN, toN *yaml.Node) bool {
	fromKeys, ok := singleKeyMappingFirstKeys(fromN.Content)
	if !ok {
		return false
	}
	toKeys, ok := singleKeyMappingFirstKeys(toN.Content)
	if !ok {
		return false
	}
	if len(fromKeys) == 0 || len(toKeys) == 0 {
		return false
	}
	for k := range toKeys {
		fromKeys[k] = true
	}
	return len(fromKeys) > 1
}

// singleKeyMappingFirstKeys collects the first-key set across items. The
// second return is true only when every item resolves to a single-key
// MappingNode (Content of length 2). On false the partially-populated key set
// flows back deliberately: areSequenceItemsHeterogeneous tests pass inputs
// like `[{a:1}, scalar]` where the partial `{a}` set, combined with skipping
// the `!ok` early return (mutant), would fold the to-side keys into the
// partial set and mis-classify the shape as heterogeneous. Returning nil here
// would silently neuter those mutation kills by making the downstream
// `len(fromKeys) == 0` guard catch the mutated path. resolveNode handles
// DocumentNode/AliasNode wrappers, including the cycle-collapse-to-nil case.
func singleKeyMappingFirstKeys(items []*yaml.Node) (map[string]bool, bool) {
	keys := make(map[string]bool, len(items))
	for _, item := range items {
		item = resolveNode(item)
		if item == nil || item.Kind != yaml.MappingNode {
			return keys, false
		}
		if len(item.Content) != 2 {
			return keys, false
		}
		keys[item.Content[0].Value] = true
	}
	return keys, true
}

// compareSequenceNodesPositional compares sequences by index.
func compareSequenceNodesPositional(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	var diffs []Difference
	from := fromN.Content
	to := toN.Content
	minLen := min(len(from), len(to))

	for i := range minLen {
		childPath := path.Append(strconv.Itoa(i))
		diffs = append(diffs, compareNodes(childPath, from[i], to[i], opts)...)
	}
	for i := minLen; i < len(to); i++ {
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(i)),
			Type: DiffAdded,
			From: nil,
			To:   nodeToInterface(to[i]),
		})
	}
	for i := minLen; i < len(from); i++ {
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(i)),
			Type: DiffRemoved,
			From: nodeToInterface(from[i]),
			To:   nil,
		})
	}

	return diffs
}

// compareSequenceNodesUnordered compares sequences ignoring order. Exact
// matches (via deepEqual on the materialized any view) drop out first
// regardless of position, then remaining items are paired positionally via
// compareNodes for precise nested diffs.
func compareSequenceNodesUnordered(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	from := fromN.Content
	to := toN.Content
	fromMatched := make([]bool, len(from))
	toMatched := make([]bool, len(to))

	// Materialize once per item for the deepEqual scan — slight up-front cost
	// but avoids repeated nodeToInterface walks inside the O(N*M) loop. The
	// allocation is unbounded in the sequence length; a future node-level
	// deepEqualNodes would let us skip materialization entirely for the
	// common scalar-list case.
	// TODO(node-pipeline): replace deepEqual(any,any) with deepEqualNodes here.
	fromValues := make([]any, len(from))
	for i := range from {
		fromValues[i] = nodeToInterface(from[i])
	}
	toValues := make([]any, len(to))
	for i := range to {
		toValues[i] = nodeToInterface(to[i])
	}

	for i := range from {
		for j := range to {
			if toMatched[j] {
				continue
			}
			if deepEqual(fromValues[i], toValues[j], opts) {
				fromMatched[i] = true
				toMatched[j] = true
				break
			}
		}
	}

	var diffs []Difference
	fi, tj := 0, 0
	for fi < len(from) && tj < len(to) {
		if fromMatched[fi] {
			fi++
			continue
		}
		if toMatched[tj] {
			tj++
			continue
		}
		diffs = append(diffs, compareNodes(path.Append(strconv.Itoa(fi)), from[fi], to[tj], opts)...)
		fi++
		tj++
	}
	for ; fi < len(from); fi++ {
		if fromMatched[fi] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(fi)),
			Type: DiffRemoved,
			From: fromValues[fi],
		})
	}
	for ; tj < len(to); tj++ {
		if toMatched[tj] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(tj)),
			Type: DiffAdded,
			To:   toValues[tj],
		})
	}

	return diffs
}

// detectListOrderChanges detects order changes among matched list identifiers.
// Kept on the identifier-any view since identifiers are scalar Go values.
// The < 2 short-circuit is intentionally omitted: with 0 or 1 common entries
// the to-order sort and the slices.Equal comparison both no-op into "no
// change", so the dedicated guard would be a redundant fast path.
func detectListOrderChanges(path DiffPath, fromIDs []any, fromIndex, toIndex map[any]int, toIDCount int) *Difference {
	hasUniqueIDs := len(fromIDs) == len(fromIndex) && len(toIndex) == toIDCount
	if !hasUniqueIDs {
		return nil
	}

	var commonFromOrder []any
	for _, id := range fromIDs {
		if _, ok := toIndex[id]; ok {
			commonFromOrder = append(commonFromOrder, id)
		}
	}

	commonToOrder := make([]any, len(commonFromOrder))
	copy(commonToOrder, commonFromOrder)
	slices.SortStableFunc(commonToOrder, func(a, b any) int {
		return cmp.Compare(toIndex[a], toIndex[b])
	})

	if slices.Equal(commonFromOrder, commonToOrder) {
		return nil
	}

	return &Difference{
		Path: path,
		Type: DiffOrderChanged,
		From: commonFromOrder,
		To:   commonToOrder,
	}
}

// compareUnidentifiedItems handles items in an identifier-matched list that
// don't have a usable identifier. Falls back to unordered (deepEqual) match
// then positional pairing on the remainder.
func compareUnidentifiedItems(path DiffPath, from, to []*yaml.Node, fromNoID, toNoID []int, opts *Options) []Difference {
	fromNoIDMatched := make([]bool, len(fromNoID))
	toNoIDMatched := make([]bool, len(toNoID))

	// Materialize only the unidentified items once for deepEqual reuse.
	fromVals := make(map[int]any, len(fromNoID))
	toVals := make(map[int]any, len(toNoID))
	for _, idx := range fromNoID {
		fromVals[idx] = nodeToInterface(from[idx])
	}
	for _, idx := range toNoID {
		toVals[idx] = nodeToInterface(to[idx])
	}

	for fi, fromIdx := range fromNoID {
		for tj, toIdx := range toNoID {
			if toNoIDMatched[tj] {
				continue
			}
			if deepEqual(fromVals[fromIdx], toVals[toIdx], opts) {
				fromNoIDMatched[fi] = true
				toNoIDMatched[tj] = true
				break
			}
		}
	}

	var diffs []Difference
	fi, tj := 0, 0
	for fi < len(fromNoID) && tj < len(toNoID) {
		if fromNoIDMatched[fi] {
			fi++
			continue
		}
		if toNoIDMatched[tj] {
			tj++
			continue
		}
		diffs = append(diffs, compareNodes(path.Append(strconv.Itoa(fromNoID[fi])), from[fromNoID[fi]], to[toNoID[tj]], opts)...)
		fi++
		tj++
	}
	for ; fi < len(fromNoID); fi++ {
		if fromNoIDMatched[fi] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(fromNoID[fi])),
			Type: DiffRemoved,
			From: fromVals[fromNoID[fi]],
		})
	}
	for ; tj < len(toNoID); tj++ {
		if toNoIDMatched[tj] {
			continue
		}
		diffs = append(diffs, Difference{
			Path: path.Append(strconv.Itoa(toNoID[tj])),
			Type: DiffAdded,
			To:   toVals[toNoID[tj]],
		})
	}
	return diffs
}

// compareSequenceNodesByIdentifier compares sequences by their identifier
// field (typically "name"/"id" or AdditionalIdentifiers). Items with matching
// identifiers are diffed at the child path (dyff-style); unmatched items are
// reported at the parent path.
func compareSequenceNodesByIdentifier(path DiffPath, fromN, toN *yaml.Node, opts *Options) []Difference {
	from := fromN.Content
	to := toN.Content
	var diffs []Difference

	fromIndex := make(map[any]int, len(from))
	fromIDs := make([]any, 0, len(from))
	var fromNoID []int
	for i, item := range from {
		id := getIdentifierNode(item, opts)
		if isComparableIdentifier(id) {
			fromIndex[id] = i
			fromIDs = append(fromIDs, id)
			continue
		}
		fromNoID = append(fromNoID, i)
	}

	toIndex := make(map[any]int, len(to))
	// toIDs[i] caches the comparable identifier for to[i] when present (nil
	// otherwise), so the addition loop below can preserve to-side source
	// order without re-running getIdentifierNode on every item.
	toIDs := make([]any, len(to))
	var toNoID []int
	toIDCount := 0
	for i, item := range to {
		id := getIdentifierNode(item, opts)
		if isComparableIdentifier(id) {
			toIDCount++
			toIndex[id] = i
			toIDs[i] = id
			continue
		}
		toNoID = append(toNoID, i)
	}

	if !opts.IgnoreOrderChanges {
		if orderDiff := detectListOrderChanges(path, fromIDs, fromIndex, toIndex, toIDCount); orderDiff != nil {
			diffs = append(diffs, *orderDiff)
		}
	}

	// Modified items (matched identifier).
	for _, id := range fromIDs {
		fromIdx := fromIndex[id]
		fromItem := from[fromIdx]
		toIdx, ok := toIndex[id]
		if !ok {
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffRemoved,
				From: nodeToInterface(fromItem),
				To:   nil,
			})
			continue
		}
		toItem := to[toIdx]
		idStr := sprintIdentifier(id)
		childPath := path.Append(idStr)
		diffs = append(diffs, compareNodes(childPath, fromItem, toItem, opts)...)
	}

	// Added items (unmatched identifier on the to side), preserving to-side
	// order. Identifiers were cached in toIDs during the indexing pass; a nil
	// entry means the item lacked a comparable identifier and was already
	// routed to toNoID for the unidentified-fallback path.
	for i, toItem := range to {
		id := toIDs[i]
		if id == nil {
			continue
		}
		if _, inFrom := fromIndex[id]; !inFrom {
			diffs = append(diffs, Difference{
				Path: path,
				Type: DiffAdded,
				From: nil,
				To:   nodeToInterface(toItem),
			})
		}
	}

	// Fallback for items without usable identifiers.
	diffs = append(diffs, compareUnidentifiedItems(path, from, to, fromNoID, toNoID, opts)...)

	return diffs
}

// canMatchByIdentifier is retained as an exported-test-touched alias around
// the node-based canMatchByIdentifierNodes for any remaining any-shaped
// callers. The live pipeline uses canMatchByIdentifierNodes directly.
func canMatchByIdentifier(list []any, opts *Options) bool {
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	return CanMatchByIdentifierWithAdditional(list, additional)
}

// getIdentifier returns the identifier value from an any-shaped map (legacy
// helper kept for the public CanMatchByIdentifierWithAdditional path and for
// in-package tests).
func getIdentifier(val any, opts *Options) any {
	var additional []string
	if opts != nil {
		additional = opts.AdditionalIdentifiers
	}
	if om, ok := val.(*OrderedMap); ok {
		return getIdentifierFromOrderedMap(om, additional)
	}
	if m, ok := val.(map[string]any); ok {
		return IdentifierWithAdditional(m, additional)
	}
	return nil
}

// equalValues compares two scalar values for equality, honoring the relevant
// Options flags (FormatStrings JSON-canonical compare, IgnoreWhitespaceChanges).
func equalValues(from, to any, opts *Options) bool {
	if opts != nil {
		if fromStr, ok := from.(string); ok {
			if toStr, ok := to.(string); ok {
				if fromStr == toStr {
					return true
				}

				if opts.FormatStrings && couldBeJSON(fromStr) && couldBeJSON(toStr) {
					if equal, matched := jsonCanonicalEqual(fromStr, toStr); matched {
						return equal
					}
				}

				if opts.IgnoreWhitespaceChanges {
					return strings.TrimSpace(fromStr) == strings.TrimSpace(toStr)
				}

				return false
			}
		}
	}

	return from == to
}

// couldBeJSON returns true if s starts with '{' or '[', indicating it might be
// a JSON object or array. Used as a cheap pre-check to avoid expensive
// json.Unmarshal calls on strings that are clearly not JSON.
func couldBeJSON(s string) bool {
	return len(s) >= 2 && (s[0] == '{' || s[0] == '[')
}

// jsonCanonicalEqual attempts to parse both strings as JSON and compares their
// canonical forms. Returns (equal, true) if both parsed; (false, false) if not.
func jsonCanonicalEqual(a, b string) (bool, bool) {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false, false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false, false
	}
	ca, _ := json.Marshal(va)
	cb, _ := json.Marshal(vb)
	return bytes.Equal(ca, cb), true
}

// deepEqualOrderedMaps checks deep equality between two OrderedMaps.
func deepEqualOrderedMaps(from, to *OrderedMap, opts *Options) bool {
	if len(from.Values) != len(to.Values) {
		return false
	}
	for k, fv := range from.Values {
		tv, ok := to.Values[k]
		if !ok || !deepEqual(fv, tv, opts) {
			return false
		}
	}
	return true
}

// deepEqualMaps checks deep equality between two maps.
func deepEqualMaps(from, to map[string]any, opts *Options) bool {
	if len(from) != len(to) {
		return false
	}
	for k, fv := range from {
		tv, ok := to[k]
		if !ok || !deepEqual(fv, tv, opts) {
			return false
		}
	}
	return true
}

// deepEqualSlices checks deep equality between two slices.
func deepEqualSlices(from, to []any, opts *Options) bool {
	if len(from) != len(to) {
		return false
	}
	for i := range from {
		if !deepEqual(from[i], to[i], opts) {
			return false
		}
	}
	return true
}

// deepEqual compares two values deeply with options. Kept as an any-based
// utility: the node comparator materializes its operands once via
// nodeToInterface for the rare unordered-list / unidentified-item paths.
func deepEqual(from, to any, opts *Options) bool {
	switch fromVal := from.(type) {
	case *OrderedMap:
		if toVal, ok := to.(*OrderedMap); ok {
			return deepEqualOrderedMaps(fromVal, toVal, opts)
		}
		return false
	case map[string]any:
		if toVal, ok := to.(map[string]any); ok {
			return deepEqualMaps(fromVal, toVal, opts)
		}
		return false
	case []any:
		if toVal, ok := to.([]any); ok {
			return deepEqualSlices(fromVal, toVal, opts)
		}
		return false
	default:
		return equalValues(from, to, opts)
	}
}

// sprintIdentifier converts an identifier value to a string.
// Fast-paths common types (string, int) to avoid fmt.Sprint overhead.
func sprintIdentifier(id any) string {
	switch v := id.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprint(id)
	}
}
