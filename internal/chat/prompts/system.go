package prompts

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/recrsn/coder/internal/tools"
	"text/template"
)

//go:embed default.md
var DefaultPromptTemplate string

// PromptData contains data to be injected into the prompt template
type PromptData struct {
	KnowsTools       bool
	Tools            []*tools.Tool
	WorkingDirectory string
	Platform         string
	Date             string
}

// RenderSystemPrompt renders a prompt template with the given data
func RenderSystemPrompt(data PromptData) (string, error) {
	tmpl, err := template.New("prompt").Parse(DefaultPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
