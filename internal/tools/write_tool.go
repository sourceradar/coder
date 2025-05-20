package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"github.com/sergi/go-diff/diffmatchpatch"
	"os"
	"path/filepath"
	"strings"
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
		Explain: func(input map[string]any) ExplainResult {
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

			title := fmt.Sprintf("Write(%s)", path)

			// Try to read the current file content for diff
			var explainContent string
			existingContent, err := os.ReadFile(path)
			if err == nil {
				diffText := generatePrettyDiff(string(existingContent), content)
				explainContent = fmt.Sprintf("Will write %s to '%s'\n\nDiff:\n```diff\n%s\n```",
					contentDesc, path, diffText)
			} else {
				// If file doesn't exist or can't be read, just show the new content
				explainContent = fmt.Sprintf("Will write %s to '%s'\n\nNew content:\n```\n%s\n```",
					contentDesc, path, content)
			}

			return ExplainResult{
				Title:   title,
				Context: explainContent,
			}
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

// generatePrettyDiff creates a colorized diff using the go-diff library
func generatePrettyDiff(oldText, newText string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, true)

	// Create a string builder to hold the diff output
	var diffOutput strings.Builder
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			diffOutput.WriteString(d.Text)
		case diffmatchpatch.DiffInsert:
			diffOutput.WriteString(fmt.Sprintf("\x1b[32m%s\x1b[0m", d.Text)) // Green for insertions
		case diffmatchpatch.DiffDelete:
			diffOutput.WriteString(fmt.Sprintf("\x1b[31m%s\x1b[0m", d.Text)) // Red for deletions
		}
		diffOutput.WriteString("\n")
	}

	return diffOutput.String()
}
