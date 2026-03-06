package compare

import (
	"fmt"
	"strings"
	"testing"

	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/parse"
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/types"
)

// ---------------------------------------------------------------------------
// DeepEqual tests
// ---------------------------------------------------------------------------

func TestDeepEqual_OrderedMaps_DifferentLengths(t *testing.T) {
	a := &types.OrderedMap{Values: map[string]interface{}{"x": 1, "y": 2}}
	b := &types.OrderedMap{Values: map[string]interface{}{"x": 1}}
	if DeepEqual(a, b, nil) {
		t.Error("expected OrderedMaps with different lengths to not be DeepEqual")
	}
}

func TestDeepEqual_Slices_Equal(t *testing.T) {
	a := []interface{}{"x", "y", "z"}
	b := []interface{}{"x", "y", "z"}
	if !DeepEqual(a, b, nil) {
		t.Error("expected equal slices to be DeepEqual")
	}
}

func TestDeepEqual_Slices_DifferentValues(t *testing.T) {
	a := []interface{}{"x", "y"}
	b := []interface{}{"x", "z"}
	if DeepEqual(a, b, nil) {
		t.Error("expected slices with different values to not be DeepEqual")
	}
}

func TestDeepEqual_Slices_DifferentLengths(t *testing.T) {
	a := []interface{}{"x"}
	b := []interface{}{"x", "y"}
	if DeepEqual(a, b, nil) {
		t.Error("expected slices with different lengths to not be DeepEqual")
	}
}

func TestDeepEqual_Slices_Nested(t *testing.T) {
	a := []interface{}{[]interface{}{"a", "b"}}
	b := []interface{}{[]interface{}{"a", "b"}}
	if !DeepEqual(a, b, nil) {
		t.Error("expected nested equal slices to be DeepEqual")
	}
}

// ---------------------------------------------------------------------------
// AreListItemsHeterogeneous tests
// ---------------------------------------------------------------------------

func TestAreListItemsHeterogeneous_OrderedMaps(t *testing.T) {
	from := []interface{}{
		&types.OrderedMap{Keys: []string{"namespaceSelector"}, Values: map[string]interface{}{"namespaceSelector": "ns1"}},
	}
	to := []interface{}{
		&types.OrderedMap{Keys: []string{"ipBlock"}, Values: map[string]interface{}{"ipBlock": "10.0.0.0/8"}},
	}
	if !AreListItemsHeterogeneous(from, to) {
		t.Error("expected heterogeneous for maps with different single keys")
	}
}

func TestAreListItemsHeterogeneous_OrderedMapsMultipleKeys(t *testing.T) {
	from := []interface{}{
		&types.OrderedMap{Keys: []string{"a", "b"}, Values: map[string]interface{}{"a": "1", "b": "2"}},
	}
	to := []interface{}{
		&types.OrderedMap{Keys: []string{"c"}, Values: map[string]interface{}{"c": "3"}},
	}
	if AreListItemsHeterogeneous(from, to) {
		t.Error("expected not heterogeneous when an item has multiple keys")
	}
}

// ---------------------------------------------------------------------------
// CompareListsByIdentifier tests
// ---------------------------------------------------------------------------

func TestCompareListsByIdentifier_NoIDFallback(t *testing.T) {
	from := []interface{}{
		&types.OrderedMap{
			Keys:   []string{"name", "value"},
			Values: map[string]interface{}{"name": "a", "value": "1"},
		},
		"scalar-from-only",
		"shared-scalar",
	}
	to := []interface{}{
		&types.OrderedMap{
			Keys:   []string{"name", "value"},
			Values: map[string]interface{}{"name": "a", "value": "2"},
		},
		"new-scalar",
		"shared-scalar",
	}
	diffs := CompareListsByIdentifier("items", from, to, nil)
	var removed, added int
	for _, d := range diffs {
		switch d.Type {
		case types.DiffRemoved:
			removed++
		case types.DiffAdded:
			added++
		}
	}
	if removed < 1 {
		t.Errorf("expected at least 1 removed diff (scalar-from-only), got %d removed", removed)
	}
	if added < 1 {
		t.Errorf("expected at least 1 added diff (new-scalar), got %d added", added)
	}
}

