package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"os"
	"path/filepath"
)

// NewWriteTool creates a new tool for writing files
func NewWriteTool() *Tool {
	return &Tool{
		Name:        "write",
		Description: "Write content to a file",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"path": {
					Type:        "string",
					Description: "The path to the file to write",
				},
				"content": {
					Type:        "string",
					Description: "The content to write to the file",
				},
			},
			Required: []string{"path", "content"},
		},
		Explain: func(input map[string]any) string {
			path, _ := input["path"].(string)
			content, _ := input["content"].(string)

			contentLength := len(content)
			var contentDesc string
			if contentLength == 0 {
				contentDesc = "an empty file"
			} else if contentLength == 1 {
				contentDesc = "1 byte"
			} else {
				contentDesc = fmt.Sprintf("%d bytes", contentLength)
			}

			return fmt.Sprintf("Will write %s to '%s'", contentDesc, path)
		},
		Execute: func(input map[string]any) (string, error) {
			// Extract parameters
			path, _ := input["path"].(string)
			content, _ := input["content"].(string)

			// Ensure the directory exists
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}

			// Write the file
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}

			return fmt.Sprintf("File written to %s (%d bytes)", path, len(content)), nil
		},
	}
}
