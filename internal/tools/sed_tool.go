package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"os"
	"regexp"
	"strings"
)

// NewSedTool creates a tool to perform string replacement in files
func NewSedTool() *Tool {
	return &Tool{
		Name:        "sed",
		Description: "Replace text in files",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"file": {
					Type:        "string",
					Description: "The file to modify",
				},
				"pattern": {
					Type:        "string",
					Description: "The pattern to search for",
				},
				"replacement": {
					Type:        "string",
					Description: "The replacement text",
				},
				"useRegex": {
					Type:        "boolean",
					Description: "Whether to use regex for pattern matching",
				},
			},
			Required: []string{"file", "pattern", "replacement"},
		},
		Execute: func(input map[string]any) (string, error) {
			file := input["file"].(string)
			pattern := input["pattern"].(string)
			replacement := input["replacement"].(string)
			useRegex, ok := input["useRegex"].(bool)
			if !ok {
				useRegex = false
			}

			content, err := os.ReadFile(file)
			if err != nil {
				return "", err
			}

			var newContent string
			var count int

			if useRegex {
				regex, err := regexp.Compile(pattern)
				if err != nil {
					return "", err
				}

				newContentBytes := regex.ReplaceAll(content, []byte(replacement))
				newContent = string(newContentBytes)

				// Count replacements
				count = strings.Count(string(content), pattern) - strings.Count(newContent, pattern)
			} else {
				// Simple string replacement
				newContent = strings.ReplaceAll(string(content), pattern, replacement)
				count = strings.Count(string(content), pattern)
			}

			err = os.WriteFile(file, []byte(newContent), 0644)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Made %d replacements in %s", count, file), nil
		},
	}
}
