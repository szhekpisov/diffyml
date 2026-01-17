package diffyml

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Data generators
// ---------------------------------------------------------------------------

// generateServiceList generates YAML with n named services (~12 lines each).
func generateServiceList(n int) []byte {
	var b strings.Builder
	b.WriteString("services:\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "- name: service-%03d\n", i)
		fmt.Fprintf(&b, "  version: 1.0.%d\n", i%10)
		fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%5))
		fmt.Fprintf(&b, "  memory: %dMi\n", 256+(i%4)*128)
		fmt.Fprintf(&b, "  cpu: %dm\n", 100+(i%4)*50)
		b.WriteString("  enabled: true\n")
		fmt.Fprintf(&b, "  port: %d\n", 8000+i)
		b.WriteString("  protocol: http\n")
		fmt.Fprintf(&b, "  timeout: %d\n", 30+(i%3)*10)
		b.WriteString("  labels:\n")
		b.WriteString("    tier: backend\n")
		fmt.Fprintf(&b, "    team: team-%d\n", i%5)
	}
	return []byte(b.String())
}

// generateServiceListModified generates a modified variant of generateServiceList
// (~20% of values changed, some added/removed).
func generateServiceListModified(n int) []byte {
	var b strings.Builder
	b.WriteString("services:\n")
	// Skip first 2 services (removed), add the rest with modifications
	removed := 2
	added := n / 10
	if added < 1 {
		added = 1
	}
	for i := removed; i < n; i++ {
		fmt.Fprintf(&b, "- name: service-%03d\n", i)
		if i%5 == 0 {
			// ~20% have version changes
			fmt.Fprintf(&b, "  version: 2.0.%d\n", i%10)
		} else {
			fmt.Fprintf(&b, "  version: 1.0.%d\n", i%10)
		}
		if i%5 == 1 {
			fmt.Fprintf(&b, "  replicas: %d\n", 3+(i%3))
		} else {
			fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%5))
		}
		fmt.Fprintf(&b, "  memory: %dMi\n", 256+(i%4)*128)
		fmt.Fprintf(&b, "  cpu: %dm\n", 100+(i%4)*50)
		b.WriteString("  enabled: true\n")
		fmt.Fprintf(&b, "  port: %d\n", 8000+i)
		b.WriteString("  protocol: http\n")
		fmt.Fprintf(&b, "  timeout: %d\n", 30+(i%3)*10)
		b.WriteString("  labels:\n")
		b.WriteString("    tier: backend\n")
		fmt.Fprintf(&b, "    team: team-%d\n", i%5)
	}
	// Add new services
	for i := n; i < n+added; i++ {
		fmt.Fprintf(&b, "- name: service-%03d\n", i)
		fmt.Fprintf(&b, "  version: 1.0.%d\n", i%10)
		fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%5))
		fmt.Fprintf(&b, "  memory: %dMi\n", 256+(i%4)*128)
		fmt.Fprintf(&b, "  cpu: %dm\n", 100+(i%4)*50)
		b.WriteString("  enabled: true\n")
		fmt.Fprintf(&b, "  port: %d\n", 8000+i)
		b.WriteString("  protocol: http\n")
		fmt.Fprintf(&b, "  timeout: %d\n", 30+(i%3)*10)
		b.WriteString("  labels:\n")
		b.WriteString("    tier: backend\n")
		fmt.Fprintf(&b, "    team: team-%d\n", i%5)
	}
	return []byte(b.String())
}