// ---------------------------------------------------------------------------
// CompareListsPositional tests
// ---------------------------------------------------------------------------

func TestCompareListsPositional_ToLonger(t *testing.T) {
	from := []interface{}{"a", "b"}
	to := []interface{}{"a", "b", "c", "d"}
	diffs := CompareListsPositional("list", from, to, nil)
	added := 0
	for _, d := range diffs {
		if d.Type == types.DiffAdded {
			added++
		}
	}
	if added != 2 {
		t.Errorf("expected 2 added items, got %d", added)
	}
}

func TestCompareListsPositional_FromLonger(t *testing.T) {
	from := []interface{}{"a", "b", "c"}
	to := []interface{}{"a"}
	diffs := CompareListsPositional("list", from, to, nil)
	removed := 0
	for _, d := range diffs {
		if d.Type == types.DiffRemoved {
			removed++
		}
	}
	if removed != 2 {
		t.Errorf("expected 2 removed items, got %d", removed)
	}
}

// ---------------------------------------------------------------------------
// ExtractPathOrder tests
// ---------------------------------------------------------------------------

func TestExtractPathOrder_OrderedMap(t *testing.T) {
	om := &types.OrderedMap{
		Keys:   []string{"beta", "alpha"},
		Values: map[string]interface{}{"beta": "2", "alpha": "1"},
	}
	docs := []interface{}{om}
	order := ExtractPathOrder(docs, nil, nil)
	if len(order) == 0 {
		t.Fatal("expected non-empty path order")
	}
	if _, ok := order["alpha"]; !ok {
		t.Error("expected 'alpha' in path order")
	}
	if _, ok := order["beta"]; !ok {
		t.Error("expected 'beta' in path order")
	}
}

func TestExtractPathOrder_OrderedMapNested(t *testing.T) {
	child := &types.OrderedMap{
		Keys:   []string{"child"},
		Values: map[string]interface{}{"child": "val"},
	}
	om := &types.OrderedMap{
		Keys:   []string{"parent"},
		Values: map[string]interface{}{"parent": child},
	}
	docs := []interface{}{om}
	order := ExtractPathOrder(docs, nil, nil)
	if _, ok := order["parent"]; !ok {
		t.Error("expected 'parent' in path order")
	}
	if _, ok := order["parent.child"]; !ok {
		t.Error("expected 'parent.child' in path order")
	}
}

func TestExtractPathOrder_OrderedMapIndexIncrement(t *testing.T) {
	child1 := &types.OrderedMap{Keys: []string{"child1"}, Values: map[string]interface{}{"child1": "v1"}}
	child2 := &types.OrderedMap{Keys: []string{"child2"}, Values: map[string]interface{}{"child2": "v2"}}
	child3 := &types.OrderedMap{Keys: []string{"child3"}, Values: map[string]interface{}{"child3": "v3"}}
	om := &types.OrderedMap{
		Keys:   []string{"alpha", "beta", "gamma"},
		Values: map[string]interface{}{"alpha": child1, "beta": child2, "gamma": child3},
	}
	docs := []interface{}{om}
	order := ExtractPathOrder(docs, nil, nil)
	if order["alpha"] >= order["beta"] {
		t.Errorf("expected alpha (%d) < beta (%d)", order["alpha"], order["beta"])
	}
	if order["beta"] >= order["gamma"] {
		t.Errorf("expected beta (%d) < gamma (%d)", order["beta"], order["gamma"])
	}
}

