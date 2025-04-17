package tools

import (
	"os/exec"
)

// NewShellTool creates a tool to execute shell commands
func NewShellTool() Tool[map[string]any, map[string]any] {
	return Tool[map[string]any, map[string]any]{
		Name:        "shell",
		Description: "Execute shell commands",
		Usage:       "shell --command=\"ls -la\"",
		Example:     "shell --command=\"echo hello world\"",
		InputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"command": {
					Type:        "string",
					Description: "The shell command to execute",
				},
			},
			Required: []string{"command"},
		},
		OutputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"stdout": {
					Type:        "string",
					Description: "Command standard output",
				},
				"stderr": {
					Type:        "string",
					Description: "Command standard error",
				},
				"exitCode": {
					Type:        "integer",
					Description: "Command exit code",
				},
			},
			Required: []string{"stdout", "stderr", "exitCode"},
		},
		Execute: func(input map[string]any) (map[string]any, error) {
			command := input["command"].(string)
			cmd := exec.Command("sh", "-c", command)
			
			stdout, err := cmd.Output()
			if err != nil {
				var exitCode int
				var stderr string
				
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
					stderr = string(exitErr.Stderr)
				} else {
					exitCode = 1
					stderr = err.Error()
				}
				
				return map[string]any{
					"stdout":   string(stdout),
					"stderr":   stderr,
					"exitCode": exitCode,
				}, nil
			}
			
			return map[string]any{
				"stdout":   string(stdout),
				"stderr":   "",
				"exitCode": 0,
			}, nil
		},
	}
}