// generateK8sMultiDoc generates multi-document YAML with n K8s Deployments.
func generateK8sMultiDoc(n int) []byte {
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString("---\n")
		}
		b.WriteString("apiVersion: apps/v1\n")
		b.WriteString("kind: Deployment\n")
		b.WriteString("metadata:\n")
		fmt.Fprintf(&b, "  name: app-%03d\n", i)
		fmt.Fprintf(&b, "  namespace: ns-%d\n", i%3)
		b.WriteString("  labels:\n")
		b.WriteString("    app: myapp\n")
		fmt.Fprintf(&b, "    version: v%d\n", i%5)
		b.WriteString("spec:\n")
		fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%4))
		b.WriteString("  selector:\n")
		b.WriteString("    matchLabels:\n")
		fmt.Fprintf(&b, "      app: app-%03d\n", i)
		b.WriteString("  template:\n")
		b.WriteString("    metadata:\n")
		b.WriteString("      labels:\n")
		fmt.Fprintf(&b, "        app: app-%03d\n", i)
		b.WriteString("    spec:\n")
		b.WriteString("      containers:\n")
		fmt.Fprintf(&b, "      - name: app-%03d\n", i)
		fmt.Fprintf(&b, "        image: myregistry/app-%03d:v1\n", i)
		b.WriteString("        ports:\n")
		fmt.Fprintf(&b, "        - containerPort: %d\n", 8080+i)
		b.WriteString("        resources:\n")
		b.WriteString("          requests:\n")
		b.WriteString("            cpu: 100m\n")
		b.WriteString("            memory: 128Mi\n")
		b.WriteString("          limits:\n")
		b.WriteString("            cpu: 200m\n")
		b.WriteString("            memory: 256Mi\n")
	}
	return []byte(b.String())
}

// generateK8sMultiDocModified generates a modified variant of K8s multi-doc YAML.
func generateK8sMultiDocModified(n int) []byte {
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString("---\n")
		}
		b.WriteString("apiVersion: apps/v1\n")
		b.WriteString("kind: Deployment\n")
		b.WriteString("metadata:\n")
		fmt.Fprintf(&b, "  name: app-%03d\n", i)
		fmt.Fprintf(&b, "  namespace: ns-%d\n", i%3)
		b.WriteString("  labels:\n")
		b.WriteString("    app: myapp\n")
		if i%3 == 0 {
			fmt.Fprintf(&b, "    version: v%d\n", (i%5)+1) // changed
		} else {
			fmt.Fprintf(&b, "    version: v%d\n", i%5)
		}
		b.WriteString("spec:\n")
		if i%4 == 0 {
			fmt.Fprintf(&b, "  replicas: %d\n", 2+(i%4)) // changed
		} else {
			fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%4))
		}
		b.WriteString("  selector:\n")
		b.WriteString("    matchLabels:\n")
		fmt.Fprintf(&b, "      app: app-%03d\n", i)
		b.WriteString("  template:\n")
		b.WriteString("    metadata:\n")
		b.WriteString("      labels:\n")
		fmt.Fprintf(&b, "        app: app-%03d\n", i)
		b.WriteString("    spec:\n")
		b.WriteString("      containers:\n")
		fmt.Fprintf(&b, "      - name: app-%03d\n", i)
		if i%5 == 0 {
			fmt.Fprintf(&b, "        image: myregistry/app-%03d:v2\n", i) // changed
		} else {
			fmt.Fprintf(&b, "        image: myregistry/app-%03d:v1\n", i)
		}
		b.WriteString("        ports:\n")
		fmt.Fprintf(&b, "        - containerPort: %d\n", 8080+i)
		b.WriteString("        resources:\n")
		b.WriteString("          requests:\n")
		b.WriteString("            cpu: 100m\n")
		b.WriteString("            memory: 128Mi\n")
		b.WriteString("          limits:\n")
		if i%4 == 0 {
			b.WriteString("            cpu: 400m\n") // changed
		} else {
			b.WriteString("            cpu: 200m\n")
		}
		b.WriteString("            memory: 256Mi\n")
	}
	return []byte(b.String())
}

// buildOrderedMap creates a flat OrderedMap with n keys.
func buildOrderedMap(n int) *OrderedMap {
	om := NewOrderedMap()
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%04d", i)
		om.Keys = append(om.Keys, key)
		om.Values[key] = fmt.Sprintf("value-%d", i)
	}
	return om
}

