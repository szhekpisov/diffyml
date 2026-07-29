// summarizer.go - AI-powered summary generation for YAML differences.
//
// Uses the Anthropic Messages API to generate natural language summaries.
// Key types: Summarizer, httpDoer interface.
// Key functions: NewSummarizer, Summarize, buildPrompt.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/szhekpisov/diffyml/pkg/diffyml"
)

const (
	defaultModel     = "claude-haiku-4-5-20251001"
	anthropicAPIURL  = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	maxPromptLen     = 8000
	summaryTimeout   = 30 * time.Second
)

// httpDoer abstracts HTTP request execution for testability.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Summarizer generates AI-powered summaries of YAML differences.
type Summarizer struct {
	client httpDoer
	apiKey string
	model  string
	apiURL string // overridable for testing; defaults to anthropicAPIURL
}

// NewSummarizer creates a summarizer with the specified model.
// If model is empty, defaults to claude-haiku-4-5-20251001.
// Reads ANTHROPIC_API_KEY from the environment.
func NewSummarizer(model string) *Summarizer {
	if model == "" {
		model = defaultModel
	}
	return &Summarizer{
		client: &http.Client{},
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		model:  model,
		apiURL: anthropicAPIURL,
	}
}

// NewSummarizerWithClient creates a summarizer with an injected httpDoer.
// Used in tests to supply a mock HTTP client.
func NewSummarizerWithClient(model string, apiKey string, client httpDoer) *Summarizer {
	if model == "" {
		model = defaultModel
	}
	return &Summarizer{
		client: client,
		apiKey: apiKey,
		model:  model,
		apiURL: anthropicAPIURL,
	}
}

// messagesRequest is the Anthropic Messages API request body.
type messagesRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system"`
	Messages  []messageParam `json:"messages"`
}

