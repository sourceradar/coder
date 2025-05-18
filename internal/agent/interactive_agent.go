package agent

import (
	"context"
	"fmt"
	"github.com/recrsn/coder/internal/llm"
	"github.com/recrsn/coder/internal/tools"
)

// UICallbacks contains functions for interacting with the UI
type UICallbacks struct {
	// PrintMessage displays a message from the agent
	PrintMessage func(message string)

	// PrintToolCall displays a tool call execution
	PrintToolCall func(toolName string, args map[string]any, result string, err error)

	// AskToolCallConfirmation asks for confirmation before executing a tool
	// Returns whether to execute the tool and any alternative instructions
	AskToolCallConfirmation func(explanation string) (bool, string)
}

// InteractiveAgent wraps an llm.Agent and provides UI integration
type InteractiveAgent struct {
	name        string
	agent       *llm.Agent
	registry    *tools.Registry
	uiCallbacks UICallbacks
	cancelFunc  context.CancelFunc
}

// NewInteractiveAgent creates a new interactive agent
func NewInteractiveAgent(
	name string,
	systemPrompt string,
	registry *tools.Registry,
	client *llm.Client,
	uiCallbacks UICallbacks,
	config llm.ModelConfig,
) *InteractiveAgent {
	ia := &InteractiveAgent{
		name:        name,
		registry:    registry,
		uiCallbacks: uiCallbacks,
	}

	// Create the underlying agent
	agent := llm.NewAgent(
		name,
		systemPrompt,
		registry.ListTools(),
		config,
		client,
		ia.handleToolCalls, // Tool call callback
		ia.handleMessage,   // Message callback
	)

	ia.agent = agent
	return ia
}

// AddMessage adds a message to the agent
func (ia *InteractiveAgent) AddMessage(role string, content string) {
	ia.agent.AddMessage(role, content)
}

// ClearContext clears the agent's conversation context
func (ia *InteractiveAgent) ClearContext() {
	ia.agent.ClearContext()
}

// Run executes the agent with the current context
func (ia *InteractiveAgent) Run(ctx context.Context) (llm.Message, error) {
	// Create a cancellable context
	runCtx, cancel := context.WithCancel(ctx)
	ia.cancelFunc = cancel

	// Run the agent and capture the result
	result, err := ia.agent.Run(runCtx)

	// Clear the cancel function when done
	ia.cancelFunc = nil

	return result, err
}

// Interrupt interrupts the current agent execution if running
func (ia *InteractiveAgent) Interrupt() bool {
	if ia.cancelFunc != nil {
		ia.cancelFunc()
		return true
	}
	return false
}

// handleMessage displays messages from the agent
func (ia *InteractiveAgent) handleMessage(message string) {
	if ia.uiCallbacks.PrintMessage != nil {
		ia.uiCallbacks.PrintMessage(message)
	}
}

// handleToolCalls processes tool calls from the agent
func (ia *InteractiveAgent) handleToolCalls(ctx context.Context, toolName string, args map[string]any) (string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("tool execution interrupted")
	default:
		// Continue processing
	}

	// Get the tool from the registry
	tool, ok := ia.registry.Get(toolName)
	if !ok {
		errorMsg := fmt.Sprintf("Tool not found: %s", toolName)
		return errorMsg, nil
	}

	// Ask for confirmation if callback is provided
	var execute bool = true
	var alternate string = ""

	if ia.uiCallbacks.AskToolCallConfirmation != nil {
		execute, alternate = ia.uiCallbacks.AskToolCallConfirmation(tool.Explain(args))
	}

	if execute {
		// Execute the tool
		result, err := tool.Run(args)

		// Display the result if callback is provided
		if ia.uiCallbacks.PrintToolCall != nil {
			ia.uiCallbacks.PrintToolCall(toolName, args, result, err)
		}

		// Return result or error message
		if err != nil {
			return fmt.Sprintf("Error executing %s: %s", toolName, err.Error()), nil
		}
		return result, nil
	}

	// Return alternate instructions if execution was rejected
	return "The user doesn't want to proceed with this tool use. " +
		"The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file)." +
		"STOP what you are doing and do this instead\n" + alternate, nil
}

// GetMessages returns the agent's message history
func (ia *InteractiveAgent) GetMessages() []llm.Message {
	return ia.agent.Messages
}
