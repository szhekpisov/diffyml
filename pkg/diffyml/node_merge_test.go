package diffyml

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// TestResolveMergeKeys_NodeToInterfaceEquivalence is the contract test for
// Stage 2: for every merge-key shape we care about, running nodeToInterface
// on the resolveMergeKeys-rewritten tree must equal running it on the raw
// tree (which carries the legacy merge handling inside nodeToInterfaceImpl).
// This pins that the parse-time rewrite is observationally indistinguishable
// from the previous on-the-fly merge resolution.
func TestResolveMergeKeys_NodeToInterfaceEquivalence(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "simple_merge",
			yaml: `
base: &b
  a: 1
  b: 2
host:
  <<: *b
  c: 3
`,
		},
		{
			name: "explicit_key_before_merge_wins",
			yaml: `
base: &b
  a: 1
  b: 2
host:
  a: 99
  <<: *b
`,
		},
		{
			name: "explicit_key_after_merge_appears_duplicate",
			yaml: `
base: &b
  a: 1
  b: 2
host:
  <<: *b
  b: 99
`,
		},
		{
			name: "nested_merge_in_source",
			yaml: `
inner: &inner
  x: 1
outer: &outer
  <<: *inner
  y: 2
host:
  <<: *outer
  z: 3
`,
		},
		{
			name: "merge_in_sequence_item",
			yaml: `
defaults: &d
  retries: 3
items:
  - <<: *d
    name: a
  - <<: *d
    name: b
    retries: 9
`,
		},
		{
			name: "missing_anchor_silently_dropped",
			// yaml.v3 errors on unresolved aliases at Decode, so this case
			// instead exercises a non-MappingNode merge source.
			yaml: `
seq: &s
  - 1
  - 2
host:
  <<: *s
  a: 1
`,
		},
		{
			name: "no_merge_keys_at_all",
			yaml: `
plain:
  a: 1
  b: 2
list:
  - x
  - y
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			legacy := decodeOne(t, tc.yaml)
			resolved := decodeOne(t, tc.yaml)
			resolveMergeKeys(resolved)

			gotLegacy := nodeToInterface(legacy)
			gotResolved := nodeToInterface(resolved)
			if !reflect.DeepEqual(gotLegacy, gotResolved) {
				t.Errorf("nodeToInterface mismatch after resolveMergeKeys\n--- legacy ---\n%#v\n--- resolved ---\n%#v", gotLegacy, gotResolved)
			}
		})
	}
}

// TestResolveMergeKeys_Idempotent confirms that resolving twice produces the
// same tree as resolving once: a second pass finds no "<<" keys and is a no-op.
func TestResolveMergeKeys_Idempotent(t *testing.T) {
	src := `
base: &b
  a: 1
host:
  <<: *b
  c: 2
`
	n := decodeOne(t, src)
	resolveMergeKeys(n)
	once := nodeToInterface(n)

	resolveMergeKeys(n)
	twice := nodeToInterface(n)

	if !reflect.DeepEqual(once, twice) {
		t.Errorf("resolveMergeKeys is not idempotent\nonce:  %#v\ntwice: %#v", once, twice)
	}
}

// TestResolveMergeKeys_NilSafe pins the nil-input guard so coverage of the
// dispatch's early return is explicit.
func TestResolveMergeKeys_NilSafe(t *testing.T) {
	resolveMergeKeys(nil) // must not panic
}

// TestResolveAlias_Cycle confirms that an alias whose target is itself
// terminates rather than recursing forever. yaml.v3 doesn't construct such
// trees by default, so this exercise is synthetic.
func TestResolveAlias_Cycle(t *testing.T) {
	a := &yaml.Node{Kind: yaml.AliasNode}
	a.Alias = a
	if got := resolveAlias(a); got != nil {
		t.Errorf("self-aliasing chain must terminate at nil, got %v", got)
	}
}

// TestResolveMergeKeys_CyclicAnchorTerminates pins the cycle break for a
// self-referential merge anchor (regression from a fuzz-discovered hang).
// `&a {<<: *a, k: v}` would otherwise recurse forever into the same mapping.
func TestResolveMergeKeys_CyclicAnchorTerminates(t *testing.T) {
	src := []byte("&self\n<<: *self\nk: v\n")
	done := make(chan struct{})
	go func() {
		_, _ = Compare(src, src, nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Compare on self-referential merge hung > 2s")
	}
}

// decodeOne parses a single-document YAML string into its DocumentNode,
// without running resolveMergeKeys.
func decodeOne(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader([]byte(src))).Decode(&n); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	return &n
}
