package diffyml

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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
			Path: DiffPath{"spec", "containers", "0"},
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

	diffs := compareListsByIdentifier(DiffPath{"items"}, from, to, nil)

	// "a" matched by name → modified value (1 → 2)
	// "shared-scalar" matched by deepEqual in fallback → no diff
	// "scalar-from-only" and "new-scalar" both lack identifiers, unmatched by deepEqual
	// → compared positionally producing a modification
	var modified int
	for _, d := range diffs {
		if d.Type == DiffModified {
			modified++
		}
	}

	// Expect 2 modifications: "a".value (1→2) and scalar-from-only→new-scalar
	if modified != 2 {
		t.Errorf("expected 2 modified diffs, got %d; diffs: %v", modified, diffs)
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
			Path: DiffPath{"items", "0"},
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
			Path: DiffPath{"items", "0"},
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
			Path: DiffPath{"items", "0"},
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

// --- computeLineDiff direct unit tests (detailed_formatter_linediff.go) ---

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
	diffs := compareListsPositional(DiffPath{"list"}, from, to, nil)

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
	diffs := compareListsPositional(DiffPath{"list"}, from, to, nil)

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
	// Verifies that a match at the last position of toLines is found.
	// The algorithm must consider all positions to find the optimal edit script.
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
		t.Errorf("expected 1 keep (match at last position), got %d", keeps)
	}

	// Verify the edit ops correctly reconstruct both from and to.
	// This kills the backtrack mutant at line 155:45 (< → <=) which produces
	// an incorrect edit script when prev[k-1+offset] == prev[k+1+offset].
	var reconstructedFrom, reconstructedTo []string
	for _, op := range ops {
		switch op.Type {
		case editKeep:
			reconstructedFrom = append(reconstructedFrom, op.Line)
			reconstructedTo = append(reconstructedTo, op.Line)
		case editDelete:
			reconstructedFrom = append(reconstructedFrom, op.Line)
		case editInsert:
			reconstructedTo = append(reconstructedTo, op.Line)
		}
	}
	if strings.Join(reconstructedFrom, ",") != strings.Join(from, ",") {
		t.Errorf("reconstructed from %v != original %v", reconstructedFrom, from)
	}
	if strings.Join(reconstructedTo, ",") != strings.Join(to, ",") {
		t.Errorf("reconstructed to %v != original %v", reconstructedTo, to)
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

	diffs := compareListsByIdentifier(DiffPath{"items"}, from, to, nil)
	for _, d := range diffs {
		if d.Type == DiffRemoved || d.Type == DiffAdded {
			t.Errorf("unexpected diff: %+v", d)
		}
	}
}

func TestCompareListsByIdentifier_NoIDExcessAdded(t *testing.T) {
	// to has more unidentified items than from, exercising the excess-added loop
	// in compareUnidentifiedItems.
	from := []any{
		&OrderedMap{
			Keys:   []string{"name", "v"},
			Values: map[string]any{"name": "x", "v": "1"},
		},
		"only-in-from",
	}
	to := []any{
		&OrderedMap{
			Keys:   []string{"name", "v"},
			Values: map[string]any{"name": "x", "v": "1"},
		},
		"new-a",
		"new-b",
	}

	diffs := compareListsByIdentifier(DiffPath{"items"}, from, to, nil)

	// "only-in-from" vs "new-a" → positional modification
	// "new-b" has no counterpart → added
	var modified, added int
	for _, d := range diffs {
		switch d.Type {
		case DiffModified:
			modified++
		case DiffAdded:
			added++
		}
	}
	if modified != 1 {
		t.Errorf("expected 1 modified diff, got %d; diffs: %v", modified, diffs)
	}
	if added != 1 {
		t.Errorf("expected 1 added diff, got %d; diffs: %v", added, diffs)
	}
}

