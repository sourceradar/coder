package prompts

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/recrsn/coder/internal/tools"
	"text/template"
)

//go:embed default.go.tmpl
var DefaultPromptTemplate string

// PromptData contains data to be injected into the prompt template
type PromptData struct {
	KnowsTools bool
	Tools      []*tools.Tool
}

// Manager handles loading and rendering prompt templates
type Manager struct{}

// NewManager creates a new prompt manager
func NewManager(_ string) *Manager {
	return &Manager{}
}

// EnsureDefaultPromptExists is a no-op in the simplified implementation
func (m *Manager) EnsureDefaultPromptExists() error {
	return nil
}

// LoadPrompt simply returns the default prompt
func (m *Manager) LoadPrompt(_ string) (string, error) {
	return DefaultPromptTemplate, nil
}

// RenderPrompt renders a prompt template with the given data
func (m *Manager) RenderPrompt(templateContent string, data PromptData) (string, error) {
	tmpl, err := template.New("prompt").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

func GetToolsForPrompt(registry *tools.Registry) []*tools.Tool {
	return registry.GetAll()
}
