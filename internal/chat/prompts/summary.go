package prompts

import _ "embed"

//go:embed summary.md
var summaryPrompt string

func RenderSummaryPrompt() string {
	return summaryPrompt
}