func TestCompareUnidentifiedItems_CursorSkipMatchedTo(t *testing.T) {
	// Exercises the toNoIDMatched skip branch in the cursor loop:
	// "shared" exact-matches, so the cursor must skip it in to before pairing
	// "only-from" with "only-to".
	from := []any{
		&OrderedMap{
			Keys:   []string{"name"},
			Values: map[string]any{"name": "x"},
		},
		"only-from",
		"shared",
	}
	to := []any{
		&OrderedMap{
			Keys:   []string{"name"},
			Values: map[string]any{"name": "x"},
		},
		"shared",
		"only-to",
	}

	diffs := compareListsByIdentifier(DiffPath{"items"}, from, to, nil)

	// "shared" matches exactly. Remaining: "only-from" vs "only-to" → modification.
	var modified int
	for _, d := range diffs {
		if d.Type == DiffModified {
			modified++
		}
	}
	if modified != 1 {
		t.Errorf("expected 1 modified diff, got %d; diffs: %v", modified, diffs)
	}
}

func TestCompareUnidentifiedItems_ExcessFromWithMatchedSkip(t *testing.T) {
	// Exercises the fromNoIDMatched skip in the excess-from tail loop:
	// from has more unidentified items than to, and a matched item ("shared")
	// appears between unmatched items in the from-side cursor walk.
	from := []any{
		&OrderedMap{
			Keys:   []string{"name"},
			Values: map[string]any{"name": "x"},
		},
		"removed-a",
		"shared",
		"removed-b",
	}
	to := []any{
		&OrderedMap{
			Keys:   []string{"name"},
			Values: map[string]any{"name": "x"},
		},
		"shared",
	}

	diffs := compareListsByIdentifier(DiffPath{"items"}, from, to, nil)

	// "shared" matches exactly. Remaining from: ["removed-a", "removed-b"] vs to: [].
	// Both are excess → 2 modifications? No — no to items left, so "removed-a" has
	// nothing to pair with, and "removed-b" has nothing either. But wait — the cursor
	// loop pairs positionally: no unmatched to items, so both go to excess-from.
	// Actually: cursor loop finds "removed-a" in from, no unmatched to → exits loop.
	// Excess-from: "removed-a" (removed), skip "shared" (matched), "removed-b" (removed).
	var removed int
	for _, d := range diffs {
		if d.Type == DiffRemoved {
			removed++
		}
	}
	if removed != 2 {
		t.Errorf("expected 2 removed diffs, got %d; diffs: %v", removed, diffs)
	}
}

