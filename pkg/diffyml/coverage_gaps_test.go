package diffyml

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Tests targeting remaining coverage gaps identified by gremlins mutation testing.

// --- deepEqual: *OrderedMap different lengths ---

func TestDeepEqual_OrderedMaps_DifferentLengths(t *testing.T) {
	a := &OrderedMap{Values: map[string]any{"x": 1, "y": 2}}
	b := &OrderedMap{Values: map[string]any{"x": 1}}
	if deepEqual(a, b, nil) {
		t.Error("expected OrderedMaps with different lengths to not be deepEqual")
	}
}

// --- deepEqual: []any slice case ---

func TestDeepEqual_Slices_Equal(t *testing.T) {
	a := []any{"x", "y", "z"}
	b := []any{"x", "y", "z"}
	if !deepEqual(a, b, nil) {
		t.Error("expected equal slices to be deepEqual")
	}
}

func TestDeepEqual_Slices_DifferentValues(t *testing.T) {
	a := []any{"x", "y"}
	b := []any{"x", "z"}
	if deepEqual(a, b, nil) {
		t.Error("expected slices with different values to not be deepEqual")
	}
}

func TestDeepEqual_Slices_DifferentLengths(t *testing.T) {
	a := []any{"x"}
	b := []any{"x", "y"}
	if deepEqual(a, b, nil) {
		t.Error("expected slices with different lengths to not be deepEqual")
	}
}

func TestDeepEqual_Slices_Nested(t *testing.T) {
	a := []any{[]any{"a", "b"}}
	b := []any{[]any{"a", "b"}}
	if !deepEqual(a, b, nil) {
		t.Error("expected nested equal slices to be deepEqual")
	}
}

// --- extractPathOrder: map[string]any branch ---

func TestExtractPathOrder_PlainMap(t *testing.T) {
	docs := []any{
		map[string]any{
			"beta":  "2",
			"alpha": "1",
		},
	}
	order := extractPathOrder(docs, nil, nil)

	if len(order) == 0 {
		t.Fatal("expected non-empty path order for plain map")
	}
	if _, ok := order["alpha"]; !ok {
		t.Error("expected 'alpha' in path order")
	}
	if _, ok := order["beta"]; !ok {
		t.Error("expected 'beta' in path order")
	}
}

func TestExtractPathOrder_PlainMapNested(t *testing.T) {
	docs := []any{
		map[string]any{
			"parent": map[string]any{"child": "val"},
		},
	}
	order := extractPathOrder(docs, nil, nil)

	if _, ok := order["parent"]; !ok {
		t.Error("expected 'parent' in path order")
	}
	if _, ok := order["parent.child"]; !ok {
		t.Error("expected 'parent.child' in path order")
	}
}

// --- areListItemsHeterogeneous: map[string]any items ---

func TestAreListItemsHeterogeneous_PlainMaps(t *testing.T) {
	from := []any{
		map[string]any{"namespaceSelector": "ns1"},
	}
	to := []any{
		map[string]any{"ipBlock": "10.0.0.0/8"},
	}

	if !areListItemsHeterogeneous(from, to) {
		t.Error("expected heterogeneous for plain maps with different single keys")
	}
}

func TestAreListItemsHeterogeneous_PlainMapsMultipleKeys(t *testing.T) {
	from := []any{
		map[string]any{"a": "1", "b": "2"},
	}
	to := []any{
		map[string]any{"c": "3"},
	}

	// from item has 2 keys, so checkSingleDistinctKeys returns false
	if areListItemsHeterogeneous(from, to) {
		t.Error("expected not heterogeneous when an item has multiple keys")
	}
}

// --- clamp: min/max boundary branches ---

func TestClamp_BelowMin(t *testing.T) {
	if got := clamp(-10, 0, 255); got != 0 {
		t.Errorf("clamp(-10, 0, 255) = %d, want 0", got)
	}
}

func TestClamp_AboveMax(t *testing.T) {
	if got := clamp(300, 0, 255); got != 255 {
		t.Errorf("clamp(300, 0, 255) = %d, want 255", got)
	}
}

func TestClamp_InRange(t *testing.T) {
	if got := clamp(128, 0, 255); got != 128 {
		t.Errorf("clamp(128, 0, 255) = %d, want 128", got)
	}
}

// --- ContextColorCode: true color path ---

func TestContextColorCode_TrueColor(t *testing.T) {
	code := ContextColorCode(true)
	if !strings.HasPrefix(code, "\033[38;2;") {
		t.Errorf("expected true color ANSI prefix, got %q", code)
	}
}

