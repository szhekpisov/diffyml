package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

// mockHTTPDoer implements httpDoer for testing.
type mockHTTPDoer struct {
	statusCode int
	body       string
	err        error
	lastReq    *http.Request
}

func (m *mockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.body)),
	}, nil
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

// --- runDirectory: real filesystem paths ---

func TestRunDirectory_RealFilesystem(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	// Create test YAML files: one shared (modified), one only-from, one only-to
	writeTestFile(t, filepath.Join(fromDir, "common.yaml"), "key: old\n")
	writeTestFile(t, filepath.Join(toDir, "common.yaml"), "key: new\n")
	writeTestFile(t, filepath.Join(fromDir, "removed.yaml"), "gone: true\n")
	writeTestFile(t, filepath.Join(toDir, "added.yaml"), "fresh: true\n")

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

	writeTestFile(t, filepath.Join(fromDir, "deleted.yaml"), "old: data\n")
	writeTestFile(t, filepath.Join(toDir, "created.yaml"), "new: data\n")

	cfg := &CLIConfig{Output: "compact"}
	var stdout, stderr bytes.Buffer
	rc := &RunConfig{Stdout: &stdout, Stderr: &stderr}

	result := runDirectory(cfg, rc, fromDir, toDir)

	if result.Code == ExitCodeError {
		t.Fatalf("runDirectory failed: %v", result.Err)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
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

	types := map[string]diffyml.FilePairType{}
	for _, p := range pairs {
		types[p.Name] = p.Type
	}

	if types["both.yaml"] != diffyml.FilePairBothExist {
		t.Error("both.yaml should be FilePairBothExist")
	}
	if types["from-only.yaml"] != diffyml.FilePairOnlyFrom {
		t.Error("from-only.yaml should be FilePairOnlyFrom")
	}
	if types["to-only.yaml"] != diffyml.FilePairOnlyTo {
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

// --- buildFilePairsFromMap: one-sided file pairs ---

func TestBuildFilePairsFromMap_OnlyFrom(t *testing.T) {
	m := map[string][2][]byte{
		"deleted.yaml": {[]byte("content"), nil},
	}
	pairs := buildFilePairsFromMap(m)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Type != diffyml.FilePairOnlyFrom {
		t.Errorf("expected FilePairOnlyFrom, got %v", pairs[0].Type)
	}
}

func TestBuildFilePairsFromMap_OnlyTo(t *testing.T) {
	m := map[string][2][]byte{
		"added.yaml": {nil, []byte("content")},
	}
	pairs := buildFilePairsFromMap(m)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Type != diffyml.FilePairOnlyTo {
		t.Errorf("expected FilePairOnlyTo, got %v", pairs[0].Type)
	}
}

func TestBuildFilePairsFromMap_Mixed(t *testing.T) {
	m := map[string][2][]byte{
		"both.yaml":    {[]byte("a"), []byte("b")},
		"added.yaml":   {nil, []byte("new")},
		"deleted.yaml": {[]byte("old"), nil},
	}
	pairs := buildFilePairsFromMap(m)
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	// Sorted alphabetically: added, both, deleted
	types := map[string]diffyml.FilePairType{}
	for _, p := range pairs {
		types[p.Name] = p.Type
	}

	if types["added.yaml"] != diffyml.FilePairOnlyTo {
		t.Errorf("added.yaml: expected FilePairOnlyTo, got %v", types["added.yaml"])
	}
	if types["both.yaml"] != diffyml.FilePairBothExist {
		t.Errorf("both.yaml: expected FilePairBothExist, got %v", types["both.yaml"])
	}
	if types["deleted.yaml"] != diffyml.FilePairOnlyFrom {
		t.Errorf("deleted.yaml: expected FilePairOnlyFrom, got %v", types["deleted.yaml"])
	}
}

// === Section 2: Code coverage gap tests ===

// --- cli.go: loadContents ToFile error ---

func TestLoadContents_ToFileError(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.FromFile = "/dev/null" // valid file that loads empty
	cfg.ToFile = "/nonexistent/path/to/file.yaml"
	rc := &RunConfig{Stdout: &strings.Builder{}, Stderr: &strings.Builder{}}
	_, _, err := loadContents(cfg, rc)
	if err == nil {
		t.Fatal("expected error loading non-existent ToFile")
	}
}

// --- cli.go: Run with nil RunConfig ---

func TestRun_NilRunConfig(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.ShowHelp = true
	// Run with rc=nil should not panic; it creates a default RunConfig
	result := Run(cfg, nil)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected success for help flag, got %d", result.Code)
	}
}

// --- cli.go: Run with invalid output format ---

func TestRun_InvalidOutputFormat_SetupError(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "nonexistent_format"
	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout:      &stdout,
		Stderr:      &stderr,
		FromContent: []byte("key: old"),
		ToContent:   []byte("key: new"),
	}
	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected error for invalid output format, got %d", result.Code)
	}
}

// --- directory.go: loadFilePairContent error cases ---

