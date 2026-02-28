package diffyml

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// --- mockHTTPDoer ---

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

// --- serializeValue tests ---

func TestSerializeValue_Nil(t *testing.T) {
	got := serializeValue(nil)
	if got != "<none>" {
		t.Errorf("serializeValue(nil) = %q, want %q", got, "<none>")
	}
}

func TestSerializeValue_String(t *testing.T) {
	got := serializeValue("hello")
	if got != "hello" {
		t.Errorf("serializeValue(\"hello\") = %q, want %q", got, "hello")
	}
}

func TestSerializeValue_Int(t *testing.T) {
	got := serializeValue(42)
	if got != "42" {
		t.Errorf("serializeValue(42) = %q, want %q", got, "42")
	}
}

func TestSerializeValue_Bool(t *testing.T) {
	got := serializeValue(true)
	if got != "true" {
		t.Errorf("serializeValue(true) = %q, want %q", got, "true")
	}
}

func TestSerializeValue_Float(t *testing.T) {
	got := serializeValue(3.14)
	if got != "3.14" {
		t.Errorf("serializeValue(3.14) = %q, want %q", got, "3.14")
	}
}

func TestSerializeValue_OrderedMap(t *testing.T) {
	om := NewOrderedMap()
	om.Keys = []string{"name", "port"}
	om.Values["name"] = "http"
	om.Values["port"] = 80

	got := serializeValue(om)
	if !strings.Contains(got, "name: http") || !strings.Contains(got, "port: 80") {
		t.Errorf("serializeValue(OrderedMap) = %q, want to contain name and port", got)
	}
}

func TestSerializeValue_Map(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	got := serializeValue(m)
	if !strings.Contains(got, "key: value") {
		t.Errorf("serializeValue(map) = %q, want to contain 'key: value'", got)
	}
}

func TestSerializeValue_Slice(t *testing.T) {
	s := []interface{}{"a", "b", "c"}
	got := serializeValue(s)
	if !strings.Contains(got, "- a") || !strings.Contains(got, "- b") {
		t.Errorf("serializeValue(slice) = %q, want to contain list items", got)
	}
}

// --- buildPrompt tests ---

func TestBuildPrompt_SingleFileAdded(t *testing.T) {
	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: "spec.replicas", Type: DiffAdded, From: nil, To: 3},
			},
		},
	}

	got := buildPrompt(groups)

	if !strings.Contains(got, "File: deploy.yaml") {
		t.Errorf("buildPrompt missing file header, got: %s", got)
	}
	if !strings.Contains(got, "[ADDED]") {
		t.Errorf("buildPrompt missing [ADDED] label, got: %s", got)
	}
	if !strings.Contains(got, "spec.replicas") {
		t.Errorf("buildPrompt missing path, got: %s", got)
	}
	if !strings.Contains(got, "<none>") {
		t.Errorf("buildPrompt missing <none> for nil From, got: %s", got)
	}
}

func TestBuildPrompt_AllDiffTypes(t *testing.T) {
	groups := []DiffGroup{
		{
			FilePath: "test.yaml",
			Diffs: []Difference{
				{Path: "a", Type: DiffAdded, From: nil, To: "new"},
				{Path: "b", Type: DiffRemoved, From: "old", To: nil},
				{Path: "c", Type: DiffModified, From: "v1", To: "v2"},
				{Path: "d", Type: DiffOrderChanged, From: nil, To: nil},
			},
		},
	}

	got := buildPrompt(groups)
	for _, label := range []string{"[ADDED]", "[REMOVED]", "[MODIFIED]", "[ORDER_CHANGED]"} {
		if !strings.Contains(got, label) {
			t.Errorf("buildPrompt missing %s label, got: %s", label, got)
		}
	}
}

func TestBuildPrompt_MultipleFiles(t *testing.T) {
	groups := []DiffGroup{
		{
			FilePath: "file1.yaml",
			Diffs:    []Difference{{Path: "a", Type: DiffAdded, To: "x"}},
		},
		{
			FilePath: "file2.yaml",
			Diffs:    []Difference{{Path: "b", Type: DiffRemoved, From: "y"}},
		},
	}

	got := buildPrompt(groups)
	if !strings.Contains(got, "File: file1.yaml") || !strings.Contains(got, "File: file2.yaml") {
		t.Errorf("buildPrompt missing multiple file headers, got: %s", got)
	}
}

