package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Task 3.1: Wire summarizer into single-file comparison mode ---

func TestRun_WithSummary_AppendsSummaryToOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	// Start a mock Anthropic API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("expected x-api-key header")
		}

		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"The key value was changed from value1 to value2."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should contain standard diff output
	if !strings.Contains(output, "key") {
		t.Error("expected standard diff output containing 'key'")
	}
	// Should contain AI summary header and text
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected 'AI Summary:' header in output, got: %s", output)
	}
	if !strings.Contains(output, "value was changed") {
		t.Errorf("expected summary text in output, got: %s", output)
	}
}

func TestRun_WithSummary_NoDiffs_NoAPICall(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml := "key: value\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if apiCalled {
		t.Error("API should not be called when there are no differences")
	}
}

func TestRun_WithSummary_APIFailure_WarningOnStderr(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"internal error"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)

	// Exit code should not be affected by summary failure
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code %d, got %d", ExitCodeSuccess, result.Code)
	}
	// Standard diff output should still be present
	if !strings.Contains(stdout.String(), "key") {
		t.Error("expected standard diff output despite API failure")
	}
	// Warning on stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
	// No AI Summary in stdout
	if strings.Contains(stdout.String(), "AI Summary:") {
		t.Error("expected no AI Summary header on API failure")
	}
}

func TestRun_WithSummary_PreservesExitCode(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = true
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	// Exit code 1 should be preserved even with summary
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --set-exit-code, got %d", ExitCodeDifferences, result.Code)
	}
}

func TestRun_WithSummary_APIFailure_PreservesExitCode(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"fail"}}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = true
	cfg.SetExitCode = true
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	// Exit code should still be 1 (differences) even on API failure
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with --set-exit-code and API failure, got %d",
			ExitCodeDifferences, result.Code)
	}
}

func TestRun_WithoutSummary_NoAPICall(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Summary = false
	cfg.Color = "never"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if apiCalled {
		t.Error("API should not be called when --summary is not set")
	}
}

// --- Task 3.3: Brief format special case ---

func TestRun_BriefSummary_ReplacesOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"The key was updated."}]}`)
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
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should contain AI summary
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header, got: %s", output)
	}
	if !strings.Contains(output, "The key was updated.") {
		t.Errorf("expected summary text, got: %s", output)
	}
	// Should NOT contain brief format markers (± or "modified")
	if strings.Contains(output, "±") {
		t.Errorf("expected brief output to be suppressed, but found '±' in: %s", output)
	}
}

func TestRun_BriefSummary_FallbackOnAPIFailure(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"fail"}}`)
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
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should fall back to brief output
	if !strings.Contains(output, "±") && !strings.Contains(output, "modified") {
		t.Errorf("expected brief fallback output on API failure, got: %s", output)
	}
	// Warning on stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
	// No AI Summary header
	if strings.Contains(output, "AI Summary:") {
		t.Error("expected no AI Summary header on API failure")
	}
}

func TestRun_BriefSummary_NoDiffs_ShowsStandardOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml := "key: value\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
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
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if apiCalled {
		t.Error("API should not be called when there are no differences")
	}
}

// --- Task 4.1: End-to-end CLI flag and validation tests ---

func TestRun_SummaryValidation_NoAPIKey_ExitCode255(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := NewCLIConfig()
	cfg.FromFile = "from.yaml"
	cfg.ToFile = "to.yaml"
	cfg.Summary = true

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code %d (255) when --summary without API key, got %d", ExitCodeError, result.Code)
	}
	if !strings.Contains(stderr.String(), "ANTHROPIC_API_KEY") {
		t.Errorf("expected error mentioning ANTHROPIC_API_KEY, got: %s", stderr.String())
	}
}

func TestRun_SummaryValidation_NoAPIKey_ParseAndRun(t *testing.T) {
	// End-to-end: parse args then run
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := NewCLIConfig()
	args := []string{"--summary", "from.yaml", "to.yaml"}
	if err := cfg.ParseArgs(args); err != nil {
		t.Fatalf("failed to parse args: %v", err)
	}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr

	result := Run(cfg, rc)
	if result.Code != ExitCodeError {
		t.Errorf("expected exit code 255, got %d", result.Code)
	}
}

func TestRun_SummaryModelFlag_ParseAndRun(t *testing.T) {
	// End-to-end: parse --summary-model flag then verify it's used in API call
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	var receivedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if model, ok := req["model"].(string); ok {
			receivedModel = model
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	args := []string{"--summary", "--summary-model", "claude-sonnet-4-20250514", "from.yaml", "to.yaml"}
	if err := cfg.ParseArgs(args); err != nil {
		t.Fatalf("failed to parse args: %v", err)
	}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte("key: value1\n")
	rc.ToContent = []byte("key: value2\n")
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	if receivedModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-20250514' in API request, got %q", receivedModel)
	}
}

