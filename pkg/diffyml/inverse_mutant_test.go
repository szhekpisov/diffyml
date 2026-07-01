// White-box tests targeting specific surviving mutants in inverse.go (the
// Options.Unchanged walk). Each test is annotated with the mutant it pins.
package diffyml

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// mappingFromYAML / seqFromYAML decode a single YAML document and unwrap it to
// its root mapping/sequence node for direct white-box calls.
func mappingFromYAML(t *testing.T, src string) *yaml.Node {
	t.Helper()
	return resolveNode(nodeFromYAML(t, src))
}

// k8sCM renders a minimal ConfigMap document with a name and a data value.
func k8sCM(name, val string) string {
	return "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + name + "\ndata:\n  k: " + val + "\n"
}

// --- collectUnchangedDocs positional pairing (uneven document counts) ---

// TestCollectUnchangedDocs_UnevenPositional pins the per-side index bounds in
// the positional loop and the null guard in collectUnchanged. With uneven
// document counts one side is nil at the trailing index; the `i < len(...)`
// boundary mutants would index out of range, and the `isNullNode(...)` guards
// (and their `||`) would dereference the nil node.
// Kills CONDITIONALS_BOUNDARY at inverse.go i<len(from)/i<len(to) and the
// EXPRESSION_REMOVE/INVERT_LOGICAL mutants on the collectUnchanged null guard.
func TestCollectUnchangedDocs_UnevenPositional(t *testing.T) {
	run := func(t *testing.T, fromSrc, toSrc string) {
		t.Helper()
		from, err := parse([]byte(fromSrc))
		if err != nil {
			t.Fatalf("parse from: %v", err)
		}
		to, err := parse([]byte(toSrc))
		if err != nil {
			t.Fatalf("parse to: %v", err)
		}
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("collectUnchangedDocs panicked on uneven docs: %v", r)
			}
		}()
		diffs := collectUnchangedDocs(from, to, &Options{})
		// The shared first document (a: 1) is equal and collapses at "[0]".
		var sawDoc0 bool
		for _, d := range diffs {
			if d.Path.String() == "[0]" && d.Type == DiffUnchanged {
				sawDoc0 = true
			}
		}
		if !sawDoc0 {
			t.Errorf("expected the equal first document at [0], got %+v", diffs)
		}
	}

	t.Run("to has more docs (fromN nil at tail)", func(t *testing.T) {
		run(t, "a: 1\n", "a: 1\n---\nb: 2\n")
	})
	t.Run("from has more docs (toN nil at tail)", func(t *testing.T) {
		run(t, "a: 1\n---\nb: 2\n", "a: 1\n")
	})
}

// --- collectUnchanged kind mismatch guard ---

// TestCollectUnchanged_KindMismatchReturnsNil pins the `if fromN.Kind !=
// toN.Kind` guard. With the guard removed, a mapping-vs-sequence pair would
// descend into collectUnchangedMapping and treat the sequence's items as
// key/value pairs, spuriously reporting a match.
// Kills BRANCH_IF at inverse.go on the kind-mismatch return.
func TestCollectUnchanged_KindMismatchReturnsNil(t *testing.T) {
	from := mappingFromYAML(t, "a: b\n")   // mapping {a: b}
	to := mappingFromYAML(t, "- a\n- b\n") // sequence [a, b]
	diffs := collectUnchanged(DiffPath{}, from, to, &Options{}, false)
	if len(diffs) != 0 {
		t.Errorf("mapping vs sequence must yield no unchanged entries, got %+v", diffs)
	}
}

// --- collectUnchanged per-kind collapse branches ---

// TestCollectUnchanged_ScalarBranch pins the default (scalar) branch of
// collectUnchanged's per-kind equality dispatch, which replaced the generic
// deepEqualNodes pre-check: equal scalars collapse to a single entry, unequal
// scalars yield nothing.
func TestCollectUnchanged_ScalarBranch(t *testing.T) {
	equal := collectUnchanged(DiffPath{"k"}, scalarNode("v"), scalarNode("v"), &Options{}, false)
	if len(equal) != 1 || equal[0].Type != DiffUnchanged || equal[0].To != "v" {
		t.Fatalf("equal scalars must collapse to one unchanged entry, got %+v", equal)
	}

	if unequal := collectUnchanged(DiffPath{"k"}, scalarNode("v"), scalarNode("w"), &Options{}, false); len(unequal) != 0 {
		t.Errorf("unequal scalars must yield nothing, got %+v", unequal)
	}
}

