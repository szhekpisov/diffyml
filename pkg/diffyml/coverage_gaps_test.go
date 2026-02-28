package diffyml

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tests targeting remaining coverage gaps identified by gremlins mutation testing.

// --- deepEqual: []interface{} slice case ---

func TestDeepEqual_Slices_Equal(t *testing.T) {
	a := []interface{}{"x", "y", "z"}
	b := []interface{}{"x", "y", "z"}
	if !deepEqual(a, b, nil) {
		t.Error("expected equal slices to be deepEqual")
	}
}

func TestDeepEqual_Slices_DifferentValues(t *testing.T) {
	a := []interface{}{"x", "y"}
	b := []interface{}{"x", "z"}
	if deepEqual(a, b, nil) {
		t.Error("expected slices with different values to not be deepEqual")
	}
}

func TestDeepEqual_Slices_DifferentLengths(t *testing.T) {
	a := []interface{}{"x"}
	b := []interface{}{"x", "y"}
	if deepEqual(a, b, nil) {
		t.Error("expected slices with different lengths to not be deepEqual")
	}
}

func TestDeepEqual_Slices_Nested(t *testing.T) {
	a := []interface{}{[]interface{}{"a", "b"}}
	b := []interface{}{[]interface{}{"a", "b"}}
	if !deepEqual(a, b, nil) {
		t.Error("expected nested equal slices to be deepEqual")
	}
}

// --- extractPathOrder: map[string]interface{} branch ---

