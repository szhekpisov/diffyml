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
	doc.Values["items"] = []any{item1, item2}

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

	from := []any{fromDoc}
	to := []any{toDoc}

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
	from := []any{
		mkK8sConfigMap("from-0", []string{"a", "b", "c"}),
		mkK8sConfigMap("from-1", []string{"a", "b", "d"}),
	}
	to := []any{
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
	from := []any{mkK8sConfigMap("from-config", []string{
		"key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8",
	})}
	to := []any{mkK8sConfigMap("to-config", []string{
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
	from := make([]any, 51)
	to := make([]any, 51)
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
	from := []any{mkK8sConfigMap("a", nil)}
	to := []any{mkK8sConfigMap("b", nil)}

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
	from := []any{mkK8sConfigMap("a", nil)}
	to := []any{mkK8sConfigMap("b", nil)}
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
	from := []any{nonK8sDoc, k8sDoc}
	to := []any{nonK8sDoc, k8sDoc}

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
	doc := map[string]any{
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

	from := []any{smallDoc}
	to := []any{largeDoc}

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
	from := []any{
		mkK8sConfigMap("from-0", []string{"shared1", "shared2", "shared3"}),
		mkK8sConfigMap("from-1", []string{"shared1", "shared2", "shared3"}),
	}
	to := []any{
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
	from := make([]any, 50)
	to := make([]any, 50)
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
	from := make([]any, 1)
	to := make([]any, 51)
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

	from := []any{mkDoc("aaa-from", "ns-from")}
	to := []any{mkDoc("bbb-to", "ns-to")}

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
	from := []any{largeDoc}
	to := []any{smallDoc}

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

	from := []any{smallDoc}
	to := []any{largeDoc}
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
	from := make([]any, n)
	to := make([]any, n)
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

func TestNewSimilarityIndex_WhitespaceOnlyLinesSkipped(t *testing.T) {
	// Each line consists solely of whitespace and MUST be skipped (numLines == 0).
	// Covers the three byte comparisons in the hasContent check (' ', '\t', '\r')
	// and the two && operators joining them: any mutation that lets a space, tab,
	// or carriage-return count as content makes numLines > 0.
	cases := map[string]string{
		"spaces only":      "   ",
		"tabs only":        "\t\t",
		"carriage returns": "\r\r",
		"mixed whitespace": " \t\r",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			idx := newSimilarityIndex([]byte(in))
			if idx.numLines != 0 {
				t.Errorf("whitespace-only line %q: numLines=%d, want 0", in, idx.numLines)
			}
		})
	}
}

func TestNewSimilarityIndex_BlankLineBeforeContent(t *testing.T) {
	// A whitespace-only line is skipped via `continue`, NOT `break`: scanning must
	// proceed to the content lines that follow it. If `continue` becomes `break`,
	// the scan aborts at the blank line and the trailing content is never indexed.
	idx := newSimilarityIndex([]byte("   \nfoo\nbar"))
	if idx.numLines != 2 {
		t.Errorf("expected 2 content lines after a leading blank line, got %d", idx.numLines)
	}
}

func TestDetectRenames_SizeRatioSkipThenMatch(t *testing.T) {
	// In buildRenamePairs, a toIdx rejected by the size-ratio guard is skipped via
	// `continue` so later candidates are still considered. to[0] is far too large
	// to match from[0] (ratio < 60%) and is skipped; to[1] is identical and matches.
	// If `continue` becomes `break`, the inner loop aborts at to[0] and to[1] is
	// never reached, so no match is found.
	from := []any{mkK8sConfigMap("cfg-a", []string{"k1"})}
	to := []any{
		mkK8sConfigMap("cfg-huge", []string{
			"a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
			"k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
		}),
		mkK8sConfigMap("cfg-a", []string{"k1"}),
	}

	opts := &Options{DetectRenames: true}
	matched, _, _ := detectRenames(from, to, []int{0}, []int{0, 1}, opts)

	if toIdx, ok := matched[0]; !ok || toIdx != 1 {
		t.Errorf("expected from[0]→to[1] (to[0] skipped by size ratio), got matched[0]=%v ok=%v", toIdx, ok)
	}
}

func TestDetectRenames_DisabledReturnsNonNilMap(t *testing.T) {
	// The early-return paths return the map allocated at the top of detectRenames.
	// If that allocation statement is removed, the function returns a nil map.
	from := []any{mkK8sConfigMap("a", nil)}
	to := []any{mkK8sConfigMap("b", nil)}

	opts := &Options{DetectRenames: false}
	matched, _, _ := detectRenames(from, to, []int{0}, []int{0}, opts)

	if matched == nil {
		t.Error("expected a non-nil (initialized) match map on the early-return path, got nil")
	}
}

func TestDetectRenames_EmptyFromPreservesToOrder(t *testing.T) {
	// With an empty unmatchedFrom, detectRenames must early-return immediately,
	// handing back unmatchedTo unchanged. If the `len(unmatchedFrom) == 0` guard is
	// disabled (removed, or its || turned into &&), the function instead proceeds
	// into K8s filtering, which reorders the indices (non-K8s before K8s).
	nonK8s := NewOrderedMap()
	nonK8s.Keys = append(nonK8s.Keys, "someKey")
	nonK8s.Values["someKey"] = "someValue"

	to := []any{mkK8sConfigMap("k8s-cfg", []string{"k1"}), nonK8s} // index 0 = K8s, 1 = non-K8s

	opts := &Options{DetectRenames: true}
	_, _, remainTo := detectRenames([]any{}, to, []int{}, []int{0, 1}, opts)

	if len(remainTo) != 2 || remainTo[0] != 0 || remainTo[1] != 1 {
		t.Errorf("expected remainingTo to preserve input order [0 1], got %v", remainTo)
	}
}

func TestDetectRenames_EmptyToPreservesFromOrder(t *testing.T) {
	// Symmetric to the empty-from case: an empty unmatchedTo must early-return with
	// unmatchedFrom unchanged. If the `len(unmatchedTo) == 0` guard is disabled, the
	// function proceeds into K8s filtering and reorders the from indices.
	nonK8s := NewOrderedMap()
	nonK8s.Keys = append(nonK8s.Keys, "someKey")
	nonK8s.Values["someKey"] = "someValue"

	from := []any{mkK8sConfigMap("k8s-cfg", []string{"k1"}), nonK8s} // index 0 = K8s, 1 = non-K8s

	opts := &Options{DetectRenames: true}
	_, remainFrom, _ := detectRenames(from, []any{}, []int{0, 1}, []int{}, opts)

	if len(remainFrom) != 2 || remainFrom[0] != 0 || remainFrom[1] != 1 {
		t.Errorf("expected remainingFrom to preserve input order [0 1], got %v", remainFrom)
	}
}

func TestNewSimilarityIndex_LeadingNewline(t *testing.T) {
	// The scan loop starts at index 0, so a leading '\n' is recognized as an empty
	// first line and skipped. If the loop instead starts at index 1, data[0]=='\n'
	// is never detected and the first content line is hashed *with* the leading
	// newline, changing its hash. A line "foo" preceded by a newline must hash
	// identically to a bare "foo".
	withLeading := newSimilarityIndex([]byte("\nfoo\n"))
	plain := newSimilarityIndex([]byte("foo\n"))
	if got := withLeading.score(plain); got != 100 {
		t.Errorf("leading-newline line must hash identically to a plain line: score=%d, want 100", got)
	}
}

// mkOrderedDoc builds an OrderedMap from ordered key/value string pairs.
func mkOrderedDoc(kv ...[2]string) *OrderedMap {
	d := NewOrderedMap()
	for _, p := range kv {
		d.Keys = append(d.Keys, p[0])
		d.Values[p[0]] = p[1]
	}
	return d
}

func TestBuildRenamePairs_ScoreJustBelowThreshold(t *testing.T) {
	// Two equally-sized docs sharing 13 of 22 lines score exactly 59 — one point
	// below the 60 threshold. The byte-size ratio is 100, so the size-ratio guard
	// is neutral and only the `s >= renameScoreThreshold` check decides. At
	// threshold 60, score 59 yields no pair; if the threshold is decremented to
	// 59, a spurious pair appears.
	a := mkOrderedDoc()
	b := mkOrderedDoc()
	for i := 0; i < 13; i++ { // shared lines
		k := fmt.Sprintf("shared%02d", i)
		a.Keys = append(a.Keys, k)
		a.Values[k] = "v"
		b.Keys = append(b.Keys, k)
		b.Values[k] = "v"
	}
	for i := 0; i < 9; i++ { // unique lines per side
		ka, kb := fmt.Sprintf("aonly%02d", i), fmt.Sprintf("bonly%02d", i)
		a.Keys = append(a.Keys, ka)
		a.Values[ka] = "v"
		b.Keys = append(b.Keys, kb)
		b.Values[kb] = "v"
	}

	aData, bData := serializeDocument(a), serializeDocument(b)
	score := newSimilarityIndex(aData).score(newSimilarityIndex(bData))
	if score != renameScoreThreshold-1 {
		t.Fatalf("precondition: score=%d, want %d (threshold-1)", score, renameScoreThreshold-1)
	}
	if ratio := min(len(aData), len(bData)) * 100 / max(len(aData), len(bData)); ratio < renameScoreThreshold {
		t.Fatalf("precondition: size ratio=%d must be >= threshold so the size guard stays neutral", ratio)
	}

	pairs := buildRenamePairs([]any{a}, []any{b}, []int{0}, []int{0})
	if len(pairs) != 0 {
		t.Errorf("score %d is below threshold %d; expected no pair, got %d", score, renameScoreThreshold, len(pairs))
	}
}

func TestBuildRenamePairs_SizeRatioMultiplierLower(t *testing.T) {
	// Size-ratio guard: `minLen*100/maxLen < threshold`. Construct a pair whose
	// byte ratio is exactly 60 (small=18B, large=30B). With the *100 multiplier
	// the ratio is 60, so `60 < 60` is false and the pair is kept (similarity 75
	// >= 60). If the multiplier drops to 99 the ratio becomes 59, `59 < 60` is
	// true, and the pair is wrongly rejected.
	small := mkOrderedDoc([2]string{"s0", "v"}, [2]string{"s1", "v"}, [2]string{"s2", "v"})
	large := mkOrderedDoc([2]string{"s0", "v"}, [2]string{"s1", "v"}, [2]string{"s2", "v"},
		[2]string{"p", strings.Repeat("x", 8)})

	sData, lData := serializeDocument(small), serializeDocument(large)
	minLen, maxLen := min(len(sData), len(lData)), max(len(sData), len(lData))
	if minLen*100/maxLen != renameScoreThreshold || minLen*99/maxLen != renameScoreThreshold-1 {
		t.Fatalf("precondition: want ratio*100=%d and ratio*99=%d, got *100=%d *99=%d",
			renameScoreThreshold, renameScoreThreshold-1, minLen*100/maxLen, minLen*99/maxLen)
	}
	if sim := newSimilarityIndex(sData).score(newSimilarityIndex(lData)); sim < renameScoreThreshold {
		t.Fatalf("precondition: similarity %d must be >= threshold so a pair forms when the guard passes", sim)
	}

	pairs := buildRenamePairs([]any{small}, []any{large}, []int{0}, []int{0})
	if len(pairs) != 1 {
		t.Errorf("byte ratio 60 passes the size guard (60 < 60 is false); expected 1 pair, got %d", len(pairs))
	}
}

func TestBuildRenamePairs_SizeRatioMultiplierHigher(t *testing.T) {
	// Size-ratio guard: `minLen*100/maxLen < threshold`. Construct a pair whose
	// byte ratio is 59 (small=25B, large=42B). With the *100 multiplier the ratio
	// is 59, `59 < 60` is true, and the pair is rejected. If the multiplier rises
	// to 101 the ratio becomes 60, `60 < 60` is false, and a pair wrongly forms
	// (similarity 80 >= 60).
	small := mkOrderedDoc([2]string{"s0", "v"}, [2]string{"s1", "v"}, [2]string{"s2", "v"},
		[2]string{"f", "y"})
	large := mkOrderedDoc([2]string{"s0", "v"}, [2]string{"s1", "v"}, [2]string{"s2", "v"},
		[2]string{"f", "y"}, [2]string{"p", strings.Repeat("x", 13)})

	sData, lData := serializeDocument(small), serializeDocument(large)
	minLen, maxLen := min(len(sData), len(lData)), max(len(sData), len(lData))
	if minLen*100/maxLen != renameScoreThreshold-1 || minLen*101/maxLen < renameScoreThreshold {
		t.Fatalf("precondition: want ratio*100=%d and ratio*101>=%d, got *100=%d *101=%d",
			renameScoreThreshold-1, renameScoreThreshold, minLen*100/maxLen, minLen*101/maxLen)
	}
	if sim := newSimilarityIndex(sData).score(newSimilarityIndex(lData)); sim < renameScoreThreshold {
		t.Fatalf("precondition: similarity %d must be >= threshold so a pair would form if the guard passed", sim)
	}

	pairs := buildRenamePairs([]any{small}, []any{large}, []int{0}, []int{0})
	if len(pairs) != 0 {
		t.Errorf("byte ratio 59 fails the size guard (59 < 60 is true); expected no pair, got %d", len(pairs))
	}
}

func TestDetectRenames_SizeRatioBoundaryExact(t *testing.T) {
	// Targets CONDITIONALS_BOUNDARY mutant at rename.go:152
	// which changes `minLen*100/maxLen < threshold` to `<= threshold`.
	//
	// We construct two K8s docs where the byte-size ratio is exactly 60%
	// (the threshold). With `<`, ratio 60 < 60 is false → pair passes.
	// With `<=`, ratio 60 <= 60 is true → pair is rejected (mutant behavior).
	// Both docs share enough lines for similarity >= 60%.

	// Build small doc: shared structure + small data
	smallDoc := mkMinK8sDoc("cfg-small")
	smallData := NewOrderedMap()
	smallData.Keys = append(smallData.Keys, "key1")
	smallData.Values["key1"] = "val"
	smallDoc.Keys = append(smallDoc.Keys, "data")
	smallDoc.Values["data"] = smallData

	smallBytes := serializeDocument(smallDoc)
	smallLen := len(smallBytes)

	// We need maxLen such that smallLen*100/maxLen == 60 (integer division).
	// That means maxLen = smallLen*100/60 (rounded so integer div gives 60).
	targetMaxLen := smallLen * 100 / 60

	// Build large doc: same shared keys + extra padding in a data value
	largeDoc := mkMinK8sDoc("cfg-large")
	largeData := NewOrderedMap()
	largeData.Keys = append(largeData.Keys, "key1")
	largeData.Values["key1"] = "val"

	// Start with a padding key and adjust length
	padValue := ""
	largeData.Keys = append(largeData.Keys, "pad")
	largeData.Values["pad"] = padValue
	largeDoc.Keys = append(largeDoc.Keys, "data")
	largeDoc.Values["data"] = largeData

	// Binary search for the right padding length
	lo, hi := 0, 1000
	for lo < hi {
		mid := (lo + hi) / 2
		largeData.Values["pad"] = strings.Repeat("x", mid)
		largeBytes := serializeDocument(largeDoc)
		if len(largeBytes) < targetMaxLen {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	// Fine-tune: try values around lo to find exact ratio=60
	found := false
	for delta := -2; delta <= 2; delta++ {
		padLen := lo + delta
		if padLen < 0 {
			continue
		}
		largeData.Values["pad"] = strings.Repeat("x", padLen)
		largeBytes := serializeDocument(largeDoc)
		ratio := smallLen * 100 / len(largeBytes)
		if ratio == renameScoreThreshold {
			found = true
			break
		}
	}
	if !found {
		// Fallback: adjust smallDoc padding to hit exact ratio
		t.Fatalf("could not construct docs with byte-size ratio exactly %d%%; smallLen=%d, targetMaxLen=%d",
			renameScoreThreshold, smallLen, targetMaxLen)
	}

	// Verify preconditions
	largeBytes := serializeDocument(largeDoc)
	minLen := min(smallLen, len(largeBytes))
	maxLen := max(smallLen, len(largeBytes))
	sizeRatio := minLen * 100 / maxLen
	if sizeRatio != renameScoreThreshold {
		t.Fatalf("precondition: size ratio = %d%%, want exactly %d%%", sizeRatio, renameScoreThreshold)
	}

	fromIdx := newSimilarityIndex(smallBytes)
	toIdx := newSimilarityIndex(largeBytes)
	similarity := fromIdx.score(toIdx)
	if similarity < renameScoreThreshold {
		t.Fatalf("precondition: similarity %d%% < %d%%, need >= threshold", similarity, renameScoreThreshold)
	}

	// Size ratio == threshold → original code passes (60 < 60 = false), mutant rejects (60 <= 60 = true)
	from := []any{smallDoc}
	to := []any{largeDoc}
	opts := &Options{DetectRenames: true}
	matched, remainFrom, remainTo := detectRenames(from, to, []int{0}, []int{0}, opts)

	if len(matched) != 1 {
		t.Errorf("expected 1 match at size ratio exactly %d%% (passes < check), got %d; "+
			"sizeRatio=%d%%, similarity=%d%%", renameScoreThreshold, len(matched), sizeRatio, similarity)
	}
	if len(remainFrom) != 0 {
		t.Errorf("expected 0 remaining from, got %d", len(remainFrom))
	}
	if len(remainTo) != 0 {
		t.Errorf("expected 0 remaining to, got %d", len(remainTo))
	}
}
