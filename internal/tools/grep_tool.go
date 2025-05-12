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

// isTextFile checks if a file is likely to be a text file
func isTextFile(filePath string) bool {
	// Skip binary file extensions
	binaryExtensions := []string{
		".bin", ".exe", ".dll", ".so", ".dylib", ".o", ".obj",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff",
		".pdf", ".zip", ".tar", ".gz", ".7z", ".rar",
		".mp3", ".mp4", ".avi", ".mov", ".flv", ".wav",
		".class", ".jar", ".pyc", ".pyo", ".pyd",
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	for _, binExt := range binaryExtensions {
		if ext == binExt {
			return false
		}
	}

	// Try to read a bit of the file to check if it's binary
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 512 bytes to determine file type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	// Check if it contains null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return false
		}
	}

	return true
}

// Helper function for grep tool
func searchFile(filePath string, regex *regexp.Regexp) []map[string]interface{} {
	// Only search text files
	if !isTextFile(filePath) {
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var matches []map[string]interface{}

	for i, line := range lines {
		if regex.MatchString(line) {
			// Trim long lines and add ellipsis if needed
			trimmedLine := line
			if len(trimmedLine) > 1024 {
				trimmedLine = trimmedLine[:1020] + "..."
			}

			matches = append(matches, map[string]interface{}{
				"file":    filePath,
				"line":    i + 1,
				"content": trimmedLine,
			})
		}
	}

	return matches
}
