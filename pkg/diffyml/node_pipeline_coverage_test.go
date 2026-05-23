package diffyml

import (
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// TestParseWithOrder_Error pins the parseNodes-error propagation through the
// public ParseWithOrder wrapper.
func TestParseWithOrder_Error(t *testing.T) {
	if _, err := ParseWithOrder([]byte("a: [1, 2")); err == nil {
		t.Error("expected parse error for malformed YAML")
	}
}

// TestUnwrapDocOrAlias covers the empty-DocumentNode and AliasNode branches
// that real-world YAML rarely produces but the contract requires handling.
func TestUnwrapDocOrAlias(t *testing.T) {
	// Empty DocumentNode → nil (path is treated as the document root with no content).
	emptyDoc := &yaml.Node{Kind: yaml.DocumentNode}
	if got := unwrapDocOrAlias(emptyDoc); got != nil {
		t.Errorf("empty DocumentNode should unwrap to nil, got %v", got)
	}

	// AliasNode → target (chain dereference).
	target := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "hello"}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: target}
	if got := unwrapDocOrAlias(alias); got != target {
		t.Errorf("AliasNode should unwrap to its target, got %v", got)
	}

	// nil input → nil (no panic).
	if got := unwrapDocOrAlias(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}
}

// TestResolveNode_Branches covers DocumentNode unwrap and AliasNode resolution
// for the comparator-side helper.
func TestResolveNode_Branches(t *testing.T) {
	emptyDoc := &yaml.Node{Kind: yaml.DocumentNode}
	if got := resolveNode(emptyDoc); got != nil {
		t.Errorf("empty DocumentNode should resolve to nil, got %v", got)
	}

	target := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "42"}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: target}
	if got := resolveNode(alias); got != target {
		t.Errorf("AliasNode should resolve to its target, got %v", got)
	}

	if got := resolveNode(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}
}