func TestAreListItemsHeterogeneous_SingleKeyHomogeneous(t *testing.T) {
	// Kills CONDITIONALS_BOUNDARY at comparator.go areListItemsHeterogeneous:
	// len(allKeys) > 1 mutated to >= 1.
	// Single-key items with the SAME key are homogeneous → positional comparison.
	// The mutation would wrongly classify them as heterogeneous → unordered comparison.
	// With unordered: {a:1} exact-matches {a:1} in to, leaving {a:2} vs {a:3} → 1 diff.
	// With positional: {a:1} vs {a:1} → 0 diffs, {a:2} vs {a:3} → 1 diff → 1 diff total.
	// But {a:1} vs {a:3} and {a:2} vs {a:1} → 2 diffs with positional when order differs.
	from := `---
items:
  - a: "1"
  - a: "2"
`
	to := `---
items:
  - a: "2"
  - a: "3"
`
	diffs, err := Compare([]byte(from), []byte(to), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Positional: items.0.a (1→2) + items.1.a (2→3) = 2 modifications
	// Unordered would produce: items.0.a (1→3) = 1 modification (a:2 matches exactly)
	if len(diffs) != 2 {
		t.Errorf("expected 2 diffs (positional comparison), got %d: %v", len(diffs), diffs)
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
		{Path: DiffPath{"spec", "containers", "0"}, Type: DiffAdded, From: nil, To: om},
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
		{Path: DiffPath{"spec", "containers", "0"}, Type: DiffAdded, From: nil, To: om},
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

// === Section 4: Code coverage gap tests ===

// --- chroot.go: parsePath and splitPath edge cases ---

func TestParsePath_EmptyPath(t *testing.T) {
	segments, err := parsePath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 0 {
		t.Errorf("expected 0 segments for empty path, got %d", len(segments))
	}
}

func TestParsePath_ConsecutiveDots(t *testing.T) {
	// "a..b" has an empty part between the two dots that should be skipped
	segments, err := parsePath("a..b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}
	if segments[0].key != "a" || segments[1].key != "b" {
		t.Errorf("expected [a, b], got [%s, %s]", segments[0].key, segments[1].key)
	}
}

func TestSplitPath_NestedBrackets(t *testing.T) {
	_, err := splitPath("a[[0]]")
	if err == nil {
		t.Fatal("expected error for nested brackets")
	}
}

func TestSplitPath_UnmatchedClosingBracket(t *testing.T) {
	_, err := splitPath("a]b")
	if err == nil {
		t.Fatal("expected error for unmatched closing bracket")
	}
}

// --- chroot.go: applyChrootToDocs empty path ---

func TestApplyChrootToDocs_EmptyPath(t *testing.T) {
	docs := []any{map[string]any{"key": "value"}}
	result, err := applyChrootToDocs(docs, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(result))
	}
}

// --- comparator.go: detectListOrderChanges with non-unique IDs ---

func TestDetectListOrderChanges_NonUniqueIDs(t *testing.T) {
	// Duplicate IDs → fromIDs length != fromIndex length → return nil
	fromIDs := []any{"a", "a", "b"}
	fromIndex := map[any]int{"a": 0, "b": 2} // len=2, but fromIDs len=3
	toIndex := map[any]int{"a": 0, "b": 1}
	result := detectListOrderChanges(DiffPath{"items"}, fromIDs, fromIndex, toIndex, 2)
	if result != nil {
		t.Error("expected nil for non-unique IDs")
	}
}

// --- diffyml.go: Compare error paths ---

func TestCompare_InvalidFromYAML(t *testing.T) {
	_, err := Compare([]byte("{{invalid"), []byte("key: val"), nil)
	if err == nil {
		t.Fatal("expected parse error for invalid 'from' YAML")
	}
}

func TestCompare_InvalidToYAML(t *testing.T) {
	_, err := Compare([]byte("key: val"), []byte("{{invalid"), nil)
	if err == nil {
		t.Fatal("expected parse error for invalid 'to' YAML")
	}
}

func TestCompare_ChrootError(t *testing.T) {
	// Chroot with a path that doesn't exist in the from doc → error
	from := []byte("key: val")
	to := []byte("key: val")
	opts := &Options{Chroot: "nonexistent.deep.path"}
	_, err := Compare(from, to, opts)
	if err == nil {
		t.Fatal("expected chroot error for missing path in 'from'")
	}
}

func TestCompare_ChrootToErrorWithChroot(t *testing.T) {
	// Chroot path exists in 'from' but not in 'to'
	from := []byte("nonexistent:\n  deep:\n    path: val")
	to := []byte("other: val")
	opts := &Options{Chroot: "nonexistent.deep.path"}
	_, err := Compare(from, to, opts)
	if err == nil {
		t.Fatal("expected chroot error for missing path in 'to'")
	}
}

func TestCompare_ChrootFromError(t *testing.T) {
	// ChrootFrom with non-existent path → error (else branch, from)
	from := []byte("key: val")
	to := []byte("key: val")
	opts := &Options{ChrootFrom: "nonexistent.path"}
	_, err := Compare(from, to, opts)
	if err == nil {
		t.Fatal("expected chroot error for ChrootFrom with missing path")
	}
}

func TestCompare_ChrootToError(t *testing.T) {
	// ChrootTo with non-existent path → error (else branch, to)
	from := []byte("key: val")
	to := []byte("key: val")
	opts := &Options{ChrootTo: "nonexistent.path"}
	_, err := Compare(from, to, opts)
	if err == nil {
		t.Fatal("expected chroot error for ChrootTo with missing path")
	}
}

// --- diffyml.go: hasIdentifierField branches ---

func TestHasIdentifierField_OrderedMapWithID(t *testing.T) {
	om := &OrderedMap{
		Keys:   []string{"id", "value"},
		Values: map[string]any{"id": "123", "value": "x"},
	}
	if !hasIdentifierField(om) {
		t.Error("expected true for OrderedMap with 'id' field")
	}
}

func TestHasIdentifierField_PlainMapWithName(t *testing.T) {
	m := map[string]any{"name": "myapp", "value": "1"}
	if !hasIdentifierField(m) {
		t.Error("expected true for plain map with 'name' field")
	}
}

func TestHasIdentifierField_PlainMapWithID(t *testing.T) {
	m := map[string]any{"id": "123", "value": "x"}
	if !hasIdentifierField(m) {
		t.Error("expected true for plain map with 'id' field")
	}
}

// --- diffyml.go: compareByExactOrParentOrder !okI && okJ branch ---

func TestCompareByExactOrParentOrder_OnlyJInOrder(t *testing.T) {
	pathOrder := map[string]int{
		"known": 0,
	}
	// pathI is not in pathOrder, pathJ is → should return 1 (!okI && okJ)
	result := compareByExactOrParentOrder(DiffPath{"unknown"}, DiffPath{"known"}, pathOrder, func(path DiffPath) (int, bool) {
		return 0, false
	})
	if result != 1 {
		t.Errorf("expected 1 when only J is in order, got %d", result)
	}
}

// --- kubernetes.go: detectK8sOrderChanges same order → nil ---

// --- chroot.go: parsePath additional edge cases ---

func TestParsePath_InvalidBracketSyntax(t *testing.T) {
	// Multiple brackets like "key[0][1]" triggers invalid bracket syntax error
	_, err := parsePath("key[0][1]")
	if err == nil {
		t.Fatal("expected error for invalid bracket syntax")
	}
}

func TestParsePath_EmptyListIndex(t *testing.T) {
	// "key[]" has empty index string → error
	_, err := parsePath("key[]")
	if err == nil {
		t.Fatal("expected error for empty list index")
	}
}

// --- chroot.go: navigateToPath index on non-list ---

func TestNavigateToPath_IndexOnNonList(t *testing.T) {
	// Navigate with index accessor on a string value → error
	doc := &OrderedMap{
		Keys:   []string{"key"},
		Values: map[string]any{"key": "not-a-list"},
	}
	_, err := navigateToPath(doc, "key[0]")
	if err == nil {
		t.Fatal("expected error when indexing a non-list value")
	}
}

func TestDetectK8sOrderChanges_SameOrder(t *testing.T) {
	// Two matched docs in the same order → orderChanged is false → return nil
	matched := map[int]int{0: 0, 1: 1}
	from := []any{
		&OrderedMap{
			Keys:   []string{"apiVersion", "kind", "metadata"},
			Values: map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": &OrderedMap{Keys: []string{"name"}, Values: map[string]any{"name": "svc1"}}},
		},
		&OrderedMap{
			Keys:   []string{"apiVersion", "kind", "metadata"},
			Values: map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": &OrderedMap{Keys: []string{"name"}, Values: map[string]any{"name": "svc2"}}},
		},
	}
	result := detectK8sOrderChanges(matched, from, false)
	if result != nil {
		t.Error("expected nil when docs are in same order")
	}
}

// --- sameScalarType: default fallback for rare types ---

func TestSameScalarType_DefaultFallback(t *testing.T) {
	// time.Time is produced by yaml.v3 decoder for !!timestamp
	a := time.Now()
	b := time.Now()
	if !sameScalarType(a, b) {
		t.Error("expected same type for two time.Time values")
	}
	if sameScalarType(a, "string") {
		t.Error("expected different type for time.Time vs string")
	}
}

// --- deepEqual: type mismatch branches ---

func TestDeepEqual_TypeMismatch_OrderedMapVsString(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = append(om.Keys, "k")
	om.Values["k"] = "v"
	if deepEqual(om, "not-a-map", nil) {
		t.Error("expected false for *OrderedMap vs string")
	}
}

func TestDeepEqual_TypeMismatch_MapVsString(t *testing.T) {
	m := map[string]any{"k": "v"}
	if deepEqual(m, "not-a-map", nil) {
		t.Error("expected false for map vs string")
	}
}

func TestDeepEqual_TypeMismatch_SliceVsString(t *testing.T) {
	s := []any{"a", "b"}
	if deepEqual(s, "not-a-slice", nil) {
		t.Error("expected false for slice vs string")
	}
}

func TestDeepEqual_TypeMismatch_ScalarTypes(t *testing.T) {
	if deepEqual("hello", 42, nil) {
		t.Error("expected false for string vs int")
	}
	if deepEqual(3.14, true, nil) {
		t.Error("expected false for float64 vs bool")
	}
}

// --- resolveScalar: YAML float special values ---

func TestResolveScalar_SpecialFloats(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"positive infinity", ".inf"},
		{"negative infinity", "-.inf"},
		{"not a number", ".nan"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: tt.value}
			got := resolveScalar(node)
			if got == nil {
				t.Errorf("resolveScalar(%q) returned nil", tt.value)
			}
		})
	}
}

