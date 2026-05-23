// Tests in this file target specific surviving mutants discovered by
// gomutants. Each test is annotated with the mutant it pins. Adjacent
// behavioral tests live alongside their feature files; this file collects
// the ones whose purpose is mutation-kill rather than feature coverage.
package diffyml

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

// --- chroot.go ---

// TestNavigateToPath_AliasMappingValue exercises the in-loop `resolveNode`
// call in navigateToPath. lookupMappingValueNode returns the AliasNode as-is,
// so without the loop-top resolve the next segment would see Kind=AliasNode
// and bail with "expected map". Kills the loop-top STATEMENT_REMOVE mutant.
func TestNavigateToPath_AliasMappingValue(t *testing.T) {
	doc := nodeFromYAML(t, `
defaults: &d
  port: 80
config: *d
`)
	result, err := navigateToPath(doc, "config.port")
	if err != nil {
		t.Fatalf("navigateToPath(\"config.port\") failed: %v", err)
	}
	if got := nodeToInterface(result); got != 80 {
		t.Errorf("expected port=80, got %v", got)
	}
}

// cyclicAliasMapping returns a synthetic DocumentNode whose top-level mapping
// has one key `key` whose value is a self-referential AliasNode. resolveNode
// will resolve that alias to nil via resolveAlias's cycle guard, exercising
// the nil-current branches of navigateToPath.
func cyclicAliasMapping(key string) *yaml.Node {
	cyclic := &yaml.Node{Kind: yaml.AliasNode}
	cyclic.Alias = cyclic
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
					cyclic,
				},
			},
		},
	}
}

// TestNavigateToPath_CyclicAliasBeforeIndex pins the `current == nil` guard
// on the index branch of navigateToPath: after the loop-top unwrap resolves a
// cyclic alias to nil, the index check must short-circuit instead of
// dereferencing `current.Kind`.
// Kills EXPRESSION_REMOVE `current == nil → false` at chroot.go:52:7.
func TestNavigateToPath_CyclicAliasBeforeIndex(t *testing.T) {
	doc := cyclicAliasMapping("items")
	_, err := navigateToPath(doc, "items[0]")
	if err == nil {
		t.Fatal("expected error navigating index into cyclic-alias value, got nil")
	}
}

// TestNavigateToPath_CyclicAliasBeforeKey is the map-key twin of the above.
// Kills EXPRESSION_REMOVE `current == nil → false` at chroot.go:69:6.
func TestNavigateToPath_CyclicAliasBeforeKey(t *testing.T) {
	doc := cyclicAliasMapping("config")
	_, err := navigateToPath(doc, "config.field")
	if err == nil {
		t.Fatal("expected error navigating key into cyclic-alias value, got nil")
	}
}

// TestApplyChroot_CyclicAliasListToDocuments covers the `target != nil` guard
// in applyChroot's listToDocuments branch. With a chroot that lands on a
// cyclic alias, resolveNode yields nil and the SequenceNode-expand path
// must be skipped — the function falls through to the single-doc wrap.
// Kills the `target != nil → true` EXPRESSION_REMOVE mutant.
func TestApplyChroot_CyclicAliasListToDocuments(t *testing.T) {
	doc := cyclicAliasMapping("items")
	result, err := applyChroot(doc, "items", true)
	if err != nil {
		t.Fatalf("applyChroot on cyclic-alias value failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected single-doc wrap (target unwraps to nil, not a sequence), got %d docs", len(result))
	}
}

// --- comparator.go ---

// TestResolveNode_DocumentScalar pins that a non-empty DocumentNode hands
// back its single Content entry. Locks the `n = n.Content[0]` step against
// STATEMENT_REMOVE / DocumentNode-branch BRANCH_IF mutations.
func TestResolveNode_DocumentScalar(t *testing.T) {
	scalar := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{scalar}}
	if got := resolveNode(doc); got != scalar {
		t.Errorf("DocumentNode should resolve to its single content entry, got %v", got)
	}
}

// TestResolveNode_DocumentWithNilContentEntry pins the nil-tolerance after
// the `n = n.Content[0]` step when Content[0] is nil.
func TestResolveNode_DocumentWithNilContentEntry(t *testing.T) {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{nil}}
	if got := resolveNode(doc); got != nil {
		t.Errorf("DocumentNode with nil Content[0] should resolve to nil, got %v", got)
	}
}

