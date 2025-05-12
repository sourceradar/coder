package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"io/fs"
	"os"
	"path/filepath"
)

// NewLSTool creates a tool to list files and directories
func NewLSTool() *Tool {
	return &Tool{
		Name:        "ls",
		Description: "List files and directories",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"path": {
					Type:        "string",
					Description: "The directory path to list",
				},
				"recursive": {
					Type:        "boolean",
					Description: "Whether to list directories recursively",
				},
			},
			Required: []string{"path"},
		},
		Explain: func(input map[string]any) string {
			path, _ := input["path"].(string)
			recursive, ok := input["recursive"].(bool)
			if !ok {
				recursive = false
			}

			if recursive {
				return fmt.Sprintf("Will list all files and directories recursively in '%s'", path)
			}
			return fmt.Sprintf("Will list files and directories in '%s'", path)
		},
		Execute: func(input map[string]any) (string, error) {
			path := input["path"].(string)
			recursive, ok := input["recursive"].(bool)
			if !ok {
				recursive = false
			}

			var files []string
			var walkErr error

			if recursive {
				walkErr = filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
					if err != nil {
						return err
					}
					files = append(files, path)
					return nil
				})
			} else {
				entries, err := os.ReadDir(path)
				if err != nil {
					return "", err
				}

				for _, entry := range entries {
					files = append(files, filepath.Join(path, entry.Name()))
				}
			}

			if walkErr != nil {
				return "", walkErr
			}

			// Format the result as a readable text
			result := fmt.Sprintf("Found %d files in %s:\n\n", len(files), path)
			for _, file := range files {
				result += file + "\n"
			}

			return result, nil
		},
	}
}