func TestExtractPathOrder_EmptyPrefix(t *testing.T) {
	om := types.NewOrderedMap()
	om.Keys = []string{"beta", "alpha"}
	om.Values["beta"] = "val1"
	om.Values["alpha"] = "val2"
	docs := []interface{}{om}
	opts := &types.Options{}
	pathOrder := ExtractPathOrder(docs, nil, opts)
	betaIdx, hasBeta := pathOrder["beta"]
	alphaIdx, hasAlpha := pathOrder["alpha"]
	if !hasBeta || !hasAlpha {
		t.Fatalf("pathOrder missing keys: beta=%v alpha=%v", hasBeta, hasAlpha)
	}
	if betaIdx >= alphaIdx {
		t.Errorf("beta (idx=%d) should have lower index than alpha (idx=%d)", betaIdx, alphaIdx)
	}
	if _, hasEmpty := pathOrder[""]; hasEmpty {
		t.Error("empty prefix should not be registered in pathOrder")
	}
}

func TestExtractPathOrder_IndexIncrement(t *testing.T) {
	om := types.NewOrderedMap()
	om.Keys = []string{"first", "second", "third"}
	om.Values["first"] = "a"
	om.Values["second"] = "b"
	om.Values["third"] = "c"
	docs := []interface{}{om}
	opts := &types.Options{}
	pathOrder := ExtractPathOrder(docs, nil, opts)
	firstIdx := pathOrder["first"]
	secondIdx := pathOrder["second"]
	thirdIdx := pathOrder["third"]
	if firstIdx >= secondIdx {
		t.Errorf("first (idx=%d) should be less than second (idx=%d)", firstIdx, secondIdx)
	}
	if secondIdx >= thirdIdx {
		t.Errorf("second (idx=%d) should be less than third (idx=%d)", secondIdx, thirdIdx)
	}
}

func TestExtractPathOrder_ListIndexIncrement(t *testing.T) {
	list := []interface{}{"item0", "item1"}
	om := types.NewOrderedMap()
	om.Keys = []string{"items"}
	om.Values["items"] = list
	docs := []interface{}{om}
	opts := &types.Options{}
	pathOrder := ExtractPathOrder(docs, nil, opts)
	itemsIdx, hasItems := pathOrder["items"]
	idx0, has0 := pathOrder["items.0"]
	if !hasItems || !has0 {
		t.Fatalf("missing path entries: items=%v items.0=%v", hasItems, has0)
	}
	if itemsIdx >= idx0 {
		t.Errorf("items prefix (idx=%d) should have lower index than items.0 (idx=%d)", itemsIdx, idx0)
	}
}

// ---------------------------------------------------------------------------
// CompareK8sDocs tests
// ---------------------------------------------------------------------------

func TestCompareK8sDocs_IgnoreApiVersion_Differences(t *testing.T) {
	fromDoc := map[string]interface{}{
		"apiVersion": "apps/v1beta1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
		"spec":       map[string]interface{}{"replicas": 3},
	}
	toDoc := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app", "namespace": "default"},
		"spec":       map[string]interface{}{"replicas": 5},
	}
	opts := &types.Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs := CompareK8sDocs([]interface{}{fromDoc}, []interface{}{toDoc}, opts, CompareNodes)
	hasApiVersionDiff := false
	hasReplicasDiff := false
	for _, d := range diffs {
		if d.Type == types.DiffModified && d.From == "apps/v1beta1" && d.To == "apps/v1" {
			hasApiVersionDiff = true
		}
		if d.Type == types.DiffModified && d.From == 3 && d.To == 5 {
			hasReplicasDiff = true
		}
	}
	if !hasApiVersionDiff {
		t.Error("expected apiVersion to be reported as a modified field (apps/v1beta1 -> apps/v1)")
	}
	if !hasReplicasDiff {
		t.Error("expected spec.replicas to be reported as a modified field (3 -> 5)")
	}
}

