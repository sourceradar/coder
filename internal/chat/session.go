package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/recrsn/coder/internal/config"
	"github.com/recrsn/coder/internal/llm"
	"github.com/recrsn/coder/internal/logger"
	"github.com/recrsn/coder/internal/prompts"
	"github.com/recrsn/coder/internal/tools"
	"github.com/recrsn/coder/internal/ui"
	"os"
	"path/filepath"
	"strings"
)

// Session represents a chat session
type Session struct {
	ui          *ui.UI
	config      config.Config
	registry    *tools.Registry
	messages    []llm.Message
	client      *llm.OpenAIClient
	history     []string
	historyFile string
	apiLogger   *logger.APILogger
	// For cancellation
	cancelFunc context.CancelFunc
	// Stores the most recent summary of conversation
	conversationSummary string
}

// NewSession creates a new chat session
func NewSession(userInterface *ui.UI, cfg config.Config, registry *tools.Registry) (*Session, error) {

	// Create OpenAI client
	client := llm.NewClient(cfg.Provider.Endpoint, cfg.Provider.APIKey)

	// Load and render system prompt
	systemPrompt, err := getSystemPrompt(registry)
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

	// Set up history file in config directory
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "/tmp" // Fallback
	}
	configDir := filepath.Join(userConfigDir, "coder")
	historyFile := filepath.Join(configDir, "history")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Warning: couldn't create config directory: %v\n", err)
	}

	// Initialize API logger
	apiLogger := logger.NewAPILogger(configDir)

	return &Session{
		ui:                  userInterface,
		config:              cfg,
		registry:            registry,
		messages:            messages,
		client:              client,
		history:             []string{},
		historyFile:         historyFile,
		apiLogger:           apiLogger,
		conversationSummary: "",
	}, nil
}

// Start starts the chat session
func (s *Session) Start() error {
	s.ui.ShowHeader()
	s.ui.PrintSuccess("Welcome to Coder! Type your programming questions or /help for commands.")

	// Load history from file
	s.loadHistory()

	for {
		// Get user input with history support
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
				if err.Error() == "interrupt" {
					continue
				}
				s.ui.PrintError(fmt.Sprintf("Error executing command: %v", err))
			}
			continue
		}

		// Display user message
		s.ui.PrintUserMessage(userInput)

		// Add user message to messages list
		s.messages = append(s.messages, llm.Message{
			Role:    "user",
			Content: userInput,
		})

		// Add input to command history
		s.addToHistory(userInput)

		// Save history to file
		s.saveHistory()

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
	case "/interrupt":
		// Cancel any ongoing operations
		if s.cancelFunc != nil {
			s.cancelFunc()
			s.ui.PrintSuccess("Interrupted current operation")
		} else {
			s.ui.PrintSuccess("No operation to interrupt")
		}
		return fmt.Errorf("interrupt")
	case "/clear":
		s.ui.ClearScreen()
		return nil
	case "/summarize":
		// Summarize previous messages and add to context
		if err := s.AddSummaryToContext(); err != nil {
			return fmt.Errorf("summarizing messages: %w", err)
		}
		s.ui.PrintSuccess("Conversation summarized and added to context")
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
		for _, tool := range s.registry.ListTools() {
			fmt.Printf("- %s: %s\n", tool.Function.Name, tool.Function.Description)
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
	case "provider.endpoint":
		s.config.Provider.Endpoint = value
		// Update client with new endpoint
		s.client = llm.NewClient(value, s.config.Provider.APIKey)
	case "provider.api_key":
		s.config.Provider.APIKey = value
		// Update client with new API key
		s.client = llm.NewClient(s.config.Provider.Endpoint, value)
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
	// Prompt template file option is no longer needed with simplified implementation
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
	// Create a new context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Store the cancel function so it can be called if the user interrupts
	s.cancelFunc = cancel

	// Start the conversation flow, which will handle all tool calls
	// and yield the final response only when the conversation is complete
	err := s.continueConversation(ctx)

	// Clear the cancel function when done
	s.cancelFunc = nil

	return err
}

