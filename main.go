package main

import (
	"fmt"
	"github.com/recrsn/coder/internal/chat"
	config2 "github.com/recrsn/coder/internal/config"
	"github.com/recrsn/coder/internal/tools"
	"github.com/recrsn/coder/internal/ui"
	"os"
)

func main() {
	// Load configuration
	cfg, err := config2.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Using default configuration")
		cfg = config2.DefaultConfig()
	}

	// Create a registry with all tools
	registry := tools.NewRegistry()

	// Register all tools
	registry.Register("shell", tools.NewShellTool())
	registry.Register("ls", tools.NewLSTool())
	registry.Register("glob", tools.NewGlobTool())
	registry.Register("sed", tools.NewSedTool())
	registry.Register("grep", tools.NewGrepTool())

	// Create UI
	userInterface := ui.NewUI(cfg.UI)

	// Create and start chat session
	session, err := chat.NewSession(userInterface, cfg, registry)
	if err != nil {
		fmt.Printf("Error creating chat session: %v\n", err)
		os.Exit(1)
	}

	// Start the session
	if err := session.Start(); err != nil {
		fmt.Printf("Error in chat session: %v\n", err)
		os.Exit(1)
	}
}