func TestCompareK8sDocs_AgnosticDuplicates_ReportedAsAddedRemoved(t *testing.T) {
	mkDoc := func(apiVer, name string, replicas int) map[string]interface{} {
		return map[string]interface{}{
			"apiVersion": apiVer,
			"kind":       "Deployment",
			"metadata":   map[string]interface{}{"name": name, "namespace": "default"},
			"spec":       map[string]interface{}{"replicas": replicas},
		}
	}
	fromDocs := []interface{}{mkDoc("apps/v1", "my-app", 3)}
	toDocs := []interface{}{mkDoc("apps/v1", "my-app", 3), mkDoc("apps/v1beta1", "my-app", 1)}
	opts := &types.Options{
		DetectKubernetes: true,
		IgnoreApiVersion: true,
	}
	diffs := CompareK8sDocs(fromDocs, toDocs, opts, CompareNodes)
	hasAdded := false
	for _, d := range diffs {
		if d.Type == types.DiffAdded {
			hasAdded = true
		}
	}
	if !hasAdded {
		t.Error("expected unmatched duplicate to be reported as DiffAdded")
	}
}

func TestCompareK8sDocs_RenameDetection_SingleDoc(t *testing.T) {
	fromDoc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "app-config-abc123"},
		"data":       map[string]interface{}{"key": "value"},
	}
	toDoc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "app-config-def456"},
		"data":       map[string]interface{}{"key": "value"},
	}
	opts := &types.Options{
		DetectKubernetes: true,
		DetectRenames:    true,
	}
	diffs := CompareK8sDocs([]interface{}{fromDoc}, []interface{}{toDoc}, opts, CompareNodes)
	hasNameChange := false
	hasBulkAdd := false
	hasBulkRemove := false
	for _, d := range diffs {
		if d.Type == types.DiffModified && d.From == "app-config-abc123" && d.To == "app-config-def456" {
			hasNameChange = true
			if strings.HasPrefix(d.Path, "[") {
				t.Errorf("single-doc rename should not have document index prefix, got path %q", d.Path)
			}
		}
		if d.Type == types.DiffAdded && d.Path == "[0]" {
			hasBulkAdd = true
		}
		if d.Type == types.DiffRemoved && d.Path == "[0]" {
			hasBulkRemove = true
		}
	}
	if !hasNameChange {
		t.Error("expected metadata.name field-level diff (rename detection)")
	}
	if hasBulkAdd || hasBulkRemove {
		t.Error("expected no bulk add/remove for rename-matched single-doc pair")
	}
}

func TestCompareK8sDocs_RenameDetection_MultiDoc(t *testing.T) {
	sharedDoc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata":   map[string]interface{}{"name": "my-service"},
		"spec":       map[string]interface{}{"port": 80},
	}
	fromDoc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "app-config-abc123"},
		"data":       map[string]interface{}{"key": "value"},
	}
	toDoc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "app-config-def456"},
		"data":       map[string]interface{}{"key": "value"},
	}
	opts := &types.Options{
		DetectKubernetes: true,
		DetectRenames:    true,
	}
	diffs := CompareK8sDocs([]interface{}{sharedDoc, fromDoc}, []interface{}{sharedDoc, toDoc}, opts, CompareNodes)
	hasNameChange := false
	for _, d := range diffs {
		if d.Type == types.DiffModified && d.From == "app-config-abc123" && d.To == "app-config-def456" {
			hasNameChange = true
			if !strings.HasPrefix(d.Path, "[1]") {
				t.Errorf("multi-doc rename should have [1] prefix, got path %q", d.Path)
			}
		}
	}
	if !hasNameChange {
		t.Error("expected metadata.name field-level diff (rename detection)")
	}
}

// ---------------------------------------------------------------------------
// Benchmark helpers
// ---------------------------------------------------------------------------

func buildOrderedMap(n int) *types.OrderedMap {
	om := types.NewOrderedMap()
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%03d", i)
		om.Keys = append(om.Keys, key)
		om.Values[key] = fmt.Sprintf("value-%d", i)
	}
	return om
}

func buildOrderedMapModified(n int) *types.OrderedMap {
	om := types.NewOrderedMap()
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%03d", i)
		om.Keys = append(om.Keys, key)
		if i%5 == 0 {
			om.Values[key] = fmt.Sprintf("modified-value-%d", i)
		} else {
			om.Values[key] = fmt.Sprintf("value-%d", i)
		}
	}
	return om
}

