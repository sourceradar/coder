package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"os"
	"path/filepath"
	"strings"
)

// NewReadTool creates a tool for reading files
func NewReadTool() *Tool {
	return &Tool{
		Name:        "read",
		Description: "Read content from a file",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"path": {
					Type:        "string",
					Description: "The path to the file to read",
				},
				"start": {
					Type:        "integer",
					Description: "The line number to start reading from (1-based, optional)",
				},
				"end": {
					Type:        "integer",
					Description: "The line number to end reading at (1-based, inclusive, optional)",
				},
			},
			Required: []string{"path"},
		},
		Explain: func(input map[string]any) ExplainResult {
			path, _ := input["path"].(string)
			start, hasStart := input["start"].(float64)
			end, hasEnd := input["end"].(float64)

			var title string
			var content string

			if hasStart && hasEnd {
				title = fmt.Sprintf("Read(%s, %d-%d)", path, int(start), int(end))
				content = fmt.Sprintf("Will read lines %d to %d from '%s'", int(start), int(end), path)
			} else if hasStart {
				title = fmt.Sprintf("Read(%s, %d+)", path, int(start))
				content = fmt.Sprintf("Will read from line %d to the end of '%s'", int(start), path)
			} else if hasEnd {
				title = fmt.Sprintf("Read(%s, 1-%d)", path, int(end))
				content = fmt.Sprintf("Will read from the beginning to line %d of '%s'", int(end), path)
			} else {
				title = fmt.Sprintf("Read(%s)", path)
				content = fmt.Sprintf("Will read the entire contents of '%s'", path)
			}

			return ExplainResult{
				Title:   title,
				Context: content,
			}
		},
		Execute: func(input map[string]any) (string, error) {
			// Extract parameters
			path, _ := input["path"].(string)
			startFloat, hasStart := input["start"].(float64)
			endFloat, hasEnd := input["end"].(float64)

			start := 1 // Default to first line
			if hasStart {
				start = int(startFloat)
				if start < 1 {
					return "", fmt.Errorf("start line must be at least 1")
				}
			}

			// Ensure the file exists
			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			fileInfo, err := os.Stat(absPath)
			if err != nil {
				return "", fmt.Errorf("failed to access file: %w", err)
			}

			if fileInfo.IsDir() {
				return "", fmt.Errorf("path is a directory, not a file")
			}

			// Read the file content
			content, err := os.ReadFile(absPath)
			if err != nil {
				return "", fmt.Errorf("failed to read file: %w", err)
			}

			// Convert to string and split by lines
			lines := strings.Split(string(content), "\n")

			// Determine the end line if specified or default to the last line
			end := len(lines)
			if hasEnd {
				end = int(endFloat)
				if end > len(lines) {
					end = len(lines)
				}
			}

			// Adjust start to be within bounds
			if start > len(lines) {
				start = len(lines)
			}

			// Return empty string if the range is invalid
			if start > end {
				return "", fmt.Errorf("start line (%d) is after end line (%d)", start, end)
			}

			// Get the subset of lines
			resultLines := lines[start-1 : end]

			// Format lines with line numbers
			var result strings.Builder
			for _, line := range resultLines {
				result.WriteString(line)
				result.WriteString("\n")
			}

			return result.String(), nil
		},
	}
}