// TestIsNullNode_DocumentWrappingNullScalar pins the leading
// `n = resolveNode(n)` step in isNullNode: without it, a DocumentNode wrapper
// would mask the underlying !!null scalar and the function would report false.
// Kills STATEMENT_REMOVE at comparator.go:95:2.
func TestIsNullNode_DocumentWrappingNullScalar(t *testing.T) {
	nullScalar := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "~"}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{nullScalar}}
	if !isNullNode(doc) {
		t.Error("DocumentNode wrapping a !!null scalar should be reported as null")
	}
}

// TestIsNullNode_NonScalarWithNullTag pins the Kind == ScalarNode guard:
// a MappingNode (or any non-scalar) tagged "!!null" must NOT be reported as
// null — only scalars with that tag qualify.
// Kills EXPRESSION_REMOVE `n.Kind == yaml.ScalarNode → true` at comparator.go:99:5.
func TestIsNullNode_NonScalarWithNullTag(t *testing.T) {
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!null"}
	if isNullNode(mapping) {
		t.Error("MappingNode with !!null tag should not be reported as null")
	}
}

// TestIsNullNode_NullScalar pins that a plain ScalarNode tagged !!null reports
// true. Covers the `n.Kind == yaml.ScalarNode` and `n.Tag == "!!null"` halves
// of the conjunct, plus the return-true block.
// Kills CONDITIONALS_NEGATION at 99:12 and BRANCH_IF at 99:52.
func TestIsNullNode_NullScalar(t *testing.T) {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "~"}
	if !isNullNode(n) {
		t.Error("ScalarNode tagged !!null should be reported as null")
	}
}

// oddContentMapping constructs a malformed MappingNode whose Content slice has
// odd length — a single key without a paired value. Real YAML never produces
// this shape, but comparator/lookup loops bound by `i+1 < len(Content)` must
// still tolerate it; the boundary mutant `< → <=` would slip past the end
// of the slice and panic on the implicit value access.
func oddContentMapping(key string) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		},
	}
}

// TestCompareMappingNodes_OddContentFrom pins the `i+1 < len(fromN.Content)`
// boundary in compareMappingNodes' from-iteration loop.
// Kills CONDITIONALS_BOUNDARY at comparator.go:232:18.
func TestCompareMappingNodes_OddContentFrom(t *testing.T) {
	from := oddContentMapping("lonely")
	to := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("compareMappingNodes panicked on odd-Content from: %v", r)
		}
	}()
	diffs := compareMappingNodes(DiffPath{}, from, to, &Options{})
	if len(diffs) != 0 {
		t.Errorf("expected no diffs from odd-Content trailing key, got %d", len(diffs))
	}
}

// TestCompareMappingNodes_OddContentTo pins the `i+1 < len(toN.Content)`
// boundary in compareMappingNodes' to-iteration loop.
// Kills CONDITIONALS_BOUNDARY at comparator.go:253:18.
func TestCompareMappingNodes_OddContentTo(t *testing.T) {
	from := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	to := oddContentMapping("lonely")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("compareMappingNodes panicked on odd-Content to: %v", r)
		}
	}()
	diffs := compareMappingNodes(DiffPath{}, from, to, &Options{})
	if len(diffs) != 0 {
		t.Errorf("expected no diffs from odd-Content trailing to-side key, got %d", len(diffs))
	}
}

// TestCompareSequenceNodesByIdentifier_AllRemovedAccumulate pins the
// `continue` in the matched-id loop: with `break` instead, only the first
// removed item would be reported.
// Kills INVERT_LOOP_CTRL at comparator.go:594:4.
func TestCompareSequenceNodesByIdentifier_AllRemovedAccumulate(t *testing.T) {
	from := nodeFromYAML(t, "- name: a\n- name: b\n- name: c\n").Content[0]
	to := nodeFromYAML(t, "- name: d\n").Content[0]
	diffs := compareSequenceNodes(DiffPath{}, from, to, &Options{})
	removed := 0
	for _, d := range diffs {
		if d.Type == DiffRemoved {
			removed++
		}
	}
	if removed != 3 {
		t.Errorf("expected 3 DiffRemoved entries (a, b, c), got %d (diffs=%+v)", removed, diffs)
	}
}