type messageParam struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesResponse is the relevant subset of the Anthropic Messages API response.
type messagesResponse struct {
	Content []contentBlock `json:"content"`
	Error   *apiError      `json:"error,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Summarize generates a natural language summary of the given differences.
// Returns the summary text or an error if the API call fails.
func (s *Summarizer) Summarize(ctx context.Context, groups []diffyml.DiffGroup) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, summaryTimeout)
	defer cancel()

	prompt := buildPrompt(groups)

	reqBody := messagesRequest{
		Model:     s.model,
		MaxTokens: 512,
		System:    systemPrompt(),
		Messages:  []messageParam{{Role: "user", Content: prompt}},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Check context before making the request
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", fmt.Errorf("request timed out")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("unexpected response format")
	}

	// Handle HTTP error status codes
	if err := checkHTTPError(resp.StatusCode, &result); err != nil {
		return "", err
	}

	// Extract text from first text content block
	for _, block := range result.Content {
		if block.Type == "text" {
			if block.Text == "" {
				return "", fmt.Errorf("unexpected response format: empty text")
			}
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("unexpected response format: no text content")
}

// checkHTTPError converts HTTP error status codes into descriptive errors.
func checkHTTPError(statusCode int, result *messagesResponse) error {
	//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gomutants to misclassify mutations as NOT COVERED
	if statusCode == 401 {
		return fmt.Errorf("invalid API key")
	} else if statusCode == 429 {
		return fmt.Errorf("rate limited")
	} else if statusCode >= 500 {
		msg := "unknown error"
		if result.Error != nil && result.Error.Message != "" {
			msg = result.Error.Message
		}
		return fmt.Errorf("server error: %s", msg)
	} else if statusCode != 200 {
		msg := fmt.Sprintf("HTTP %d", statusCode)
		if result.Error != nil && result.Error.Message != "" {
			msg = result.Error.Message
		}
		return fmt.Errorf("API error: %s", msg)
	}
	return nil
}

// diffTypeLabel returns the prompt label for a DiffType.
func diffTypeLabel(dt diffyml.DiffType) string {
	switch dt {
	case diffyml.DiffAdded:
		return "ADDED"
	case diffyml.DiffRemoved:
		return "REMOVED"
	case diffyml.DiffModified:
		return "MODIFIED"
	case diffyml.DiffOrderChanged:
		return "ORDER_CHANGED"
	case diffyml.DiffUnchanged:
		return "UNCHANGED"
	default:
		return "UNKNOWN"
	}
}

// remainingPromptItems counts changes and files that have not yet been written.
// diffIndex is the first unwritten difference in groups[groupIndex].
func remainingPromptItems(groups []diffyml.DiffGroup, groupIndex, diffIndex int) (changes, files int) {
	for i := groupIndex; i < len(groups); i++ {
		start := 0
		if i == groupIndex {
			start = diffIndex
		}
		if start >= len(groups[i].Diffs) {
			continue
		}
		changes += len(groups[i].Diffs) - start
		files++
	}
	return changes, files
}

func truncationMarker(remainingChanges, remainingFiles int) string {
	return fmt.Sprintf("\n... and %d more changes across %d files (truncated)\n", remainingChanges, remainingFiles)
}

// buildTruncatedPrompt rebuilds an oversized prompt while reserving space for
// an exact truncation marker. Only complete headers and differences are
// written, so the marker's remaining-change count always matches the content.
func buildTruncatedPrompt(groups []diffyml.DiffGroup) string {
	var sb strings.Builder
	remainingChanges, remainingFiles := remainingPromptItems(groups, 0, 0)

	for _, group := range groups {
		marker := truncationMarker(remainingChanges, remainingFiles)
		header := fmt.Sprintf("File: %s\n", group.FilePath)
		if sb.Len()+len(header)+len(marker) > maxPromptLen {
			return sb.String() + marker
		}
		sb.WriteString(header)

		for diffIndex, diff := range group.Diffs {
			from := diffyml.SerializeValue(diff.From)
			to := diffyml.SerializeValue(diff.To)
			line := fmt.Sprintf("- [%s] %s: %q → %q\n", diffTypeLabel(diff.Type), diff.Path, from, to)

			changesAfter := remainingChanges - 1
			filesAfter := remainingFiles
			if diffIndex == len(group.Diffs)-1 {
				filesAfter--
			}

			reserved := ""
			if changesAfter > 0 {
				reserved = truncationMarker(changesAfter, filesAfter)
			}
			if sb.Len()+len(line)+len(reserved) > maxPromptLen {
				return sb.String() + marker
			}

			sb.WriteString(line)
			remainingChanges = changesAfter
			remainingFiles = filesAfter
			marker = reserved
		}

		if sb.Len()+1+len(marker) <= maxPromptLen {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// buildPrompt serializes DiffGroups into structured text for the API while
// keeping the complete request prompt within maxPromptLen bytes. It writes one
// difference at a time so a single large file cannot bypass the limit.
func buildPrompt(groups []diffyml.DiffGroup) string {
	var sb strings.Builder

	for _, group := range groups {
		header := fmt.Sprintf("File: %s\n", group.FilePath)
		if sb.Len()+len(header) > maxPromptLen {
			return buildTruncatedPrompt(groups)
		}
		sb.WriteString(header)

		for _, diff := range group.Diffs {
			from := diffyml.SerializeValue(diff.From)
			to := diffyml.SerializeValue(diff.To)
			line := fmt.Sprintf("- [%s] %s: %q → %q\n", diffTypeLabel(diff.Type), diff.Path, from, to)
			if sb.Len()+len(line) > maxPromptLen {
				return buildTruncatedPrompt(groups)
			}
			sb.WriteString(line)
		}

		if sb.Len() < maxPromptLen {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// systemPrompt returns the system prompt instructing the model on summary style.
func systemPrompt() string {
	return "You are a YAML diff summarizer. Given a list of structural differences between YAML files, produce a concise natural language summary (2-5 sentences). Focus on the most important changes and their likely impact. Do not repeat raw paths or values — describe the changes at a conceptual level. If changes span multiple files, mention the affected files."
}
