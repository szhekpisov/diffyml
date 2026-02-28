// summarizer.go - AI-powered summary generation for YAML differences.
//
// Uses the Anthropic Messages API to generate natural language summaries.
// Key types: Summarizer, httpDoer interface.
// Key functions: NewSummarizer, Summarize, buildPrompt, serializeValue.
package diffyml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
func (s *Summarizer) Summarize(ctx context.Context, groups []DiffGroup) (string, error) {
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
	//nolint:gocritic // if-else kept intentionally: switch/case conditions fall outside Go coverage blocks, causing gremlins to misclassify mutations as NOT COVERED
	if resp.StatusCode == 401 {
		return "", fmt.Errorf("invalid API key")
	} else if resp.StatusCode == 429 {
		return "", fmt.Errorf("rate limited")
	} else if resp.StatusCode >= 500 {
		msg := "unknown error"
		if result.Error != nil && result.Error.Message != "" {
			msg = result.Error.Message
		}
		return "", fmt.Errorf("server error: %s", msg)
	} else if resp.StatusCode != 200 {
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if result.Error != nil && result.Error.Message != "" {
			msg = result.Error.Message
		}
		return "", fmt.Errorf("API error: %s", msg)
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

// diffTypeLabel returns the prompt label for a DiffType.
func diffTypeLabel(dt DiffType) string {
	switch dt {
	case DiffAdded:
		return "ADDED"
	case DiffRemoved:
		return "REMOVED"
	case DiffModified:
		return "MODIFIED"
	case DiffOrderChanged:
		return "ORDER_CHANGED"
	default:
		return "UNKNOWN"
	}
}

// buildPrompt serializes DiffGroups into structured text for the API.
func buildPrompt(groups []DiffGroup) string {
	var sb strings.Builder
	totalLen := 0
	groupsWritten := 0
	totalGroups := len(groups)
	totalRemainingDiffs := 0

	for _, group := range groups {
		// Serialize this group into a temporary buffer
		var groupBuf strings.Builder
		fmt.Fprintf(&groupBuf, "File: %s\n", group.FilePath)
		for _, diff := range group.Diffs {
			from := serializeValue(diff.From)
			to := serializeValue(diff.To)
			fmt.Fprintf(&groupBuf, "- [%s] %s: %q → %q\n", diffTypeLabel(diff.Type), diff.Path, from, to)
		}
		groupBuf.WriteString("\n")

		groupText := groupBuf.String()

		// Check truncation before adding
		if totalLen+len(groupText) > maxPromptLen && groupsWritten > 0 {
			// Count remaining diffs
			for _, g := range groups[groupsWritten:] {
				totalRemainingDiffs += len(g.Diffs)
			}
			remainingFiles := totalGroups - groupsWritten
			fmt.Fprintf(&sb, "... and %d more changes across %d more files (truncated)\n", totalRemainingDiffs, remainingFiles)
			break
		}

		sb.WriteString(groupText)
		totalLen += len(groupText)
		groupsWritten++
	}

	return sb.String()
}

// serializeValue serializes a Difference.From or Difference.To value into a
// human-readable string for prompt inclusion.
func serializeValue(val interface{}) string {
	if val == nil {
		return "<none>"
	}

	switch v := val.(type) {
	case *OrderedMap:
		out, err := yaml.Marshal(orderedMapToGeneric(v))
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return strings.TrimRight(string(out), "\n")
	case map[string]interface{}, []interface{}:
		out, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return strings.TrimRight(string(out), "\n")
	default:
		return fmt.Sprintf("%v", val)
	}
}

// orderedMapToGeneric converts an OrderedMap to a yaml.v3-serializable
// structure that preserves key order.
func orderedMapToGeneric(om *OrderedMap) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, key := range om.Keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
		valNode := valueToYAMLNode(om.Values[key])
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node
}

// valueToYAMLNode converts a Go value to a yaml.Node for serialization.
func valueToYAMLNode(val interface{}) *yaml.Node {
	switch v := val.(type) {
	case *OrderedMap:
		return orderedMapToGeneric(v)
	default:
		n := &yaml.Node{}
		_ = n.Encode(val)
		return n
	}
}

// systemPrompt returns the system prompt instructing the model on summary style.
func systemPrompt() string {
	return "You are a YAML diff summarizer. Given a list of structural differences between YAML files, produce a concise natural language summary (2-5 sentences). Focus on the most important changes and their likely impact. Do not repeat raw paths or values — describe the changes at a conceptual level. If changes span multiple files, mention the affected files."
}

// formatSummaryOutput formats the AI summary for display.
func formatSummaryOutput(summary string, opts *FormatOptions) string {
	var sb strings.Builder
	sb.WriteString("\n")

	if opts != nil && opts.Color {
		sb.WriteString(styleBold + colorCyan)
		sb.WriteString("AI Summary:")
		sb.WriteString(colorReset)
	} else {
		sb.WriteString("AI Summary:")
	}
	sb.WriteString("\n")
	sb.WriteString(summary)
	sb.WriteString("\n")

	return sb.String()
}
