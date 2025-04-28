package tools

import (
	"github.com/recrsn/coder/internal/schema"
)

type Tool struct {
	Name        string
	Description string
	InputSchema schema.Schema
	Execute     func(input map[string]any) (string, error)
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
