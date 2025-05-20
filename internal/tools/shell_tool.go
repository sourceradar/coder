package tools

import (
	"fmt"
	"github.com/recrsn/coder/internal/schema"
	"os/exec"
)

// NewShellTool creates a tool to execute shell commands
func NewShellTool() *Tool {
	return &Tool{
		Name:        "shell",
		Description: "Execute shell commands",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"command": {
					Type:        "string",
					Description: "The shell command to execute",
				},
				"why": {
					Type:        "string",
					Description: "A very short reason for executing this command",
				},
			},
			Required: []string{"command"},
		},
		Explain: func(input map[string]any) ExplainResult {
			command, _ := input["command"].(string)
			why, _ := input["why"].(string)
			return ExplainResult{
				Title:   fmt.Sprintf("Shell(%s)", command),
				Context: why,
			}
		},
		Execute: func(input map[string]any) (string, error) {
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

				result := "Command: " + command + "\n"
				result += fmt.Sprintf("Exit Code: %d\n", exitCode)
				if len(stdout) > 0 {
					result += "\nStandard Output:\n" + string(stdout) + "\n"
				}
				if stderr != "" {
					result += "\nStandard Error:\n" + stderr + "\n"
				}
				return result, nil
			}

			result := "Command: " + command + "\n"
			result += "Exit Code: 0\n"
			if len(stdout) > 0 {
				result += "\nOutput:\n" + string(stdout) + "\n"
			}
			return result, nil
		},
	}
}