// TestCompareSequenceNodesByIdentifier_AddedAfterUnidentifiedItem pins the
// `continue` in the to-side added-items loop: with `break`, an added item
// appearing after an item without a usable identifier would be missed.
// Kills INVERT_LOOP_CTRL at comparator.go:606:4.
func TestCompareSequenceNodesByIdentifier_AddedAfterUnidentifiedItem(t *testing.T) {
	from := nodeFromYAML(t, "- name: z\n").Content[0]
	// to[0] has no name/id (just a value field) so its identifier is nil and
	// the matched-id added loop continues past it; to[1] has a name and must
	// still be flagged as added.
	to := nodeFromYAML(t, "- value: 1\n- name: y\n").Content[0]
	diffs := compareSequenceNodes(DiffPath{}, from, to, &Options{})
	hasAddedY := false
	for _, d := range diffs {
		if d.Type != DiffAdded {
			continue
		}
		om, ok := d.To.(*OrderedMap)
		if !ok {
			continue
		}
		if name, _ := om.Values["name"].(string); name == "y" {
			hasAddedY = true
		}
	}
	if !hasAddedY {
		t.Errorf("expected DiffAdded for {name: y} after the unidentified item, got diffs=%+v", diffs)
	}
}

// TestAreSequenceItemsHeterogeneous_NilItem pins the resolveNode-then-nil
// guard in singleKeyMappingFirstKeys: a cyclic-alias item resolves to nil and
// must short-circuit to a non-heterogeneous result. Also doubles as the
// EXPRESSION_REMOVE kill for the `item == nil` half of the conjunct.
func TestAreSequenceItemsHeterogeneous_NilItem(t *testing.T) {
	cyclic := &yaml.Node{Kind: yaml.AliasNode}
	cyclic.Alias = cyclic
	from := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{cyclic}}
	to := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "k"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "v"},
		}},
	}}
	if areSequenceItemsHeterogeneous(from, to) {
		t.Error("a cyclic-alias item should disqualify the heterogeneous-shape check")
	}
}

// TestAreSequenceItemsHeterogeneous_AliasResolvedToMapping pins the
// resolveNode dispatch: alias items pointing at single-key mappings must be
// extracted (post-resolve) and contribute to the distinct-keys union.
func TestAreSequenceItemsHeterogeneous_AliasResolvedToMapping(t *testing.T) {
	doc := nodeFromYAML(t, `
defs:
  - &one
    a: 1
  - &two
    b: 2
from:
  - *one
to:
  - *two
`)
	root := doc.Content[0] // inside DocumentNode
	from := lookupMappingValueNode(root, "from")
	to := lookupMappingValueNode(root, "to")
	if !areSequenceItemsHeterogeneous(from, to) {
		t.Error("alias-to-mapping items with distinct first keys should be heterogeneous")
	}
}

// TestAreSequenceItemsHeterogeneous_MultiKeyDisqualifies pins the
// `len(item.Content) != 2` guard: a mapping with two keys breaks the single-
// key invariant and the function must return false.
func TestAreSequenceItemsHeterogeneous_MultiKeyDisqualifies(t *testing.T) {
	from := nodeFromYAML(t, "- a: 1\n  b: 2\n").Content[0]
	to := nodeFromYAML(t, "- c: 3\n").Content[0]
	if areSequenceItemsHeterogeneous(from, to) {
		t.Error("multi-key item must veto the heterogeneous-shape check")
	}
}

// TestAreSequenceItemsHeterogeneous_EmptyFromMultiToKey pins the
// `len(fromKeys) == 0` half of the empty-side guard. With the guard removed
// (mutant), the function would fold to-side keys into an empty from-set and
// report heterogeneous when the from list is in fact silent.
// Kills EXPRESSION_REMOVE / BRANCH_IF at comparator.go:276 (fromKeys half).
func TestAreSequenceItemsHeterogeneous_EmptyFromMultiToKey(t *testing.T) {
	from := &yaml.Node{Kind: yaml.SequenceNode}
	to := nodeFromYAML(t, "- a: 1\n- b: 2\n").Content[0]
	if areSequenceItemsHeterogeneous(from, to) {
		t.Error("empty from-side must not be reported as heterogeneous")
	}
}