// --- resolveScalar: bool and null edge cases ---

func TestResolveScalar_BoolTrue(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
	got := resolveScalar(node)
	if got != true {
		t.Errorf("expected true, got %v", got)
	}
}

func TestResolveScalar_BoolFalse(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
	got := resolveScalar(node)
	if got != false {
		t.Errorf("expected false, got %v (%T)", got, got)
	}
}

func TestResolveScalar_NullVariants(t *testing.T) {
	// !!null always returns nil regardless of value
	for _, v := range []string{"", "null", "~", "Null", "NULL", "anything"} {
		node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: v}
		got := resolveScalar(node)
		if got != nil {
			t.Errorf("resolveScalar(tag=!!null, value=%q) = %v, want nil", v, got)
		}
	}
}

func TestResolveScalar_IntAndFloat(t *testing.T) {
	intNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "42"}
	got := resolveScalar(intNode)
	if got != 42 {
		t.Errorf("expected 42, got %v", got)
	}

	floatNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "3.14"}
	gotF := resolveScalar(floatNode)
	if gotF != 3.14 {
		t.Errorf("expected 3.14, got %v", gotF)
	}
}

// --- Mutation-killing tests for resolveScalar fast paths ---
// These verify the fast path returns the same type as the yaml.v3 decoder
// to kill mutants that negate err == nil conditions (swapping fast/slow path).

