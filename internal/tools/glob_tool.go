package tools

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// NewGlobTool creates a tool to find files by glob pattern
func NewGlobTool() Tool[map[string]any, map[string]any] {
	return Tool[map[string]any, map[string]any]{
		Name:        "glob",
		Description: "Find files matching a glob pattern",
		Usage:       "glob --pattern=\"**/*.go\" --root=\".\"",
		Example:     "glob --pattern=\"*.txt\" --root=\"/path/to/dir\"",
		InputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
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
		OutputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"matches": {
					Type:        "array",
					Description: "List of matched file paths",
					Items: &PropertyItems{
						Type: "string",
					},
				},
			},
			Required: []string{"matches"},
		},
		Execute: func(input map[string]any) (map[string]any, error) {
			pattern := input["pattern"].(string)
			root, ok := input["root"].(string)
			if !ok {
				root = "."
			}
			
			matches, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				return nil, err
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
						return nil, err
					}
					
					matches = allMatches
				}
			}
			
			return map[string]any{
				"matches": matches,
			}, nil
		},
	}
}