func TestExtractPathOrder_PlainMap(t *testing.T) {
	docs := []interface{}{
		map[string]interface{}{
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
	docs := []interface{}{
		map[string]interface{}{
			"parent": map[string]interface{}{"child": "val"},
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

// --- areListItemsHeterogeneous: map[string]interface{} items ---

func TestAreListItemsHeterogeneous_PlainMaps(t *testing.T) {
	from := []interface{}{
		map[string]interface{}{"namespaceSelector": "ns1"},
	}
	to := []interface{}{
		map[string]interface{}{"ipBlock": "10.0.0.0/8"},
	}

	if !areListItemsHeterogeneous(from, to) {
		t.Error("expected heterogeneous for plain maps with different single keys")
	}
}

func TestAreListItemsHeterogeneous_PlainMapsMultipleKeys(t *testing.T) {
	from := []interface{}{
		map[string]interface{}{"a": "1", "b": "2"},
	}
	to := []interface{}{
		map[string]interface{}{"c": "3"},
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

// --- GetContextColorCode: true color path ---

func TestGetContextColorCode_TrueColor(t *testing.T) {
	code := GetContextColorCode(true)
	if !strings.HasPrefix(code, "\033[38;2;") {
		t.Errorf("expected true color ANSI prefix, got %q", code)
	}
}

func TestGetContextColorCode_Basic(t *testing.T) {
	code := GetContextColorCode(false)
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

// --- ExitResult.String(): nil error and unknown exit code ---

func TestExitResult_String_ErrorNilErr(t *testing.T) {
	result := NewExitResult(ExitCodeError, nil)
	got := result.String()
	if !strings.Contains(got, "unknown error") {
		t.Errorf("expected 'unknown error', got %q", got)
	}
}

func TestExitResult_String_UnknownCode(t *testing.T) {
	result := NewExitResult(99, nil)
	got := result.String()
	if !strings.Contains(got, "unknown exit code") || !strings.Contains(got, "99") {
		t.Errorf("expected 'unknown exit code: 99', got %q", got)
	}
}

// --- renderFirstKeyValueYAML: []interface{} value ---

func TestDetailedFormatter_ListValueInFirstKey(t *testing.T) {
	// The first key of a list entry maps to a list value,
	// exercising the []interface{} case in renderFirstKeyValueYAML.
	om := &OrderedMap{
		Keys:   []string{"ports", "protocol"},
		Values: map[string]interface{}{"ports": []interface{}{"80", "443"}, "protocol": "TCP"},
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
	from := []interface{}{
		&OrderedMap{
			Keys:   []string{"name", "value"},
			Values: map[string]interface{}{"name": "a", "value": "1"},
		},
		"scalar-from-only",
		"shared-scalar",
	}
	to := []interface{}{
		&OrderedMap{
			Keys:   []string{"name", "value"},
			Values: map[string]interface{}{"name": "a", "value": "2"},
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

// --- runDirectory: real filesystem paths ---

func TestRunDirectory_RealFilesystem(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	// Create test YAML files: one shared (modified), one only-from, one only-to
	writeFile(t, filepath.Join(fromDir, "common.yaml"), "key: old\n")
	writeFile(t, filepath.Join(toDir, "common.yaml"), "key: new\n")
	writeFile(t, filepath.Join(fromDir, "removed.yaml"), "gone: true\n")
	writeFile(t, filepath.Join(toDir, "added.yaml"), "fresh: true\n")

	cfg := &CLIConfig{Output: "compact"}
	var stdout, stderr bytes.Buffer
	rc := &RunConfig{Stdout: &stdout, Stderr: &stderr}

	result := runDirectory(cfg, rc, fromDir, toDir)

	if result.Code == ExitCodeError {
		t.Fatalf("runDirectory failed: %v\nstderr: %s", result.Err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "common.yaml") {
		t.Error("expected common.yaml in output")
	}
}

func TestRunDirectory_RealFilesystem_OnlyFromAndOnlyTo(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	writeFile(t, filepath.Join(fromDir, "deleted.yaml"), "old: data\n")
	writeFile(t, filepath.Join(toDir, "created.yaml"), "new: data\n")

	cfg := &CLIConfig{Output: "compact"}
	var stdout, stderr bytes.Buffer
	rc := &RunConfig{Stdout: &stdout, Stderr: &stderr}

	result := runDirectory(cfg, rc, fromDir, toDir)

	if result.Code == ExitCodeError {
		t.Fatalf("runDirectory failed: %v", result.Err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// --- GetTrueColorCode: exercises clamp through boundary values ---

func TestGetTrueColorCode_Clamped(t *testing.T) {
	// Values out of range should be clamped
	code := GetTrueColorCode(-1, 256, 128)
	expected := fmt.Sprintf("\033[38;2;%d;%d;%dm", 0, 255, 128)
	if code != expected {
		t.Errorf("expected clamped color code %q, got %q", expected, code)
	}
}

// === Section 2: Kill LIVED mutants ===

// --- extractPathOrder: index++ increment (diffyml.go:155) ---

func TestExtractPathOrder_PlainMapIndexIncrement(t *testing.T) {
	// Kills INCREMENT_DECREMENT at diffyml.go:155 (index++ → index--)
	// Uses nested maps so recursion enters the map[string]interface{} case at line 150,
	// where index++ (line 155) is executed for each parent path.
	// With the mutation (index--), all parent paths get the same order value (0),
	// so the strict ordering assertion catches it.
	docs := []interface{}{
		map[string]interface{}{
			"alpha": map[string]interface{}{"child1": "v1"},
			"beta":  map[string]interface{}{"child2": "v2"},
			"gamma": map[string]interface{}{"child3": "v3"},
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
	// The first key's value is a map[string]interface{}, so renderFirstKeyValueYAML
	// enters the map case (line 291) and renders children at indent+4 (=8 spaces).
	// With the mutation (indent-4), children would be at 0 spaces instead.
	diffs := []Difference{
		{
			Path: "items.0",
			Type: DiffAdded,
			From: nil,
			To:   map[string]interface{}{"aaa": map[string]interface{}{"child": "value"}},
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
			To:   map[string]interface{}{"aaa": "val1", "zzz": "val2"},
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
		Values: map[string]interface{}{"config": "line1\nline2\nline3"},
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

// --- directory.go:232 trueColor mode ---

func TestRunDirectory_TrueColorMode(t *testing.T) {
	// Kills CONDITIONALS_NEGATION at directory.go:232 (== ColorModeAlways → !=)
	cfg := &CLIConfig{
		Output:    "detailed",
		Color:     "always",
		TrueColor: "always",
	}
	var stdout, stderr bytes.Buffer
	rc := &RunConfig{
		Stdout: &stdout,
		Stderr: &stderr,
		FilePairs: map[string][2][]byte{
			"test.yaml": {[]byte("key: old"), []byte("key: new")},
		},
	}

	_ = runDirectory(cfg, rc, "", "")

	output := stdout.String()
	// True color uses \033[38;2;R;G;Bm format
	if !strings.Contains(output, "\033[38;2;") {
		t.Errorf("expected true color escape codes with TrueColor=always, got:\n%s", output)
	}
}

// --- directory.go:362 summary not called when no diffs ---

func TestRunDirectory_SummaryNotCalledWhenNoDiffs(t *testing.T) {
	// Kills CONDITIONALS_BOUNDARY at directory.go:362 (len(groups) > 0 → >= 0)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(500)
	}))
	defer server.Close()

	cfg := &CLIConfig{
		Output:  "github",
		Summary: true,
		Color:   "never",
	}
	var stdout, stderr bytes.Buffer
	rc := &RunConfig{
		Stdout: &stdout,
		Stderr: &stderr,
		FilePairs: map[string][2][]byte{
			"same.yaml": {[]byte("key: same"), []byte("key: same")},
		},
		SummaryAPIURL: server.URL,
	}

	_ = runDirectory(cfg, rc, "", "")

	if apiCalled {
		t.Error("summarizer should not be called when there are no diffs")
	}
}

// --- cli.go:638 brief+summary defers output ---

func TestRun_BriefSummary_DefersOutput(t *testing.T) {
	// Kills CONDITIONALS_NEGATION at cli.go:638 (== "brief" → != "brief")
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"AI summary of changes."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "brief"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte("key: old")
	rc.ToContent = []byte("key: new")
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	output := stdout.String()

	if result.Code == ExitCodeError {
		t.Fatalf("Run failed: %v\nstderr: %s", result.Err, stderr.String())
	}

	// When brief+summary succeeds, the AI summary replaces brief output
	if !strings.Contains(output, "AI summary of changes.") {
		t.Errorf("expected AI summary in output, got:\n%s", output)
	}

	// The brief diff output must NOT appear — it should be deferred and replaced.
	// With the mutation (== "brief" → != "brief"), isBriefSummary becomes false,
	// so the brief output ("1 modified") is printed alongside the AI summary.
	if strings.Contains(output, "modified") {
		t.Errorf("expected brief diff output to be absent when AI summary succeeds, got:\n%s", output)
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
	from := []interface{}{"a", "b"}
	to := []interface{}{"a", "b", "c", "d"}
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
	from := []interface{}{"a", "b", "c"}
	to := []interface{}{"a"}
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

// --- buildFilePairsFromMap: all pair types (directory.go:179,181) ---

func TestBuildFilePairsFromMap_AllTypes(t *testing.T) {
	m := map[string][2][]byte{
		"both.yaml":      {[]byte("a"), []byte("b")},
		"from-only.yaml": {[]byte("a"), nil},
		"to-only.yaml":   {nil, []byte("b")},
	}
	pairs := buildFilePairsFromMap(m)

	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	types := map[string]FilePairType{}
	for _, p := range pairs {
		types[p.Name] = p.Type
	}

	if types["both.yaml"] != FilePairBothExist {
		t.Error("both.yaml should be FilePairBothExist")
	}
	if types["from-only.yaml"] != FilePairOnlyFrom {
		t.Error("from-only.yaml should be FilePairOnlyFrom")
	}
	if types["to-only.yaml"] != FilePairOnlyTo {
		t.Error("to-only.yaml should be FilePairOnlyTo")
	}
}

// --- summarizer: status 502 (summarizer.go:150) ---

func TestSummarize_ServerError502(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 502,
		body:       `{"type":"error","error":{"type":"api_error","message":"bad gateway"}}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(t.Context(), groups)
	if err == nil {
		t.Fatal("expected error for 502")
	}
	if !strings.Contains(err.Error(), "server error") {
		t.Errorf("expected 'server error' for 502, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bad gateway") {
		t.Errorf("expected 'bad gateway' message for 502, got: %v", err)
	}
}

// --- remote.go: constant value assertions (remote.go:14,16) ---

func TestRemoteConstants(t *testing.T) {
	if MaxResponseSize != 10*1024*1024 {
		t.Errorf("MaxResponseSize should be 10485760, got %d", MaxResponseSize)
	}
	if DefaultTimeout != 30*time.Second {
		t.Errorf("DefaultTimeout should be 30s, got %v", DefaultTimeout)
	}
}
