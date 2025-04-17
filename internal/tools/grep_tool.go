package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// NewGrepTool creates a tool to search for patterns in files
func NewGrepTool() Tool[map[string]any, map[string]any] {
	return Tool[map[string]any, map[string]any]{
		Name:        "grep",
		Description: "Search for patterns in files",
		Usage:       "grep --pattern=\"func\" --paths=\"[\"file1.go\", \"file2.go\"]\"",
		Example:     "grep --pattern=\"error\" --paths=\"[\"main.go\"]\" --recursive=true",
		InputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"pattern": {
					Type:        "string",
					Description: "The regex pattern to search for",
				},
				"paths": {
					Type:        "array",
					Description: "Paths to search in",
					Items: &PropertyItems{
						Type: "string",
					},
				},
				"recursive": {
					Type:        "boolean",
					Description: "Whether to search directories recursively",
				},
			},
			Required: []string{"pattern", "paths"},
		},
		OutputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"matches": {
					Type:        "array",
					Description: "List of matches with file, line number and content",
					Items: &PropertyItems{
						Type: "object",
					},
				},
			},
			Required: []string{"matches"},
		},
		Execute: func(input map[string]any) (map[string]any, error) {
			pattern := input["pattern"].(string)
			pathsAny := input["paths"].([]interface{})
			recursive, ok := input["recursive"].(bool)
			if !ok {
				recursive = false
			}
			
			paths := make([]string, len(pathsAny))
			for i, p := range pathsAny {
				paths[i] = p.(string)
			}
			
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, err
			}
			
			var matches []map[string]interface{}
			
			for _, path := range paths {
				fileInfo, err := os.Stat(path)
				if err != nil {
					continue
				}
				
				if fileInfo.IsDir() {
					if recursive {
						err := filepath.Walk(path, func(filePath string, info fs.FileInfo, err error) error {
							if err != nil {
								return err
							}
							
							if !info.IsDir() {
								fileMatches := searchFile(filePath, regex)
								matches = append(matches, fileMatches...)
							}
							
							return nil
						})
						
						if err != nil {
							return nil, err
						}
					}
				} else {
					fileMatches := searchFile(path, regex)
					matches = append(matches, fileMatches...)
				}
			}
			
			return map[string]any{
				"matches": matches,
			}, nil
		},
	}
}

// Helper function for grep tool
func searchFile(filePath string, regex *regexp.Regexp) []map[string]interface{} {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	
	lines := strings.Split(string(content), "\n")
	var matches []map[string]interface{}
	
	for i, line := range lines {
		if regex.MatchString(line) {
			matches = append(matches, map[string]interface{}{
				"file":    filePath,
				"line":    i + 1,
				"content": line,
			})
		}
	}
	
	return matches
}