package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

type ModelConfig struct {
	Model       string
	Temperature float64
}

type Agent struct {
	Name             string
	systemPrompt     string
	tools            []Tool
	Messages         []Message
	config           ModelConfig
	client           *Client
	toolCallCallback func(ctx context.Context, toolName string, args map[string]any) (string, error)
	messageCallback  func(message string)
}

func NewAgent(name string,
	systemPrompt string,
	tools []Tool, config ModelConfig,
	client *Client,
	toolCallCallback func(ctx context.Context, toolName string, args map[string]any) (string, error),
	messageCallback func(message string),
) *Agent {
	messages := []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	return &Agent{
		Name:             name,
		systemPrompt:     systemPrompt,
		tools:            tools,
		Messages:         messages,
		config:           config,
		client:           client,
		toolCallCallback: toolCallCallback,
		messageCallback:  messageCallback,
	}
}

func (a *Agent) ClearContext() {
	a.Messages = []Message{
		{
			Role:    "system",
			Content: a.systemPrompt,
		},
	}
}

func (a *Agent) AddMessage(role string, content string) {
	a.Messages = append(a.Messages, Message{
		Role:    role,
		Content: content,
	})
}

// Run executes the agent's logic, executes the tools, and returns the final message
func (a *Agent) Run(ctx context.Context) (Message, error) {
	for {
		select {
		case <-ctx.Done():
			return Message{}, fmt.Errorf("operation interrupted")
		default:
			// Continue processing
		}

		// Create chat completion request with tools
		req := ChatCompletionRequest{
			Model:       a.config.Model,
			Temperature: a.config.Temperature,
			Messages:    a.Messages,
			Tools:       a.tools,
		}

		response, err := a.client.CreateChatCompletion(ctx, req)

		if err != nil {
			// Check if the error was due to context cancellation
			if ctx.Err() != nil {
				return Message{}, fmt.Errorf("operation interrupted")
			}
			return Message{}, fmt.Errorf("calling API: %w", err)
		}

		if len(response.Choices) == 0 {
			return Message{}, fmt.Errorf("no response choices")
		}

		choice := response.Choices[0]
		message := choice.Message
		a.Messages = append(a.Messages, message)

		if a.messageCallback != nil {
			a.messageCallback(message.Content)
		}

		// Handle response based on finish reason
		if choice.FinishReason == "tool_calls" && len(message.ToolCalls) > 0 {
			// Handle tool calls
			err = a.handleToolCalls(ctx, message.ToolCalls)
			if err != nil {
				return Message{}, fmt.Errorf("handling tool calls: %w", err)
			}
		} else if choice.FinishReason == "stop" {
			return message, nil
		} else if choice.FinishReason == "length" {
			// Handle length limit reached
			a.AddMessage("user", "Please continue exactly from where you left off.")
		}
	}
}

// handleToolCalls processes tool calls from the LLM
func (a *Agent) handleToolCalls(ctx context.Context, toolCalls []ToolCall) error {
	if a.toolCallCallback == nil {
		return fmt.Errorf("tool call callback not set")
	}

	for _, toolCall := range toolCalls {
		// Check if the context has been cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("tool execution interrupted")
		default:
			// Continue processing
		}

		toolName := toolCall.Function.Name
		toolArgs := toolCall.Function.Arguments

		// Parse tool arguments
		var args map[string]any
		err := json.Unmarshal([]byte(toolArgs), &args)

		if err != nil {
			// Add tool response to Messages
			toolResponse := fmt.Sprintf("Error parsing arguments for %a: %a", toolName, err.Error())
			a.Messages = append(a.Messages, Message{
				Role:       "tool",
				Content:    toolResponse,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		result, err := a.toolCallCallback(ctx, toolName, args)
		if err != nil {
			return fmt.Errorf("tool call failed: %w", err)
		}
		a.Messages = append(a.Messages, Message{
			Role:       "tool",
			Content:    result,
			ToolCallID: toolCall.ID,
		})
	}

	return nil
}

func (a *Agent) Clone() *Agent {
	clone := *a
	clone.Messages = make([]Message, len(a.Messages))
	copy(clone.Messages, a.Messages)
	return &clone
}
