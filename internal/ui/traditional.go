package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/recrsn/coder/internal/config"

	md "github.com/MichaelMure/go-term-markdown"
	"github.com/chzyer/readline"
	"github.com/pterm/pterm"
)

// TraditionalUI handles the terminal user interface using pterm and readline
type TraditionalUI struct {
	config      config.UIConfig
	readline    *readline.Instance
	exitHandler func()
}

// NewTraditionalUI creates a new TraditionalUI instance
func NewTraditionalUI(cfg config.UIConfig, exitHandler func()) (*TraditionalUI, error) {
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

	return &TraditionalUI{
		config:      cfg,
		readline:    instance,
		exitHandler: exitHandler,
	}, nil
}

// ShowHeader displays the application header
func (u *TraditionalUI) ShowHeader() {
	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).WithMargin(10)
	header.Println("Coder - Your Programming Sidekick")

	// Show info about API logging
	configDir, _ := getConfigDir()
	if configDir != "" {
		pterm.Info.Println("API requests and responses are being logged to the logs directory: " + configDir + "/logs/")
	}
}

// StartSpinner starts a spinner with the given text
func (u *TraditionalUI) StartSpinner(text string) *pterm.SpinnerPrinter {
	if !u.config.ShowSpinner {
		fmt.Println(text + "...")
		return nil
	}

	spinner, _ := pterm.DefaultSpinner.Start(text)
	return spinner
}

// StopSpinner stops a spinner with success
func (u *TraditionalUI) StopSpinner(spinner *pterm.SpinnerPrinter, text string) {
	if spinner == nil {
		fmt.Println(text)
		return
	}

	spinner.Success(text)
}

// StopSpinnerFail stops a spinner with failure
func (u *TraditionalUI) StopSpinnerFail(spinner *pterm.SpinnerPrinter, text string) {
	if spinner == nil {
		pterm.Error.Println(text)
		return
	}

	spinner.Fail(text)
}

// PrintUserMessage prints a user message
func (u *TraditionalUI) PrintUserMessage(message string) {
	pterm.FgLightGreen.Println("You: " + message)
}

// parseMarkdown processes basic markdown formatting
func parseMarkdown(text string) string {
	return string(md.Render(text, 80, 0))
}

// PrintAssistantMessage prints an assistant message with markdown formatting
func (u *TraditionalUI) PrintAssistantMessage(message string) {
	fmt.Print("$ ")
	fmt.Println(parseMarkdown(message))
}

// PrintCodeBlock prints a code block with a highlighted box
func (u *TraditionalUI) PrintCodeBlock(code, language string) {
	fmt.Println()
	pterm.DefaultBox.WithTitle(language).Println(code)
	fmt.Println()
}

// PrintToolCall prints information about a tool call
func (u *TraditionalUI) PrintToolCall(toolName string, args map[string]any, result string, err error) {
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
func (u *TraditionalUI) PrintHelp() {
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
func (u *TraditionalUI) PrintError(message string) {
	pterm.Error.Println(message)
}

// PrintSuccess prints a success message
func (u *TraditionalUI) PrintSuccess(message string) {
	pterm.Success.Println(message)
}

// PrintInfo prints an informational message
func (u *TraditionalUI) PrintInfo(message string) {
	pterm.Info.Println(message)
}

// AskInput asks for user input with a prompt
func (u *TraditionalUI) AskInput(prompt string) string {
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
func (u *TraditionalUI) AskMultiLineInput(prompt string) string {
	text, _ := pterm.DefaultInteractiveTextInput.WithMultiLine(true).Show(prompt)
	return text
}

// ClearScreen clears the terminal screen
func (u *TraditionalUI) ClearScreen() {
	pterm.DefaultArea.Clear()
}

func (u *TraditionalUI) AskToolCallConfirmation(explanation string) (bool, string) {
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

// AskPermission asks the user for permission with a title and context
func (u *TraditionalUI) AskPermission(explanation string) (bool, string) {
	pterm.DefaultBox.WithTitle("Permission Request").
		Println(explanation)

	confirmation, _ := pterm.DefaultInteractiveConfirm.
		WithConfirmText("Yes, allow this action").
		WithRejectText("No, deny this action").
		Show()

	if confirmation {
		return true, ""
	}

	return false, u.AskInput("What should I do instead?")
}
