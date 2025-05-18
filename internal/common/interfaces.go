package common

import (
	"context"
)

// ToolRegistry provides an interface for accessing tools
type ToolRegistry interface {
	// Get returns a tool by name
	Get(name string) (Tool, bool)

	// ListTools returns all available tools in a format suitable for LLM API
	ListTools() []any
}

// Tool represents a tool that can be executed
type Tool interface {
	// Explain returns a human-readable explanation of what the tool will do
	Explain(args map[string]any) string

	// Run executes the tool with the given arguments
	Run(args map[string]any) (string, error)
}

// Agent represents an interactive agent
type Agent interface {
	// AddMessage adds a message to the agent's conversation
	AddMessage(role string, content string)

	// ClearContext clears the agent's conversation context
	ClearContext()

	// Run executes the agent and returns the final response
	Run(ctx context.Context) (any, error)

	// Interrupt interrupts the current execution
	Interrupt() bool
}