// TestCollectUnchanged_SequenceCollapse pins the sequence whole-collapse branch:
// a fully (positionally) equal sequence collapses to a single entry rather than
// descending into its items. Since the per-kind dispatch replaced the generic
// deepEqualNodes pre-check, this branch is what preserves the highest-equal-node
// collapse for lists.
func TestCollectUnchanged_SequenceCollapse(t *testing.T) {
	from := resolveNode(nodeFromYAML(t, "- 1\n- 2\n"))
	to := resolveNode(nodeFromYAML(t, "- 1\n- 2\n"))
	diffs := collectUnchanged(DiffPath{"list"}, from, to, &Options{}, false)
	if len(diffs) != 1 || diffs[0].Type != DiffUnchanged {
		t.Fatalf("equal sequence must collapse to one unchanged entry, got %+v", diffs)
	}
}

// --- collectUnchangedMapping pair-iteration loop ---

// TestCollectUnchangedMapping_OddContentFrom pins the `i+1 < len(fromN.Content)`
// boundary and the `i+1` offset in the from-iteration loop. An odd-Content
// mapping holds a trailing key with no value; the key is also present in `to`
// so the `!inTo` short-circuit does not fire, and `<=` or `i+0` would advance
// into the missing value slot and panic.
// Kills CONDITIONALS_BOUNDARY and INTEGER_DECREMENT('1') at inverse.go:158.
func TestCollectUnchangedMapping_OddContentFrom(t *testing.T) {
	from := oddContentMapping("lonely")
	to := mappingFromYAML(t, "lonely: x\n")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("collectUnchangedMapping panicked on odd-Content from: %v", r)
		}
	}()
	if diffs := collectUnchangedMapping(DiffPath{}, from, to, &Options{}); len(diffs) != 0 {
		t.Errorf("odd-Content trailing key must be skipped, got %+v", diffs)
	}
}

// TestCollectUnchangedMapping_StepTwoSkipsValueNodes pins the `i += 2` stride.
// from has two keys, so with `i += 1` the loop would visit the first value
// node "hit" as a key; to has a "hit" key whose value is also "hit", which the
// mutant would spuriously report at path "hit".
// Kills INTEGER_DECREMENT('2') at inverse.go:158.
func TestCollectUnchangedMapping_StepTwoSkipsValueNodes(t *testing.T) {
	from := mappingFromYAML(t, "m: hit\nz: 9\n")
	to := mappingFromYAML(t, "m: x\nhit: hit\nz: 9\n")
	diffs := collectUnchangedMapping(DiffPath{}, from, to, &Options{})
	if hasPath(diffs, "hit") {
		t.Errorf("i += 1 mutant leaked a value node ('hit') as a key, got %+v", diffs)
	}
	if !hasPath(diffs, "z") {
		t.Errorf("sanity: expected the genuinely-equal key 'z', got %+v", diffs)
	}
}

// TestCollectUnchangedMapping_KeyAbsentInToSkipped pins the `if !inTo { continue }`
// guard. Without it, a from-only key would index into the to mapping using the
// zero position and spuriously match.
// Kills BRANCH_IF at inverse.go on the `!inTo` continue.
func TestCollectUnchangedMapping_KeyAbsentInToSkipped(t *testing.T) {
	from := mappingFromYAML(t, "a: 5\ngone: 1\n")
	to := mappingFromYAML(t, "a: 1\n")
	diffs := collectUnchangedMapping(DiffPath{}, from, to, &Options{})
	if len(diffs) != 0 {
		t.Errorf("from-only key must not match against position 0, got %+v", diffs)
	}
}

