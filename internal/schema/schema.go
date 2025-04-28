package schema

import "fmt"

// JSONSchemaDraft2020_12 is the URI for the JSON Schema draft 2020-12
const JSONSchemaDraft2020_12 = "https://json-schema.org/draft/2020-12/schema"

// Schema represents a minimal subset of JSON Schema Draft 2020-12
type Schema struct {
	Type                 string              `json:"type"`
	Properties           map[string]Property `json:"properties,omitempty"`
	Required             []string            `json:"required,omitempty"`
	AdditionalProperties any                 `json:"additionalProperties,omitempty"`
	SchemaURI            string              `json:"$schema,omitempty"`
}

type Property struct {
	Type                 string        `json:"type,omitempty"`
	Description          string        `json:"description,omitempty"`
	Items                *Schema       `json:"items,omitempty"`
	Format               string        `json:"format,omitempty"`
	Enum                 []interface{} `json:"enum,omitempty"`
	AdditionalProperties any           `json:"additionalProperties,omitempty"`
	Required             []string      `json:"required,omitempty"`
	Properties           map[string]Property `json:"properties,omitempty"`
}

func (s Schema) Validate(input map[string]interface{}) error {
	// Check required fields
	for _, requiredField := range s.Required {
		if _, ok := input[requiredField]; !ok {
			return fmt.Errorf("missing required field: %s", requiredField)
		}
	}

	// Validate fields based on schema properties
	for fieldName, value := range input {
		// Check if field is defined in properties
		property, ok := s.Properties[fieldName]
		
		// Handle additionalProperties
		if !ok {
			if s.AdditionalProperties == false {
				return fmt.Errorf("unexpected field: %s", fieldName)
			}
			// If additionalProperties is true or a schema, field is allowed
			continue
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
			arr, ok := value.([]interface{})
			if !ok {
				return fmt.Errorf("field %s must be an array", fieldName)
			}
			// Validate items if schema is provided
			if property.Items != nil {
				for i, item := range arr {
					if itemObj, ok := item.(map[string]interface{}); ok {
						if err := property.Items.Validate(itemObj); err != nil {
							return fmt.Errorf("array item %d in field %s: %w", i, fieldName, err)
						}
					}
				}
			}
		case "object":
			obj, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("field %s must be an object", fieldName)
			}
			// Validate nested object if it has properties defined
			if len(property.Properties) > 0 {
				// Create a temporary schema to validate against
				nestedSchema := Schema{
					Type:                 "object",
					Properties:           property.Properties,
					Required:             property.Required,
					AdditionalProperties: property.AdditionalProperties,
				}
				if err := nestedSchema.Validate(obj); err != nil {
					return fmt.Errorf("in nested object %s: %w", fieldName, err)
				}
			}
		}
	}

	return nil
}

// NewSchema creates a new Schema with the latest JSON Schema draft
func NewSchema(schemaType string) *Schema {
	return &Schema{
		Type:      schemaType,
		SchemaURI: JSONSchemaDraft2020_12,
	}
}
