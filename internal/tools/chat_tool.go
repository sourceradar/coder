package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/recrsn/coder/internal/llm"
)

// NewChatTool creates a new chat tool for executing LLM calls
func NewChatTool(registry *Registry) Tool[map[string]any, map[string]any] {
	return Tool[map[string]any, map[string]any]{
		Name:        "chat",
		Description: "Chat with an LLM to get help with programming tasks",
		Usage:       "chat --message=\"How do I parse JSON in Go?\" --model=\"gpt-4o\"",
		Example:     "chat --message=\"Write a function to check if a string is a palindrome in Python\" --model=\"gpt-3.5-turbo\"",
		InputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"message": {
					Type:        "string",
					Description: "The message to send to the LLM",
				},
				"model": {
					Type:        "string",
					Description: "The model to use (default: gpt-3.5-turbo)",
				},
				"api_key": {
					Type:        "string",
					Description: "OpenAI API key (will use config if not provided)",
				},
				"temperature": {
					Type:        "number",
					Description: "Temperature for response generation (default: 0.7)",
				},
				"max_tokens": {
					Type:        "integer",
					Description: "Maximum tokens in the response (default: 1000)",
				},
				"system_prompt": {
					Type:        "string",
					Description: "System prompt to use",
				},
			},
			Required: []string{"message"},
		},
		OutputSchema: Schema{
			Type: "object",
			Properties: map[string]Property{
				"response": {
					Type:        "string",
					Description: "LLM response text",
				},
				"tool_calls": {
					Type:        "array",
					Description: "Tool calls executed by the LLM",
					Items: &PropertyItems{
						Type: "object",
					},
				},
				"model": {
					Type:        "string",
					Description: "Model that generated the response",
				},
			},
			Required: []string{"response", "model"},
		},
		Execute: func(input map[string]any) (map[string]any, error) {
			message := input["message"].(string)

			model := "gpt-3.5-turbo"
			if modelInput, ok := input["model"].(string); ok && modelInput != "" {
				model = modelInput
			}

			apiKey := ""
			if keyInput, ok := input["api_key"].(string); ok && keyInput != "" {
				apiKey = keyInput
			} else {
				// In a real implementation, you would get the API key from config
				return nil, errors.New("API key not provided and not found in config")
			}

			temperature := 0.7
			if tempInput, ok := input["temperature"].(float64); ok {
				temperature = tempInput
			}

			maxTokens := 1000
			if tokensInput, ok := input["max_tokens"].(int); ok {
				maxTokens = tokensInput
			}

			systemPrompt := "You are Coder, a helpful AI assistant for programming tasks. You provide clear, concise code examples and explanations."
			if promptInput, ok := input["system_prompt"].(string); ok && promptInput != "" {
				systemPrompt = promptInput
			}

			client := llm.NewClient(apiKey)

			// Create messages array with system prompt and user message
			messages := []llm.Message{
				{
					Role:    "system",
					Content: systemPrompt,
				},
				{
					Role:    "user",
					Content: message,
				},
			}

			// Define available tools based on registry
			tools := make([]llm.Tool, 0)
			for _, toolName := range registry.ListTools() {
				if toolInterface, ok := registry.Get(toolName); ok {
					if tool, ok := toolInterface.(Tool[map[string]any, map[string]any]); ok {
						// Convert tool to OpenAI tool format
						params := convertSchemaToJsonSchema(tool.InputSchema)
						tools = append(tools, llm.Tool{
							Type: "function",
							Function: llm.FunctionDefinition{
								Name:        toolName,
								Description: tool.Description,
								Parameters:  params,
							},
						})
					}
				}
			}

			// Create chat completion request
			req := llm.ChatCompletionRequest{
				Model:       model,
				Messages:    messages,
				Tools:       tools,
				Temperature: temperature,
				MaxTokens:   maxTokens,
			}

			// Call the OpenAI API
			resp, err := client.CreateChatCompletion(req)
			if err != nil {
				return nil, fmt.Errorf("calling API: %w", err)
			}

			// Handle tool calls if any
			toolCallResults := make([]map[string]any, 0)
			if len(resp.Choices) > 0 && len(resp.Choices[0].ToolCalls) > 0 {
				for _, toolCall := range resp.Choices[0].ToolCalls {
					toolName := toolCall.Function.Name
					toolArgs := toolCall.Function.Arguments

					// Parse tool arguments
					var args map[string]any
					if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
						toolCallResults = append(toolCallResults, map[string]any{
							"tool":   toolName,
							"args":   toolArgs,
							"error":  fmt.Sprintf("Failed to parse args: %s", err.Error()),
							"result": nil,
						})
						continue
					}

					// Execute the tool
					toolInterface, ok := registry.Get(toolName)
					if !ok {
						toolCallResults = append(toolCallResults, map[string]any{
							"tool":   toolName,
							"args":   args,
							"error":  "Tool not found",
							"result": nil,
						})
						continue
					}

					if tool, ok := toolInterface.(Tool[map[string]any, map[string]any]); ok {
						result, err := tool.Run(args)
						if err != nil {
							toolCallResults = append(toolCallResults, map[string]any{
								"tool":   toolName,
								"args":   args,
								"error":  err.Error(),
								"result": nil,
							})
						} else {
							toolCallResults = append(toolCallResults, map[string]any{
								"tool":   toolName,
								"args":   args,
								"error":  nil,
								"result": result,
							})
						}
					} else {
						toolCallResults = append(toolCallResults, map[string]any{
							"tool":   toolName,
							"args":   args,
							"error":  "Invalid tool type",
							"result": nil,
						})
					}
				}
			}

			// Extract response text
			responseText := ""
			if len(resp.Choices) > 0 {
				responseText = resp.Choices[0].Message.Content
			}

			return map[string]any{
				"response":   responseText,
				"tool_calls": toolCallResults,
				"model":      model,
			}, nil
		},
	}
}

// convertSchemaToJsonSchema converts our schema to JSON Schema format for OpenAI
func convertSchemaToJsonSchema(schema Schema) map[string]any {
	properties := make(map[string]any)
	for name, prop := range schema.Properties {
		propSchema := map[string]any{
			"type":        prop.Type,
			"description": prop.Description,
		}
		
		// Handle array items if present
		if prop.Type == "array" && prop.Items != nil {
			propSchema["items"] = map[string]any{
				"type": prop.Items.Type,
			}
		}
		
		// Handle enum values if present
		if len(prop.Enum) > 0 {
			propSchema["enum"] = prop.Enum
		}
		
		// Handle format if present
		if prop.Format != "" {
			propSchema["format"] = prop.Format
		}
		
		properties[name] = propSchema
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   schema.Required,
	}
}