// TestCollectUnchangedMapping_ContinueNotBreak pins that a missing key uses
// `continue`, not `break`: a later equal key must still be reported.
// Kills INVERT_LOOP_CTRL (continue → break) at inverse.go on the `!inTo` skip.
func TestCollectUnchangedMapping_ContinueNotBreak(t *testing.T) {
	from := mappingFromYAML(t, "gone: 1\nkeep: 2\n")
	to := mappingFromYAML(t, "keep: 2\n")
	diffs := collectUnchangedMapping(DiffPath{}, from, to, &Options{})
	var sawKeep bool
	for _, d := range diffs {
		if d.Path.String() == "keep" {
			sawKeep = true
		}
	}
	if !sawKeep {
		t.Errorf("a key after a missing one must still be reported, got %+v", diffs)
	}
}

// --- collectUnchangedSequence identifier dispatch ---

// TestCollectUnchangedSequence_DispatchRequiresBothSides pins the
// `canMatchByIdentifierNodes(from) && canMatchByIdentifierNodes(to)` dispatch.
// When only one side is identifier-matchable, positional pairing must be used;
// the identifier path would pair nothing. The two sub-cases distinguish each
// operand and the `&&`.
// Kills the two EXPRESSION_REMOVE mutants and the INVERT_LOGICAL on inverse.go:176.
func TestCollectUnchangedSequence_DispatchRequiresBothSides(t *testing.T) {
	// from identifier-matchable, to not: positional pairs index 0, where the
	// shared key v is equal → items.0.v. The identifier path would emit nothing.
	t.Run("from matchable only", func(t *testing.T) {
		from := mappingSeq(t, "- name: a\n  v: 1\n")
		to := mappingSeq(t, "- v: 1\n  other: x\n")
		diffs := collectUnchangedSequence(DiffPath{}, from, to, &Options{})
		if !hasPath(diffs, "0.v") {
			t.Errorf("expected positional match at 0.v, got %+v", diffs)
		}
	})
	// to identifier-matchable, from not: still positional.
	t.Run("to matchable only", func(t *testing.T) {
		from := mappingSeq(t, "- v: 1\n  other: x\n")
		to := mappingSeq(t, "- name: a\n  v: 1\n")
		diffs := collectUnchangedSequence(DiffPath{}, from, to, &Options{})
		if !hasPath(diffs, "0.v") {
			t.Errorf("expected positional match at 0.v, got %+v", diffs)
		}
	})
}

// TestCollectUnchangedSequenceByIdentifier_KeylessItemContinues pins the
// `continue` after a keyless item is recorded: a later identified item must
// still be paired. With `break`, the trailing `name: a` item would be skipped.
// Kills INVERT_LOOP_CTRL at inverse.go:218.
func TestCollectUnchangedSequenceByIdentifier_KeylessItemContinues(t *testing.T) {
	from := mappingSeq(t, "- x: 1\n- name: a\n  v: 9\n")
	to := mappingSeq(t, "- name: a\n  v: 9\n")
	diffs := collectUnchangedSequence(DiffPath{}, from, to, &Options{})
	if !hasPath(diffs, "a") {
		t.Errorf("identified item after a keyless one must still be paired, got %+v", diffs)
	}
}

// TestCollectUnchangedSequenceByIdentifier_UnmatchedIdContinues pins the
// `if !ok { continue }` guard for an identifier present on only one side. The
// "gone" item is first (its v matches to[0].v); without the continue the mutant
// would pair it against to[0] and leak gone.v, and with `break` the later "a"
// item would be skipped.
// Kills BRANCH_IF and INVERT_LOOP_CTRL at inverse.go:221-222.
func TestCollectUnchangedSequenceByIdentifier_UnmatchedIdContinues(t *testing.T) {
	from := mappingSeq(t, "- name: gone\n  v: 9\n- name: a\n  v: 9\n")
	to := mappingSeq(t, "- name: a\n  v: 9\n")
	diffs := collectUnchangedSequence(DiffPath{}, from, to, &Options{})
	if !hasPath(diffs, "a") {
		t.Errorf("a matched item after an unmatched id must still be reported, got %+v", diffs)
	}
	for _, d := range diffs {
		if strings.Contains(d.Path.String(), "gone") {
			t.Errorf("unmatched id 'gone' must not be paired against to[0], got %+v", diffs)
		}
	}
}

