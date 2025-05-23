package chat

import (
	"bufio"
	"context"
	"fmt"
	"github.com/recrsn/coder/internal/chat/prompts"
	"github.com/recrsn/coder/internal/common"
	"github.com/recrsn/coder/internal/config"
	"github.com/recrsn/coder/internal/llm"
	"github.com/recrsn/coder/internal/platform"
	"github.com/recrsn/coder/internal/tools"
	"github.com/recrsn/coder/internal/ui"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session represents a chat session
type Session struct {
	ui                ui.UserInterface
	config            config.Config
	registry          *tools.Registry
	agent             *llm.Agent
	client            *llm.Client
	history           []string
	historyFile       string
	apiLogger         llm.APILogger
	permissionManager *common.PermissionManager
	// For cancellation
	cancelFunc context.CancelFunc
}

// NewSession creates a new chat session
func NewSession(userInterface ui.UserInterface, cfg config.Config, registry *tools.Registry, client *llm.Client, permissionManager *common.PermissionManager) (*Session, error) {
	// Set up history file in config directory
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "/tmp" // Fallback
	}
	configDir := filepath.Join(userConfigDir, "coder")
	historyFile := filepath.Join(configDir, "history")

	apiLogger := llm.NewAPILogger(configDir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Warning: couldn't create config directory: %v\n", err)
	}

	session := &Session{
		ui:                userInterface,
		config:            cfg,
		registry:          registry,
		client:            client,
		history:           []string{},
		historyFile:       historyFile,
		apiLogger:         apiLogger,
		permissionManager: permissionManager,
	}

	return session, nil
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

		s.agent.AddMessage("user", userInput)
		s.addToHistory(userInput)
		s.saveHistory()

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
		summary, err := s.SummarizeMessages()
		if err != nil {
			return fmt.Errorf("summarizing messages: %w", err)
		}
		s.ui.PrintAssistantMessage(summary)
		s.ui.PrintSuccess("Conversation summarized and added to context")
		return nil
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

// processUserMessage processes a user message and gets a response
func (s *Session) processUserMessage() error {
	// Create a new context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Store the cancel function so it can be called if the user interrupts
	s.cancelFunc = cancel

	// Start the conversation flow, which will handle all tool calls
	// and yield the final response only when the conversation is complete
	_, err := s.agent.Run(ctx)

	if err != nil {
		s.ui.PrintError(fmt.Sprintf("Error processing message: %v", err))
	}

	// Clear the cancel function when done
	s.cancelFunc = nil

	return err
}

// HandleToolCalls processes tool calls from the LLM
func (s *Session) HandleToolCalls(ctx context.Context, toolName string, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("tool execution interrupted")
	default:
		// Continue processing
	}

	tool, ok := s.registry.Get(toolName)
	if !ok {
		errorMsg := fmt.Sprintf("Tool not found: %s", toolName)
		s.ui.PrintError(errorMsg)
		return errorMsg, nil
	}

	// Create permission request
	result := tool.Explain(args)
	request := common.PermissionRequest{
		ToolName:  toolName,
		Arguments: args,
		Title:     result.Title,
		Context:   result.Context,
	}

	response := s.permissionManager.RequestPermission(request)
	execute := response.Granted
	alternate := response.AlternateAction

	if execute {
		result, err := tool.Run(args)

		// Print result or error
		if err != nil {
			s.ui.PrintToolCall(toolName, args, "", err)
			return fmt.Sprintf("Error executing %s: %s", toolName, err.Error()), nil
		} else {
			s.ui.PrintToolCall(toolName, args, result, nil)
			return result, nil
		}
	}

	// If the user chooses not to execute, we can either return an alternate response
	return "The user doesn't want to proceed with this tool use. " +
		"The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file)." +
		"STOP what you are doing and do this instead\n" + alternate, nil
}

// GetSystemPrompt loads and renders the system prompt
func GetSystemPrompt(registry *tools.Registry) (string, error) {
	toolsList := registry.GetAll()

	platformInfo := platform.GetPlatformInfo()
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	promptData := prompts.PromptData{
		KnowsTools:       len(toolsList) > 0,
		Tools:            toolsList,
		Platform:         fmt.Sprintf("%s %s (%s)", platformInfo.Name, platformInfo.Version, platformInfo.Arch),
		Date:             time.Now().Format("2006-01-02"),
		WorkingDirectory: dir,
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

// HandleMessage handles messages from the agent
func (s *Session) HandleMessage(message string) {
	s.ui.PrintAssistantMessage(message)
}

// SetAgent sets the agent for this session
func (s *Session) SetAgent(agent *llm.Agent) {
	s.agent = agent
}
