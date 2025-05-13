package main

import (
	"fmt"
	"github.com/recrsn/coder/internal/chat"
	"github.com/recrsn/coder/internal/config"
	"github.com/recrsn/coder/internal/tools"
	"github.com/recrsn/coder/internal/tools/lsp"
	"github.com/recrsn/coder/internal/ui"
	"os"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Using default configuration")
		cfg = config.DefaultConfig()
	}

	// Create a registry with all tools
	registry := tools.NewRegistry()

	// Register all tools
	registry.Register("shell", tools.NewShellTool())
	registry.Register("ls", tools.NewLSTool())
	registry.Register("glob", tools.NewGlobTool())
	registry.Register("sed", tools.NewSedTool())
	registry.Register("grep", tools.NewGrepTool())
	registry.Register("write", tools.NewWriteTool())
	registry.Register("read", tools.NewReadTool())
	registry.Register("search_replace", tools.NewSearchReplaceTool())
	registry.Register("tree", tools.NewTreeTool())
	registry.Register("outline", tools.NewOutlineTool())

	// Register LSP tools
	lspManager, err := lsp.NewManager()
	if err != nil {
		fmt.Printf("Error initializing LSP manager: %v\n", err)
		fmt.Println("LSP features may not work properly")
	} else {
		defer lspManager.StopAllServers() // Ensure all LSP servers are stopped on exit
		registry.Register("lsp_definition", lsp.NewDefinitionTool(lspManager))
		registry.Register("lsp_references", lsp.NewReferencesTool(lspManager))
		registry.Register("lsp_callhierarchy", lsp.NewCallHierarchyTool(lspManager))
	}

	var session *chat.Session

	// Create UI with exit handler
	userInterface, err := ui.NewUI(cfg.UI, func() {
		if session != nil {
			session.Exit()
		} else {
			os.Exit(0)
		}
	})
	if err != nil {
		fmt.Printf("Error creating UI: %v\n", err)
		os.Exit(1)
	}

	// Create and start chat session
	session, err = chat.NewSession(userInterface, cfg, registry)
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
