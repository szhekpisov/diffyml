package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

// --- buildPrompt tests ---

func TestBuildPrompt_SingleFileAdded(t *testing.T) {
	groups := []diffyml.DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []diffyml.Difference{
				{Path: diffyml.DiffPath{"spec", "replicas"}, Type: diffyml.DiffAdded, From: nil, To: 3},
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
	groups := []diffyml.DiffGroup{
		{
			FilePath: "test.yaml",
			Diffs: []diffyml.Difference{
				{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, From: nil, To: "new"},
				{Path: diffyml.DiffPath{"b"}, Type: diffyml.DiffRemoved, From: "old", To: nil},
				{Path: diffyml.DiffPath{"c"}, Type: diffyml.DiffModified, From: "v1", To: "v2"},
				{Path: diffyml.DiffPath{"d"}, Type: diffyml.DiffOrderChanged, From: nil, To: nil},
				{Path: diffyml.DiffPath{"e"}, Type: diffyml.DiffUnchanged, From: "same", To: "same"},
			},
		},
	}

	got := buildPrompt(groups)
	for _, label := range []string{"[ADDED]", "[REMOVED]", "[MODIFIED]", "[ORDER_CHANGED]", "[UNCHANGED]"} {
		if !strings.Contains(got, label) {
			t.Errorf("buildPrompt missing %s label, got: %s", label, got)
		}
	}
}

func TestBuildPrompt_MultipleFiles(t *testing.T) {
	groups := []diffyml.DiffGroup{
		{
			FilePath: "file1.yaml",
			Diffs:    []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "x"}},
		},
		{
			FilePath: "file2.yaml",
			Diffs:    []diffyml.Difference{{Path: diffyml.DiffPath{"b"}, Type: diffyml.DiffRemoved, From: "y"}},
		},
	}

	got := buildPrompt(groups)
	if !strings.Contains(got, "File: file1.yaml") || !strings.Contains(got, "File: file2.yaml") {
		t.Errorf("buildPrompt missing multiple file headers, got: %s", got)
	}
}