// buildOrderedMapModified creates a modified variant (every 5th key changed).
func buildOrderedMapModified(n int) *OrderedMap {
	om := NewOrderedMap()
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%04d", i)
		om.Keys = append(om.Keys, key)
		if i%5 == 0 {
			om.Values[key] = fmt.Sprintf("modified-value-%d", i)
		} else {
			om.Values[key] = fmt.Sprintf("value-%d", i)
		}
	}
	return om
}

// buildServiceList creates a parsed []interface{} list with n named items
// for direct use in compareLists benchmarks.
func buildServiceList(n int) []interface{} {
	list := make([]interface{}, n)
	for i := 0; i < n; i++ {
		om := NewOrderedMap()
		om.Keys = append(om.Keys, "name", "version", "replicas")
		om.Values["name"] = fmt.Sprintf("service-%03d", i)
		om.Values["version"] = fmt.Sprintf("1.0.%d", i%10)
		om.Values["replicas"] = 1 + (i % 5)
		list[i] = om
	}
	return list
}

// buildServiceListModified creates a modified list for compareLists benchmarks.
func buildServiceListModified(n int) []interface{} {
	removed := 2
	added := n / 10
	if added < 1 {
		added = 1
	}
	list := make([]interface{}, 0, n-removed+added)
	for i := removed; i < n; i++ {
		om := NewOrderedMap()
		om.Keys = append(om.Keys, "name", "version", "replicas")
		om.Values["name"] = fmt.Sprintf("service-%03d", i)
		if i%5 == 0 {
			om.Values["version"] = fmt.Sprintf("2.0.%d", i%10)
		} else {
			om.Values["version"] = fmt.Sprintf("1.0.%d", i%10)
		}
		om.Values["replicas"] = 1 + (i % 5)
		list = append(list, om)
	}
	for i := n; i < n+added; i++ {
		om := NewOrderedMap()
		om.Keys = append(om.Keys, "name", "version", "replicas")
		om.Values["name"] = fmt.Sprintf("service-%03d", i)
		om.Values["version"] = fmt.Sprintf("1.0.%d", i%10)
		om.Values["replicas"] = 1 + (i % 5)
		list = append(list, om)
	}
	return list
}

// buildScalarList creates a list of n scalar values (no identifiers).
func buildScalarList(n int) []interface{} {
	list := make([]interface{}, n)
	for i := 0; i < n; i++ {
		list[i] = fmt.Sprintf("item-%d", i)
	}
	return list
}

// buildScalarListModified creates a modified scalar list for unordered comparison.
func buildScalarListModified(n int) []interface{} {
	removed := 2
	added := n / 10
	if added < 1 {
		added = 1
	}
	list := make([]interface{}, 0, n-removed+added)
	for i := removed; i < n; i++ {
		list = append(list, fmt.Sprintf("item-%d", i))
	}
	for i := n; i < n+added; i++ {
		list = append(list, fmt.Sprintf("item-%d", i))
	}
	return list
}

// buildNestedMap builds a deeply nested OrderedMap for deepEqual benchmarks.
func buildNestedMap(depth int) *OrderedMap {
	if depth == 0 {
		om := NewOrderedMap()
		om.Keys = append(om.Keys, "leaf")
		om.Values["leaf"] = "value"
		return om
	}
	om := NewOrderedMap()
	key := fmt.Sprintf("level-%d", depth)
	om.Keys = append(om.Keys, key)
	om.Values[key] = buildNestedMap(depth - 1)
	return om
}

// ---------------------------------------------------------------------------
// Benchmarks: Parsing
// ---------------------------------------------------------------------------

