package ui

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/recrsn/coder/internal/config"
)

// UserInterface is the interface for any UI implementation
type UserInterface interface {
	ShowHeader()
	StartSpinner(text string) *pterm.SpinnerPrinter
	StopSpinner(spinner *pterm.SpinnerPrinter, text string)
	StopSpinnerFail(spinner *pterm.SpinnerPrinter, text string)
	PrintUserMessage(message string)
	PrintAssistantMessage(message string)
	PrintCodeBlock(code, language string)
	PrintToolCall(toolName string, args map[string]any, result string, err error)
	PrintHelp()
	PrintError(message string)
	PrintSuccess(message string)
	PrintInfo(message string)
	AskInput(prompt string) string
	AskMultiLineInput(prompt string) string
	ClearScreen()
	AskToolCallConfirmation(explanation string) (bool, string)
	AskPermission(explanation string) (bool, string)
}

// NewUI creates a new UI instance based on config
func NewUI(cfg config.UIConfig, exitHandler func()) (UserInterface, error) {
	if cfg.UseBubbleTea {
		fmt.Println("Using BubbleTea UI...")
		return NewBubbleTeaUI(cfg, exitHandler)
	}

	// Fall back to traditional UI
	return NewTraditionalUI(cfg, exitHandler)
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