// TestAreSequenceItemsHeterogeneous_MultiFromKeyEmptyTo mirrors the previous
// test for the toKeys half of the guard. With the to-side check removed, the
// function would inspect from-side distinctness while ignoring the empty
// to-side and mis-classify the shape as heterogeneous.
// Kills EXPRESSION_REMOVE at comparator.go:276:27 and INVERT_LOGICAL at 276:24.
func TestAreSequenceItemsHeterogeneous_MultiFromKeyEmptyTo(t *testing.T) {
	from := nodeFromYAML(t, "- a: 1\n- b: 2\n").Content[0]
	to := &yaml.Node{Kind: yaml.SequenceNode}
	if areSequenceItemsHeterogeneous(from, to) {
		t.Error("empty to-side must not be reported as heterogeneous")
	}
}

// TestAreSequenceItemsHeterogeneous_BadItemAfterGoodFrom pins the !ok return
// on the from-side singleKeyMappingFirstKeys call. With the partial-key
// return semantics, the from list `[{a:1}, scalar]` yields keys={a} but
// ok=false; the caller must still bail out on !ok rather than fold a-key
// into a to-side union and falsely report heterogeneous.
// Kills BRANCH_IF at comparator.go:269:9.
func TestAreSequenceItemsHeterogeneous_BadItemAfterGoodFrom(t *testing.T) {
	from := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a"},
			{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
		}},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "scalar-item"},
	}}
	to := nodeFromYAML(t, "- x: 1\n- y: 2\n").Content[0]
	if areSequenceItemsHeterogeneous(from, to) {
		t.Error("from list with a non-mapping tail must disqualify even with partial keys collected")
	}
}

// TestAreSequenceItemsHeterogeneous_BadItemAfterGoodTo is the symmetric kill
// for the to-side !ok return.
// Kills BRANCH_IF at comparator.go:273:9.
func TestAreSequenceItemsHeterogeneous_BadItemAfterGoodTo(t *testing.T) {
	from := nodeFromYAML(t, "- x: 1\n- y: 2\n").Content[0]
	to := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a"},
			{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
		}},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "scalar-item"},
	}}
	if areSequenceItemsHeterogeneous(from, to) {
		t.Error("to list with a non-mapping tail must disqualify even with partial keys collected")
	}
}

// TestAreSequenceItemsHeterogeneous_SequenceItemAsMapping pins the
// `item.Kind != yaml.MappingNode` half of singleKeyMappingFirstKeys' shape
// guard. With the guard removed (mutant), a 2-element SequenceNode would
// slip through and have its first element's Value harvested as a "key".
// Kills EXPRESSION_REMOVE at comparator.go:295:21.
func TestAreSequenceItemsHeterogeneous_SequenceItemAsMapping(t *testing.T) {
	from := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
		{Kind: yaml.SequenceNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "fromKey"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "fromVal"},
		}},
	}}
	to := nodeFromYAML(t, "- toKey: v\n").Content[0]
	if areSequenceItemsHeterogeneous(from, to) {
		t.Error("a 2-element SequenceNode item must not be treated as a single-key mapping")
	}
}

// --- identifier_node.go ---

// TestGetIdentifierNode_NilInput pins the `if n == nil { return nil }` guard
// post-resolveAlias. resolveAlias(nil)=nil, so the guard fires.
// Kills BRANCH_IF / EXPRESSION_REMOVE / CONDITIONALS_NEGATION on the nil
// check at identifier_node.go:37 (post-split).
func TestGetIdentifierNode_NilInput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("getIdentifierNode(nil) panicked: %v", r)
		}
	}()
	if got := getIdentifierNode(nil, nil); got != nil {
		t.Errorf("expected nil for nil input, got %v", got)
	}
}

