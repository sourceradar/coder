package agent

import (
	"context"
	"testing"
)

func TestNewInteractiveAgent(t *testing.T) {
	// Create a minimal test to verify the package compiles correctly
	callbacks := UICallbacks{
		PrintMessage:  func(message string) {},
		PrintToolCall: func(toolName string, args map[string]any, result string, err error) {},
		AskToolCallConfirmation: func(explanation string) (bool, string) {
			return true, ""
		},
	}

	// Just test that we can create an agent - no actual LLM calls in this test
	ia := &InteractiveAgent{
		name:        "TestAgent",
		uiCallbacks: callbacks,
	}

	if ia.name != "TestAgent" {
		t.Errorf("Expected agent name to be TestAgent, got %s", ia.name)
	}
}

func TestInterruptAgent(t *testing.T) {
	// Test the interrupt functionality
	ia := &InteractiveAgent{}

	// No cancel function set
	if ia.Interrupt() {
		t.Error("Expected Interrupt to return false when no cancel function is set")
	}

	// Set a cancel function
	ctx, cancel := context.WithCancel(context.Background())
	ia.cancelFunc = cancel

	// Interrupt should return true and cancel the context
	if !ia.Interrupt() {
		t.Error("Expected Interrupt to return true when cancel function is set")
	}

	// Context should be cancelled
	if ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}
