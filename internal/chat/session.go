package chat

import (
	"encoding/json"
	"fmt"
	"github.com/recrsn/coder/internal/config"
	"github.com/recrsn/coder/internal/llm"
	"github.com/recrsn/coder/internal/prompts"
	tools2 "github.com/recrsn/coder/internal/tools"
	"github.com/recrsn/coder/internal/ui"
	"path/filepath"
	"strings"
)

// Session represents a chat session
type Session struct {
	ui            *ui.UI
	config        config.Config
	registry      *tools2.Registry
	messages      []llm.Message
	client        *llm.OpenAIClient
	promptManager *prompts.Manager
}

// NewSession creates a new chat session
func NewSession(userInterface *ui.UI, cfg config.Config, registry *tools2.Registry) (*Session, error) {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("getting data directory: %w", err)
	}

	// Create prompts directory
	promptsDir := filepath.Join(dataDir, "prompts")

	// Initialize prompt manager
	promptManager := prompts.NewManager(promptsDir)

	// Ensure default prompt exists
	if err := promptManager.EnsureDefaultPromptExists(); err != nil {
		return nil, fmt.Errorf("ensuring default prompt: %w", err)
	}

	// Create OpenAI client
	client := llm.NewClient(cfg.Provider.APIKey)

	// Load and render system prompt
	systemPrompt, err := getSystemPrompt(promptManager, cfg, registry)
	if err != nil {
		return nil, fmt.Errorf("getting system prompt: %w", err)
	}

	// Initialize messages with system prompt
	messages := []llm.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	return &Session{
		ui:            userInterface,
		config:        cfg,
		registry:      registry,
		messages:      messages,
		client:        client,
		promptManager: promptManager,
	}, nil
}

// Start starts the chat session
func (s *Session) Start() error {
	s.ui.ShowHeader()
	s.ui.PrintSuccess("Welcome to Coder! Type your programming questions or /help for commands.")

	for {
		// Get user input
		userInput := s.ui.AskInput("> ")
		userInput = strings.TrimSpace(userInput)

		if userInput == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(userInput, "/") {
			if err := s.handleCommand(userInput); err != nil {
				if err.Error() == "exit" {
					return nil
				}
				s.ui.PrintError(fmt.Sprintf("Error executing command: %v", err))
			}
			continue
		}

		// Display user message
		s.ui.PrintUserMessage(userInput)

		// Add user message to history
		s.messages = append(s.messages, llm.Message{
			Role:    "user",
			Content: userInput,
		})

		// Process user message
		if err := s.processUserMessage(); err != nil {
			s.ui.PrintError(fmt.Sprintf("Error processing message: %v", err))
		}
	}
}