func TestRun_WithSummary_AllFormats(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Summary text."}]}`)
	}))
	defer server.Close()

	formats := []string{"compact", "github", "gitlab", "gitea", "detailed"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := NewCLIConfig()
			cfg.Output = format
			cfg.Summary = true
			cfg.Color = "never"

			rc := NewRunConfig()
			var stdout, stderr strings.Builder
			rc.Stdout = &stdout
			rc.Stderr = &stderr
			rc.FromContent = []byte(yaml1)
			rc.ToContent = []byte(yaml2)
			rc.SummaryAPIURL = server.URL

			result := Run(cfg, rc)
			if result.Err != nil {
				t.Fatalf("unexpected error for format %s: %v", format, result.Err)
			}

			output := stdout.String()
			if !strings.Contains(output, "AI Summary:") {
				t.Errorf("expected AI Summary header for format %s, got: %s", format, output)
			}
		})
	}
}

// --- Task 4.2: End-to-end integration tests for summary flows ---

func TestRun_WithSummary_ColorEnabled(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Colored summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "always"

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := stdout.String()
	// Should contain colored AI Summary header (cyan = \033[36m)
	if !strings.Contains(output, "\033[36m") {
		t.Errorf("expected cyan color in AI Summary header with color=always, got: %s", output)
	}
	// Should contain bold style (\033[1m)
	if !strings.Contains(output, "\033[1m") {
		t.Errorf("expected bold style in AI Summary header with color=always, got: %s", output)
	}
	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected AI Summary header, got: %s", output)
	}
}

func TestRun_WithSummary_WithFilter_OnlyFilteredDiffsSent(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml1 := "config:\n  a: 1\n  b: 2\n"
	yaml2 := "config:\n  a: 10\n  b: 20\n"

	var receivedPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 0 {
			receivedPrompt = req.Messages[0].Content
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"Filtered summary."}]}`)
	}))
	defer server.Close()

	cfg := NewCLIConfig()
	cfg.Output = "compact"
	cfg.Summary = true
	cfg.Color = "never"
	cfg.Filter = []string{"config.a"}

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)
	rc.SummaryAPIURL = server.URL

	Run(cfg, rc)

	// Should contain config.a in prompt
	if !strings.Contains(receivedPrompt, "config.a") {
		t.Errorf("expected config.a in API prompt, got: %s", receivedPrompt)
	}
	// Should NOT contain config.b in prompt (filtered out)
	if strings.Contains(receivedPrompt, "config.b") {
		t.Errorf("expected config.b NOT in API prompt (filtered), got: %s", receivedPrompt)
	}
	// Output should contain the summary
	if !strings.Contains(stdout.String(), "AI Summary:") {
		t.Errorf("expected AI Summary in output, got: %s", stdout.String())
	}
}

func TestRun_WithSummary_BriefNoDiffs_StandardOutput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	yaml := "key: value\n"

	apiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
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
	rc.FromContent = []byte(yaml)
	rc.ToContent = []byte(yaml)
	rc.SummaryAPIURL = server.URL

	result := Run(cfg, rc)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected exit code 0, got %d", result.Code)
	}
	if apiCalled {
		t.Error("API should not be called when there are no differences (brief+summary)")
	}
	// Standard brief output should be shown (no diffs, so formatter handles it)
	if strings.Contains(stdout.String(), "AI Summary:") {
		t.Error("should not show AI Summary when there are no diffs")
	}
}

func TestRun_DetailedWithSummaryOutputNotDeferred(t *testing.T) {
	yaml1 := "key: value1\n"
	yaml2 := "key: value2\n"

	cfg := NewCLIConfig()
	cfg.Output = "detailed"
	cfg.Summary = true
	// Don't set ANTHROPIC_API_KEY — summary will fail, but detailed output should already be written

	rc := NewRunConfig()
	var stdout, stderr strings.Builder
	rc.Stdout = &stdout
	rc.Stderr = &stderr
	rc.FromContent = []byte(yaml1)
	rc.ToContent = []byte(yaml2)

	Run(cfg, rc)

	output := stdout.String()
	// Detailed output should be present (not deferred like brief+summary)
	if !strings.Contains(output, "key") {
		t.Error("expected detailed diff output to be written even with --summary, but output is empty or missing")
	}
}
