package tools

import (
	"os"
	"regexp"
	"strings"
)

// NewSedTool creates a tool to perform string replacement in files
func NewSedTool() Tool[map[string]any, map[string]any] {
	return Tool[map[string]any, map[string]any]{
		Name:        "sed",
		Description: "Replace text in files",
		Usage:       "sed --file=\"file.txt\" --pattern=\"hello\" --replacement=\"world\"",
		Example:     "sed --file=\"config.json\" --pattern=\"localhost\" --replacement=\"127.0.0.1\"",
		InputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
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
		OutputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"replacements": {
					Type:        "integer",
					Description: "Number of replacements made",
				},
				"content": {
					Type:        "string",
					Description: "New file content",
				},
			},
			Required: []string{"replacements", "content"},
		},
		Execute: func(input map[string]any) (map[string]any, error) {
			file := input["file"].(string)
			pattern := input["pattern"].(string)
			replacement := input["replacement"].(string)
			useRegex, ok := input["useRegex"].(bool)
			if !ok {
				useRegex = false
			}
			
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			
			var newContent string
			var count int
			
			if useRegex {
				regex, err := regexp.Compile(pattern)
				if err != nil {
					return nil, err
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
				return nil, err
			}
			
			return map[string]any{
				"replacements": count,
				"content":      newContent,
			}, nil
		},
	}
}