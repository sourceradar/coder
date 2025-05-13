package lsp

import (
	"fmt"
	"github.com/recrsn/coder/internal/lsp"
	"path/filepath"
	"strings"

	"github.com/recrsn/coder/internal/schema"
	"github.com/recrsn/coder/internal/tools"
)

// NewReferencesTool creates a tool for finding references using LSP
func NewReferencesTool(manager *lsp.Manager) *tools.Tool {
	return &tools.Tool{
		Name:        "references",
		Description: "Find all references to a symbol using the Language Server Protocol",
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
				"include_declaration": {
					Type:        "boolean",
					Description: "Whether to include the declaration in the results (optional)",
				},
			},
			Required: []string{"file_path", "line", "character"},
		},
		Explain: func(input map[string]any) string {
			filePath, _ := input["file_path"].(string)
			line, _ := input["line"].(float64)
			character, _ := input["character"].(float64)
			includeDeclaration, hasIncludeDeclaration := input["include_declaration"].(bool)

			explanation := fmt.Sprintf("Will find all references to the symbol at %s:%d:%d",
				filePath, int(line), int(character))

			if hasIncludeDeclaration {
				if includeDeclaration {
					explanation += " (including declaration)"
				} else {
					explanation += " (excluding declaration)"
				}
			}

			return explanation
		},
		Execute: func(input map[string]any) (string, error) {
			// Extract parameters
			filePath, _ := input["file_path"].(string)
			line, _ := input["line"].(float64)
			character, _ := input["character"].(float64)
			includeDeclaration, hasIncludeDeclaration := input["include_declaration"].(bool)

			if !hasIncludeDeclaration {
				includeDeclaration = true // Default to include declaration
			}

			// Convert to absolute path if needed
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			// Get references
			locations, err := manager.GetReferences(absPath, int(line), int(character), includeDeclaration)
			if err != nil {
				return "", err
			}

			if len(locations) == 0 {
				return "No references found", nil
			}

			// Format the results
			var result strings.Builder
			result.WriteString(fmt.Sprintf("Found %d references:\n\n", len(locations)))

			// Group references by file
			fileRefs := make(map[string][]string)
			for _, location := range locations {
				// Convert URI to file path
				refPath := strings.TrimPrefix(location.URI.String(), "file://")

				// Get line and column (add 1 to convert from 0-based to 1-based)
				startLine := location.Range.Start.Line + 1
				startChar := location.Range.Start.Character + 1

				// Add to file group
				fileRefs[refPath] = append(fileRefs[refPath],
					fmt.Sprintf("Line %d, Col %d", startLine, startChar))
			}

			// Print grouped references
			for file, refs := range fileRefs {
				result.WriteString(fmt.Sprintf("File: %s\n", file))
				for _, ref := range refs {
					result.WriteString(fmt.Sprintf("  - %s\n", ref))
				}
				result.WriteString("\n")
			}

			return result.String(), nil
		},
	}
}