// handleCommand handles chat commands
func (s *Session) handleCommand(cmd string) error {
	parts := strings.SplitN(cmd, " ", 2)
	command := parts[0]

	switch command {
	case "/help":
		s.ui.PrintHelp()
		return nil
	case "/exit":
		s.ui.PrintSuccess("Goodbye!")
		return fmt.Errorf("exit")
	case "/clear":
		s.ui.ClearScreen()
		return nil
	case "/config":
		// Show config command
		if len(parts) == 1 {
			// Print current config
			configJSON, err := json.MarshalIndent(s.config, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(configJSON))
			return nil
		}

		// Edit config command
		subCommand := parts[1]
		return s.handleConfigCommand(subCommand)
	case "/tools":
		// List available tools
		fmt.Println("Available tools:")
		for _, name := range s.registry.ListTools() {
			if toolInterface, ok := s.registry.Get(name); ok {
				if tool, ok := toolInterface.(tools2.Tool[map[string]any, map[string]any]); ok {
					fmt.Printf("- %s: %s\n", name, tool.Description)
				}
			}
		}
		return nil
	case "/version":
		fmt.Println("Coder v0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleConfigCommand handles config-related commands
func (s *Session) handleConfigCommand(subCommand string) error {
	// Parse key=value format
	parts := strings.SplitN(subCommand, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid config command format, use: /config key=value")
	}

	key := parts[0]
	value := parts[1]

	switch key {
	case "provider.api_key":
		s.config.Provider.APIKey = value
		// Update client with new API key
		s.client = llm.NewClient(value)
	case "provider.model":
		s.config.Provider.Model = value
	case "provider.temperature":
		var temp float64
		if _, err := fmt.Sscanf(value, "%f", &temp); err != nil {
			return fmt.Errorf("invalid temperature value: %s", value)
		}
	case "ui.color_enabled":
		var enabled bool
		if value == "true" {
			enabled = true
		} else if value == "false" {
			enabled = false
		} else {
			return fmt.Errorf("invalid boolean value: %s, use true or false", value)
		}
		s.config.UI.ColorEnabled = enabled
	case "ui.show_spinner":
		var enabled bool
		if value == "true" {
			enabled = true
		} else if value == "false" {
			enabled = false
		} else {
			return fmt.Errorf("invalid boolean value: %s, use true or false", value)
		}
		s.config.UI.ShowSpinner = enabled
	case "prompt.template_file":
		s.config.Prompt.TemplateFile = value

		// Update system message with new prompt
		systemPrompt, err := getSystemPrompt(s.promptManager, s.config, s.registry)
		if err != nil {
			return fmt.Errorf("updating system prompt: %w", err)
		}

		// Update first message in history (system message)
		if len(s.messages) > 0 && s.messages[0].Role == "system" {
			s.messages[0].Content = systemPrompt
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	// Save updated config
	if err := config.SaveConfig(s.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	s.ui.PrintSuccess(fmt.Sprintf("Config updated: %s = %s", key, value))
	return nil
}

// processUserMessage processes a user message and gets a response
func (s *Session) processUserMessage() error {
	// Start spinner
	spinner := s.ui.StartSpinner("Thinking...")

	// Define available tools based on registry
	var tools []llm.Tool

	for _, name := range s.registry.ListTools() {
		if toolInterface, ok := s.registry.Get(name); ok {
			if tool, ok := toolInterface.(tools.Tool[map[string]any, map[string]any]); ok {
				// Convert tool schema to parameter schema
				params := convertSchemaToJsonSchema(tool.InputSchema)
				tools = append(tools, llm.Tool{
					Type: "function",
					Function: llm.FunctionDefinition{
						Name:        name,
						Description: tool.Description,
						Parameters:  params,
					},
				})
			}
		}
	}

	// Create chat completion request
	req := llm.ChatCompletionRequest{
		Model:    s.config.Provider.Model,
		Messages: s.messages,
		Tools:    tools,
	}

	// Send request to OpenAI
	resp, err := s.client.CreateChatCompletion(req)
	if err != nil {
		s.ui.StopSpinnerFail(spinner, "Failed to get response")
		return fmt.Errorf("calling API: %w", err)
	}

	// Handle the response
	if len(resp.Choices) == 0 {
		s.ui.StopSpinnerFail(spinner, "No response received")
		return fmt.Errorf("no response choices")
	}

	choice := resp.Choices[0]

	// Add assistant message to history
	s.messages = append(s.messages, choice.Message)

	// Stop spinner
	s.ui.StopSpinner(spinner, "Response received")

	// Handle response based on finish reason
	if choice.FinishReason == "tool_calls" && len(choice.ToolCalls) > 0 {
		// Handle tool calls
		return s.handleToolCalls(choice.ToolCalls)
	} else {
		// Print assistant message
		s.ui.PrintAssistantMessage(choice.Message.Content)
	}

	return nil
}

// handleToolCalls processes tool calls from the LLM
func (s *Session) handleToolCalls(toolCalls []llm.ToolCall) error {
	for _, toolCall := range toolCalls {
		toolName := toolCall.Function.Name
		toolArgs := toolCall.Function.Arguments

		// Parse tool arguments
		var args map[string]any
		if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
			s.ui.PrintError(fmt.Sprintf("Failed to parse tool args: %v", err))

			// Add tool response to messages
			toolResponse := fmt.Sprintf("Error parsing arguments for %s: %s", toolName, err.Error())
			s.messages = append(s.messages, llm.Message{
				Role:    "tool",
				Content: toolResponse,
			})
			continue
		}

		// Get the tool
		toolInterface, ok := s.registry.Get(toolName)
		if !ok {
			s.ui.PrintError(fmt.Sprintf("Tool not found: %s", toolName))

			// Add tool response to messages
			toolResponse := fmt.Sprintf("Tool not found: %s", toolName)
			s.messages = append(s.messages, llm.Message{
				Role:    "tool",
				Content: toolResponse,
			})
			continue
		}

		// Execute the tool
		if tool, ok := toolInterface.(tools2.Tool[map[string]any, map[string]any]); ok {
			fmt.Printf("Executing tool: %s\n", toolName)

			// Execute the tool
			result, err := tool.Run(args)

			// Print result or error
			if err != nil {
				s.ui.PrintToolCall(toolName, args, nil, err)

				// Add tool response to messages
				toolResponse := fmt.Sprintf("Error executing %s: %s", toolName, err.Error())
				s.messages = append(s.messages, llm.Message{
					Role:    "tool",
					Content: toolResponse,
				})
			} else {
				s.ui.PrintToolCall(toolName, args, result, nil)

				// Convert result to string for tool response
				resultJSON, _ := json.Marshal(result)
				s.messages = append(s.messages, llm.Message{
					Role:    "tool",
					Content: string(resultJSON),
				})
			}
		} else {
			s.ui.PrintError(fmt.Sprintf("Invalid tool type for: %s", toolName))

			// Add tool response to messages
			toolResponse := fmt.Sprintf("Invalid tool type for: %s", toolName)
			s.messages = append(s.messages, llm.Message{
				Role:    "tool",
				Content: toolResponse,
			})
		}
	}

	// Get follow-up response
	return s.processFollowUp()
}

// processFollowUp gets a follow-up response after tool calls
func (s *Session) processFollowUp() error {
	// Start spinner
	spinner := s.ui.StartSpinner("Processing tool results...")

	// Create chat completion request without tools
	req := llm.ChatCompletionRequest{
		Model:    s.config.Provider.Model,
		Messages: s.messages,
	}

	// Send request to OpenAI
	resp, err := s.client.CreateChatCompletion(req)
	if err != nil {
		s.ui.StopSpinnerFail(spinner, "Failed to get follow-up response")
		return fmt.Errorf("calling API: %w", err)
	}

	// Handle the response
	if len(resp.Choices) == 0 {
		s.ui.StopSpinnerFail(spinner, "No follow-up response received")
		return fmt.Errorf("no response choices")
	}

	choice := resp.Choices[0]

	// Add assistant message to history
	s.messages = append(s.messages, choice.Message)

	// Stop spinner
	s.ui.StopSpinner(spinner, "Follow-up response received")

	// Print assistant message
	s.ui.PrintAssistantMessage(choice.Message.Content)

	// Check if there are more tool calls
	if choice.FinishReason == "tool_calls" && len(choice.ToolCalls) > 0 {
		return s.handleToolCalls(choice.ToolCalls)
	}

	return nil
}

// convertSchemaToJsonSchema converts our schema to JSON Schema format
func convertSchemaToJsonSchema(schema tools2.Schema) map[string]any {
	properties := make(map[string]any)
	for name, prop := range schema.Properties {
		propSchema := map[string]any{
			"type":        prop.Type,
			"description": prop.Description,
		}
		properties[name] = propSchema
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   schema.Required,
	}
}

// getSystemPrompt loads and renders the system prompt
func getSystemPrompt(promptManager *prompts.Manager, cfg config.Config, registry *tools2.Registry) (string, error) {
	// Load template content
	templateContent, err := promptManager.LoadPrompt(cfg.Prompt.TemplateFile)
	if err != nil {
		return "", fmt.Errorf("loading prompt template: %w", err)
	}

	// Get tools data for prompt
	toolsList := prompts.GetToolsForPrompt(registry)

	// Render template
	promptData := prompts.PromptData{
		KnowsTools: len(toolsList) > 0,
		Tools:      toolsList,
	}

	return promptManager.RenderPrompt(templateContent, promptData)
}
