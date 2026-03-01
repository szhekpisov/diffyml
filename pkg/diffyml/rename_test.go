package diffyml

import (
	"fmt"
	"strings"
	"testing"
)

// Task 1.1: Document serialization tests

func TestSerializeDocument_OrderedMap(t *testing.T) {
	meta := NewOrderedMap()
	meta.Keys = append(meta.Keys, "name", "namespace")
	meta.Values["name"] = "my-config"
	meta.Values["namespace"] = "default"

	doc := NewOrderedMap()
	doc.Keys = append(doc.Keys, "apiVersion", "kind", "metadata")
	doc.Values["apiVersion"] = "v1"
	doc.Values["kind"] = "ConfigMap"
	doc.Values["metadata"] = meta

	data, err := serializeDocument(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(data)

	// Verify key order is preserved
	apiIdx := strings.Index(result, "apiVersion")
	kindIdx := strings.Index(result, "kind")
	metaIdx := strings.Index(result, "metadata")
	if apiIdx >= kindIdx || kindIdx >= metaIdx {
		t.Errorf("expected key order apiVersion < kind < metadata, got positions %d, %d, %d",
			apiIdx, kindIdx, metaIdx)
	}

	// Verify values present
	if !strings.Contains(result, "my-config") {
		t.Error("expected 'my-config' in output")
	}
	if !strings.Contains(result, "default") {
		t.Error("expected 'default' in output")
	}
}

func TestSerializeDocument_Scalars(t *testing.T) {
	doc := NewOrderedMap()
	doc.Keys = append(doc.Keys, "str", "num", "float", "flag", "empty")
	doc.Values["str"] = "hello"
	doc.Values["num"] = 42
	doc.Values["float"] = 3.14
	doc.Values["flag"] = true
	doc.Values["empty"] = nil

	data, err := serializeDocument(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "hello") {
		t.Error("expected string value")
	}
	if !strings.Contains(result, "42") {
		t.Error("expected int value")
	}
	if !strings.Contains(result, "3.14") {
		t.Error("expected float value")
	}
	if !strings.Contains(result, "true") {
		t.Error("expected bool value")
	}
}

func TestSerializeDocument_NestedSequence(t *testing.T) {
	item1 := NewOrderedMap()
	item1.Keys = append(item1.Keys, "name")
	item1.Values["name"] = "item-a"

	item2 := NewOrderedMap()
	item2.Keys = append(item2.Keys, "name")
	item2.Values["name"] = "item-b"

	doc := NewOrderedMap()
	doc.Keys = append(doc.Keys, "items")
	doc.Values["items"] = []interface{}{item1, item2}

	data, err := serializeDocument(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "item-a") || !strings.Contains(result, "item-b") {
		t.Errorf("expected both items in output, got:\n%s", result)
	}
}

// Task 1.2: Similarity index tests

func TestSimilarityIndex_IdenticalDocs(t *testing.T) {
	data := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")
	idx1 := newSimilarityIndex(data)
	idx2 := newSimilarityIndex(data)

	score := idx1.score(idx2)
	if score != 100 {
		t.Errorf("expected score 100 for identical docs, got %d", score)
	}
}

func TestSimilarityIndex_CompletelyDifferent(t *testing.T) {
	data1 := []byte("aaa bbb ccc\nddd eee fff\n")
	data2 := []byte("xxx yyy zzz\nwww vvv uuu\n")
	idx1 := newSimilarityIndex(data1)
	idx2 := newSimilarityIndex(data2)

	score := idx1.score(idx2)
	if score != 0 {
		t.Errorf("expected score 0 for completely different docs, got %d", score)
	}
}

func TestSimilarityIndex_PartialMatch(t *testing.T) {
	data1 := []byte("line1\nline2\nline3\nline4\n")
	data2 := []byte("line1\nline2\nlineX\nlineY\n")
	idx1 := newSimilarityIndex(data1)
	idx2 := newSimilarityIndex(data2)

	score := idx1.score(idx2)
	if score <= 0 || score >= 100 {
		t.Errorf("expected score between 0 and 100, got %d", score)
	}
	// 2 out of 4 lines match = 50%
	if score != 50 {
		t.Errorf("expected score 50, got %d", score)
	}
}

func TestSimilarityIndex_EmptyDocs(t *testing.T) {
	idx1 := newSimilarityIndex([]byte(""))
	idx2 := newSimilarityIndex([]byte(""))

	score := idx1.score(idx2)
	if score != 0 {
		t.Errorf("expected score 0 for empty docs, got %d", score)
	}
}

// Task 1.3: Rename detection orchestrator tests

func mkK8sConfigMap(name string, dataKeys []string) *OrderedMap {
	meta := NewOrderedMap()
	meta.Keys = append(meta.Keys, "name")
	meta.Values["name"] = name

	dataMap := NewOrderedMap()
	for _, k := range dataKeys {
		dataMap.Keys = append(dataMap.Keys, k)
		dataMap.Values[k] = "value"
	}

	doc := NewOrderedMap()
	doc.Keys = append(doc.Keys, "apiVersion", "kind", "metadata", "data")
	doc.Values["apiVersion"] = "v1"
	doc.Values["kind"] = "ConfigMap"
	doc.Values["metadata"] = meta
	doc.Values["data"] = dataMap
	return doc
}

func TestDetectRenames_BasicMatch(t *testing.T) {
	fromDoc := mkK8sConfigMap("app-config-abc123", []string{"key1"})
	toDoc := mkK8sConfigMap("app-config-def456", []string{"key1"})

	from := []interface{}{fromDoc}
	to := []interface{}{toDoc}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, []int{0}, opts)

	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	if matched[0] != 0 {
		t.Errorf("expected from[0]→to[0], got from[0]→to[%d]", matched[0])
	}
	if len(remainFrom) != 0 {
		t.Errorf("expected 0 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 0 {
		t.Errorf("expected 0 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_GreedyMatching(t *testing.T) {
	// from[0] has data keys a,b,c
	// from[1] has data keys a,b,d
	// to[0] has data keys a,b,d → most similar to from[1] (87%) and from[0] (75%)
	// to[1] has data keys x,y,z → dissimilar to both (below threshold)
	// Greedy: from[1]→to[0] wins (87% > 75%), from[0] stays unmatched
	from := []interface{}{
		mkK8sConfigMap("from-0", []string{"a", "b", "c"}),
		mkK8sConfigMap("from-1", []string{"a", "b", "d"}),
	}
	to := []interface{}{
		mkK8sConfigMap("to-0", []string{"a", "b", "d"}),
		mkK8sConfigMap("to-1", []string{"x", "y", "z"}),
	}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0, 1}, []int{0, 1}, opts)

	// from[1]→to[0] should be the greedy winner
	if toIdx, ok := matched[1]; !ok || toIdx != 0 {
		t.Errorf("expected from[1]→to[0] (greedy), got matched[1]=%v ok=%v", toIdx, ok)
	}
	// from[0] should NOT be matched (to[0] taken, to[1] below threshold)
	if _, ok := matched[0]; ok {
		t.Error("expected from[0] to NOT be matched")
	}
	if len(remainFrom) != 1 || remainFrom[0] != 0 {
		t.Errorf("expected remainFrom=[0], got %v", remainFrom)
	}
	if len(remainTo) != 1 || remainTo[0] != 1 {
		t.Errorf("expected remainTo=[1], got %v", remainTo)
	}
}

func TestDetectRenames_BelowThreshold(t *testing.T) {
	// Documents with mostly different content should not match
	from := []interface{}{mkK8sConfigMap("from-config", []string{
		"key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8",
	})}
	to := []interface{}{mkK8sConfigMap("to-config", []string{
		"x1", "x2", "x3", "x4", "x5", "x6", "x7", "x8",
	})}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, []int{0}, opts)

	if len(matched) != 0 {
		t.Errorf("expected 0 matches for dissimilar docs, got %d", len(matched))
	}
	if len(remainFrom) != 1 {
		t.Errorf("expected 1 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 1 {
		t.Errorf("expected 1 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_ExceedsLimit(t *testing.T) {
	mkMinK8sDoc := func(name string) *OrderedMap {
		meta := NewOrderedMap()
		meta.Keys = append(meta.Keys, "name")
		meta.Values["name"] = name

		doc := NewOrderedMap()
		doc.Keys = append(doc.Keys, "apiVersion", "kind", "metadata")
		doc.Values["apiVersion"] = "v1"
		doc.Values["kind"] = "ConfigMap"
		doc.Values["metadata"] = meta
		return doc
	}

	// 51 documents exceeds rename limit of 50
	from := make([]interface{}, 51)
	to := make([]interface{}, 51)
	unmatchedFrom := make([]int, 51)
	unmatchedTo := make([]int, 51)
	for i := 0; i < 51; i++ {
		from[i] = mkMinK8sDoc(fmt.Sprintf("from-%d", i))
		to[i] = mkMinK8sDoc(fmt.Sprintf("to-%d", i))
		unmatchedFrom[i] = i
		unmatchedTo[i] = i
	}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, unmatchedFrom, unmatchedTo, opts)

	if len(matched) != 0 {
		t.Errorf("expected 0 matches when exceeding limit, got %d", len(matched))
	}
	if len(remainFrom) != 51 {
		t.Errorf("expected 51 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 51 {
		t.Errorf("expected 51 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_Disabled(t *testing.T) {
	from := []interface{}{mkK8sConfigMap("a", nil)}
	to := []interface{}{mkK8sConfigMap("b", nil)}

	opts := &Options{DetectRenames: false}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, []int{0}, opts)

	if len(matched) != 0 {
		t.Errorf("expected 0 matches when disabled, got %d", len(matched))
	}
	if len(remainFrom) != 1 {
		t.Errorf("expected 1 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 1 {
		t.Errorf("expected 1 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_EmptyUnmatched(t *testing.T) {
	from := []interface{}{mkK8sConfigMap("a", nil)}
	to := []interface{}{mkK8sConfigMap("b", nil)}
	opts := &Options{DetectRenames: true}

	// Empty from side
	matched, remainFrom, remainTo := detectRenames(from, to, []int{}, []int{0}, opts)
	if len(matched) != 0 {
		t.Errorf("expected 0 matches with empty from, got %d", len(matched))
	}
	if len(remainFrom) != 0 {
		t.Errorf("expected 0 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 1 {
		t.Errorf("expected 1 remaining to, got %d", len(remainTo))
	}

	// Empty to side
	matched, remainFrom, remainTo = detectRenames(from, to, []int{0}, []int{}, opts)
	if len(matched) != 0 {
		t.Errorf("expected 0 matches with empty to, got %d", len(matched))
	}
	if len(remainFrom) != 1 {
		t.Errorf("expected 1 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 0 {
		t.Errorf("expected 0 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_NonK8sFiltered(t *testing.T) {
	k8sDoc := mkK8sConfigMap("my-config", []string{"key1"})

	nonK8sDoc := NewOrderedMap()
	nonK8sDoc.Keys = append(nonK8sDoc.Keys, "someKey")
	nonK8sDoc.Values["someKey"] = "someValue"

	// Index 0 = non-K8s, Index 1 = K8s
	from := []interface{}{nonK8sDoc, k8sDoc}
	to := []interface{}{nonK8sDoc, k8sDoc}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0, 1}, []int{0, 1}, opts)

	// K8s docs (index 1) should match each other
	if len(matched) != 1 {
		t.Fatalf("expected 1 match (K8s doc), got %d", len(matched))
	}
	if matched[1] != 1 {
		t.Errorf("expected from[1]→to[1], got from[1]→to[%d]", matched[1])
	}

	// Non-K8s doc (index 0) should be in remaining
	foundNonK8sFrom := false
	for _, idx := range remainFrom {
		if idx == 0 {
			foundNonK8sFrom = true
		}
	}
	if !foundNonK8sFrom {
		t.Errorf("expected non-K8s doc (index 0) in remainingFrom, got %v", remainFrom)
	}

	foundNonK8sTo := false
	for _, idx := range remainTo {
		if idx == 0 {
			foundNonK8sTo = true
		}
	}
	if !foundNonK8sTo {
		t.Errorf("expected non-K8s doc (index 0) in remainingTo, got %v", remainTo)
	}
}