// TestGetIdentifierNode_SequenceWithNameKey pins the `Kind != MappingNode`
// guard. A synthetic 2-element SequenceNode whose first child has Value
// "name" would, without the kind guard, look like a mapping with key "name"
// and yield the second child's value. The kind guard must short-circuit
// before lookupMappingValueNode walks the sequence as pairs.
// Kills the kind-half BRANCH_IF/EXPRESSION_REMOVE mutants at
// identifier_node.go:37.
func TestGetIdentifierNode_SequenceWithNameKey(t *testing.T) {
	n := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "matched"},
	}}
	if got := getIdentifierNode(n, nil); got != nil {
		t.Errorf("non-mapping input must yield nil identifier, got %#v", got)
	}
}

// TestGetIdentifierNode_MappingReturnsValue pins the happy path so the
// CONDITIONALS_NEGATION mutants on the nil/Kind guards (which would flip
// them into rejecting valid mappings) are observable.
func TestGetIdentifierNode_MappingReturnsValue(t *testing.T) {
	doc := nodeFromYAML(t, "name: alice\n")
	if got := getIdentifierNode(doc.Content[0], nil); got != "alice" {
		t.Errorf("expected identifier \"alice\", got %#v", got)
	}
}

// TestCanMatchByIdentifierNodes_AliasItemResolved pins the resolveAlias call
// inside canMatchByIdentifierNodes' loop. With the call removed (mutant),
// alias-to-mapping items fall through to the Kind guard as AliasNodes and
// the function reports false even though the list is well-formed.
// Kills STATEMENT_REMOVE at identifier_node.go:66:3.
func TestCanMatchByIdentifierNodes_AliasItemResolved(t *testing.T) {
	doc := nodeFromYAML(t, `
defs:
  - &one
    name: a
list:
  - *one
`)
	root := doc.Content[0]
	list := lookupMappingValueNode(root, "list")
	if !canMatchByIdentifierNodes(list.Content, nil) {
		t.Error("alias-to-mapping list items must be resolved before the kind check")
	}
}

// TestCanMatchByIdentifierNodes_CyclicAliasItem pins the `item == nil` guard
// post-resolveAlias: a cyclic alias resolves to nil and the loop must
// short-circuit to false rather than dereference `item.Kind`.
// Kills EXPRESSION_REMOVE at identifier_node.go:67:6 (post-split).
func TestCanMatchByIdentifierNodes_CyclicAliasItem(t *testing.T) {
	cyclic := &yaml.Node{Kind: yaml.AliasNode}
	cyclic.Alias = cyclic
	items := []*yaml.Node{cyclic}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("canMatchByIdentifierNodes panicked on cyclic alias: %v", r)
		}
	}()
	if canMatchByIdentifierNodes(items, nil) {
		t.Error("cyclic-alias item should disqualify the list")
	}
}

// TestCanMatchByIdentifierNodes_MixedListNonMappingDisqualifies pins the
// `Kind != MappingNode` guard. With the guard removed, a list with one
// good mapping and one sequence item would still report true because the
// good mapping contributes a usable identifier — but the function's
// contract is "every item must be a mapping".
// Kills BRANCH_IF / EXPRESSION_REMOVE on the kind half at
// identifier_node.go:67.
func TestCanMatchByIdentifierNodes_MixedListNonMappingDisqualifies(t *testing.T) {
	items := []*yaml.Node{
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "alice"},
		}},
		{Kind: yaml.SequenceNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"},
		}},
	}
	if canMatchByIdentifierNodes(items, nil) {
		t.Error("a non-mapping item must veto the whole list")
	}
}

// TestCanMatchByIdentifierNodes_AllMappings pins the happy path so that
// CONDITIONALS_NEGATION mutants flipping `Kind != Mapping` to `Kind ==
// Mapping` (which would reject valid mappings) are observable.
func TestCanMatchByIdentifierNodes_AllMappings(t *testing.T) {
	items := []*yaml.Node{
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "alice"},
		}},
	}
	if !canMatchByIdentifierNodes(items, nil) {
		t.Error("a list of mapping items with identifiers must be matchable")
	}
}

// TestLookupMappingValueNode_OddContent pins the `i+1 < len(n.Content)`
// boundary in lookupMappingValueNode. With `<=`, the loop would access
// Content[len] on the trailing dangling key and panic.
// Kills CONDITIONALS_BOUNDARY at identifier_node.go:86:18.
func TestLookupMappingValueNode_OddContent(t *testing.T) {
	n := oddContentMapping("dangling")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("lookupMappingValueNode panicked on odd Content: %v", r)
		}
	}()
	if got := lookupMappingValueNode(n, "dangling"); got != nil {
		t.Errorf("expected nil for trailing dangling key, got %v", got)
	}
}

