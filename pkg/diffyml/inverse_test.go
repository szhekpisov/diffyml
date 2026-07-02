package diffyml_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

// unchangedByPath indexes DiffUnchanged entries by their path string and fails
// the test if any non-unchanged entry is present (inverse mode must only ever
// emit DiffUnchanged).
func unchangedByPath(t *testing.T, diffs []diffyml.Difference) map[string]diffyml.Difference {
	t.Helper()
	out := make(map[string]diffyml.Difference, len(diffs))
	for _, d := range diffs {
		if d.Type != diffyml.DiffUnchanged {
			t.Fatalf("inverse mode emitted non-unchanged type %v at %s", d.Type, d.Path)
		}
		out[d.Path.String()] = d
	}
	return out
}

func mustCompareUnchanged(t *testing.T, from, to string, opts *diffyml.Options) []diffyml.Difference {
	t.Helper()
	if opts == nil {
		opts = &diffyml.Options{}
	}
	opts.Unchanged = true
	diffs, err := diffyml.Compare([]byte(from), []byte(to), opts)
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	return diffs
}

func TestInverse_CollapsesFullyEqualDocument(t *testing.T) {
	doc := "a: 1\nb:\n  c: 2\n"
	diffs := mustCompareUnchanged(t, doc, doc, nil)
	if len(diffs) != 1 {
		t.Fatalf("expected a single collapsed entry, got %d: %+v", len(diffs), diffs)
	}
	got := unchangedByPath(t, diffs)
	if _, ok := got[""]; !ok {
		t.Errorf("expected collapse at root (empty path), got paths %v", keys(got))
	}
}

func TestInverse_PartialMapEmitsOnlyEqualLeaves(t *testing.T) {
	from := "name: app\nimage:\n  repo: nginx\n  tag: \"1.0\"\n"
	to := "name: app\nimage:\n  repo: nginx\n  tag: \"2.0\"\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)

	// name is equal (collapses), image.repo is equal, image.tag differs.
	wantPaths := []string{"name", "image.repo"}
	if len(got) != len(wantPaths) {
		t.Fatalf("expected %d unchanged entries, got %d: %v", len(wantPaths), len(got), keys(got))
	}
	for _, p := range wantPaths {
		d, ok := got[p]
		if !ok {
			t.Errorf("expected unchanged entry at %q; have %v", p, keys(got))
			continue
		}
		if d.From == nil || d.From != d.To {
			t.Errorf("entry %q: expected From==To non-nil, got From=%v To=%v", p, d.From, d.To)
		}
	}
	if _, bad := got["image.tag"]; bad {
		t.Error("image.tag differs and must not appear as unchanged")
	}
}

func TestInverse_SubtreeCollapse(t *testing.T) {
	// image is wholly equal -> one collapsed entry at "image", not per-leaf.
	from := "replicas: 1\nimage:\n  repo: nginx\n  tag: \"1.0\"\n"
	to := "replicas: 2\nimage:\n  repo: nginx\n  tag: \"1.0\"\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)
	if len(got) != 1 {
		t.Fatalf("expected single collapsed entry, got %v", keys(got))
	}
	if _, ok := got["image"]; !ok {
		t.Errorf("expected collapse at 'image', got %v", keys(got))
	}
}

func TestInverse_OnlyOneSideKeyNotEmitted(t *testing.T) {
	from := "shared: yes\nonlyleft: x\n"
	to := "shared: yes\nonlyright: y\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)
	if len(got) != 1 {
		t.Fatalf("expected only 'shared', got %v", keys(got))
	}
	if _, ok := got["shared"]; !ok {
		t.Errorf("expected 'shared' unchanged, got %v", keys(got))
	}
}

func TestInverse_KindMismatchNotEmitted(t *testing.T) {
	from := "k: scalar\n"
	to := "k:\n  nested: 1\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	if len(diffs) != 0 {
		t.Fatalf("kind mismatch must not be unchanged, got %+v", diffs)
	}
}