func BenchmarkParseWithOrder(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"Small", 5},
		{"Medium", 50},
		{"Large", 500},
		{"XLarge", 2000},
	}

	for _, sz := range sizes {
		data := generateServiceList(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ParseWithOrder(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: End-to-end Compare
// ---------------------------------------------------------------------------

func BenchmarkCompare(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"Small", 5},
		{"Medium", 50},
		{"Large", 500},
		{"XLarge", 2000},
	}

	for _, sz := range sizes {
		from := generateServiceList(sz.n)
		to := generateServiceListModified(sz.n)
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Compare(from, to, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCompare_Identical(b *testing.B) {
	data := generateServiceList(50)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Compare(data, data, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_PerfTestData(b *testing.B) {
	from, err := os.ReadFile("../../testdata/perf/test1/file1.yaml")
	if err != nil {
		b.Skipf("perf test data not found: %v", err)
	}
	to, err := os.ReadFile("../../testdata/perf/test1/file2.yaml")
	if err != nil {
		b.Skipf("perf test data not found: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Compare(from, to, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_WithOptions(b *testing.B) {
	from := generateServiceList(50)
	to := generateServiceListModified(50)

	b.Run("Chroot", func(b *testing.B) {
		opts := &Options{Chroot: "services"}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Compare(from, to, opts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("IgnoreOrder", func(b *testing.B) {
		opts := &Options{IgnoreOrderChanges: true}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Compare(from, to, opts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("K8s", func(b *testing.B) {
		fromK8s := generateK8sMultiDoc(10)
		toK8s := generateK8sMultiDocModified(10)
		opts := &Options{DetectKubernetes: true}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Compare(fromK8s, toK8s, opts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Benchmarks: Internal map comparison
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
				compareOrderedMaps("root", from, to, nil)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: List comparison strategies
// ---------------------------------------------------------------------------

func BenchmarkCompareLists_ByIdentifier(b *testing.B) {
	sizes := []int{10, 50, 500}

	for _, n := range sizes {
		from := buildServiceList(n)
		to := buildServiceListModified(n)
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compareListsByIdentifier("services", from, to, nil)
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
				compareListsUnordered("items", from, to, nil)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: K8s multi-document comparison
// ---------------------------------------------------------------------------

func BenchmarkCompare_K8sMultiDoc(b *testing.B) {
	sizes := []int{3, 10, 50}

	for _, n := range sizes {
		from := generateK8sMultiDoc(n)
		to := generateK8sMultiDocModified(n)
		opts := &Options{DetectKubernetes: true}
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Compare(from, to, opts)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: Deep equality
// ---------------------------------------------------------------------------

func BenchmarkDeepEqual(b *testing.B) {
	b.Run("Identical", func(b *testing.B) {
		om := buildOrderedMap(50)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			deepEqual(om, om, nil)
		}
	})

	b.Run("Different", func(b *testing.B) {
		om1 := buildOrderedMap(50)
		om2 := buildOrderedMapModified(50)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			deepEqual(om1, om2, nil)
		}
	})

	b.Run("Nested", func(b *testing.B) {
		nested1 := buildNestedMap(20)
		nested2 := buildNestedMap(20)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			deepEqual(nested1, nested2, nil)
		}
	})
}

// ---------------------------------------------------------------------------
// Benchmarks: Path ordering and diff sorting
// ---------------------------------------------------------------------------

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
		docs, err := ParseWithOrder(data)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				extractPathOrder(docs, docs, nil)
			}
		})
	}
}

func BenchmarkSortDiffsWithOrder(b *testing.B) {
	// Generate diffs of known sizes
	makeDiffs := func(n int) ([]Difference, map[string]int) {
		diffs := make([]Difference, n)
		pathOrder := make(map[string]int)
		for i := 0; i < n; i++ {
			path := fmt.Sprintf("root.section-%03d.key-%03d", i%10, i)
			diffs[i] = Difference{
				Path: path,
				Type: DiffModified,
				From: "old",
				To:   "new",
			}
			pathOrder[path] = i
		}
		return diffs, pathOrder
	}

	sizes := []int{100, 1000}
	for _, n := range sizes {
		diffs, pathOrder := makeDiffs(n)
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			// Make a copy to sort each iteration
			buf := make([]Difference, len(diffs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				copy(buf, diffs)
				sortDiffsWithOrder(buf, pathOrder)
			}
		})
	}
}