func TestBuildPrompt_Truncation(t *testing.T) {
	// Create enough diffs to exceed ~8000 chars
	var diffs []diffyml.Difference
	for i := 0; i < 500; i++ {
		diffs = append(diffs, diffyml.Difference{
			Path: diffyml.DiffPath(strings.Split(strings.Repeat("very.long.path.segment.", 5)+"key", ".")),
			Type: diffyml.DiffModified,
			From: strings.Repeat("old-value-", 10),
			To:   strings.Repeat("new-value-", 10),
		})
	}

	groups := []diffyml.DiffGroup{
		{FilePath: "file1.yaml", Diffs: diffs[:250]},
		{FilePath: "file2.yaml", Diffs: diffs[250:]},
	}

	got := buildPrompt(groups)
	if !strings.Contains(got, "truncated") {
		t.Errorf("buildPrompt should truncate large input, got length: %d", len(got))
	}
	if len(got) > maxPromptLen {
		t.Errorf("buildPrompt length = %d, want <= %d", len(got), maxPromptLen)
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

	groups := []diffyml.DiffGroup{
		{
			FilePath: "deploy.yaml",
			Diffs: []diffyml.Difference{
				{Path: diffyml.DiffPath{"spec", "replicas"}, Type: diffyml.DiffModified, From: 3, To: 5},
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
		err: context.DeadlineExceeded,
	}
	s := NewSummarizerWithClient("test-model", "test-key", mock)

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
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
	var diffs []diffyml.Difference
	for i := 0; i < 500; i++ {
		diffs = append(diffs, diffyml.Difference{
			Path: diffyml.DiffPath(strings.Split(strings.Repeat("very.long.path.", 10)+"key", ".")),
			Type: diffyml.DiffModified,
			From: strings.Repeat("old-value-", 20),
			To:   strings.Repeat("new-value-", 20),
		})
	}

	groups := []diffyml.DiffGroup{
		{FilePath: "single-big.yaml", Diffs: diffs},
	}

	got := buildPrompt(groups)

	if !strings.Contains(got, "File: single-big.yaml") {
		t.Error("single oversized group should retain its file header")
	}
	if !strings.Contains(got, "truncated") {
		t.Error("single oversized group should be truncated")
	}
	if len(got) > maxPromptLen {
		t.Errorf("buildPrompt length = %d, want <= %d", len(got), maxPromptLen)
	}
	if !utf8.ValidString(got) {
		t.Error("truncated prompt must remain valid UTF-8")
	}
}

func TestBuildPrompt_TruncationRemainingCount(t *testing.T) {
	// summarizer.go:220 — remaining file count must be correct
	var bigDiffs []diffyml.Difference
	for i := 0; i < 200; i++ {
		bigDiffs = append(bigDiffs, diffyml.Difference{
			Path: diffyml.DiffPath(strings.Split(strings.Repeat("path.", 10)+"key", ".")),
			Type: diffyml.DiffModified,
			From: strings.Repeat("a", 50),
			To:   strings.Repeat("b", 50),
		})
	}

	groups := []diffyml.DiffGroup{
		{FilePath: "file1.yaml", Diffs: bigDiffs},
		{FilePath: "file2.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"x"}, Type: diffyml.DiffAdded, To: "v"}}},
		{FilePath: "file3.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"y"}, Type: diffyml.DiffAdded, To: "w"}}},
		{FilePath: "file4.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"z"}, Type: diffyml.DiffAdded, To: "u"}}},
	}

	got := buildPrompt(groups)

	// The first file should be represented before truncation.
	if !strings.Contains(got, "File: file1.yaml") {
		t.Error("first group should be present")
	}

	if !strings.Contains(got, "truncated") {
		t.Fatal("expected prompt to be truncated")
	}
	writtenChanges := strings.Count(got, "\n- [")
	remainingChanges := len(bigDiffs) + 3 - writtenChanges
	expectedMarker := fmt.Sprintf("%d more changes across 4 files", remainingChanges)
	if !strings.Contains(got, expectedMarker) {
		t.Errorf("truncation marker should contain %q, got: %s", expectedMarker, got)
	}
	if len(got) > maxPromptLen {
		t.Errorf("buildPrompt length = %d, want <= %d", len(got), maxPromptLen)
	}
}

func TestRemainingPromptItems_SkipsEmptyAndConsumedGroups(t *testing.T) {
	groups := []diffyml.DiffGroup{
		{FilePath: "empty.yaml"},
		{
			FilePath: "changes.yaml",
			Diffs: []diffyml.Difference{
				{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: 1},
				{Path: diffyml.DiffPath{"b"}, Type: diffyml.DiffAdded, To: 2},
			},
		},
	}

	changes, files := remainingPromptItems(groups, 0, 0)
	if changes != 2 || files != 1 {
		t.Errorf("remainingPromptItems() = (%d, %d), want (2, 1)", changes, files)
	}

	changes, files = remainingPromptItems(groups, 1, len(groups[1].Diffs))
	if changes != 0 || files != 0 {
		t.Errorf("consumed group = (%d, %d), want (0, 0)", changes, files)
	}
}

func TestBuildPrompt_HeaderExceedsLimit(t *testing.T) {
	groups := []diffyml.DiffGroup{{
		FilePath: strings.Repeat("x", maxPromptLen),
		Diffs:    []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: 1}},
	}}

	got := buildPrompt(groups)
	if len(got) > maxPromptLen {
		t.Errorf("buildPrompt length = %d, want <= %d", len(got), maxPromptLen)
	}
	if strings.Contains(got, "File: ") {
		t.Error("oversized file header should not be written")
	}
	if !strings.Contains(got, "1 more change across 1 file (truncated)") {
		t.Errorf("expected exact truncation marker, got %q", got)
	}
}

func TestBuildTruncatedPrompt_InputFits(t *testing.T) {
	groups := []diffyml.DiffGroup{{
		FilePath: "small.yaml",
		Diffs:    []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: 1}},
	}}

	got := buildTruncatedPrompt(groups)
	want := buildPrompt(groups)
	if got != want {
		t.Errorf("buildTruncatedPrompt() = %q, want %q", got, want)
	}
}

func TestBuildPrompt_ExactBoundary(t *testing.T) {
	// summarizer.go:215 — `> maxPromptLen` → `>= maxPromptLen`
	// If mutated, a prompt that totals exactly maxPromptLen would be truncated.
	// We construct groups that sum to exactly maxPromptLen to detect this.

	// First, build a single group and measure its serialized length
	singleDiff := diffyml.Difference{
		Path: diffyml.DiffPath{"test", "path"},
		Type: diffyml.DiffModified,
		From: "oldval",
		To:   "newval",
	}
	// Measure the size of a single-diff group
	testGroup := diffyml.DiffGroup{
		FilePath: "test.yaml",
		Diffs:    []diffyml.Difference{singleDiff},
	}
	singlePrompt := buildPrompt([]diffyml.DiffGroup{testGroup})
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
	group2 := diffyml.DiffGroup{
		FilePath: "pad.yaml",
		Diffs: []diffyml.Difference{
			{Path: diffyml.DiffPath{padPath}, Type: diffyml.DiffModified, From: "a", To: "b"},
		},
	}

	groups := []diffyml.DiffGroup{testGroup, group2}
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
	// Companion test: verify that one byte of content over the boundary DOES
	// truncate. The serializer may omit the optional final blank line, so add
	// two bytes to the complete representation to put the diff line itself one
	// byte over the limit.
	singleDiff := diffyml.Difference{
		Path: diffyml.DiffPath{"test", "path"},
		Type: diffyml.DiffModified,
		From: "oldval",
		To:   "newval",
	}
	testGroup := diffyml.DiffGroup{
		FilePath: "test.yaml",
		Diffs:    []diffyml.Difference{singleDiff},
	}
	singlePrompt := buildPrompt([]diffyml.DiffGroup{testGroup})
	singleLen := len(singlePrompt)

	remaining := maxPromptLen - singleLen + 2
	if remaining <= 0 {
		t.Skip("single group already exceeds maxPromptLen")
	}

	overhead := len("File: pad.yaml\n") + len("- [MODIFIED] ") + len(": \"a\" \xe2\x86\x92 \"b\"\n") + len("\n")
	pathLen := remaining - overhead
	if pathLen <= 0 {
		t.Skip("can't construct boundary+1 test")
	}

	padPath := strings.Repeat("x", pathLen)
	group2 := diffyml.DiffGroup{
		FilePath: "pad.yaml",
		Diffs: []diffyml.Difference{
			{Path: diffyml.DiffPath{padPath}, Type: diffyml.DiffModified, From: "a", To: "b"},
		},
	}

	groups := []diffyml.DiffGroup{testGroup, group2}
	got := buildPrompt(groups)

	if !strings.Contains(got, "File: test.yaml") {
		t.Error("first group should be present")
	}
	if !strings.Contains(got, "truncated") {
		t.Error("should truncate when total exceeds maxPromptLen by 1")
	}
	if len(got) > maxPromptLen {
		t.Errorf("buildPrompt length = %d, want <= %d", len(got), maxPromptLen)
	}
}

func TestSummarize_APIKeyNotInError(t *testing.T) {
	mock := &mockHTTPDoer{
		statusCode: 401,
		body:       `{"type":"error","error":{"type":"authentication_error","message":"invalid"}}`,
	}
	s := NewSummarizerWithClient("test-model", "secret-api-key-12345", mock)

	groups := []diffyml.DiffGroup{
		{FilePath: "f.yaml", Diffs: []diffyml.Difference{{Path: diffyml.DiffPath{"a"}, Type: diffyml.DiffAdded, To: "v"}}},
	}

	_, err := s.Summarize(context.Background(), groups)
	if err != nil && strings.Contains(err.Error(), "secret-api-key-12345") {
		t.Error("API key should never appear in error messages")
	}
}
