package tools

import (
	"github.com/recrsn/coder/internal/schema"
)

// ExplainResult represents the result of the Tool.Explain function
type ExplainResult struct {
	// Title is a short description of what the tool will do
	Title string
	// Context provides a more detailed explanation of the tool operation
	Context string
}

type Tool struct {
	Name        string
	Description string
	InputSchema schema.Schema
	Execute     func(input map[string]any) (string, error)
	Explain     func(input map[string]any) ExplainResult
}

func (t *Tool) Validate(input map[string]any) error {
	return t.InputSchema.Validate(input)
}

func (t *Tool) Run(input map[string]any) (string, error) {
	if err := t.Validate(input); err != nil {
		return "", err
	}
	return t.Execute(input)
}