func TestContextColorCode_Basic(t *testing.T) {
	code := ContextColorCode(false)
	if code != "\033[90m" {
		t.Errorf("expected gray ANSI code \\033[90m, got %q", code)
	}
}

// --- ChrootError.Error() ---

func TestChrootError_Error(t *testing.T) {
	err := &ChrootError{Path: "spec.containers", Message: "key not found"}
	got := err.Error()
	if !strings.Contains(got, "spec.containers") {
		t.Errorf("expected path in error, got %q", got)
	}
	if !strings.Contains(got, "key not found") {
		t.Errorf("expected message in error, got %q", got)
	}
}

// --- renderFirstKeyValueYAML: []any value ---

func TestDetailedFormatter_ListValueInFirstKey(t *testing.T) {
	// The first key of a list entry maps to a list value,
	// exercising the []any case in renderFirstKeyValueYAML.
	om := &OrderedMap{
		Keys:   []string{"ports", "protocol"},
		Values: map[string]any{"ports": []any{"80", "443"}, "protocol": "TCP"},
	}

	diffs := []Difference{
		{
			Path: "spec.containers.0",
			Type: DiffAdded,
			From: nil,
			To:   om,
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	if !strings.Contains(result, "ports") {
		t.Errorf("expected 'ports' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "80") || !strings.Contains(result, "443") {
		t.Errorf("expected list items '80' and '443' in output, got:\n%s", result)
	}
}

// --- compareListsByIdentifier: fallback for items without identifiers ---

func TestCompareListsByIdentifier_NoIDFallback(t *testing.T) {
	// Mix identified and unidentified items.
	// Items with "name" get identifier-based matching; scalars use fallback.
	from := []any{
		&OrderedMap{
			Keys:   []string{"name", "value"},
			Values: map[string]any{"name": "a", "value": "1"},
		},
		"scalar-from-only",
		"shared-scalar",
	}
	to := []any{
		&OrderedMap{
			Keys:   []string{"name", "value"},
			Values: map[string]any{"name": "a", "value": "2"},
		},
		"new-scalar",
		"shared-scalar",
	}

	diffs := compareListsByIdentifier("items", from, to, nil)

	// "a" matched by name → modified value
	// "scalar-from-only" has no identifier → removed (fallback)
	// "new-scalar" has no identifier → added (fallback)
	// "shared-scalar" matched by deepEqual in fallback → no diff
	var removed, added int
	for _, d := range diffs {
		switch d.Type {
		case DiffRemoved:
			removed++
		case DiffAdded:
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

// --- TrueColorCode: exercises clamp through boundary values ---

func TestTrueColorCode_Clamped(t *testing.T) {
	// Values out of range should be clamped
	code := TrueColorCode(-1, 256, 128)
	expected := fmt.Sprintf("\033[38;2;%d;%d;%dm", 0, 255, 128)
	if code != expected {
		t.Errorf("expected clamped color code %q, got %q", expected, code)
	}
}

// === Section 2: Kill LIVED mutants ===

// --- extractPathOrder: index++ increment (diffyml.go:155) ---

func TestExtractPathOrder_PlainMapIndexIncrement(t *testing.T) {
	// Kills INCREMENT_DECREMENT at diffyml.go:155 (index++ → index--)
	// Uses nested maps so recursion enters the map[string]any case at line 150,
	// where index++ (line 155) is executed for each parent path.
	// With the mutation (index--), all parent paths get the same order value (0),
	// so the strict ordering assertion catches it.
	docs := []any{
		map[string]any{
			"alpha": map[string]any{"child1": "v1"},
			"beta":  map[string]any{"child2": "v2"},
			"gamma": map[string]any{"child3": "v3"},
		},
	}
	order := extractPathOrder(docs, nil, nil)

	// Keys are sorted alphabetically for plain maps, so alpha < beta < gamma
	if order["alpha"] >= order["beta"] {
		t.Errorf("expected alpha (%d) < beta (%d)", order["alpha"], order["beta"])
	}
	if order["beta"] >= order["gamma"] {
		t.Errorf("expected beta (%d) < gamma (%d)", order["beta"], order["gamma"])
	}
}

// --- DetailedFormatter: map continuation indent (detailed_formatter.go:239,294) ---

func TestDetailedFormatter_MapContinuationIndent(t *testing.T) {
	// Kills ARITHMETIC_BASE at detailed_formatter.go:294 (indent+4 → indent-4)
	// The first key's value is a map[string]any, so renderFirstKeyValueYAML
	// enters the map case (line 291) and renders children at indent+4 (=8 spaces).
	// With the mutation (indent-4), children would be at 0 spaces instead.
	diffs := []Difference{
		{
			Path: "items.0",
			Type: DiffAdded,
			From: nil,
			To:   map[string]any{"aaa": map[string]any{"child": "value"}},
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	// Find the "child:" line and verify it has exactly 8 spaces of indentation.
	lines := strings.Split(result, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "child:") {
			found = true
			trimmed := strings.TrimLeft(line, " ")
			indent := len(line) - len(trimmed)
			if indent != 8 {
				t.Errorf("expected child key at indent 8, got %d: %q", indent, line)
			}
		}
	}
	if !found {
		t.Fatalf("expected 'child:' in output, got:\n%s", result)
	}
}

func TestDetailedFormatter_MapContinuationKeyIndent(t *testing.T) {
	// Kills ARITHMETIC_BASE at detailed_formatter.go:239 (indent+2 → indent-2)
	// A map with 2+ keys: the continuation key is rendered via renderKeyValueYAML
	// at indent+2 (=6 spaces). With the mutation, it would be at 2 spaces.
	diffs := []Difference{
		{
			Path: "items.0",
			Type: DiffAdded,
			From: nil,
			To:   map[string]any{"aaa": "val1", "zzz": "val2"},
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	// The continuation key (no "- " prefix) should be at exactly 6 spaces (indent 4+2).
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if !strings.HasPrefix(trimmed, "-") && (strings.Contains(line, "aaa:") || strings.Contains(line, "zzz:")) {
			if indent != 6 {
				t.Errorf("expected continuation key at indent 6, got %d: %q", indent, line)
			}
		}
	}
}

// --- DetailedFormatter: multiline first key indent (detailed_formatter.go:303) ---

func TestDetailedFormatter_FirstKeyMultilineIndent(t *testing.T) {
	// Kills ARITHMETIC_BASE at detailed_formatter.go:303 (indent+2 → indent-2)
	// renderFirstKeyValueYAML calls renderMultilineValue with indent+2 (=6),
	// which adds indent+2 (=8) padding. With mutation (indent-2 = 2),
	// padding becomes 2+2=4 spaces instead of 8.
	om := &OrderedMap{
		Keys:   []string{"config"},
		Values: map[string]any{"config": "line1\nline2\nline3"},
	}
	diffs := []Difference{
		{
			Path: "items.0",
			Type: DiffAdded,
			From: nil,
			To:   om,
		},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	if !strings.Contains(result, "line1") {
		t.Fatalf("expected multiline content in output, got:\n%s", result)
	}
	// Assert exact indentation: continuation lines must have exactly 8 spaces.
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "line2") || strings.Contains(line, "line3") {
			trimmed := strings.TrimLeft(line, " ")
			indent := len(line) - len(trimmed)
			if indent != 8 {
				t.Errorf("expected 8 spaces for multiline continuation, got %d: %q", indent, line)
			}
		}
	}
}

// --- color.go:77,81 IsTerminal return value ---

func TestIsTerminal_ReturnValue(t *testing.T) {
	// Pipe/CI sanity check: stdout is a pipe, so IsTerminal returns false.
	// The actual mutant-killing tests for lines 77/81 are in color_test.go
	// (TestIsTerminal_WithCharDevice, TestIsTerminal_StatError) which use
	// the injectable stdoutStatFn to mock a real terminal environment.
	got := IsTerminal(os.Stdout.Fd())
	if got {
		t.Skip("running in a real terminal; cannot test pipe behavior")
	}
	if got != false {
		t.Error("IsTerminal should return false for pipe stdout")
	}
}

// === Section 3: Target NOT COVERED mutants ===

// --- computeLineDiff direct unit tests (detailed_formatter.go:464-483) ---

func TestComputeLineDiff_PartialMatch(t *testing.T) {
	from := []string{"a", "b", "c"}
	to := []string{"a", "x", "c"}
	ops := computeLineDiff(from, to)

	var keeps, deletes, inserts int
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			keeps++
		case editDelete:
			deletes++
		case editInsert:
			inserts++
		}
	}
	if keeps != 2 || deletes != 1 || inserts != 1 {
		t.Errorf("expected 2 keeps, 1 delete, 1 insert; got %d/%d/%d", keeps, deletes, inserts)
	}
}

func TestComputeLineDiff_AllDifferent(t *testing.T) {
	from := []string{"a", "b"}
	to := []string{"c", "d"}
	ops := computeLineDiff(from, to)

	var deletes, inserts int
	for _, op := range ops {
		switch op.Type {
		case editDelete:
			deletes++
		case editInsert:
			inserts++
		}
	}
	if deletes != 2 || inserts != 2 {
		t.Errorf("expected 2 deletes, 2 inserts; got %d/%d", deletes, inserts)
	}
}

func TestComputeLineDiff_EmptyFrom(t *testing.T) {
	ops := computeLineDiff([]string{}, []string{"a", "b"})
	inserts := 0
	for _, op := range ops {
		if op.Type == editInsert {
			inserts++
		}
	}
	if inserts != 2 {
		t.Errorf("expected 2 inserts, got %d", inserts)
	}
}

func TestComputeLineDiff_EmptyTo(t *testing.T) {
	ops := computeLineDiff([]string{"a", "b"}, []string{})
	deletes := 0
	for _, op := range ops {
		if op.Type == editDelete {
			deletes++
		}
	}
	if deletes != 2 {
		t.Errorf("expected 2 deletes, got %d", deletes)
	}
}

func TestComputeLineDiff_IdenticalLines(t *testing.T) {
	ops := computeLineDiff([]string{"a", "b", "c"}, []string{"a", "b", "c"})
	for _, op := range ops {
		if op.Type != editKeep {
			t.Errorf("expected all keeps for identical input, got %v", op.Type)
		}
	}
	if len(ops) != 3 {
		t.Errorf("expected 3 ops, got %d", len(ops))
	}
}

// --- compareListsPositional: different-length lists (comparator.go:353,361) ---

func TestCompareListsPositional_ToLonger(t *testing.T) {
	from := []any{"a", "b"}
	to := []any{"a", "b", "c", "d"}
	diffs := compareListsPositional("list", from, to, nil)

	added := 0
	for _, d := range diffs {
		if d.Type == DiffAdded {
			added++
		}
	}
	if added != 2 {
		t.Errorf("expected 2 added items, got %d", added)
	}
}

func TestCompareListsPositional_FromLonger(t *testing.T) {
	from := []any{"a", "b", "c"}
	to := []any{"a"}
	diffs := compareListsPositional("list", from, to, nil)

	removed := 0
	for _, d := range diffs {
		if d.Type == DiffRemoved {
			removed++
		}
	}
	if removed != 2 {
		t.Errorf("expected 2 removed items, got %d", removed)
	}
}

// --- remote.go: constant value assertions (remote.go:14,16) ---

func TestComputeLineDiff_MatchAtLastPosition(t *testing.T) {
	// Kills CONDITIONALS_BOUNDARY at detailed_formatter.go:490 (j <= n → j < n)
	// With j < n, dp[*][n] stays 0 — the LCS match at the last position of toLines is lost.
	from := []string{"a", "b"}
	to := []string{"c", "a"}
	ops := computeLineDiff(from, to)

	keeps := 0
	for _, op := range ops {
		if op.Type == editKeep {
			keeps++
		}
	}
	if keeps != 1 {
		t.Errorf("expected 1 keep (LCS match at last position), got %d", keeps)
	}
}

func TestDetectRenames_AsymmetricTiebreaker(t *testing.T) {
	// Kills sort tiebreaker mutations in rename.go (lines 165, 167-168)
	// With 3×2 identical ConfigMaps, all 6 pairs have the same score.
	// Mutations that invert the fromIdx tiebreaker change greedy assignment:
	// normal → {0:0, 1:1}, remaining=[2]; reversed → {2:0, 1:1}, remaining=[0]
	from := []any{
		mkK8sConfigMap("cm", []string{"key1", "key2", "key3"}),
		mkK8sConfigMap("cm", []string{"key1", "key2", "key3"}),
		mkK8sConfigMap("cm", []string{"key1", "key2", "key3"}),
	}
	to := []any{
		mkK8sConfigMap("cm", []string{"key1", "key2", "key3"}),
		mkK8sConfigMap("cm", []string{"key1", "key2", "key3"}),
	}

	opts := &Options{DetectRenames: true}
	matched, remainingFrom, _ := detectRenames(from, to, []int{0, 1, 2}, []int{0, 1}, opts)

	if len(matched) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matched))
	}
	if _, ok := matched[0]; !ok {
		t.Error("expected from[0] to be matched")
	}
	if len(remainingFrom) != 1 || remainingFrom[0] != 2 {
		t.Errorf("expected remaining=[2], got %v", remainingFrom)
	}
}

func TestHasK8sDocuments_OnlyToHasK8s(t *testing.T) {
	from := []any{map[string]any{"key": "value"}}
	to := []any{map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": map[string]any{"name": "svc"}}}
	if !hasK8sDocuments(from, to) {
		t.Error("expected true when only 'to' has K8s documents")
	}
}

func TestGetIdentifier_PlainMap(t *testing.T) {
	m := map[string]any{"name": "myapp", "value": "1"}
	id := getIdentifier(m, nil)
	if id != "myapp" {
		t.Errorf("expected 'myapp', got %v", id)
	}
}

func TestDeepEqual_BothNil(t *testing.T) {
	if !deepEqual(nil, nil, nil) {
		t.Error("deepEqual(nil, nil) should be true")
	}
}

func TestDeepEqual_TypeMismatch(t *testing.T) {
	if deepEqual("str", 42, nil) {
		t.Error("deepEqual with different types should be false")
	}
}

func TestCompareListsByIdentifier_NoIDMatchedSkip(t *testing.T) {
	// Two unidentified items in from that match toNoID items.
	// The second fromNoID item must iterate past the already-matched slot
	// to hit the toNoIDMatched continue branch (line 562).
	from := []any{
		&OrderedMap{
			Keys:   []string{"name", "v"},
			Values: map[string]any{"name": "x", "v": "1"},
		},
		"shared-a",
		"shared-b",
	}
	to := []any{
		&OrderedMap{
			Keys:   []string{"name", "v"},
			Values: map[string]any{"name": "x", "v": "1"},
		},
		"shared-a",
		"shared-b",
	}

	diffs := compareListsByIdentifier("items", from, to, nil)
	for _, d := range diffs {
		if d.Type == DiffRemoved || d.Type == DiffAdded {
			t.Errorf("unexpected diff: %+v", d)
		}
	}
}

func TestDetailedFormatter_RenderKeyValueYAML_ListIndent(t *testing.T) {
	// Kills ARITHMETIC_BASE at detailed_formatter_render.go:58 (indent+2 → other)
	// renderKeyValueYAML, case []any: list items are rendered at indent+2.
	// The OrderedMap first key "items" has a []any value. Since "items" is the
	// first key, it goes through renderFirstKeyValueYAML at indent=4 (base),
	// which calls renderListItems at indent+4=8. But we want to test the
	// renderKeyValueYAML []any branch (line 58), so "items" must be a
	// CONTINUATION key (not the first key).
	om := &OrderedMap{
		Keys:   []string{"name", "items"},
		Values: map[string]any{"name": "test", "items": []any{"val1", "val2"}},
	}

	diffs := []Difference{
		{Path: "spec.containers.0", Type: DiffAdded, From: nil, To: om},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	// "name" is first key → renderFirstKeyValueYAML at indent=4: "    - name: test"
	// "items" is continuation key → renderKeyValueYAML at indent=4+2=6: "      items:"
	// List items are rendered at indent+2=8 via renderListItems: "        - val1"
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "- val1") || strings.Contains(line, "- val2") {
			trimmed := strings.TrimLeft(line, " ")
			indent := len(line) - len(trimmed)
			if indent != 8 {
				t.Errorf("expected list items at indent 8, got %d: %q", indent, line)
			}
		}
	}
}

func TestDetailedFormatter_RenderFirstKeyValueYAML_ListIndent(t *testing.T) {
	// Kills ARITHMETIC_BASE at detailed_formatter_render.go:86 (indent+4 → other)
	// renderFirstKeyValueYAML, case []any: nested list items inside a list entry's
	// first key should be rendered at indent+4.
	om := &OrderedMap{
		Keys:   []string{"commands"},
		Values: map[string]any{"commands": []any{"cmd1", "cmd2"}},
	}

	diffs := []Difference{
		{Path: "spec.containers.0", Type: DiffAdded, From: nil, To: om},
	}

	f := &DetailedFormatter{}
	opts := &FormatOptions{Color: false}
	result := f.Format(diffs, opts)

	// "commands" is the first key → renderFirstKeyValueYAML at indent=4.
	// List items are rendered at indent+4=8 via renderListItems.
	// Each list item line: "        - cmd1" (8 spaces + "- cmd1")
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "- cmd1") || strings.Contains(line, "- cmd2") {
			trimmed := strings.TrimLeft(line, " ")
			indent := len(line) - len(trimmed)
			if indent != 8 {
				t.Errorf("expected list items at indent 8, got %d: %q", indent, line)
			}
		}
	}
}

func TestRemoteConstants(t *testing.T) {
	if MaxResponseSize != 10*1024*1024 {
		t.Errorf("MaxResponseSize should be 10485760, got %d", MaxResponseSize)
	}
	if DefaultTimeout != 30*time.Second {
		t.Errorf("DefaultTimeout should be 30s, got %v", DefaultTimeout)
	}
}
