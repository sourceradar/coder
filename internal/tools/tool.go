package tools

import (
	"fmt"
)

type Tool[Input, Output map[string]any] struct {
	Name         string
	Description  string
	Usage        string
	Example      string
	InputSchema  Schema
	OutputSchema Schema
	Execute      func(input Input) (Output, error)
}

type Schema struct {
	Type       string            `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string          `json:"required,omitempty"`
}

type Property struct {
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Items       *PropertyItems    `json:"items,omitempty"`
	Format      string            `json:"format,omitempty"`
	Enum        []interface{}     `json:"enum,omitempty"`
}

type PropertyItems struct {
	Type string `json:"type"`
}

func (t *Tool[Input, Output]) Validate(input Input) error {
	for _, requiredField := range t.InputSchema.Required {
		if _, ok := input[requiredField]; !ok {
			return fmt.Errorf("missing required field: %s", requiredField)
		}
	}

	for fieldName, value := range input {
		property, ok := t.InputSchema.Properties[fieldName]
		if !ok {
			return fmt.Errorf("unexpected field: %s", fieldName)
		}

		// Type validation
		switch property.Type {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field %s must be a string", fieldName)
			}
		case "number", "integer":
			switch value.(type) {
			case int, int32, int64, float32, float64:
				// Valid numeric types
			default:
				return fmt.Errorf("field %s must be a number", fieldName)
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("field %s must be a boolean", fieldName)
			}
		case "array":
			if _, ok := value.([]interface{}); !ok {
				return fmt.Errorf("field %s must be an array", fieldName)
			}
		case "object":
			if _, ok := value.(map[string]interface{}); !ok {
				return fmt.Errorf("field %s must be an object", fieldName)
			}
		}
	}

	return nil
}

func (t *Tool[Input, Output]) Run(input Input) (Output, error) {
	if err := t.Validate(input); err != nil {
		return nil, err
	}
	output, err := t.Execute(input)
	if err != nil {
		return nil, err
	}
	return output, nil
}