func TestBuildPrompt_Truncation(t *testing.T) {
	// Create enough diffs to exceed ~8000 chars
	var diffs []Difference
	for i := 0; i < 500; i++ {
		diffs = append(diffs, Difference{
			Path: strings.Repeat("very.long.path.segment.", 5) + "key",
			Type: DiffModified,
			From: strings.Repeat("old-value-", 10),
			To:   strings.Repeat("new-value-", 10),
		})
	}

	groups := []DiffGroup{
		{FilePath: "file1.yaml", Diffs: diffs[:250]},
		{FilePath: "file2.yaml", Diffs: diffs[250:]},
	}

	got := buildPrompt(groups)
	if !strings.Contains(got, "truncated") {
		t.Errorf("buildPrompt should truncate large input, got length: %d", len(got))
	}
}

// --- systemPrompt tests ---

func TestSystemPrompt_NotEmpty(t *testing.T) {
	got := systemPrompt()
	if got == "" {
		t.Error("systemPrompt() should not be empty")
	}
	if !strings.Contains(got, "YAML") {
		t.Error("systemPrompt() should mention YAML")
	}
}

// --- formatSummaryOutput tests ---

func TestFormatSummaryOutput_NoColor(t *testing.T) {
	opts := &FormatOptions{Color: false}
	got := formatSummaryOutput("Test summary text.", opts)

	if !strings.Contains(got, "AI Summary:") {
		t.Errorf("formatSummaryOutput missing header, got: %s", got)
	}
	if !strings.Contains(got, "Test summary text.") {
		t.Errorf("formatSummaryOutput missing body, got: %s", got)
	}
	if !strings.HasPrefix(got, "\n") {
		t.Errorf("formatSummaryOutput should start with blank line, got: %s", got)
	}
}

func TestFormatSummaryOutput_WithColor(t *testing.T) {
	opts := &FormatOptions{Color: true}
	got := formatSummaryOutput("Test summary.", opts)

	if !strings.Contains(got, colorCyan) {
		t.Errorf("formatSummaryOutput with color should use cyan, got: %s", got)
	}
	if !strings.Contains(got, styleBold) {
		t.Errorf("formatSummaryOutput with color should use bold, got: %s", got)
	}
	if !strings.Contains(got, colorReset) {
		t.Errorf("formatSummaryOutput with color should reset, got: %s", got)
	}
}

// --- Summarizer tests ---

func TestNewSummarizer_DefaultModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	s := NewSummarizer("")
	if s.model != defaultModel {
		t.Errorf("NewSummarizer(\"\").model = %q, want %q", s.model, defaultModel)
	}
}

func TestNewSummarizer_CustomModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	s := NewSummarizer("claude-sonnet-4-20250514")
	if s.model != "claude-sonnet-4-20250514" {
		t.Errorf("NewSummarizer custom model = %q, want %q", s.model, "claude-sonnet-4-20250514")
	}
}

func TestSummarize_Success(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 200,
		body:       `{"content":[{"type":"text","text":"The replicas were increased from 3 to 5."}]}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []Difference{
				{Path: "spec.replicas", Type: DiffModified, From: 3, To: 5},
			},
		},
	}

	summary, err := s.Summarize(context.Background(), groups)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if summary != "The replicas were increased from 3 to 5." {
		t.Errorf("Summarize() = %q, want expected summary", summary)
	}

	// Verify request headers
	if mock.lastReq.Header.Get("x-api-key") != "test-key" {
		t.Error("request missing x-api-key header")
	}
	if mock.lastReq.Header.Get("anthropic-version") != anthropicVersion {
		t.Error("request missing anthropic-version header")
	}
	if mock.lastReq.Header.Get("Content-Type") != "application/json" {
		t.Error("request missing Content-Type header")
	}
}

func TestSummarize_NetworkError(t *testing.T) {
	mock := &mockHTTPDoer{
		err: io.ErrUnexpectedEOF,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for network failure")
	}
}

func TestSummarize_Auth401(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 401,
		body:       `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for 401")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("error should mention 'invalid API key', got: %v", err)
	}
}

func TestSummarize_RateLimit429(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 429,
		body:       `{"type":"error","error":{"type":"rate_limit_error","message":"rate limited"}}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for 429")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error should mention 'rate limited', got: %v", err)
	}
}

func TestSummarize_ServerError500(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 500,
		body:       `{"type":"error","error":{"type":"api_error","message":"internal error"}}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for 500")
	}
	if !strings.Contains(err.Error(), "server error") {
		t.Errorf("error should mention 'server error', got: %v", err)
	}
}

func TestSummarize_MalformedResponse(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 200,
		body:       `{"content":[]}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for malformed response")
	}
	if !strings.Contains(err.Error(), "unexpected response format") {
		t.Errorf("error should mention 'unexpected response format', got: %v", err)
	}
}

func TestSummarize_EmptyTextBlock(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 200,
		body:       `{"content":[{"type":"text","text":""}]}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for empty text")
	}
}

