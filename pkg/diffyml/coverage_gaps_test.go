package diffyml

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
