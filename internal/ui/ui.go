package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/recrsn/coder/internal/config"

	md "github.com/MichaelMure/go-term-markdown"
	"github.com/chzyer/readline"
	"github.com/pterm/pterm"
)

// UI handles the terminal user interface
type UI struct {
	config      config.UIConfig
	readline    *readline.Instance
	exitHandler func()
}

// NewUI creates a new UI instance
func NewUI(cfg config.UIConfig, exitHandler func()) (*UI, error) {
	// Configure PTerm based on config
	if !cfg.ColorEnabled {
		pterm.DisableColor()
	}

	// Create history file path
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %v", err)
	}

	historyFile := configDir + "/history"

	// Configure readline with history support and path completion
	rlConfig := &readline.Config{
		Prompt:          "> ",
		HistoryFile:     historyFile,
		HistoryLimit:    1000,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    GetPathCompleter(),
	}

	instance, err := readline.NewEx(rlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create readline instance: %v", err)
	}

	return &UI{
		config:      cfg,
		readline:    instance,
		exitHandler: exitHandler,
	}, nil
}

// ShowHeader displays the application header
func (u *UI) ShowHeader() {
	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).WithMargin(10)
	header.Println("Coder - Your Programming Sidekick")

	// Show info about API logging
	configDir, _ := getConfigDir()
	if configDir != "" {
		pterm.Info.Println("API requests and responses are being logged to the logs directory: " + configDir + "/logs/")
	}
}

// getConfigDir gets the config directory path and ensures it exists
func getConfigDir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "/tmp"
	}

	configDir := userConfigDir + "/coder"

	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	return configDir, nil
}

// StartSpinner starts a spinner with the given text
func (u *UI) StartSpinner(text string) *pterm.SpinnerPrinter {
	if !u.config.ShowSpinner {
		fmt.Println(text + "...")
		return nil
	}

	spinner, _ := pterm.DefaultSpinner.Start(text)
	return spinner
}

// StopSpinner stops a spinner with success
func (u *UI) StopSpinner(spinner *pterm.SpinnerPrinter, text string) {
	if spinner == nil {
		fmt.Println(text)
		return
	}

	spinner.Success(text)
}

// StopSpinnerFail stops a spinner with failure
func (u *UI) StopSpinnerFail(spinner *pterm.SpinnerPrinter, text string) {
	if spinner == nil {
		pterm.Error.Println(text)
		return
	}

	spinner.Fail(text)
}

// PrintUserMessage prints a user message
func (u *UI) PrintUserMessage(message string) {
	pterm.FgLightGreen.Println("You: " + message)
}

// parseMarkdown processes basic markdown formatting
func parseMarkdown(text string) string {
	return string(md.Render(text, 80, 0))
}

// PrintAssistantMessage prints an assistant message with markdown formatting
func (u *UI) PrintAssistantMessage(message string) {
	fmt.Print("$ ")
	fmt.Println(parseMarkdown(message))
}

// PrintCodeBlock prints a code block with a highlighted box
func (u *UI) PrintCodeBlock(code, language string) {
	fmt.Println()
	pterm.DefaultBox.WithTitle(language).Println(code)
	fmt.Println()
}

// PrintToolCall prints information about a tool call
func (u *UI) PrintToolCall(toolName string, args map[string]any, result string, err error) {
	panel := pterm.DefaultBox.WithTitle("Tool: " + toolName)

	var content strings.Builder

	// Print arguments
	content.WriteString("Arguments:\n")
	for k, v := range args {
		content.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
	}

	// Print error if any
	if err != nil {
		content.WriteString("\nError: " + err.Error() + "\n")
	} else if result != "" {
		// Print result as text
		content.WriteString("\nResult:\n" + result + "\n")
	}

	panel.Println(content.String())
}

// PrintHelp prints the help message
func (u *UI) PrintHelp() {
	table := pterm.TableData{
		{"Command", "Description"},
		{"/help", "Show this help message"},
		{"/exit", "Exit the application"},
		{"/clear", "Clear the screen"},
		{"/config", "Show or edit configuration"},
		{"/tools", "List available tools"},
		{"/prompt", "Edit the prompt template"},
		{"/version", "Show version information"},
		{"Ctrl+C", "Interrupt current operation"},
		{"Ctrl+D", "Exit the application"},
	}

	err := pterm.DefaultTable.WithHasHeader().WithData(table).Render()
	if err != nil {
		return
	}
}

// PrintError prints an error message
func (u *UI) PrintError(message string) {
	pterm.Error.Println(message)
}

// PrintSuccess prints a success message
func (u *UI) PrintSuccess(message string) {
	pterm.Success.Println(message)
}

// PrintInfo prints an informational message
func (u *UI) PrintInfo(message string) {
	pterm.Info.Println(message)
}

// AskInput asks for user input with a prompt
func (u *UI) AskInput(prompt string) string {
	u.readline.SetPrompt(prompt)
	defer u.readline.SetPrompt("> ")

	text, err := u.readline.Readline()
	if err != nil {
		if err == io.EOF && u.exitHandler != nil {
			fmt.Println("exit")
			u.exitHandler()
			return "/exit"
		}
		if err == readline.ErrInterrupt {
			fmt.Println("^C")
			return "/interrupt"
		}
		pterm.Error.Println("Error reading input:", err)
		return ""
	}

	// Add non-empty, non-command input to history
	if text != "" && !strings.HasPrefix(text, "/") {
		u.readline.SaveHistory(text)
	}

	return text
}

// AskMultiLineInput asks for multi-line user input with a prompt
func (u *UI) AskMultiLineInput(prompt string) string {
	text, _ := pterm.DefaultInteractiveTextInput.WithMultiLine(true).Show(prompt)
	return text
}

// ClearScreen clears the terminal screen
func (u *UI) ClearScreen() {
	pterm.DefaultArea.Clear()
}

func (u *UI) AskToolCallConfirmation(explanation string) (bool, string) {
	pterm.DefaultBox.WithTitle("Confirm tool call").
		Println(explanation)

	confirmation, _ := pterm.DefaultInteractiveConfirm.
		WithRejectText("No, and tell what to do instead").
		WithDefaultText(explanation).
		Show()

	if confirmation {
		return true, ""
	}

	return false, u.AskInput("What should I do instead?")
}