// mappingSeq decodes a YAML sequence document to its root sequence node.
func mappingSeq(t *testing.T, src string) *yaml.Node {
	t.Helper()
	return resolveNode(nodeFromYAML(t, src))
}

func hasPath(diffs []Difference, path string) bool {
	for _, d := range diffs {
		if d.Path.String() == path && d.Type == DiffUnchanged {
			return true
		}
	}
	return false
}

// --- collectMatchedK8sUnchanged document-index / prefix / metadata ---

// TestCollectMatchedK8sUnchanged_UseToIdxAffectsPath pins the `if useToIdx`
// branch and the `docIdx = toIdx` assignment for rename-matched pairs. With
// fromIdx=0, toIdx=1 and useToIdx=true, the collapsed equal document must be
// reported at [1] with DocumentIndex 1.
// Kills BRANCH_IF and STATEMENT_REMOVE on inverse.go:88-89.
func TestCollectMatchedK8sUnchanged_UseToIdxAffectsPath(t *testing.T) {
	from := nodesFromYAMLT(t, k8sCM("cfg", "same")+"---\n"+k8sCM("other", "x"))
	to := nodesFromYAMLT(t, k8sCM("other", "x")+"---\n"+k8sCM("cfg", "same"))
	matched := map[int]int{0: 1}

	diffs := collectMatchedK8sUnchanged(matched, from, to, materializeK8sDocs(to), &Options{DetectKubernetes: true}, true)
	d := requireUnchanged(t, diffs)
	if !strings.HasPrefix(d.Path.String(), "[1]") {
		t.Errorf("path = %q, want [1] prefix (toIdx)", d.Path.String())
	}
	if d.DocumentIndex != 1 {
		t.Errorf("DocumentIndex = %d, want 1 (toIdx)", d.DocumentIndex)
	}
}

// TestCollectMatchedK8sUnchanged_PrefixFromHasMore pins the
// `len(fromNodes) > 1 || len(toNodes) > 1` prefix guard when only the from side
// has multiple documents. The prefix must still be set.
// Kills EXPRESSION_REMOVE(len(fromNodes)>1), INVERT_LOGICAL(||),
// INTEGER_INCREMENT, BRANCH_IF and the pathPrefix STATEMENT_REMOVE on inverse.go:93-94.
func TestCollectMatchedK8sUnchanged_PrefixFromHasMore(t *testing.T) {
	from := nodesFromYAMLT(t, k8sCM("cfg", "same")+"---\n"+k8sCM("extra", "x"))
	to := nodesFromYAMLT(t, k8sCM("cfg", "same"))
	matched := map[int]int{0: 0}

	diffs := collectMatchedK8sUnchanged(matched, from, to, materializeK8sDocs(to), &Options{DetectKubernetes: true}, false)
	d := requireUnchanged(t, diffs)
	if !strings.HasPrefix(d.Path.String(), "[0]") {
		t.Errorf("path = %q, want [0] prefix even when only from has multiple docs", d.Path.String())
	}
}

// TestCollectMatchedK8sUnchanged_PrefixToHasMore is the to-side twin, pinning
// the `len(toNodes) > 1` operand.
// Kills EXPRESSION_REMOVE(len(toNodes)>1) on inverse.go:93.
func TestCollectMatchedK8sUnchanged_PrefixToHasMore(t *testing.T) {
	from := nodesFromYAMLT(t, k8sCM("cfg", "same"))
	to := nodesFromYAMLT(t, k8sCM("extra", "x")+"---\n"+k8sCM("cfg", "same"))
	matched := map[int]int{0: 1}

	diffs := collectMatchedK8sUnchanged(matched, from, to, materializeK8sDocs(to), &Options{DetectKubernetes: true}, false)
	d := requireUnchanged(t, diffs)
	if !strings.HasPrefix(d.Path.String(), "[0]") {
		t.Errorf("path = %q, want [0] prefix when only to has multiple docs", d.Path.String())
	}
}

