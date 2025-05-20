package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/tools/outline"
	"os"
	"path/filepath"
	"strings"

	"github.com/recrsn/coder/internal/schema"
)

// NewOutlineTool creates a tool to generate an outline of a file
func NewOutlineTool() *Tool {
	return &Tool{
		Name:        "outline",
		Description: "Generate an outline of symbols in a file (both public and private)",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"file": {
					Type:        "string",
					Description: "Path to the file to analyze",
				},
			},
			Required: []string{"file"},
		},
		Explain: func(input map[string]any) ExplainResult {
			filePath, _ := input["file"].(string)

			title := fmt.Sprintf("Outline(%s)", filePath)
			content := fmt.Sprintf("Will generate an outline showing the structure and symbols in '%s'", filePath)

			return ExplainResult{
				Title:   title,
				Context: content,
			}
		},
		Execute: func(input map[string]any) (string, error) {
			filePath := input["file"].(string)

			// Check if file exists
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return "", fmt.Errorf("file not found: %v", err)
			}
			if fileInfo.IsDir() {
				return "", fmt.Errorf("expected a file, got directory")
			}

			// Read file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("error reading file: %v", err)
			}

			// Detect language based on file extension
			ext := strings.ToLower(filepath.Ext(filePath))
			var language string

			switch ext {
			case ".go":
				language = "go"
			case ".js":
				language = "javascript"
			case ".jsx":
				language = "javascript"
			case ".ts":
				language = "typescript"
			case ".tsx":
				language = "tsx"
			case ".py":
				language = "python"
			default:
				return "", fmt.Errorf("unsupported file extension: %s", ext)
			}

			if err != nil {
				return "", fmt.Errorf("error setting language parser: %v", err)
			}

			// Extract symbols based on language
			outline, err := outline.ExtractOutline(content, language)

			if err != nil {
				return "", fmt.Errorf("error extracting outline: %v", err)
			}

			result := fmt.Sprintf("Language: %s\n\n%s", language, outline)
			return result, nil
		},
	}
}
