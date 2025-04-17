package tools

import (
	"io/fs"
	"os"
	"path/filepath"
)

// NewLSTool creates a tool to list files and directories
func NewLSTool() Tool[map[string]any, map[string]any] {
	return Tool[map[string]any, map[string]any]{
		Name:        "ls",
		Description: "List files and directories",
		Usage:       "ls --path=\"/path/to/dir\" --recursive=true",
		Example:     "ls --path=\".\" --recursive=false",
		InputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
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
		OutputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"files": {
					Type:        "array",
					Description: "List of file paths",
					Items: &PropertyItems{
						Type: "string",
					},
				},
			},
			Required: []string{"files"},
		},
		Execute: func(input map[string]any) (map[string]any, error) {
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
					return nil, err
				}
				
				for _, entry := range entries {
					files = append(files, filepath.Join(path, entry.Name()))
				}
			}
			
			if walkErr != nil {
				return nil, walkErr
			}
			
			return map[string]any{
				"files": files,
			}, nil
		},
	}
}