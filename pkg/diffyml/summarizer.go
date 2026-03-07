// summarizer.go - Summary output formatting.
//
// Contains FormatSummaryOutput which uses internal color constants.
// The Summarizer type and AI integration live in pkg/diffyml/cli/.
package diffyml

import "strings"

// FormatSummaryOutput formats the AI summary for display.
func FormatSummaryOutput(summary string, opts *FormatOptions) string {
	var sb strings.Builder
	sb.WriteString("\n")

	if opts == nil {
		opts = DefaultFormatOptions()
	}
	sb.WriteString(colorStart(opts, styleBold+colorCyan))
	sb.WriteString("AI Summary:")
	sb.WriteString(colorEnd(opts))
	sb.WriteString("\n")
	sb.WriteString(summary)
	sb.WriteString("\n")

	return sb.String()
}