// --- diffyml.go pathWalker.walk ---

// TestPathWalker_NestedAliasNilTarget pins the `if target == nil { return }`
// short-circuit in pathWalker.walk's AliasNode case when the alias appears
// mid-tree (non-root buf). Without the guard the function would recurse into
// walk(nil), trigger w.register() with the non-empty buf, and pollute
// pathOrder with the parent path of the broken alias.
// Kills BRANCH_IF at diffyml.go:219:20.
func TestPathWalker_NestedAliasNilTarget(t *testing.T) {
	nilAlias := &yaml.Node{Kind: yaml.AliasNode} // Alias == nil
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "broken"},
		nilAlias,
	}}
	w := pathWalker{
		pathOrder: make(map[string]int),
		opts:      &Options{},
		buf:       make([]byte, 0, 32),
	}
	w.walk(mapping)
	if _, ok := w.pathOrder["broken"]; ok {
		t.Errorf("nil-target alias must not register a path for its parent key; pathOrder=%v", w.pathOrder)
	}
}

// TestPathWalker_AliasSeenCleanupAcrossSiblings pins the
// `delete(w.aliasSeen, target)` cleanup. Two sibling mapping values that
// both reference the same anchored mapping must both descend into the
// anchor and register their nested paths. Without the cleanup the second
// reference would be blocked by the aliasSeen entry left over from the
// first descent.
// Kills STATEMENT_REMOVE at diffyml.go:230:3.
func TestPathWalker_AliasSeenCleanupAcrossSiblings(t *testing.T) {
	src := []byte(`
template: &t
  inner: v
a:
  ref: *t
b:
  ref: *t
`)
	nodes, err := parseNodes(src)
	if err != nil {
		t.Fatal(err)
	}
	order := extractPathOrder(nodes, nil, &Options{})
	if _, ok := order["b.ref.inner"]; !ok {
		t.Errorf("expected b.ref.inner registered after sibling-alias cleanup; pathOrder=%v", order)
	}
}

// TestPathWalker_MappingOddContent pins the `i+1 < len(n.Content)` boundary
// in pathWalker.walk's MappingNode case. With `<=`, the loop would access
// Content[len] on a trailing dangling key and panic.
// Kills CONDITIONALS_BOUNDARY at diffyml.go:233:19.
func TestPathWalker_MappingOddContent(t *testing.T) {
	n := oddContentMapping("dangling")
	w := pathWalker{
		pathOrder: make(map[string]int),
		opts:      &Options{},
		buf:       make([]byte, 0, 32),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("pathWalker.walk panicked on odd-Content mapping: %v", r)
		}
	}()
	w.walk(n)
}

// --- node_merge.go ---

// TestResolveMappingMergeKeys_OuterOddContent pins the
// `i+1 < len(n.Content)` boundary in resolveMappingMergeKeys' outer pair
// iteration. A malformed mapping with a trailing dangling key would,
// under `<=`, push the loop past the end of Content and panic.
// Kills CONDITIONALS_BOUNDARY at node_merge.go:70:18.
func TestResolveMappingMergeKeys_OuterOddContent(t *testing.T) {
	host := oddContentMapping("dangling")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("resolveMappingMergeKeys panicked on odd Content: %v", r)
		}
	}()
	resolveMappingMergeKeys(host, map[*yaml.Node]bool{})
}

// TestResolveMappingMergeKeys_SourceOddContent pins the matching boundary in
// the inner source-content loop. A merge whose source happens to be a
// malformed mapping with odd Content would index past the end under `<=`.
// Kills CONDITIONALS_BOUNDARY at node_merge.go:89:20.
func TestResolveMappingMergeKeys_SourceOddContent(t *testing.T) {
	source := oddContentMapping("lone")
	host := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		source,
	}}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("resolveMappingMergeKeys panicked on odd-Content source: %v", r)
		}
	}()
	resolveMappingMergeKeys(host, map[*yaml.Node]bool{})
}