// TestCollectMatchedK8sUnchanged_SetsDocFields pins the DocumentIndex and
// DocumentName stamping. The matched document sits at index 1, so DocumentIndex
// must be 1 (not the zero value) and DocumentName must be populated.
// Kills STATEMENT_REMOVE on docName=displayName(), DocumentIndex and
// DocumentName assignments at inverse.go:100,104,105.
func TestCollectMatchedK8sUnchanged_SetsDocFields(t *testing.T) {
	from := nodesFromYAMLT(t, k8sCM("a", "x")+"---\n"+k8sCM("cfg", "same"))
	to := nodesFromYAMLT(t, k8sCM("a", "y")+"---\n"+k8sCM("cfg", "same"))
	matched := map[int]int{1: 1}

	diffs := collectMatchedK8sUnchanged(matched, from, to, materializeK8sDocs(to), &Options{DetectKubernetes: true}, false)
	d := requireUnchanged(t, diffs)
	if d.DocumentIndex != 1 {
		t.Errorf("DocumentIndex = %d, want 1", d.DocumentIndex)
	}
	if d.DocumentName == "" {
		t.Error("DocumentName must be populated for a matched K8s document")
	}
	if d.DocumentKind != "ConfigMap" {
		t.Errorf("DocumentKind = %q, want ConfigMap", d.DocumentKind)
	}
}

func requireUnchanged(t *testing.T, diffs []Difference) Difference {
	t.Helper()
	for _, d := range diffs {
		if d.Type == DiffUnchanged {
			return d
		}
	}
	t.Fatalf("expected an unchanged entry, got %+v", diffs)
	return Difference{}
}

// --- collectUnchangedUnorderedItems pairing (finding-2 unordered matcher) ---

// allIdx returns the index slice [0, 1, ..., n-1].
func allIdx(n int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	return idx
}

// TestCollectUnchangedUnorderedItems_NoToItemReuse pins the toMatched guard and
// its assignment: an already-paired to item must not match a second from item.
// from=[5,5], to=[5,9] has only one 5 on the to side, so exactly one item may
// collapse — the reuse mutants would pair the second from 5 against to[0] again.
// Kills BRANCH_IF on `if toMatched[b] { continue }` and STATEMENT_REMOVE on
// `toMatched[b] = true`.
func TestCollectUnchangedUnorderedItems_NoToItemReuse(t *testing.T) {
	from := mappingSeq(t, "- 5\n- 5\n").Content
	to := mappingSeq(t, "- 5\n- 9\n").Content
	diffs := collectUnchangedUnorderedItems(DiffPath{}, from, to, allIdx(len(from)), allIdx(len(to)), &Options{})
	if len(diffs) != 1 {
		t.Fatalf("expected exactly one matched item, got %d: %+v", len(diffs), diffs)
	}
	if diffs[0].Path.String() != "0" {
		t.Errorf("expected the match at index 0, got %q", diffs[0].Path.String())
	}
}

// TestCollectUnchangedUnorderedItems_BreakAfterMatch pins the break after a
// match: from=[5], to=[5,5] — without break the single from item would also pair
// to[1]=5, emitting a duplicate collapse at index 0.
// Kills the break removal in collectUnchangedUnorderedItems' phase-1 loop.
func TestCollectUnchangedUnorderedItems_BreakAfterMatch(t *testing.T) {
	from := mappingSeq(t, "- 5\n").Content
	to := mappingSeq(t, "- 5\n- 5\n").Content
	diffs := collectUnchangedUnorderedItems(DiffPath{}, from, to, allIdx(len(from)), allIdx(len(to)), &Options{})
	if len(diffs) != 1 {
		t.Fatalf("expected exactly one collapse (break stops after first match), got %d: %+v", len(diffs), diffs)
	}
}