func buildNestedMap(depth int) *types.OrderedMap {
	if depth == 0 {
		om := types.NewOrderedMap()
		om.Keys = append(om.Keys, "leaf")
		om.Values["leaf"] = "value"
		return om
	}
	om := types.NewOrderedMap()
	key := fmt.Sprintf("level-%d", depth)
	om.Keys = append(om.Keys, key)
	om.Values[key] = buildNestedMap(depth - 1)
	return om
}

func buildServiceList(n int) []interface{} {
	list := make([]interface{}, n)
	for i := 0; i < n; i++ {
		om := types.NewOrderedMap()
		om.Keys = []string{"name", "port"}
		om.Values["name"] = fmt.Sprintf("svc-%03d", i)
		om.Values["port"] = 8080 + i
		list[i] = om
	}
	return list
}

func buildServiceListModified(n int) []interface{} {
	list := make([]interface{}, n)
	for i := 0; i < n; i++ {
		om := types.NewOrderedMap()
		om.Keys = []string{"name", "port"}
		om.Values["name"] = fmt.Sprintf("svc-%03d", i)
		if i%3 == 0 {
			om.Values["port"] = 9090 + i
		} else {
			om.Values["port"] = 8080 + i
		}
		list[i] = om
	}
	return list
}

func buildScalarList(n int) []interface{} {
	list := make([]interface{}, n)
	for i := 0; i < n; i++ {
		list[i] = fmt.Sprintf("item-%d", i)
	}
	return list
}

func buildScalarListModified(n int) []interface{} {
	list := make([]interface{}, n)
	for i := 0; i < n; i++ {
		if i%4 == 0 {
			list[i] = fmt.Sprintf("modified-%d", i)
		} else {
			list[i] = fmt.Sprintf("item-%d", i)
		}
	}
	return list
}

func generateServiceList(n int) []byte {
	var sb strings.Builder
	sb.WriteString("services:\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "  - name: svc-%03d\n    port: %d\n", i, 8080+i)
	}
	return []byte(sb.String())
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkCompareOrderedMaps(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, n := range sizes {
		from := buildOrderedMap(n)
		to := buildOrderedMapModified(n)
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				CompareOrderedMaps("root", from, to, nil)
			}
		})
	}
}

func BenchmarkCompareLists_ByIdentifier(b *testing.B) {
	sizes := []int{10, 50, 500}
	for _, n := range sizes {
		from := buildServiceList(n)
		to := buildServiceListModified(n)
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				CompareListsByIdentifier("services", from, to, nil)
			}
		})
	}
}

func BenchmarkCompareLists_Unordered(b *testing.B) {
	sizes := []int{10, 50, 200}
	for _, n := range sizes {
		from := buildScalarList(n)
		to := buildScalarListModified(n)
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				CompareListsUnordered("items", from, to, nil)
			}
		})
	}
}

func BenchmarkDeepEqual(b *testing.B) {
	b.Run("Identical", func(b *testing.B) {
		om := buildOrderedMap(50)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			DeepEqual(om, om, nil)
		}
	})
	b.Run("Different", func(b *testing.B) {
		om1 := buildOrderedMap(50)
		om2 := buildOrderedMapModified(50)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			DeepEqual(om1, om2, nil)
		}
	})
	b.Run("Nested", func(b *testing.B) {
		nested1 := buildNestedMap(20)
		nested2 := buildNestedMap(20)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			DeepEqual(nested1, nested2, nil)
		}
	})
}

func BenchmarkExtractPathOrder(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"Medium", 50},
		{"Large", 500},
	}
	for _, sz := range sizes {
		data := generateServiceList(sz.n)
		docs, err := parse.ParseWithOrder(data)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ExtractPathOrder(docs, docs, nil)
			}
		})
	}
}