// TestCompareNodes_KindMismatch_IgnoreValueChanges covers the early-return path
// when types differ and the caller has IgnoreValueChanges set.
func TestCompareNodes_KindMismatch_IgnoreValueChanges(t *testing.T) {
	from := nodeFromYAML(t, "key: 1\n")
	to := nodeFromYAML(t, "key: [a]\n") // value is a sequence, not scalar

	opts := &Options{IgnoreValueChanges: true}
	diffs, err := Compare(
		[]byte("key: 1\n"),
		[]byte("key: [a]\n"),
		opts,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 0 {
		t.Errorf("with IgnoreValueChanges the kind-mismatch must be suppressed, got %v", diffs)
	}
	_ = from
	_ = to
}

// TestCompareNodes_ToNil_IgnoreValueChanges covers the to-is-null branch with
// IgnoreValueChanges set (mismatched multi-doc count where the missing slot
// would normally produce a DiffModified).
func TestCompareNodes_ToNil_IgnoreValueChanges(t *testing.T) {
	from := []byte("a: 1\n---\nb: 2\n")
	to := []byte("a: 1\n")
	opts := &Options{IgnoreValueChanges: true}
	diffs, err := Compare(from, to, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 0 {
		t.Errorf("with IgnoreValueChanges the missing second document must be suppressed, got %v", diffs)
	}
}

// TestPathWalker_AliasInSequence exercises the AliasNode branch of pathWalker
// (sequence items that are aliases to anchored mappings).
func TestPathWalker_AliasInSequence(t *testing.T) {
	src := []byte(`
template: &t
  name: x
  v: 1
items:
  - *t
  - name: y
    v: 2
`)
	nodes, err := parseNodes(src)
	if err != nil {
		t.Fatal(err)
	}
	order := extractPathOrder(nodes, nil, &Options{})
	// The aliased item should have been walked through; its keys appear in
	// the path-order map under the appropriate parent.
	if _, ok := order["items.x.v"]; !ok {
		t.Errorf("expected items.x.v to be registered via the alias walk; map=%v", order)
	}
}

// TestMaterializeIdentifierValue_NonScalar covers the fallback path where the
// identifier value is a sub-mapping (rare in practice but supported).
func TestMaterializeIdentifierValue_NonScalar(t *testing.T) {
	node := decodeOne(t, "name:\n  composite: true\n").Content[0]
	got := getIdentifierNode(node, nil)
	om, ok := got.(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap identifier value, got %T", got)
	}
	if v, _ := om.Values["composite"].(bool); !v {
		t.Errorf("expected composite=true, got %v", om.Values["composite"])
	}
}

// TestMaterializeIdentifierValue_NilAlias covers the resolveAlias-returns-nil
// branch when the identifier value is a self-referential AliasNode.
func TestMaterializeIdentifierValue_NilAlias(t *testing.T) {
	a := &yaml.Node{Kind: yaml.AliasNode}
	a.Alias = a
	if got := materializeIdentifierValue(a); got != nil {
		t.Errorf("expected nil for cyclic AliasNode, got %v", got)
	}
}

// TestPathWalker_EmptyDocumentNode covers the empty-DocumentNode short-circuit
// (a parsed empty document or a synthetic node).
func TestPathWalker_EmptyDocumentNode(t *testing.T) {
	w := pathWalker{
		pathOrder: make(map[string]int),
		opts:      &Options{},
		buf:       make([]byte, 0, 32),
	}
	w.walk(&yaml.Node{Kind: yaml.DocumentNode}) // Content empty
	if len(w.pathOrder) != 0 {
		t.Errorf("empty DocumentNode should register nothing, got %v", w.pathOrder)
	}
}

// TestPathWalker_AliasNilTargetTerminates covers the n.Alias == nil guard.
func TestPathWalker_AliasNilTargetTerminates(t *testing.T) {
	w := pathWalker{
		pathOrder: make(map[string]int),
		opts:      &Options{},
		buf:       make([]byte, 0, 32),
	}
	// AliasNode with nil Alias is degenerate but the walker must not crash.
	w.walk(&yaml.Node{Kind: yaml.AliasNode})
	if len(w.pathOrder) != 0 {
		t.Errorf("nil-target alias should register nothing, got %v", w.pathOrder)
	}
}

// TestPathWalker_AliasCycleTerminates covers the aliasSeen cycle break for
// pathological self-referential anchors.
func TestPathWalker_AliasCycleTerminates(t *testing.T) {
	// A self-referential alias chain that points back to a mapping containing
	// itself would otherwise loop forever.
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: mapping}
	mapping.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "self"},
		alias,
	}
	w := pathWalker{
		pathOrder: make(map[string]int),
		opts:      &Options{},
		buf:       make([]byte, 0, 32),
	}
	done := make(chan struct{})
	go func() {
		w.walk(mapping)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pathWalker.walk on cyclic alias hung > 2s")
	}
}

// TestResolveMappingMergeKeys_OuterCycleGuard exercises the cycles[n] check at
// the top of resolveMappingMergeKeys — a belt-and-suspenders for callers that
// might invoke the function with a mapping already mid-resolution.
func TestResolveMappingMergeKeys_OuterCycleGuard(t *testing.T) {
	host := &yaml.Node{
		Kind: yaml.MappingNode, Tag: "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "k"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "v"},
		},
	}
	contentBefore := host.Content
	cycles := map[*yaml.Node]bool{host: true}
	resolveMappingMergeKeys(host, cycles) // must early-return, no-op
	if &host.Content[0] != &contentBefore[0] {
		t.Error("Content should be untouched when the outer cycles guard fires")
	}
}

// TestGetIdentifier_NonMapReturnsNil covers the trailing nil-return when the
// value is not a *OrderedMap or map[string]any.
func TestGetIdentifier_NonMapReturnsNil(t *testing.T) {
	if got := getIdentifier("not-a-map", nil); got != nil {
		t.Errorf("expected nil for non-map input, got %v", got)
	}
	if got := getIdentifier(42, &Options{AdditionalIdentifiers: []string{"x"}}); got != nil {
		t.Errorf("expected nil for int input, got %v", got)
	}
}
