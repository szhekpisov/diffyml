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

	data := serializeDocument(doc)

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

	data := serializeDocument(doc)

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

	data := serializeDocument(doc)

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

func mkMinK8sDoc(name string) *OrderedMap {
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

func mkK8sConfigMap(name string, dataKeys []string) *OrderedMap {
	doc := mkMinK8sDoc(name)

	dataMap := NewOrderedMap()
	for _, k := range dataKeys {
		dataMap.Keys = append(dataMap.Keys, k)
		dataMap.Values[k] = "value"
	}

	doc.Keys = append(doc.Keys, "data")
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

func TestSerializeDocument_MapStringInterface(t *testing.T) {
	doc := map[string]interface{}{
		"beta":  "two",
		"alpha": "one",
	}
	data := serializeDocument(doc)
	result := string(data)
	// Keys should be sorted alphabetically
	alphaIdx := strings.Index(result, "alpha")
	betaIdx := strings.Index(result, "beta")
	if alphaIdx >= betaIdx {
		t.Errorf("expected sorted key order alpha < beta, got positions %d, %d", alphaIdx, betaIdx)
	}
}

func TestSerializeDocument_UnknownType(t *testing.T) {
	// Pass a type not in the switch (e.g., struct) — should fall through to Encode
	type custom struct{ X int }
	data := serializeDocument(custom{X: 42})
	result := string(data)
	if !strings.Contains(result, "42") {
		t.Errorf("expected encoded output containing 42, got: %s", result)
	}
}

func TestSimilarityIndex_AsymmetricLineCounts(t *testing.T) {
	// self has fewer lines than other → exercises other.numLines > maxLines branch
	data1 := []byte("line1\n")
	data2 := []byte("line1\nline2\nline3\n")
	idx1 := newSimilarityIndex(data1)
	idx2 := newSimilarityIndex(data2)

	score := idx1.score(idx2)
	// 1 matching out of max(1,3) = 3 → 33%
	if score != 33 {
		t.Errorf("expected score 33, got %d", score)
	}
}

func TestSimilarityIndex_DuplicateLines(t *testing.T) {
	// self has 1 occurrence but other has 2 → exercises selfCount < count branch
	data1 := []byte("aaa\nbbb\n")
	data2 := []byte("aaa\naaa\nbbb\n")
	idx1 := newSimilarityIndex(data1)
	idx2 := newSimilarityIndex(data2)

	score := idx1.score(idx2)
	// matching: min(1,2)=1 for "aaa" + min(1,1)=1 for "bbb" = 2
	// max lines = 3 → 2*100/3 = 66
	if score != 66 {
		t.Errorf("expected score 66, got %d", score)
	}
}

func TestDetectRenames_SizeRatioRejection(t *testing.T) {
	// Create two K8s docs with very different sizes so size ratio < 60%
	smallDoc := mkK8sConfigMap("small", []string{"a"})
	largeDoc := mkK8sConfigMap("large", []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
		"k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
	})

	from := []interface{}{smallDoc}
	to := []interface{}{largeDoc}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, []int{0}, opts)

	if len(matched) != 0 {
		t.Errorf("expected 0 matches due to size ratio rejection, got %d", len(matched))
	}
	if len(remainFrom) != 1 {
		t.Errorf("expected 1 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 1 {
		t.Errorf("expected 1 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_SortTiebreaker(t *testing.T) {
	// All pairs have the same high score, so tiebreaker (ascending fromIdx, toIdx) decides.
	from := []interface{}{
		mkK8sConfigMap("from-0", []string{"shared1", "shared2", "shared3"}),
		mkK8sConfigMap("from-1", []string{"shared1", "shared2", "shared3"}),
	}
	to := []interface{}{
		mkK8sConfigMap("to-0", []string{"shared1", "shared2", "shared3"}),
		mkK8sConfigMap("to-1", []string{"shared1", "shared2", "shared3"}),
	}

	opts := &Options{DetectRenames: true}
	matched, _, _ := detectRenames(from, to, []int{0, 1}, []int{0, 1}, opts)

	// With tiebreaker: from[0]→to[0] is assigned first, then from[1]→to[1]
	if len(matched) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matched))
	}
	if matched[0] != 0 {
		t.Errorf("expected from[0]→to[0], got from[0]→to[%d]", matched[0])
	}
	if matched[1] != 1 {
		t.Errorf("expected from[1]→to[1], got from[1]→to[%d]", matched[1])
	}
}

func TestNewSimilarityIndex_NoTrailingNewline(t *testing.T) {
	// Data without trailing newline — last line must still be hashed
	withNewline := newSimilarityIndex([]byte("line1\nline2\n"))
	withoutNewline := newSimilarityIndex([]byte("line1\nline2"))

	if withNewline.numLines != withoutNewline.numLines {
		t.Errorf("expected same line count, got %d vs %d", withNewline.numLines, withoutNewline.numLines)
	}
	if withNewline.score(withoutNewline) != 100 {
		t.Errorf("expected score 100, got %d", withNewline.score(withoutNewline))
	}
}

