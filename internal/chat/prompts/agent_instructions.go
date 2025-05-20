package prompts

import (
	"os"
	"path/filepath"
)

// GetAgentInstructions reads instructions from AGENT.md or AGENTS.md file in the current directory
// Returns empty string if no instruction files are found
func GetAgentInstructions(workingDir string) string {
	// Check for AGENT.md
	agentMdPath := filepath.Join(workingDir, "AGENT.md")
	if content, err := os.ReadFile(agentMdPath); err == nil {
		return string(content)
	}

	// Check for AGENTS.md
	agentsMdPath := filepath.Join(workingDir, "AGENTS.md")
	if content, err := os.ReadFile(agentsMdPath); err == nil {
		return string(content)
	}

	return ""
}
