// summarizer.go - AI-powered summary generation for YAML differences.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/run"
)

const (
	defaultModel     = "claude-haiku-4-5-20251001"
	anthropicVersion = "2023-06-01"
	maxPromptLen     = 8000
)

type httpDoer = run.HttpDoer
type Summarizer = run.Summarizer

func NewSummarizer(model string) *Summarizer { return run.NewSummarizer(model) }
func NewSummarizerWithClient(model string, apiKey string, client httpDoer) *Summarizer {
	return run.NewSummarizerWithClient(model, apiKey, client)
}

func serializeValue(val interface{}) string { return run.SerializeValue(val) }
func buildPrompt(groups []DiffGroup) string { return run.BuildPrompt(groups) }
func systemPrompt() string                  { return run.SystemPrompt() }
func formatSummaryOutput(summary string, opts *FormatOptions) string {
	return run.FormatSummaryOutput(summary, opts)
}
