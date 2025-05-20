package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"os"
	"path/filepath"
	"strings"
)

// NewTreeTool creates a tool to display a directory structure in a tree format
func NewTreeTool() *Tool {
	return &Tool{
		Name:        "tree",
		Description: "Display directory structure in a tree format",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"path": {
					Type:        "string",
					Description: "The directory path to display",
				},
				"depth": {
					Type:        "integer",
					Description: "Maximum depth of directory tree to display (default: unlimited)",
				},
			},
			Required: []string{"path"},
		},
		Explain: func(input map[string]any) ExplainResult {
			path, _ := input["path"].(string)
			depth, _ := input["depth"].(int)
			return ExplainResult{
				Title:   fmt.Sprintf("Tree(%s, %d)", path, depth),
				Context: fmt.Sprintf("Will display a tree view of the directory structure for '%s' %d levels deep", path, depth),
			}
		},
		Execute: func(input map[string]any) (string, error) {
			path := input["path"].(string)

			// Set default depth to a large number if not specified
			var maxDepth int = 1000
			if depthVal, ok := input["depth"]; ok {
				switch v := depthVal.(type) {
				case int:
					maxDepth = v
				case float64:
					maxDepth = int(v)
				}
			}

			// Initialize tree with root directory
			tree := filepath.Base(path)
			content, err := buildTree(path, "", 0, maxDepth)
			if err != nil {
				return "", err
			}

			if content != "" {
				tree += "\n" + content
			}

			return tree, nil
		},
	}
}

// buildTree recursively builds a tree representation of the directory structure
func buildTree(path string, prefix string, depth int, maxDepth int) (string, error) {
	if depth >= maxDepth {
		return "", nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	entryCount := len(entries)

	for i, entry := range entries {
		// Determine if this is the last item at this level
		isLast := i == entryCount-1

		// Choose the appropriate connector symbols
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		// Choose the appropriate prefix for the next level
		nextPrefix := prefix + "│   "
		if isLast {
			nextPrefix = prefix + "    "
		}

		// Add this entry to the tree
		result.WriteString(prefix + connector + entry.Name() + "\n")

		// Recursively process directories
		if entry.IsDir() {
			entryPath := filepath.Join(path, entry.Name())

			// Get the subtree for this directory
			subtree, err := buildTree(entryPath, nextPrefix, depth+1, maxDepth)
			if err != nil {
				return "", err
			}

			result.WriteString(subtree)
		}
	}

	return result.String(), nil
}
