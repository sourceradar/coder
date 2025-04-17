package ui

import (
	"fmt"
	"github.com/recrsn/coder/internal/config"
	"strings"

	"github.com/pterm/pterm"
)

// UI handles the terminal user interface
type UI struct {
	config config.UIConfig
}

// NewUI creates a new UI instance
func NewUI(cfg config.UIConfig) *UI {
	// Configure PTerm based on config
	if !cfg.ColorEnabled {
		pterm.DisableColor()
	}

	return &UI{
		config: cfg,
	}
}

// ShowHeader displays the application header
func (u *UI) ShowHeader() {
	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).WithMargin(10)
	header.Println("Coder - Your Programming Sidekick")
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
	pterm.FgGreen.Println("You: " + message)
}

// PrintAssistantMessage prints an assistant message
func (u *UI) PrintAssistantMessage(message string) {
	pterm.FgBlue.Println("Coder: " + message)
}

// PrintCodeBlock prints a code block
func (u *UI) PrintCodeBlock(code, language string) {
	fmt.Println()
	pterm.DefaultBox.WithTitle(language).Println(code)
	fmt.Println()
}

// PrintToolCall prints information about a tool call
func (u *UI) PrintToolCall(toolName string, args map[string]any, result map[string]any, err error) {
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
	} else if result != nil {
		// Print result
		content.WriteString("\nResult:\n")
		for k, v := range result {
			content.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
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
	}

	pterm.DefaultTable.WithHasHeader().WithData(table).Render()
}

// PrintError prints an error message
func (u *UI) PrintError(message string) {
	pterm.Error.Println(message)
}

// PrintSuccess prints a success message
func (u *UI) PrintSuccess(message string) {
	pterm.Success.Println(message)
}

// AskInput asks for user input with a prompt
func (u *UI) AskInput(prompt string) string {
	text, _ := pterm.DefaultInteractiveTextInput.WithMultiLine(false).Show(prompt)
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