func TestInverse_BothNullNotEmitted(t *testing.T) {
	// Equal nulls are treated as absent, not "unchanged". A differing sibling
	// ('x') keeps the root from collapsing so we exercise the descent.
	from := "a: null\nb: keep\nx: 1\n"
	to := "a: null\nb: keep\nx: 2\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)
	if _, bad := got["a"]; bad {
		t.Error("equal nulls must not be reported as unchanged")
	}
	if _, ok := got["b"]; !ok {
		t.Errorf("expected 'b' unchanged, got %v", keys(got))
	}
	if _, bad := got["x"]; bad {
		t.Error("x differs and must not be unchanged")
	}
}

func TestInverse_SequencePositional(t *testing.T) {
	from := "ports:\n  - 80\n  - 443\n"
	to := "ports:\n  - 80\n  - 8443\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)
	if _, ok := got["ports.0"]; !ok {
		t.Errorf("expected ports.0 unchanged, got %v", keys(got))
	}
	if _, bad := got["ports.1"]; bad {
		t.Error("ports.1 differs and must not be unchanged")
	}
}

func TestInverse_UnequalScalarNotEmitted(t *testing.T) {
	diffs := mustCompareUnchanged(t, "k: 1\n", "k: 2\n", nil)
	if len(diffs) != 0 {
		t.Fatalf("unequal scalars must emit nothing, got %+v", diffs)
	}
}

func TestInverse_MultiDocSetsDocumentIndexAndPrefix(t *testing.T) {
	from := "a: 1\n---\nb: same\nc: 1\n"
	to := "a: 2\n---\nb: same\nc: 2\n"
	// Disable K8s matching path is irrelevant here (plain docs); positional pairing.
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)
	d, ok := got["[1].b"]
	if !ok {
		t.Fatalf("expected unchanged at [1].b, got %v", keys(got))
	}
	if d.DocumentIndex != 1 {
		t.Errorf("expected DocumentIndex 1, got %d", d.DocumentIndex)
	}
}

func TestInverse_HonorsOptionsEquality(t *testing.T) {
	// IgnoreWhitespaceChanges makes the trailing-space value equal, so it should
	// be reported as unchanged — proving deepEqual/equalValues receives opts. A
	// differing sibling ('other') prevents the single-key doc from collapsing to
	// the root so the entry is reported at "k".
	from := "k: \"value \"\nother: 1\n"
	to := "k: \"value\"\nother: 2\n"
	plain := mustCompareUnchanged(t, from, to, nil)
	if len(plain) != 0 {
		t.Fatalf("without ignore-whitespace the values differ, got %+v", plain)
	}
	ws := mustCompareUnchanged(t, from, to, &diffyml.Options{IgnoreWhitespaceChanges: true})
	got := unchangedByPath(t, ws)
	if _, ok := got["k"]; !ok {
		t.Errorf("expected 'k' unchanged under IgnoreWhitespaceChanges, got %v", keys(got))
	}
}

func TestInverse_DisabledIsNormalDiff(t *testing.T) {
	// Sanity: without Unchanged, no DiffUnchanged ever appears.
	diffs, err := diffyml.Compare([]byte("a: 1\n"), []byte("a: 2\n"), &diffyml.Options{})
	if err != nil {
		t.Fatalf("Compare error: %v", err)
	}
	for _, d := range diffs {
		if d.Type == diffyml.DiffUnchanged {
			t.Error("normal mode must never emit DiffUnchanged")
		}
	}
}

func TestInverse_K8sReorderedMatchesByIdentifier(t *testing.T) {
	// Service is identical but in different document order; Deployment is matched
	// across reordering and partially equal; the ConfigMap exists only in 'from'.
	from := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  strategy: RollingUpdate
---
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  type: ClusterIP
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: only-here
data:
  k: v
`
	to := `apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
  strategy: RollingUpdate
`
	diffs := mustCompareUnchanged(t, from, to, &diffyml.Options{DetectKubernetes: true})

	var sawServiceCollapse, sawDeploymentLeaf, sawConfigMap bool
	for _, d := range diffs {
		if d.Type != diffyml.DiffUnchanged {
			t.Fatalf("non-unchanged entry: %v at %s", d.Type, d.Path)
		}
		if d.DocumentKind == "ConfigMap" {
			sawConfigMap = true
		}
		if d.DocumentKind == "Service" {
			sawServiceCollapse = true
		}
		if d.DocumentKind == "Deployment" && strings.Contains(d.Path.String(), "strategy") {
			sawDeploymentLeaf = true
		}
	}
	if !sawServiceCollapse {
		t.Error("expected the reordered Service to be matched and reported")
	}
	if !sawDeploymentLeaf {
		t.Error("expected the reordered Deployment's equal spec.strategy to be reported")
	}
	if sawConfigMap {
		t.Error("a resource present on only one side must not be reported")
	}
}

func TestInverse_K8sSingleDocNoPrefix(t *testing.T) {
	// Single resource each side: matched, equal leaf reported at a bare path
	// (no [idx] prefix) — exercises the single-document branch.
	doc := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
data:
  same: yes
  diff: %s
`
	from := fmt.Sprintf(doc, "1")
	to := fmt.Sprintf(doc, "2")
	diffs := mustCompareUnchanged(t, from, to, &diffyml.Options{DetectKubernetes: true})
	got := unchangedByPath(t, diffs)
	if _, ok := got["data.same"]; !ok {
		t.Errorf("expected data.same unchanged at a bare path, got %v", keys(got))
	}
}