// TestCollectUnchangedUnorderedItems_RemainderPairsFromZero pins the `pa, pb :=
// 0, 0` init and the `<` bound of the remainder loop. A partially-equal pair (no
// exact match) must be paired positionally from index 0 and descended into; a
// non-zero init would skip the pair and a `<=` bound would index past the slice.
// Kills INTEGER mutants on the 0 inits and CONDITIONALS_BOUNDARY on the `<`.
func TestCollectUnchangedUnorderedItems_RemainderPairsFromZero(t *testing.T) {
	from := mappingSeq(t, "- x: 1\n  y: 2\n").Content
	to := mappingSeq(t, "- x: 1\n  y: 9\n").Content
	diffs := collectUnchangedUnorderedItems(DiffPath{}, from, to, allIdx(len(from)), allIdx(len(to)), &Options{})
	if !hasPath(diffs, "0.x") {
		t.Errorf("expected the equal leaf 0.x from the positional remainder, got %+v", diffs)
	}
}

// TestCollectUnchangedUnorderedItems_RemainderTruncatesToMin pins the
// `min(len(remFrom), len(remTo))` remainder bound: with more unmatched from
// items than to items the loop must stop at the shorter side. A `max` would
// index past the to remainder and panic.
func TestCollectUnchangedUnorderedItems_RemainderTruncatesToMin(t *testing.T) {
	from := mappingSeq(t, "- x: 1\n  y: 2\n- x: 1\n  y: 3\n").Content
	to := mappingSeq(t, "- x: 1\n  y: 9\n").Content
	diffs := collectUnchangedUnorderedItems(DiffPath{}, from, to, allIdx(len(from)), allIdx(len(to)), &Options{})
	if !hasPath(diffs, "0.x") {
		t.Errorf("expected 0.x from the single positional pair, got %+v", diffs)
	}
}

// TestCollectUnchangedUnorderedItems_ContinueScansRemainingToItems pins that an
// already-matched to item is SKIPPED (continue), not that the scan stops
// (break). With reordered exact matches, skipping lets each from item find its
// counterpart further along; continue→break would abandon the scan after the
// first already-matched item and drop the later matches.
// Kills INVERT_LOOP_CTRL (continue → break) on the `if toMatched[b]` guard.
func TestCollectUnchangedUnorderedItems_ContinueScansRemainingToItems(t *testing.T) {
	from := mappingSeq(t, "- 10\n- 20\n- 30\n").Content
	to := mappingSeq(t, "- 10\n- 30\n- 20\n").Content
	diffs := collectUnchangedUnorderedItems(DiffPath{}, from, to, allIdx(len(from)), allIdx(len(to)), &Options{})
	for _, p := range []string{"0", "1", "2"} {
		if !hasPath(diffs, p) {
			t.Errorf("expected all reordered items matched (continue, not break); missing %q in %+v", p, diffs)
		}
	}
}

// TestCollectUnchangedUnorderedItems_MixedMatchAndRemainder pins the remainder
// bookkeeping when phase 1 matches some items: the exact-matched to item must be
// excluded from the remainder (the `if !toMatched[b]` collector) and the
// unmatched from item must be carried into remFrom (the `if !matched` collector)
// so its partial equality is still found. from=[{a:1,z:7},{m:1}],
// to=[{m:1},{a:9,z:7}] matches the {m:1} pair across the reorder, then pairs the
// leftover {a:*,z:7} items and reports their equal z.
func TestCollectUnchangedUnorderedItems_MixedMatchAndRemainder(t *testing.T) {
	from := mappingSeq(t, "- a: 1\n  z: 7\n- m: 1\n").Content
	to := mappingSeq(t, "- m: 1\n- a: 9\n  z: 7\n").Content
	diffs := collectUnchangedUnorderedItems(DiffPath{}, from, to, allIdx(len(from)), allIdx(len(to)), &Options{})
	if !hasPath(diffs, "1") {
		t.Errorf("expected the exact-matched item collapsed at index 1, got %+v", diffs)
	}
	if !hasPath(diffs, "0.z") {
		t.Errorf("expected the leftover pair's equal leaf 0.z, got %+v", diffs)
	}
}
