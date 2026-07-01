package diffyml

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

// scalarNode / nullNode build minimal scalar nodes for the white-box guard
// tests below, where a nil *yaml.Node (an absent value) cannot be expressed in
// YAML source.
func scalarNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: val}
}

func nullNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
}

// TestDeepEqualNodes_MatchesDeepEqual pins the contract that the node-level
// deepEqualNodes (used by the inverse walk to avoid materializing subtrees on
// the partial-match path) agrees with deepEqual(nodeToInterface(a),
// nodeToInterface(b), opts) for every input. If the two ever diverge, inverse
// mode would either drop genuinely-equal values or report unequal ones.
func TestDeepEqualNodes_MatchesDeepEqual(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
	}{
		{"equal scalars", "x: 1\n", "x: 1\n"},
		{"unequal scalars", "x: 1\n", "x: 2\n"},
		{"scalar type mismatch", "x: 1\n", "x: \"1\"\n"},
		{"equal nested maps", "a:\n  b: 1\n  c: 2\n", "a:\n  c: 2\n  b: 1\n"},
		{"map differing value", "a:\n  b: 1\n", "a:\n  b: 9\n"},
		{"map differing key set", "a:\n  b: 1\n", "a:\n  b: 1\n  d: 2\n"},
		{"map extra key on from", "a:\n  b: 1\n  d: 2\n", "a:\n  b: 1\n"},
		// Same key count but a different key — pins the `if !ok` guard in
		// deepEqualMappingNodes (the len check passes, so a missing key must be
		// caught by the membership test, not by indexing position 0).
		{"same length different key", "m:\n  a: x\n", "m:\n  b: x\n"},
		{"equal sequences", "s:\n  - 1\n  - 2\n", "s:\n  - 1\n  - 2\n"},
		{"reordered sequence (positional unequal)", "s:\n  - 1\n  - 2\n", "s:\n  - 2\n  - 1\n"},
		{"sequence length mismatch", "s:\n  - 1\n", "s:\n  - 1\n  - 2\n"},
		{"kind mismatch map vs seq", "a:\n  b: 1\n", "a:\n  - 1\n"},
		// A mapping {a: x} vs a sequence [a, x] would mis-compare as equal if the
		// Kind guard were dropped (the seq's content reads as one k/v pair) —
		// pins the `fromN.Kind != toN.Kind` guard.
		{"map vs seq same pairs", "m:\n  a: x\n", "m:\n  - a\n  - x\n"},
		{"kind mismatch scalar vs map", "a: x\n", "a:\n  b: 1\n"},
		{"both null", "a: null\n", "a: null\n"},
		{"null vs scalar", "a: null\n", "a: x\n"},
		{"null vs map", "a: null\n", "a:\n  b: 1\n"},
		{"null-valued key equal", "a:\n  b: null\n  c: 1\n", "a:\n  c: 1\n  b: null\n"},
		{"duplicate keys last-write-wins equal", "a:\n  b: 1\n  b: 2\n", "a:\n  b: 9\n  b: 2\n"},
		{"whitespace differing strings", "a: \"value \"\n", "a: \"value\"\n"},
		{"json-equivalent strings", "a: '{\"x\":1,\"y\":2}'\n", "a: '{\"y\":2,\"x\":1}'\n"},
		{"deep nested partial", "a:\n  b:\n    c: 1\n    d: 2\n", "a:\n  b:\n    c: 1\n    d: 3\n"},
		// Aliases on one side only must resolve before the Kind dispatch — pins
		// the internal resolveNode calls (an unresolved AliasNode would mis-match
		// kinds against the plain value on the other side).
		{"alias on from side equal", "v: &a 1\nw: *a\n", "v: 1\nw: 1\n"},
		{"alias on to side equal", "v: 1\nw: 1\n", "v: &a 1\nw: *a\n"},
	}

	optsVariants := []struct {
		name string
		opts *Options
	}{
		{"default", &Options{}},
		{"ignore-whitespace", &Options{IgnoreWhitespaceChanges: true}},
		{"format-strings", &Options{FormatStrings: true}},
	}

	for _, tc := range cases {
		for _, ov := range optsVariants {
			t.Run(tc.name+"/"+ov.name, func(t *testing.T) {
				fromN := resolveNode(nodeFromYAML(t, tc.from))
				toN := resolveNode(nodeFromYAML(t, tc.to))

				want := deepEqual(nodeToInterface(fromN), nodeToInterface(toN), ov.opts)
				got := deepEqualNodes(fromN, toN, ov.opts)
				if got != want {
					t.Errorf("deepEqualNodes=%v, deepEqual(nodeToInterface)=%v\nfrom:\n%sto:\n%s",
						got, want, tc.from, tc.to)
				}
			})
		}
	}
}

// TestDeepEqualNodes_NullAndNilGuards pins the null/nil short-circuit at the top
// of deepEqualNodes. A nil *yaml.Node is an absent value the live callers never
// pass at the top level but the recursion can encounter (a cycle-collapsed
// alias resolves to nil); the guard must answer before the Kind dispatch, which
// would otherwise dereference the nil node and panic. The `fromNull && toNull`
// return is pinned by the scalar-vs-null asymmetry.
func TestDeepEqualNodes_NullAndNilGuards(t *testing.T) {
	opts := &Options{}
	s := scalarNode("x")
	null := nullNode()

	if deepEqualNodes(nil, s, opts) {
		t.Error("nil vs scalar must not be equal")
	}
	if deepEqualNodes(s, nil, opts) {
		t.Error("scalar vs nil must not be equal")
	}
	if !deepEqualNodes(nil, nil, opts) {
		t.Error("nil vs nil must be equal")
	}
	if deepEqualNodes(s, null, opts) {
		t.Error("scalar vs !!null must not be equal")
	}
	if deepEqualNodes(null, s, opts) {
		t.Error("!!null vs scalar must not be equal")
	}
	if !deepEqualNodes(null, null, opts) {
		t.Error("!!null vs !!null must be equal")
	}
	if !deepEqualNodes(nil, null, opts) {
		t.Error("nil and !!null are both absent -> equal")
	}
}
