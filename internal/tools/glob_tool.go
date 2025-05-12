package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"io/fs"
	"path/filepath"
	"strings"
)

// NewGlobTool creates a tool to find files by glob pattern
func NewGlobTool() *Tool {
	return &Tool{
		Name:        "glob",
		Description: "Find files matching a glob pattern",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"pattern": {
					Type:        "string",
					Description: "The glob pattern to match",
				},
				"root": {
					Type:        "string",
					Description: "Root directory to start searching from",
				},
			},
			Required: []string{"pattern"},
		},
		Explain: func(input map[string]any) string {
			pattern, _ := input["pattern"].(string)
			root, ok := input["root"].(string)
			if !ok {
				root = "."
			}

			return fmt.Sprintf("Will search for files matching pattern '%s' in directory '%s'", pattern, root)
		},
		Execute: func(input map[string]any) (string, error) {
			pattern := input["pattern"].(string)
			root, ok := input["root"].(string)
			if !ok {
				root = "."
			}

			matches, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				return "", err
			}

			// Handle ** recursively if present
			if strings.Contains(pattern, "**") {
				parts := strings.Split(pattern, "**")
				if len(parts) > 1 {
					var allMatches []string

					err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
						if err != nil {
							return err
						}

						// Check if path matches the pattern
						matched, err := filepath.Match(strings.Replace(pattern, "**", "*", -1), path)
						if err != nil {
							return err
						}

						if matched {
							allMatches = append(allMatches, path)
						}

						return nil
					})

					if err != nil {
						return "", err
					}

					matches = allMatches
				}
			}

			// Format the result as readable text
			if len(matches) == 0 {
				return "No files found matching pattern '" + pattern + "'", nil
			}

			result := fmt.Sprintf("Found %d files matching pattern '%s':\n\n", len(matches), pattern)
			for _, match := range matches {
				result += match + "\n"
			}
			return result, nil
		},
	}
}
