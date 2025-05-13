package lsp

import (
	"fmt"
	"github.com/recrsn/coder/internal/lsp"
	"path/filepath"
	"strings"

	"github.com/recrsn/coder/internal/schema"
	"github.com/recrsn/coder/internal/tools"
)

// NewDefinitionTool creates a tool for finding definitions using LSP
func NewDefinitionTool(manager *lsp.Manager) *tools.Tool {
	return &tools.Tool{
		Name:        "definition",
		Description: "Find the definition of a symbol using the Language Server Protocol",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"file_path": {
					Type:        "string",
					Description: "The path to the file containing the symbol",
				},
				"line": {
					Type:        "integer",
					Description: "The line number of the symbol (0-based)",
				},
				"character": {
					Type:        "integer",
					Description: "The character offset of the symbol (0-based)",
				},
			},
			Required: []string{"file_path", "line", "character"},
		},
		Explain: func(input map[string]any) string {
			filePath, _ := input["file_path"].(string)
			line, _ := input["line"].(float64)
			character, _ := input["character"].(float64)

			return fmt.Sprintf("Will find the definition of the symbol at %s:%d:%d",
				filePath, int(line), int(character))
		},
		Execute: func(input map[string]any) (string, error) {
			// Extract parameters
			filePath, _ := input["file_path"].(string)
			line, _ := input["line"].(float64)
			character, _ := input["character"].(float64)

			// Convert to absolute path if needed
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			// Get definitions
			locations, err := manager.GetDefinition(absPath, int(line), int(character))
			if err != nil {
				return "", err
			}

			if len(locations) == 0 {
				return "No definition found", nil
			}

			// Format the results
			var result strings.Builder
			for i, location := range locations {
				if i > 0 {
					result.WriteString("\n\n")
				}

				// Convert URI to file path
				defPath := strings.TrimPrefix(location.URI.String(), "file://")

				// Get line and column
				startLine := location.Range.Start.Line
				startChar := location.Range.Start.Character
				endLine := location.Range.End.Line
				endChar := location.Range.End.Character

				result.WriteString(fmt.Sprintf("Definition found at %s:%d:%d", defPath, startLine, startChar))
				if startLine != endLine || startChar != endChar {
					result.WriteString(fmt.Sprintf(" to %d:%d", endLine, endChar))
				}

				// Add link for easy navigation
				result.WriteString(fmt.Sprintf("\nLocation: %s:%d:%d", defPath, startLine, startChar))
			}

			return result.String(), nil
		},
	}
}