// TestResolveMappingMergeKeys_NilSourceFromCyclicAlias pins the
// `source == nil` guard. A self-referential merge alias resolves to nil via
// resolveAlias's cycle break; the loop must `continue` rather than
// dereference `source.Kind`.
// Kills EXPRESSION_REMOVE at node_merge.go:79:7.
func TestResolveMappingMergeKeys_NilSourceFromCyclicAlias(t *testing.T) {
	selfAlias := &yaml.Node{Kind: yaml.AliasNode}
	selfAlias.Alias = selfAlias
	host := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		selfAlias,
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "k"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "v"},
	}}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("resolveMappingMergeKeys panicked on cyclic merge alias: %v", r)
		}
	}()
	resolveMappingMergeKeys(host, map[*yaml.Node]bool{})
	// Even with the broken merge, the explicit pair must survive.
	if got := lookupMappingValueNode(host, "k"); got == nil || got.Value != "v" {
		t.Errorf("explicit k:v should survive a broken merge; got %v", got)
	}
}

// TestResolveMappingMergeKeys_MultipleMergesContinuePastCycle pins the
// `continue` inside the cycles[source] short-circuit. With `break`, a second
// well-formed `<<` entry following a cyclic one would be silently dropped
// and its keys would not be merged into the host.
// Kills INVERT_LOOP_CTRL at node_merge.go:84:5.
func TestResolveMappingMergeKeys_MultipleMergesContinuePastCycle(t *testing.T) {
	target := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a"},
		{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
	}}
	host := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	hostAlias := &yaml.Node{Kind: yaml.AliasNode, Alias: host}
	targetAlias := &yaml.Node{Kind: yaml.AliasNode, Alias: target}
	host.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		hostAlias, // refers back to host → cycles[host]==true → continue
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		targetAlias, // well-formed merge that must still be applied
	}
	resolveMappingMergeKeys(host, map[*yaml.Node]bool{})
	if got := lookupMappingValueNode(host, "a"); got == nil || got.Value != "1" {
		t.Errorf("merge after the cyclic-source entry should still apply; got %v", got)
	}
}

// TestResolveMappingMergeKeys_NestedMergeInSourceFlattens pins the
// recursive `resolveMappingMergeKeys(source, cycles)` call. Without it the
// source's own `<<` entries leak into the host as literal "<<" keys.
// Kills STATEMENT_REMOVE at node_merge.go:88:4.
func TestResolveMappingMergeKeys_NestedMergeInSourceFlattens(t *testing.T) {
	src := []byte(`
inner: &inner
  x: 1
outer: &outer
  <<: *inner
host:
  <<: *outer
  z: 3
`)
	nodes, err := parseNodes(src)
	if err != nil {
		t.Fatal(err)
	}
	root := nodes[0].Content[0]
	host := lookupMappingValueNode(root, "host")
	for i := 0; i+1 < len(host.Content); i += 2 {
		if host.Content[i].Value == "<<" {
			t.Errorf("expected nested merge to be flattened; found leftover \"<<\" in host at index %d", i)
		}
	}
	if got := lookupMappingValueNode(host, "x"); got == nil || got.Value != "1" {
		t.Errorf("expected inner.x=1 merged into host via outer; got %v", got)
	}
}

// TestResolveMappingMergeKeys_DuplicateKeyAcrossMerges pins the
// `seen[mk.Value] = true` bookkeeping. Without it, a key present in
// multiple merge sources would be appended once per source instead of just
// once (first source wins, matching nodeToInterface).
// Kills STATEMENT_REMOVE at node_merge.go:95:5.
func TestResolveMappingMergeKeys_DuplicateKeyAcrossMerges(t *testing.T) {
	s1 := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "shared"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "from-s1"},
	}}
	s2 := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "shared"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "from-s2"},
	}}
	host := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		{Kind: yaml.AliasNode, Alias: s1},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		{Kind: yaml.AliasNode, Alias: s2},
	}}
	resolveMappingMergeKeys(host, map[*yaml.Node]bool{})
	occurrences := 0
	for i := 0; i+1 < len(host.Content); i += 2 {
		if host.Content[i].Value == "shared" {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Errorf("expected \"shared\" to appear exactly once after dedup, got %d", occurrences)
	}
}