func TestLoadFilePairContent_ErrorCases(t *testing.T) {
	tests := []struct {
		name string
		pair diffyml.FilePair
	}{
		{
			name: "BothExist_FromError",
			pair: diffyml.FilePair{
				Name:     "test.yaml",
				Type:     diffyml.FilePairBothExist,
				FromPath: "/nonexistent/from.yaml",
				ToPath:   "/dev/null",
			},
		},
		{
			name: "BothExist_ToError",
			pair: diffyml.FilePair{
				Name:     "test.yaml",
				Type:     diffyml.FilePairBothExist,
				FromPath: "/dev/null",
				ToPath:   "/nonexistent/to.yaml",
			},
		},
		{
			name: "OnlyFrom_Error",
			pair: diffyml.FilePair{
				Name:     "test.yaml",
				Type:     diffyml.FilePairOnlyFrom,
				FromPath: "/nonexistent/from.yaml",
			},
		},
		{
			name: "OnlyTo_Error",
			pair: diffyml.FilePair{
				Name:   "test.yaml",
				Type:   diffyml.FilePairOnlyTo,
				ToPath: "/nonexistent/to.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := loadFilePairContent(tt.pair, nil)
			if err == nil {
				t.Fatal("expected error for non-existent file path")
			}
		})
	}
}

// --- directory.go: processDirPair errors ---

func TestProcessDirPair_LoadError(t *testing.T) {
	pair := diffyml.FilePair{
		Name:     "bad.yaml",
		Type:     diffyml.FilePairBothExist,
		FromPath: "/nonexistent/from.yaml",
		ToPath:   "/nonexistent/to.yaml",
	}
	_, err := processDirPair(pair, nil, &diffyml.Options{}, &diffyml.FilterOptions{})
	if err == nil {
		t.Fatal("expected error for non-existent file in processDirPair")
	}
}

func TestProcessDirPair_CompareError(t *testing.T) {
	// Invalid YAML in the filePairs map triggers a compare error
	filePairs := map[string][2][]byte{
		"bad.yaml": {[]byte("{{invalid yaml"), []byte("key: val")},
	}
	pair := diffyml.FilePair{
		Name: "bad.yaml",
		Type: diffyml.FilePairBothExist,
	}
	_, err := processDirPair(pair, filePairs, &diffyml.Options{}, &diffyml.FilterOptions{})
	if err == nil {
		t.Fatal("expected error for invalid YAML in processDirPair")
	}
}

// --- directory.go: setupDirFormatting invalid format ---

func TestSetupDirFormatting_InvalidFormat(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "nonexistent_formatter"
	_, _, err := setupDirFormatting(cfg)
	if err == nil {
		t.Fatal("expected error for invalid formatter name")
	}
}

// --- directory.go: runDirectory errors ---

func TestRunDirectory_BuildPairsError(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "compact"
	var stdout, stderr strings.Builder
	rc := &RunConfig{Stdout: &stdout, Stderr: &stderr}
	// Non-existent directories → buildDirFilePairs error
	result := runDirectory(cfg, rc, "/nonexistent/dir1", "/nonexistent/dir2")
	if result.Code != ExitCodeError {
		t.Errorf("expected error for non-existent directories, got %d", result.Code)
	}
}

func TestRunDirectory_SetupFormattingError(t *testing.T) {
	cfg := NewCLIConfig()
	cfg.Output = "nonexistent_format"
	var stdout, stderr strings.Builder
	rc := &RunConfig{
		Stdout: &stdout,
		Stderr: &stderr,
		FilePairs: map[string][2][]byte{
			"test.yaml": {[]byte("key: old"), []byte("key: new")},
		},
	}
	result := runDirectory(cfg, rc, "", "")
	if result.Code != ExitCodeError {
		t.Errorf("expected error for invalid output format, got %d", result.Code)
	}
}

// --- directory.go: emitDirectorySummary structured error ---

func TestEmitDirectorySummary_StructuredError(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"server down"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "github"
	var stdout, stderr strings.Builder
	rc := &RunConfig{Stdout: &stdout, Stderr: &stderr, SummaryAPIURL: server.URL}
	formatOpts := &diffyml.FormatOptions{}
	formatter, _ := diffyml.FormatterByName("github")

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
	}

	emitDirectorySummary(cfg, rc, groups, nil, formatOpts, formatter, true, false)

	if !strings.Contains(stderr.String(), "AI summary unavailable") {
		t.Errorf("expected warning in stderr, got: %s", stderr.String())
	}
}

// --- summarizer.go: Summarize invalid response body ---

func TestSummarize_InvalidResponseBody(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 200,
		body:       "not json at all",
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(t.Context(), groups)
	if err == nil {
		t.Fatal("expected error for non-JSON response body")
	}
	if !strings.Contains(err.Error(), "unexpected response format") {
		t.Errorf("expected 'unexpected response format' error, got: %v", err)
	}
}

// --- summarizer.go: Summarize invalid URL ---

func TestSummarize_InvalidURL(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	s := NewSummarizer("test-model")
	// URL with control character causes http.NewRequestWithContext to fail
	s.apiURL = "http://\x00invalid"

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(t.Context(), groups)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// --- summarizer.go: diffTypeLabel default case ---

func TestDiffTypeLabel_Unknown(t *testing.T) {
	label := diffTypeLabel(diffyml.DiffType(99))
	if label != "UNKNOWN" {
		t.Errorf("expected UNKNOWN for unknown DiffType, got %q", label)
	}
}
