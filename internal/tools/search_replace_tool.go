package tools

import (
	"errors"
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"os"
	"strings"
)

// NewSearchReplaceTool creates a tool to search for exact matches and replace them
func NewSearchReplaceTool() *Tool {
	return &Tool{
		Name:        "search_replace",
		Description: "Search for exact match of a given string and replace it with the given replacement",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"file": {
					Type:        "string",
					Description: "The file to modify",
				},
				"search": {
					Type:        "string",
					Description: "The exact string to search for",
				},
				"replacement": {
					Type:        "string",
					Description: "The replacement text",
				},
			},
			Required: []string{"file", "search", "replacement"},
		},
		Execute: func(input map[string]any) (string, error) {
			file := input["file"].(string)
			search := input["search"].(string)
			replacement := input["replacement"].(string)

			content, err := os.ReadFile(file)
			if err != nil {
				return "", err
			}

			fileContent := string(content)

			// Count occurrences
			count := strings.Count(fileContent, search)

			if count > 1 {
				return "", errors.New("found multiple matches for the search string")
			}

			if count == 0 {
				return "No matches found. File unchanged.", nil
			}

			// Replace exactly one occurrence
			newContent := strings.Replace(fileContent, search, replacement, 1)

			err = os.WriteFile(file, []byte(newContent), 0644)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Replaced 1 occurrence in %s", file), nil
		},
	}
}