// TestResolveMappingMergeKeys_NestedMergeInExplicitValue pins the recursive
// `resolveMergeKeysWithCycles(valNode, cycles)` call for non-merge values.
// Without it, a nested mapping under a regular key keeps its "<<" entries
// instead of having them flattened.
// Kills STATEMENT_REMOVE at node_merge.go:104:3.
func TestResolveMappingMergeKeys_NestedMergeInExplicitValue(t *testing.T) {
	src := []byte(`
inner: &inner
  x: 1
host:
  nested:
    <<: *inner
    z: 3
`)
	nodes, err := parseNodes(src)
	if err != nil {
		t.Fatal(err)
	}
	root := nodes[0].Content[0]
	host := lookupMappingValueNode(root, "host")
	nested := lookupMappingValueNode(host, "nested")
	for i := 0; i+1 < len(nested.Content); i += 2 {
		if nested.Content[i].Value == "<<" {
			t.Errorf("expected nested mapping's merge keys to be flattened; found \"<<\" at index %d", i)
		}
	}
}

// TestResolveAlias_NilInput pins the `n != nil` half of resolveAlias's loop
// condition. With the guard removed (mutant), the loop would dereference a
// nil n and panic.
// Kills EXPRESSION_REMOVE at node_merge.go:117:6.
func TestResolveAlias_NilInput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("resolveAlias(nil) panicked: %v", r)
		}
	}()
	if got := resolveAlias(nil); got != nil {
		t.Errorf("resolveAlias(nil) should return nil, got %v", got)
	}
}

// TestResolveMappingMergeKeys_OddSourceTailDropped pins the `j+1 < len` pair
// boundary inside resolveMappingMergeKeys's source-flatten loop. A merge
// source with a malformed odd-length Content must drop the trailing dangling
// entry without panicking — the boundary mutant `<=` would push the loop past
// the end and panic on `source.Content[j+1]`. Also locks the (k, v) pairing
// against any reshuffle of the appended entries.
// Kills CONDITIONALS_BOUNDARY and STATEMENT_REMOVE at the inline pair loop.
func TestResolveMappingMergeKeys_OddSourceTailDropped(t *testing.T) {
	src := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a"},
		{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "dangling"},
	}}
	host := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		{Kind: yaml.AliasNode, Alias: src},
	}}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("resolveMergeKeys panicked on odd-Content merge source: %v", r)
		}
	}()
	resolveMergeKeys(host)
	if len(host.Content) != 2 {
		t.Fatalf("expected single merged pair after dropping dangling tail, got %d entries", len(host.Content))
	}
	if k, v := host.Content[0], host.Content[1]; k.Value != "a" || v.Value != "1" {
		t.Errorf("expected merged pair a=1 in order, got %s=%s", k.Value, v.Value)
	}
}

// TestResolveMappingMergeKeys_SourceFlattenViaAliasOnly pins the recursive
// `resolveMappingMergeKeys(source, cycles)` call inside the merge branch.
// In a synthetic tree where the merge source is reachable ONLY via an alias
// (never visited as a direct mapping value), the outer recursion never
// flattens it, so the in-merge recursive call is the sole opportunity. With
// the call removed (mutant), the source's own "<<" entries leak through and
// get appended verbatim into the host's Content.
// Kills STATEMENT_REMOVE at node_merge.go:91 (the in-merge recursive
// flatten).
func TestResolveMappingMergeKeys_SourceFlattenViaAliasOnly(t *testing.T) {
	inner := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"},
		{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
	}}
	innerAlias := &yaml.Node{Kind: yaml.AliasNode, Alias: inner}
	hidden := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		innerAlias,
	}}
	hiddenAlias := &yaml.Node{Kind: yaml.AliasNode, Alias: hidden}
	host := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "<<"},
		hiddenAlias,
	}}
	resolveMergeKeys(host)
	for i := 0; i+1 < len(host.Content); i += 2 {
		if host.Content[i].Value == "<<" {
			t.Errorf("hidden source must be flattened in-place; found leftover \"<<\" in host at index %d", i)
		}
	}
	if got := lookupMappingValueNode(host, "x"); got == nil || got.Value != "1" {
		t.Errorf("expected x=1 merged into host via the hidden source; got %v", got)
	}
}