// handleToolCalls processes tool calls from the LLM
func (s *Session) handleToolCalls(ctx context.Context, toolCalls []llm.ToolCall) error {
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
		if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
			s.ui.PrintError(fmt.Sprintf("Failed to parse tool args: %v", err))

			// Add tool response to messages
			toolResponse := fmt.Sprintf("Error parsing arguments for %s: %s", toolName, err.Error())
			s.messages = append(s.messages, llm.Message{
				Role:       "tool",
				Content:    toolResponse,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		// Get the tool
		tool, ok := s.registry.Get(toolName)
		if !ok {
			s.ui.PrintError(fmt.Sprintf("Tool not found: %s", toolName))

			// Add tool response to messages
			toolResponse := fmt.Sprintf("Tool not found: %s", toolName)
			s.messages = append(s.messages, llm.Message{
				Role:       "tool",
				Content:    toolResponse,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		// Execute the tool
		fmt.Printf("Executing tool: %s\n", toolName)

		// Execute the tool
		result, err := tool.Run(args)

		// Print result or error
		if err != nil {
			s.ui.PrintToolCall(toolName, args, "", err)

			// Add tool response to messages
			toolResponse := fmt.Sprintf("Error executing %s: %s", toolName, err.Error())
			s.messages = append(s.messages, llm.Message{
				Role:       "tool",
				Content:    toolResponse,
				ToolCallID: toolCall.ID,
			})
		} else {
			s.ui.PrintToolCall(toolName, args, result, nil)

			// Result is already a formatted string, use directly
			s.messages = append(s.messages, llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
			})
		}
	}

	// Continue the conversation to get the next response
	return s.continueConversation(ctx)
}

// continueConversation continues the conversation with the LLM
// It sends the current messages to the LLM and processes the response
// If the response contains tool calls, it handles them recursively
// Only when the finish reason is "stop", it returns the final response
func (s *Session) continueConversation(ctx context.Context) error {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return fmt.Errorf("operation interrupted")
	default:
		// Continue processing
	}

	// Start spinner
	spinner := s.ui.StartSpinner("Processing...")

	// Create chat completion request with tools
	req := llm.ChatCompletionRequest{
		Model:    s.config.Provider.Model,
		Messages: s.messages,
		Tools:    s.registry.ListTools(),
	}

	// Send request to OpenAI with context for cancellation
	resp, apiErr := s.client.CreateChatCompletionWithContext(ctx, req)

	// Log the API interaction
	s.apiLogger.LogInteraction(req, resp, apiErr)

	if apiErr != nil {
		// Check if the error was due to context cancellation
		if ctx.Err() != nil {
			s.ui.StopSpinnerFail(spinner, "Operation interrupted")
			return fmt.Errorf("operation interrupted")
		}
		s.ui.StopSpinnerFail(spinner, "Failed to get response")
		return fmt.Errorf("calling API: %w", apiErr)
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
	if choice.FinishReason == "tool_calls" && len(choice.Message.ToolCalls) > 0 {
		// Handle tool calls
		return s.handleToolCalls(ctx, choice.Message.ToolCalls)
	} else if choice.FinishReason == "stop" {
		// Print assistant message for the final response
		if choice.Message.Content != "" {
			s.ui.PrintAssistantMessage(choice.Message.Content)
		}
		return nil
	} else {
		// Print assistant message for other finish reasons
		if choice.Message.Content != "" {
			s.ui.PrintAssistantMessage(choice.Message.Content)
		}
		return nil
	}
}

// getSystemPrompt loads and renders the system prompt
func getSystemPrompt(registry *tools.Registry) (string, error) {
	toolsList := registry.GetAll()

	promptData := prompts.PromptData{
		KnowsTools: len(toolsList) > 0,
		Tools:      toolsList,
	}

	return prompts.RenderSystemPrompt(promptData)
}

// loadHistory loads command history from the history file
func (s *Session) loadHistory() {
	// Check if file exists
	if _, err := os.Stat(s.historyFile); os.IsNotExist(err) {
		return
	}

	// Open file
	file, err := os.Open(s.historyFile)
	if err != nil {
		fmt.Printf("Warning: couldn't open history file: %v\n", err)
		return
	}
	defer file.Close()

	// Read line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			s.history = append(s.history, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Warning: error reading history file: %v\n", err)
	}
}

// saveHistory saves command history to the history file
func (s *Session) saveHistory() {
	// Create or truncate file
	file, err := os.Create(s.historyFile)
	if err != nil {
		fmt.Printf("Warning: couldn't create history file: %v\n", err)
		return
	}
	defer file.Close()

	// Write each history item on a new line
	writer := bufio.NewWriter(file)
	for _, cmd := range s.history {
		_, _ = writer.WriteString(cmd + "\n")
	}

	_ = writer.Flush()
}

// addToHistory adds a command to the history, avoiding duplicates
func (s *Session) addToHistory(cmd string) {
	// Don't add empty commands or duplicates of the most recent command
	if cmd == "" || (len(s.history) > 0 && s.history[len(s.history)-1] == cmd) {
		return
	}

	// Add to history, limiting to 1000 items
	s.history = append(s.history, cmd)
	if len(s.history) > 1000 {
		s.history = s.history[len(s.history)-1000:]
	}
}

// Exit gracefully exits the session
func (s *Session) Exit() {
	// Cancel any ongoing operations
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	s.ui.PrintSuccess("Goodbye!")
	s.saveHistory()
	os.Exit(0)
}

// GetConversationSummary returns the current conversation summary
// If no summary exists or it's outdated, it generates a new one
func (s *Session) GetConversationSummary() (string, error) {
	// If we already have a summary, return it
	if s.conversationSummary != "" {
		return s.conversationSummary, nil
	}

	// Otherwise, generate a new summary
	return s.SummarizeMessages()
}

// max returns the larger of x or y
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