func TestDetectRenames_ExactlyAtLimit(t *testing.T) {
	// Exactly 50 documents should be allowed (limit is >50, not >=50)
	from := make([]interface{}, 50)
	to := make([]interface{}, 50)
	unmatchedFrom := make([]int, 50)
	unmatchedTo := make([]int, 50)
	for i := 0; i < 50; i++ {
		from[i] = mkMinK8sDoc(fmt.Sprintf("config-%d", i))
		to[i] = mkMinK8sDoc(fmt.Sprintf("config-%d", i))
		unmatchedFrom[i] = i
		unmatchedTo[i] = i
	}

	opts := &Options{DetectRenames: true}
	matched, _, _ := detectRenames(from, to, unmatchedFrom, unmatchedTo, opts)

	// With 50 identical docs, all should be matched
	if len(matched) != 50 {
		t.Errorf("expected 50 matches at limit, got %d", len(matched))
	}
}

func TestDetectRenames_AsymmetricLimitFromSmall(t *testing.T) {
	// k8sFrom=1, k8sTo=51 — max is 51, exceeds limit
	from := make([]interface{}, 1)
	to := make([]interface{}, 51)
	from[0] = mkMinK8sDoc("from-0")
	for i := 0; i < 51; i++ {
		to[i] = mkMinK8sDoc(fmt.Sprintf("to-%d", i))
	}

	unmatchedTo := make([]int, 51)
	for i := 0; i < 51; i++ {
		unmatchedTo[i] = i
	}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, unmatchedTo, opts)

	// Should be skipped because max(1, 51) > 50
	if len(matched) != 0 {
		t.Errorf("expected 0 matches when to-side exceeds limit, got %d", len(matched))
	}
	if len(remainFrom) != 1 {
		t.Errorf("expected 1 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 51 {
		t.Errorf("expected 51 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_ScoreExactlyAtThreshold(t *testing.T) {
	// Craft docs so similarity score is exactly 60%.
	// We need matching*100/maxLines = 60, i.e., 3 out of 5 lines match.
	// Serialized K8s ConfigMap has lines: apiVersion, kind, metadata, name, data entries.
	// A doc with 2 data entries has ~7 non-empty lines. We need exactly 5.
	// Use a non-K8s-aware approach: build docs where exactly 3/5 lines match.

	// Build two K8s docs with controlled content. Each has:
	// apiVersion: v1          (match)
	// kind: ConfigMap         (match)
	// metadata:               (match)
	//   name: xxx             (differ)
	//   namespace: yyy        (differ)
	// That's 5 non-empty lines, 3 matching → score = 60%
	mkDoc := func(name, ns string) *OrderedMap {
		meta := NewOrderedMap()
		meta.Keys = append(meta.Keys, "name", "namespace")
		meta.Values["name"] = name
		meta.Values["namespace"] = ns

		doc := NewOrderedMap()
		doc.Keys = append(doc.Keys, "apiVersion", "kind", "metadata")
		doc.Values["apiVersion"] = "v1"
		doc.Values["kind"] = "ConfigMap"
		doc.Values["metadata"] = meta
		return doc
	}

	from := []interface{}{mkDoc("aaa-from", "ns-from")}
	to := []interface{}{mkDoc("bbb-to", "ns-to")}

	// Verify the score is indeed 60
	fromData := serializeDocument(from[0])
	toData := serializeDocument(to[0])
	fromIdx := newSimilarityIndex(fromData)
	toIdx := newSimilarityIndex(toData)
	actualScore := fromIdx.score(toIdx)
	if actualScore != 60 {
		t.Fatalf("precondition: expected score 60, got %d (from lines=%d, to lines=%d)\nfrom:\n%s\nto:\n%s",
			actualScore, fromIdx.numLines, toIdx.numLines, string(fromData), string(toData))
	}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, []int{0}, opts)

	// Score 60 == threshold → should match (>= 60, not > 60)
	if len(matched) != 1 {
		t.Errorf("expected 1 match at score exactly 60%%, got %d", len(matched))
	}
	if len(remainFrom) != 0 {
		t.Errorf("expected 0 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 0 {
		t.Errorf("expected 0 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_SizeRatioSwapOrder(t *testing.T) {
	// Ensure size ratio check works correctly when from is larger than to
	// (exercises the minLen/maxLen swap at line 233)
	largeDoc := mkK8sConfigMap("large", []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
		"k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
	})
	smallDoc := mkK8sConfigMap("small", []string{"a"})

	// from=large, to=small (opposite order from SizeRatioRejection test)
	from := []interface{}{largeDoc}
	to := []interface{}{smallDoc}

	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, []int{0}, opts)

	if len(matched) != 0 {
		t.Errorf("expected 0 matches due to size ratio rejection (from larger), got %d", len(matched))
	}
	if len(remainFrom) != 1 {
		t.Errorf("expected 1 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 1 {
		t.Errorf("expected 1 remaining to, got %d", len(remainTo))
	}
}

func TestDetectRenames_SizeRatioBypassGuard(t *testing.T) {
	// Targets mutations that disable the size-ratio early rejection:
	//   - NEGATION on `maxLen > 0` → `<= 0` (makes condition always false)
	//   - ARITHMETIC on `minLen*100/maxLen` (produces wrong ratio)
	//
	// We construct a pair where byte-size ratio < 60% but line similarity >= 60%.
	// The small doc shares most lines with the large doc, but the large doc
	// has extra entries with very long values that inflate its byte count.
	// Without the size-ratio guard, similarity scoring would match them.

	smallDoc := mkK8sConfigMap("small-cfg", []string{"k1", "k2", "k3", "k4", "k5"})

	// Large doc: same shared keys + extra entries with very long values
	largeDoc := mkMinK8sDoc("large-cfg")
	dataMap := NewOrderedMap()
	for _, k := range []string{"k1", "k2", "k3", "k4", "k5"} {
		dataMap.Keys = append(dataMap.Keys, k)
		dataMap.Values[k] = "value"
	}
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("long%d", i)
		dataMap.Keys = append(dataMap.Keys, key)
		dataMap.Values[key] = strings.Repeat("x", 300)
	}
	largeDoc.Keys = append(largeDoc.Keys, "data")
	largeDoc.Values["data"] = dataMap

	// Verify preconditions: size ratio < 60% but similarity >= 60%
	fromData := serializeDocument(smallDoc)
	toData := serializeDocument(largeDoc)
	fromIdx := newSimilarityIndex(fromData)
	toIdx := newSimilarityIndex(toData)

	sizeRatio := min(len(fromData), len(toData)) * 100 / max(len(fromData), len(toData))
	similarity := fromIdx.score(toIdx)

	if sizeRatio >= renameScoreThreshold {
		t.Fatalf("precondition failed: size ratio %d%% >= threshold %d%%, need < threshold",
			sizeRatio, renameScoreThreshold)
	}
	if similarity < renameScoreThreshold {
		t.Fatalf("precondition failed: similarity %d%% < threshold %d%%, need >= threshold",
			similarity, renameScoreThreshold)
	}

	from := []interface{}{smallDoc}
	to := []interface{}{largeDoc}
	opts := &Options{DetectRenames: true}
	matched, _, _ := detectRenames(from, to, []int{0}, []int{0}, opts)

	// Size-ratio check should reject this pair despite high similarity
	if len(matched) != 0 {
		t.Errorf("expected 0 matches (size-ratio rejection), got %d; "+
			"size ratio=%d%%, similarity=%d%%", len(matched), sizeRatio, similarity)
	}
}

func TestDetectRenames_SortTiebreaker_LargeInput(t *testing.T) {
	// Targets BOUNDARY mutants on the sort comparator (lines 163-171):
	//   - `score > score` → `>=` (non-irreflexive)
	//   - `toIdx < toIdx` → `<=` (non-irreflexive)
	// Non-irreflexive comparators can cause Go's pdqsort (used for n > 12)
	// to produce incorrect sort results.
	//
	// We use 5×5 = 25 rename pairs (all same score) so that pdqsort is
	// exercised instead of insertionSort, and verify deterministic assignment.

	n := 5
	from := make([]interface{}, n)
	to := make([]interface{}, n)
	unmatchedFrom := make([]int, n)
	unmatchedTo := make([]int, n)

	sharedKeys := []string{"shared1", "shared2", "shared3", "shared4", "shared5"}
	for i := 0; i < n; i++ {
		from[i] = mkK8sConfigMap(fmt.Sprintf("from-%d", i), sharedKeys)
		to[i] = mkK8sConfigMap(fmt.Sprintf("to-%d", i), sharedKeys)
		unmatchedFrom[i] = i
		unmatchedTo[i] = i
	}

	opts := &Options{DetectRenames: true}
	matched, _, _ := detectRenames(from, to, unmatchedFrom, unmatchedTo, opts)

	if len(matched) != n {
		t.Fatalf("expected %d matches, got %d", n, len(matched))
	}
	// With ascending fromIdx/toIdx tiebreaker: from[i]→to[i]
	for i := 0; i < n; i++ {
		if matched[i] != i {
			t.Errorf("expected from[%d]→to[%d], got from[%d]→to[%d]", i, i, i, matched[i])
		}
	}
}