func TestResolveScalar_IntFastPathType(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "99"}
	got := resolveScalar(node)
	// The fast path must return int, not string
	if _, ok := got.(int); !ok {
		t.Errorf("expected int type from fast path, got %T: %v", got, got)
	}
}

func TestResolveScalar_FloatFastPathType(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "2.718"}
	got := resolveScalar(node)
	// The fast path must return float64, not string
	if _, ok := got.(float64); !ok {
		t.Errorf("expected float64 type from fast path, got %T: %v", got, got)
	}
}

// Test that pre-sized OrderedMap capacity matches expected key count.
// Kills ARITHMETIC_BASE mutant at ordered_map.go:79 (/2 → *2).
func TestNodeToInterface_MappingPresize(t *testing.T) {
	// 6 content nodes = 3 key-value pairs → capacity should be 3
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a"},
			{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "b"},
			{Kind: yaml.ScalarNode, Tag: "!!int", Value: "2"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "c"},
			{Kind: yaml.ScalarNode, Tag: "!!int", Value: "3"},
		},
	}
	result := nodeToInterface(node)
	om, ok := result.(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap, got %T", result)
	}
	if len(om.Keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(om.Keys))
	}
	// Verify capacity was correctly pre-sized to len(Content)/2.
	// A mutant changing /2 to *2 would set capacity to 6, which we detect here.
	if cap(om.Keys) != 3 {
		t.Errorf("expected capacity 3, got %d (pre-sizing may be wrong)", cap(om.Keys))
	}
}