func TestSummarize_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	mock := &mockHTTPDoer{
		statusCode: 200,
		body:       `{"content":[{"type":"text","text":"ok"}]}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(ctx, groups)
	if err == nil {
		t.Fatal("Summarize() expected error for cancelled context")
	}
}

func TestSummarize_APIErrorInBody(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 400,
		body:       `{"type":"error","error":{"type":"invalid_request_error","message":"model not found"}}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for 400")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("error should contain API error message, got: %v", err)
	}
}

func TestSummarize_NoTextContentBlock(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 200,
		body:       `{"content":[{"type":"tool_use","text":"irrelevant"}]}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error when no text block found")
	}
}

// --- Mutation testing: summarizer.go ---

func TestNewSummarizerWithClient_CustomModelPreserved(t *testing.T) {
	// summarizer.go:60 — custom model should be preserved, not replaced by default
	mock := &mockHTTPDoer{statusCode: 200, body: `{"content":[{"type":"text","text":"ok"}]}`}
	s := NewSummarizerWithClient("my-custom-model", "test-key", mock)
	if s.model != "my-custom-model" {
		t.Errorf("NewSummarizerWithClient model = %q, want %q", s.model, "my-custom-model")
	}
}

func TestNewSummarizerWithClient_EmptyModelDefault(t *testing.T) {
	// summarizer.go:60 — empty model should be replaced by default
	mock := &mockHTTPDoer{statusCode: 200, body: `{"content":[{"type":"text","text":"ok"}]}`}
	s := NewSummarizerWithClient("", "test-key", mock)
	if s.model != defaultModel {
		t.Errorf("NewSummarizerWithClient empty model = %q, want %q", s.model, defaultModel)
	}
}

func TestSummarize_ServerError500_IncludesMessage(t *testing.T) {
	// summarizer.go:152 — error message from API body should be included in error
	mock := &mockHTTPDoer{
		statusCode: 500,
		body:       `{"type":"error","error":{"type":"api_error","message":"internal error"}}`,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err == nil {
		t.Fatal("Summarize() expected error for 500")
	}
	// Must include the specific message, not just "server error"
	if !strings.Contains(err.Error(), "internal error") {
		t.Errorf("error should contain 'internal error' from API body, got: %v", err)
	}
}

func TestBuildPrompt_SingleOversizedGroup(t *testing.T) {
	// summarizer.go:215 — single oversized group should still be included
	// (groupsWritten > 0 condition means first group is never truncated)
	var diffs []Difference
	for i := 0; i < 500; i++ {
		diffs = append(diffs, Difference{
			Path: strings.Repeat("very.long.path.", 10) + "key",
			Type: DiffModified,
			From: strings.Repeat("old-value-", 20),
			To:   strings.Repeat("new-value-", 20),
		})
	}

	groups := []DiffGroup{
		{FilePath: "single-big.yaml", Diffs: diffs},
	}

	got := buildPrompt(groups)

	// The single group must be included even if it exceeds maxPromptLen
	if !strings.Contains(got, "File: single-big.yaml") {
		t.Error("single oversized group should still be included in prompt")
	}
	// Should NOT contain truncation message since it's the only group
	if strings.Contains(got, "truncated") {
		t.Error("single oversized group should not trigger truncation")
	}
}

func TestBuildPrompt_TruncationRemainingCount(t *testing.T) {
	// summarizer.go:220 — remaining file count must be correct
	var bigDiffs []Difference
	for i := 0; i < 200; i++ {
		bigDiffs = append(bigDiffs, Difference{
			Path: strings.Repeat("path.", 10) + "key",
			Type: DiffModified,
			From: strings.Repeat("a", 50),
			To:   strings.Repeat("b", 50),
		})
	}

	groups := []DiffGroup{
		{FilePath: "file1.yaml", Diffs: bigDiffs},
		{FilePath: "file2.yaml", Diffs: []Difference{{Path: "x", Type: DiffAdded, To: "v"}}},
		{FilePath: "file3.yaml", Diffs: []Difference{{Path: "y", Type: DiffAdded, To: "w"}}},
		{FilePath: "file4.yaml", Diffs: []Difference{{Path: "z", Type: DiffAdded, To: "u"}}},
	}

	got := buildPrompt(groups)

	// file1 should be included (first group always is)
	if !strings.Contains(got, "File: file1.yaml") {
		t.Error("first group should always be included")
	}

	// If truncated, remaining count should be correct (3 files with correct diff counts)
	if strings.Contains(got, "truncated") {
		// Remaining should be 3 files (file2, file3, file4) with 3 total changes
		if !strings.Contains(got, "3 more files") {
			t.Errorf("truncation message should say '3 more files', got: %s", got)
		}
		if !strings.Contains(got, "3 more changes") {
			t.Errorf("truncation message should say '3 more changes', got: %s", got)
		}
	}
}

func TestBuildPrompt_ExactBoundary(t *testing.T) {
	// summarizer.go:215 — `> maxPromptLen` → `>= maxPromptLen`
	// If mutated, a prompt that totals exactly maxPromptLen would be truncated.
	// We construct groups that sum to exactly maxPromptLen to detect this.

	// First, build a single group and measure its serialized length
	singleDiff := Difference{
		Path: "test.path",
		Type: DiffModified,
		From: "oldval",
		To:   "newval",
	}
	// Measure the size of a single-diff group
	testGroup := DiffGroup{
		FilePath: "test.yaml",
		Diffs:    []Difference{singleDiff},
	}
	singlePrompt := buildPrompt([]DiffGroup{testGroup})
	singleLen := len(singlePrompt)

	// Now create groups that total exactly maxPromptLen.
	// We need group1 + group2 == maxPromptLen.
	// Build group1 to have a known size, then pad group2 to reach exactly maxPromptLen.
	remaining := maxPromptLen - singleLen
	if remaining <= 0 {
		t.Skip("single group already exceeds maxPromptLen")
	}

	// Build a second group whose serialized output is exactly `remaining` bytes
	// We'll use a path long enough to fill the remaining space.
	// Format: "File: X.yaml\n- [MODIFIED] PATH: \"old\" → \"new\"\n\n"
	// Header: "File: pad.yaml\n" = 15 bytes
	// Diff line: "- [MODIFIED] " + path + ": \"a\" → \"b\"\n"
	// Trailing: "\n"
	overhead := len("File: pad.yaml\n") + len("- [MODIFIED] ") + len(": \"a\" \xe2\x86\x92 \"b\"\n") + len("\n")
	pathLen := remaining - overhead
	if pathLen <= 0 {
		t.Skip("can't construct exact boundary test")
	}

	padPath := strings.Repeat("x", pathLen)
	group2 := DiffGroup{
		FilePath: "pad.yaml",
		Diffs: []Difference{
			{Path: padPath, Type: DiffModified, From: "a", To: "b"},
		},
	}

	groups := []DiffGroup{testGroup, group2}
	got := buildPrompt(groups)

	// Both groups should be included (not truncated) since total == maxPromptLen
	if !strings.Contains(got, "File: test.yaml") {
		t.Error("first group should be present")
	}
	if !strings.Contains(got, "File: pad.yaml") {
		t.Error("second group should be present when total == maxPromptLen")
	}
	if strings.Contains(got, "truncated") {
		t.Error("should not truncate when total equals exactly maxPromptLen")
	}
}

func TestBuildPrompt_OneByteOverBoundary(t *testing.T) {
	// Companion test: verify that one byte over the boundary DOES truncate.
	singleDiff := Difference{
		Path: "test.path",
		Type: DiffModified,
		From: "oldval",
		To:   "newval",
	}
	testGroup := DiffGroup{
		FilePath: "test.yaml",
		Diffs:    []Difference{singleDiff},
	}
	singlePrompt := buildPrompt([]DiffGroup{testGroup})
	singleLen := len(singlePrompt)

	remaining := maxPromptLen - singleLen + 1 // one byte over
	if remaining <= 0 {
		t.Skip("single group already exceeds maxPromptLen")
	}

	overhead := len("File: pad.yaml\n") + len("- [MODIFIED] ") + len(": \"a\" \xe2\x86\x92 \"b\"\n") + len("\n")
	pathLen := remaining - overhead
	if pathLen <= 0 {
		t.Skip("can't construct boundary+1 test")
	}

	padPath := strings.Repeat("x", pathLen)
	group2 := DiffGroup{
		FilePath: "pad.yaml",
		Diffs: []Difference{
			{Path: padPath, Type: DiffModified, From: "a", To: "b"},
		},
	}

	groups := []DiffGroup{testGroup, group2}
	got := buildPrompt(groups)

	// Second group should be truncated since total > maxPromptLen
	if !strings.Contains(got, "File: test.yaml") {
		t.Error("first group should always be present")
	}
	if strings.Contains(got, "File: pad.yaml") {
		// This is expected - second group should be truncated
	}
	if !strings.Contains(got, "truncated") {
		t.Error("should truncate when total exceeds maxPromptLen by 1")
	}
}

func TestSummarize_APIKeyNotInError(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 401,
		body:       `{"type":"error","error":{"type":"authentication_error","message":"invalid"}}`,
	}
	s := NewSummarizerWithClient("test-model", "secret-api-key-12345", mock)

	groups := []DiffGroup{
		{FilePath: "f.yaml", Diffs: []Difference{{Path: "a", Type: DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err != nil && strings.Contains(err.Error(), "secret-api-key-12345") {
		t.Error("API key should never appear in error messages")
	}
}
