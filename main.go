package main

import (
	"fmt"
	"github.com/recrsn/coder/internal/chat"
	"github.com/recrsn/coder/internal/chat/prompts"
	"github.com/recrsn/coder/internal/common"
	"github.com/recrsn/coder/internal/config"
	"github.com/recrsn/coder/internal/llm"
	"github.com/recrsn/coder/internal/lsp"
	"github.com/recrsn/coder/internal/platform"
	"github.com/recrsn/coder/internal/tools"
	lsptools "github.com/recrsn/coder/internal/tools/lsp"
	"github.com/recrsn/coder/internal/ui"
	"os"
	"path/filepath"
	"time"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Using default configuration")
		cfg = config.DefaultConfig()
	}

	registry := tools.NewRegistry()

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
	if err == nil {
		defer lspManager.StopAllServers()
		registry.Register("lsp_definition", lsptools.NewDefinitionTool(lspManager))
		registry.Register("lsp_references", lsptools.NewReferencesTool(lspManager))
		registry.Register("lsp_callhierarchy", lsptools.NewCallHierarchyTool(lspManager))
	} else {
		fmt.Printf("Error initializing LSP manager: %v\n", err)
		fmt.Println("LSP features may not work properly")
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

	// Create the UI permission handler
	uiPermissionHandler := ui.NewUIPermissionHandler(userInterface)

	// Create the permission manager
	permissionManager := common.NewPermissionManager(cfg.Permissions, uiPermissionHandler)

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "/tmp" // Fallback
	}
	configDir := filepath.Join(userConfigDir, "coder")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Warning: couldn't create config directory: %v\n", err)
	}

	apiLogger := llm.NewAPILogger(configDir)

	client := llm.NewClient(cfg.Provider.Endpoint, cfg.Provider.APIKey, apiLogger)

	// Get the working directory for the prompt
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	// Prepare a system prompt
	platformInfo := platform.GetPlatformInfo()
	// Get agent instructions from AGENT.md or AGENTS.md if they exist
	agentInstructions := prompts.GetAgentInstructions(workingDir)

	promptData := prompts.PromptData{
		KnowsTools:       len(registry.GetAll()) > 0,
		Tools:            registry.GetAll(),
		Platform:         fmt.Sprintf("%s %s (%s)", platformInfo.Name, platformInfo.Version, platformInfo.Arch),
		Date:             time.Now().Format("2006-01-02"),
		WorkingDirectory: workingDir,
		Instructions:     agentInstructions,
	}

	systemPrompt, err := prompts.RenderSystemPrompt(promptData)
	if err != nil {
		fmt.Printf("Error rendering system prompt: %v\n", err)
		os.Exit(1)
	}

	// Create a session instance with permission manager
	session, err = chat.NewSession(userInterface, cfg, registry, client, permissionManager)
	if err != nil {
		fmt.Printf("Error creating chat session: %v\n", err)
		os.Exit(1)
	}

	// Create the agent
	agent := llm.NewAgent(
		"Coder",
		systemPrompt,
		registry.ListTools(),
		llm.ModelConfig{
			Model:       cfg.Provider.Model,
			Temperature: 0.6,
		},
		client,
		session.HandleToolCalls, // Use the session's tool call handler
		session.HandleMessage,   // Use the session's message handler
	)

	// Set the agent in the session
	session.SetAgent(agent)

	// Register agent tool with the same client
	registry.Register("agent", tools.NewAgentTool(registry, client, userInterface, cfg.Provider.Model, permissionManager))

	if err := session.Start(); err != nil {
		fmt.Printf("Error in chat session: %v\n", err)
		os.Exit(1)
	}
}