func TestInverse_K8sRenameMatched(t *testing.T) {
	// Same Deployment renamed (web -> webby); identifier differs so it is matched
	// only via rename detection. Equal fields must still be reported.
	from := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.0
`
	to := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: webby
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.0
`
	diffs := mustCompareUnchanged(t, from, to, &diffyml.Options{DetectKubernetes: true, DetectRenames: true})
	if len(diffs) == 0 {
		t.Fatal("expected rename-matched resource to report equal fields")
	}
	var sawEqualField bool
	for _, d := range diffs {
		if d.Type == diffyml.DiffUnchanged && strings.Contains(d.Path.String(), "selector") {
			sawEqualField = true
		}
	}
	if !sawEqualField {
		t.Errorf("expected an equal field (selector) from the rename-matched pair, got %+v", diffs)
	}
}

func TestInverse_SequenceByIdentifierReordered(t *testing.T) {
	from := `containers:
  - name: app
    image: nginx:1.0
  - name: sidecar
    image: proxy:1.0
    port: 8080
  - name: old
    image: legacy:1.0
`
	to := `containers:
  - name: sidecar
    image: proxy:2.0
    port: 8080
  - name: app
    image: nginx:1.0
  - name: new
    image: fresh:1.0
`
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)

	if _, ok := got["containers.app"]; !ok {
		t.Errorf("expected containers.app collapsed (equal across reorder), got %v", keys(got))
	}
	if _, ok := got["containers.sidecar.port"]; !ok {
		t.Errorf("expected containers.sidecar.port unchanged, got %v", keys(got))
	}
	for p := range got {
		if strings.Contains(p, "old") || strings.Contains(p, "new") {
			t.Errorf("one-side-only list item %q must not be reported", p)
		}
	}
}

func TestInverse_SequenceUnidentifiedPositionalFallback(t *testing.T) {
	// Every item is a map (so identifier matching qualifies), but the keyless map
	// at index 1 has no name/id and falls back to positional pairing.
	from := `items:
  - name: a
    v: 1
  - other: keyless-equal
  - name: gone
`
	to := `items:
  - name: a
    v: 1
  - other: keyless-equal
  - name: new
`
	diffs := mustCompareUnchanged(t, from, to, nil)
	got := unchangedByPath(t, diffs)
	if _, ok := got["items.a"]; !ok {
		t.Errorf("expected identified item items.a, got %v", keys(got))
	}
	if _, ok := got["items.1"]; !ok {
		t.Errorf("expected positionally-paired keyless equal item items.1, got %v", keys(got))
	}
	for p := range got {
		if strings.Contains(p, "gone") || strings.Contains(p, "new") {
			t.Errorf("one-side-only identified item %q must not be reported", p)
		}
	}
}