// Kill CONDITIONALS_NEGATION mutants for int/float fast paths.
// If the mutant negates err==nil to err!=nil, invalid values would return
// zero-values (0 or 0.0) instead of falling through to the decoder.
func TestResolveScalar_InvalidIntFallsThrough(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "not_a_number"}
	got := resolveScalar(node)
	// With negated mutant: Atoi("not_a_number") fails, err!=nil is true,
	// so it returns i=0. We verify it does NOT return 0.
	if got == 0 || got == int(0) {
		t.Errorf("invalid !!int should not return 0, got %v (%T)", got, got)
	}
}

func TestResolveScalar_InvalidFloatFallsThrough(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "not_a_float"}
	got := resolveScalar(node)
	// With negated mutant: ParseFloat fails, err!=nil is true,
	// so it returns f=0.0. We verify it does NOT return 0.0.
	if got == float64(0) {
		t.Errorf("invalid !!float should not return 0.0, got %v (%T)", got, got)
	}
}

// --- sprintIdentifier coverage ---

func TestSprintIdentifier_String(t *testing.T) {
	if got := sprintIdentifier("hello"); got != "hello" {
		t.Errorf("sprintIdentifier(string) = %q, want %q", got, "hello")
	}
}

func TestSprintIdentifier_Int(t *testing.T) {
	if got := sprintIdentifier(42); got != "42" {
		t.Errorf("sprintIdentifier(int) = %q, want %q", got, "42")
	}
}

func TestSprintIdentifier_OtherType(t *testing.T) {
	if got := sprintIdentifier(3.14); got != "3.14" {
		t.Errorf("sprintIdentifier(float64) = %q, want %q", got, "3.14")
	}
}

// --- isSimpleDecimal coverage ---

func TestIsSimpleDecimal(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"-", false},
		{"+", false},
		{"42", true},
		{"-7", true},
		{"+7", true},
		{"0", true},
		{"0x1F", false},
		{"1_000", false},
		{"12345678901234567890", true}, // too large for int but still simple decimal
	}
	for _, tt := range tests {
		if got := isSimpleDecimal(tt.input); got != tt.want {
			t.Errorf("isSimpleDecimal(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- pathWalker.push coverage ---

func TestPathWalkerPush_DottedSegment(t *testing.T) {
	w := pathWalker{
		buf:     make([]byte, 0, 64),
		lengths: make([]int, 0, 4),
	}
	w.push("helm.sh/chart")
	if got := string(w.buf); got != "[helm.sh/chart]" {
		t.Errorf("push dotted = %q, want %q", got, "[helm.sh/chart]")
	}
}

func TestPathWalkerPush_BracketSegment(t *testing.T) {
	w := pathWalker{
		buf:     make([]byte, 0, 64),
		lengths: make([]int, 0, 4),
	}
	w.push("[0]")
	if got := string(w.buf); got != "[0]" {
		t.Errorf("push bracket = %q, want %q", got, "[0]")
	}
	w.push("metadata")
	if got := string(w.buf); got != "[0].metadata" {
		t.Errorf("push after bracket = %q, want %q", got, "[0].metadata")
	}
}

func TestPathWalkerPush_PopRoundtrip(t *testing.T) {
	w := pathWalker{
		buf:     make([]byte, 0, 64),
		lengths: make([]int, 0, 4),
	}
	w.push("root")
	w.push("child")
	if got := string(w.buf); got != "root.child" {
		t.Errorf("after two pushes = %q, want %q", got, "root.child")
	}
	w.pop()
	if got := string(w.buf); got != "root" {
		t.Errorf("after pop = %q, want %q", got, "root")
	}
	w.pop()
	if len(w.buf) != 0 {
		t.Errorf("after two pops, buf should be empty, got %q", string(w.buf))
	}
}

// --- resolveScalar with isSimpleDecimal guard ---

func TestResolveScalar_HexIntFallsToDecoder(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "0xFF"}
	got := resolveScalar(node)
	if got != 255 {
		t.Errorf("resolveScalar(0xFF) = %v (%T), want 255", got, got)
	}
}

func TestResolveScalar_OctalIntFallsToDecoder(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "0o755"}
	got := resolveScalar(node)
	if got != 493 {
		t.Errorf("resolveScalar(0o755) = %v (%T), want 493", got, got)
	}
}

func TestResolveScalar_SimpleDecimalInt(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "42"}
	got := resolveScalar(node)
	if got != 42 {
		t.Errorf("resolveScalar(42) = %v (%T), want 42", got, got)
	}
}

