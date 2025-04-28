package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// NewGrepTool creates a tool to search for patterns in files
func NewGrepTool() *Tool {
	return &Tool{
		Name:        "grep",
		Description: "Search for patterns in files",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"pattern": {
					Type:        "string",
					Description: "The regex pattern to search for",
				},
				"paths": {
					Type:        "array",
					Description: "Paths to search in",
					Items: &schema.Schema{
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
		Execute: func(input map[string]any) (string, error) {
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
				return "", err
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
							return "", err
						}
					}
				} else {
					fileMatches := searchFile(path, regex)
					matches = append(matches, fileMatches...)
				}
			}

			// Format the result as readable text
			result := fmt.Sprintf("Found %d matches for pattern '%s':\n\n", len(matches), pattern)

			for _, match := range matches {
				file := match["file"].(string)
				line := match["line"].(int)
				content := match["content"].(string)

				result += fmt.Sprintf("%s:%d: %s\n", file, line, content)
			}

			return result, nil
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