// TestInverse_DetailedMapSubtreeWithNameNotList verifies the full inverse walk
// tags a collapsed map subtree as a map entry even when its value carries a
// top-level `name` key. Regression: the detailed formatter's hasIdentifierField
// heuristic misfired on the raw collapsed value and rendered it as a list item.
func TestInverse_DetailedMapSubtreeWithNameNotList(t *testing.T) {
	from := "config:\n  name: foo\n  port: 8080\nother: 1\n"
	to := "config:\n  name: foo\n  port: 8080\nother: 2\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	out := (&diffyml.DetailedFormatter{}).Format(diffs, diffyml.DefaultFormatOptions())
	if strings.Contains(out, "list entry") {
		t.Errorf("collapsed map subtree must not render as a list entry, got:\n%s", out)
	}
	if !strings.Contains(out, "one map entry unchanged") {
		t.Errorf("expected a map entry batch, got:\n%s", out)
	}
}

// TestInverse_DetailedIdentifierListItemIsList verifies a collapsed
// identifier-matched list item keeps its "- " list rendering, so the
// container-kind tracking does not regress genuine list items to map style.
func TestInverse_DetailedIdentifierListItemIsList(t *testing.T) {
	from := "containers:\n  - name: app\n    image: nginx:1.0\n  - name: side\n    image: a:1\n"
	to := "containers:\n  - name: app\n    image: nginx:1.0\n  - name: side\n    image: b:2\n"
	diffs := mustCompareUnchanged(t, from, to, nil)
	out := (&diffyml.DetailedFormatter{}).Format(diffs, diffyml.DefaultFormatOptions())
	if !strings.Contains(out, "one list entry unchanged") {
		t.Errorf("collapsed identifier-matched list item must render as a list entry, got:\n%s", out)
	}
	if !strings.Contains(out, "- name: app") {
		t.Errorf("expected '- name: app' list rendering, got:\n%s", out)
	}
}

func TestInverse_IgnoreOrderChangesPlainList(t *testing.T) {
	// A reordered scalar list: without --ignore-order-changes only the
	// position-aligned value (2 at index 1) is unchanged; with the flag every
	// common value is matched order-independently. The differing 'other' sibling
	// keeps the root from collapsing so the list is descended into.
	from := "nums:\n  - 1\n  - 2\n  - 3\nother: 1\n"
	to := "nums:\n  - 3\n  - 2\n  - 1\nother: 2\n"

	plain := unchangedByPath(t, mustCompareUnchanged(t, from, to, nil))
	if _, ok := plain["nums.1"]; !ok {
		t.Errorf("positional: expected nums.1 unchanged, got %v", keys(plain))
	}
	if len(plain) != 1 {
		t.Errorf("positional: expected only nums.1, got %v", keys(plain))
	}

	ordered := unchangedByPath(t, mustCompareUnchanged(t, from, to, &diffyml.Options{IgnoreOrderChanges: true}))
	for _, p := range []string{"nums.0", "nums.1", "nums.2"} {
		if _, ok := ordered[p]; !ok {
			t.Errorf("ignore-order: expected %q unchanged, got %v", p, keys(ordered))
		}
	}
}

func TestInverse_HeterogeneousListReordered(t *testing.T) {
	// Single-key maps with distinct keys are heterogeneous, so the normal
	// comparator matches them unordered even without the flag — inverse mode
	// must do the same instead of mis-pairing positionally.
	from := "rules:\n  - a: 1\n  - b: 2\ndiffer: 1\n"
	to := "rules:\n  - b: 2\n  - a: 1\ndiffer: 2\n"

	got := unchangedByPath(t, mustCompareUnchanged(t, from, to, nil))
	for _, p := range []string{"rules.0", "rules.1"} {
		if _, ok := got[p]; !ok {
			t.Errorf("expected heterogeneous item %q matched unordered, got %v", p, keys(got))
		}
	}
}

func TestInverse_KeylessItemsReorderedWithinIdentifiedList(t *testing.T) {
	// An identifier-matched list whose keyless items are reordered: the keyless
	// fallback must match them order-independently (parity with
	// compareUnidentifiedItems), not pair them positionally.
	from := "items:\n  - name: a\n    v: 1\n  - foo: x\n  - bar: y\n"
	to := "items:\n  - name: a\n    v: 1\n  - bar: y\n  - foo: x\n"

	got := unchangedByPath(t, mustCompareUnchanged(t, from, to, nil))
	for _, p := range []string{"items.a", "items.1", "items.2"} {
		if _, ok := got[p]; !ok {
			t.Errorf("expected %q unchanged, got %v", p, keys(got))
		}
	}
}

func keys(m map[string]diffyml.Difference) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