func TestResolveScalar_NegativeInt(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "-10"}
	got := resolveScalar(node)
	if got != -10 {
		t.Errorf("resolveScalar(-10) = %v (%T), want -10", got, got)
	}
}

func TestResolveScalar_SimpleDecimalReturnsInt(t *testing.T) {
	// Verify the fast-path returns Go int (not int64 from yaml decoder fallback).
	// This kills the CONDITIONALS_NEGATION mutant on the Atoi err==nil check.
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "100"}
	got := resolveScalar(node)
	if _, ok := got.(int); !ok {
		t.Errorf("resolveScalar(100) type = %T, want int", got)
	}
}

func TestResolveScalar_OverflowDecimalFallsToDecoder(t *testing.T) {
	// A number that passes isSimpleDecimal but overflows strconv.Atoi.
	// Kills the CONDITIONALS_NEGATION mutant: with err != nil the mutant
	// would return Atoi's saturated value (math.MaxInt64) instead of
	// falling through to the yaml decoder which returns the raw string.
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "99999999999999999999"}
	got := resolveScalar(node)
	// The yaml decoder returns the raw string for values that overflow all integer types.
	// The mutant would return math.MaxInt64 (strconv.Atoi's saturated error value).
	if _, isStr := got.(string); !isStr {
		t.Errorf("overflow decimal should return string, got %v (%T)", got, got)
	}
}

// --- sortDiffsWithOrder single-element boundary ---

func TestSortDiffsWithOrder_SingleElement(t *testing.T) {
	diffs := []Difference{{
		Path: DiffPath{"root", "key"},
		Type: DiffModified,
		From: "old",
		To:   "new",
	}}
	pathOrder := map[string]int{"root.key": 0}
	sortDiffsWithOrder(diffs, pathOrder)
	if len(diffs) != 1 || diffs[0].Path.String() != "root.key" {
		t.Errorf("single-element sort should be identity, got %v", diffs)
	}
}

// --- compareByExactOrParentOrderCached: !okI && okJ branch ---

func TestCompareByExactOrParentOrderCached_OnlyJInOrder(t *testing.T) {
	pathOrder := map[string]int{"known": 0}
	result := compareByExactOrParentOrderCached(
		"unknown", "known",
		DiffPath{"unknown"}, DiffPath{"known"},
		pathOrder,
		func(path DiffPath) (int, bool) { return 0, false },
	)
	if result != 1 {
		t.Errorf("expected 1 when only J is in pathOrder, got %d", result)
	}
}

// --- pathWalker.push: empty segment branch ---

func TestPathWalkerPush_EmptySegment(t *testing.T) {
	w := pathWalker{
		buf:     make([]byte, 0, 64),
		lengths: make([]int, 0, 4),
	}
	w.push("root")
	w.push("")
	// Empty segment gets a dot prefix (len(buf)>0 && len(seg)==0 triggers second case)
	if got := string(w.buf); got != "root." {
		t.Errorf("push empty segment = %q, want %q", got, "root.")
	}
}
