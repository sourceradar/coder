package prompts

import (
	"bytes"
	_ "embed"
	"fmt"
	tools2 "github.com/recrsn/coder/internal/tools"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

// Embedded default prompt template
//
//go:embed default.go.tmpl
var DefaultPromptTemplate string

// PromptData contains data to be injected into the prompt template
type PromptData struct {
	KnowsTools bool
	Tools      []tools2.Tool[map[string]any, map[string]any]
}

// Manager handles loading and rendering prompt templates
type Manager struct {
	templateDir string
}

// NewManager creates a new prompt manager
func NewManager(templateDir string) *Manager {
	return &Manager{
		templateDir: templateDir,
	}
}

// GetDefaultPromptPath returns the path to the default prompt template
func (m *Manager) GetDefaultPromptPath() string {
	return filepath.Join(m.templateDir, "default.go.tmpl")
}

// EnsureDefaultPromptExists ensures the default prompt template exists
func (m *Manager) EnsureDefaultPromptExists() error {
	defaultPath := m.GetDefaultPromptPath()

	// Check if default prompt template exists
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		// Ensure directory exists
		if err := os.MkdirAll(m.templateDir, 0755); err != nil {
			return fmt.Errorf("creating template directory: %w", err)
		}

		// Create default prompt file
		if err := os.WriteFile(defaultPath, []byte(DefaultPromptTemplate), 0644); err != nil {
			return fmt.Errorf("creating default prompt: %w", err)
		}
	}

	return nil
}

// ListPrompts returns a list of available prompt templates
func (m *Manager) ListPrompts() ([]string, error) {
	var templates []string

	// Ensure template directory exists
	if err := os.MkdirAll(m.templateDir, 0755); err != nil {
		return nil, fmt.Errorf("creating template directory: %w", err)
	}

	// Ensure default prompt exists
	if err := m.EnsureDefaultPromptExists(); err != nil {
		return nil, err
	}

	err := filepath.WalkDir(m.templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ".tmpl" {
			// Get relative path from template directory
			relPath, err := filepath.Rel(m.templateDir, path)
			if err != nil {
				return err
			}
			templates = append(templates, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("listing prompts: %w", err)
	}

	return templates, nil
}

// LoadPrompt loads a prompt template from a file
func (m *Manager) LoadPrompt(filename string) (string, error) {
	// Ensure template directory exists
	if err := os.MkdirAll(m.templateDir, 0755); err != nil {
		return "", fmt.Errorf("creating template directory: %w", err)
	}

	// Use default template if it's the default file and doesn't exist
	if filename == "default.go.tmpl" {
		defaultPath := m.GetDefaultPromptPath()
		if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
			// Return the embedded default template
			return DefaultPromptTemplate, nil
		}
	}

	path := filepath.Join(m.templateDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && filename == "default.go.tmpl" {
			// Return the embedded default template if the file doesn't exist
			return DefaultPromptTemplate, nil
		}
		return "", fmt.Errorf("reading prompt file: %w", err)
	}

	return string(data), nil
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

// SavePrompt saves a prompt template to a file
func (m *Manager) SavePrompt(filename, content string) error {
	// Ensure template directory exists
	if err := os.MkdirAll(m.templateDir, 0755); err != nil {
		return fmt.Errorf("creating template directory: %w", err)
	}

	path := filepath.Join(m.templateDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing prompt file: %w", err)
	}

	return nil
}

// GetToolsForPrompt converts a registry of tools to a list of tools for PromptData
func GetToolsForPrompt(registry *tools2.Registry) []tools2.Tool[map[string]any, map[string]any] {
	var toolsList []tools2.Tool[map[string]any, map[string]any]

	for _, name := range registry.ListTools() {
		if toolInterface, ok := registry.Get(name); ok {
			if tool, ok := toolInterface.(tools2.Tool[map[string]any, map[string]any]); ok {
				toolsList = append(toolsList, tool)
			}
		}
	}

	return toolsList
}
