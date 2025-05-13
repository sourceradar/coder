package lsp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/pterm/pterm"
	"go.bug.st/lsp"

	"github.com/recrsn/coder/internal/platform"
)

// Config defines language server configuration
type Config struct {
	Language     string   // The programming language (go, python, typescript, etc.)
	Command      string   // The command to run the language server
	Args         []string // Command line arguments
	FilePatterns []string // File patterns that this server can handle
}

// LanguageServer represents a language server connection
type LanguageServer struct {
	Language     string
	Command      string
	Args         []string
	Client       *lsp.Client
	RootURI      string
	IsRunning    bool
	FilePatterns []string
}

// Manager handles LSP server connections
type Manager struct {
	servers       map[string]*LanguageServer
	mu            sync.RWMutex
	configs       map[string]Config
	initialized   bool
	serverManager *ServerManager
	directories   *platform.Directories
}

// NewManager creates a new LSP manager
func NewManager() (*Manager, error) {
	// Initialize platform directories
	dirs, err := platform.GetDirectories("coder")
	if err != nil {
		return nil, fmt.Errorf("failed to get application directories: %w", err)
	}

	// Create the server manager
	serverManager, err := NewServerManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create server manager: %w", err)
	}

	// Load default configurations
	if err := serverManager.LoadDefaultConfigs(); err != nil {
		return nil, fmt.Errorf("failed to load default configurations: %w", err)
	}

	// Initialize the legacy config
	m := &Manager{
		servers:       make(map[string]*LanguageServer),
		configs:       make(map[string]Config),
		serverManager: serverManager,
		directories:   dirs,
	}

	// Register default language servers
	m.RegisterLanguage(Config{
		Language:     "go",
		Command:      "gopls",
		FilePatterns: []string{"*.go"},
	})

	m.RegisterLanguage(Config{
		Language:     "typescript",
		Command:      "typescript-language-server",
		Args:         []string{"--stdio"},
		FilePatterns: []string{"*.ts", "*.tsx", "*.js", "*.jsx"},
	})

	m.RegisterLanguage(Config{
		Language:     "python",
		Command:      "pyright-langserver",
		Args:         []string{"--stdio"},
		FilePatterns: []string{"*.py"},
	})

	m.RegisterLanguage(Config{
		Language:     "rust",
		Command:      "rust-analyzer",
		FilePatterns: []string{"*.rs"},
	})

	m.RegisterLanguage(Config{
		Language:     "c",
		Command:      "clangd",
		FilePatterns: []string{"*.c", "*.h", "*.cpp", "*.hpp"},
	})

	return m, nil
}

// RegisterLanguage adds a language server configuration
func (m *Manager) RegisterLanguage(config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[config.Language] = config
}

// determineLanguageFromPath determines the language from a file path
func (m *Manager) determineLanguageFromPath(filePath string) (string, error) {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return "", fmt.Errorf("cannot determine language for file with no extension")
	}

	pattern := "*" + ext
	m.mu.RLock()
	defer m.mu.RUnlock()

	for lang, config := range m.configs {
		for _, filePattern := range config.FilePatterns {
			if filePattern == pattern {
				return lang, nil
			}
		}
	}

	return "", fmt.Errorf("no language server configured for %s files", ext)
}

// ensureServerRunning ensures that a language server is running for the given file
func (m *Manager) ensureServerRunning(filePath string) (string, error) {
	// Determine the language from the file path
	language, err := m.determineLanguageFromPath(filePath)
	if err != nil {
		return "", err
	}

	m.mu.RLock()
	server, exists := m.servers[language]
	m.mu.RUnlock()

	if exists && server.IsRunning {
		return language, nil
	}

	// Get workspace root (assume it's the Git root or parent directory)
	workspaceRoot, err := findWorkspaceRoot(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to determine workspace root: %w", err)
	}

	return language, m.startServer(language, workspaceRoot)
}

// findWorkspaceRoot finds the workspace root directory from a file path
func findWorkspaceRoot(filePath string) (string, error) {
	// Try to find Git repository root
	dir := filepath.Dir(filePath)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// We've reached the root directory
			break
		}
		dir = parentDir
	}

	// Fall back to the file's directory
	return filepath.Dir(filePath), nil
}

// startServer starts a language server
func (m *Manager) startServer(language, rootPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, ok := m.configs[language]
	if !ok {
		return fmt.Errorf("no configuration found for language %q", language)
	}

	// Create server entry if it doesn't exist
	server, exists := m.servers[language]
	if !exists {
		server = &LanguageServer{
			Language:     language,
			Command:      config.Command,
			Args:         config.Args,
			FilePatterns: config.FilePatterns,
			IsRunning:    false,
		}
		m.servers[language] = server
	}

	if server.IsRunning {
		return nil // Server already running
	}

	// Ensure the server is installed
	pterm.Info.Printfln("Checking %s language server installation...", language)

	serverPath, err := m.serverManager.EnsureServerInstalled(language)
	if err != nil {
		return fmt.Errorf("failed to install language server: %w", err)
	}

	// If server.Command doesn't match the installed path, update it
	// (but keep the original command name for error messages)
	actualCommand := serverPath
	if filepath.Base(serverPath) != server.Command {
		pterm.Info.Printfln("Using installed server at %s", serverPath)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Starting %s language server...", language))

	// Create command to start the server and establish a pipe for JSON-RPC
	cmd := exec.Command(actualCommand, server.Args...)

	cmd.Dir = rootPath
	cmd.Env = os.Environ()

	// Set up the command's standard input and output
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	client := lsp.NewClient(
		stdout,
		stdin,
		nil,
	)

	go client.Run()

	ctx := context.Background()

	// Initialize the server
	params := &lsp.InitializeParams{
		RootURI: lsp.NewDocumentURI(rootPath),
	}

	_, rpcErr, err := client.Initialize(ctx, params)

	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to initialize %s language server: %v", language, err))
		return fmt.Errorf("failed to initialize language server: %w", err)
	}

	if rpcErr != nil {
		spinner.Fail(fmt.Sprintf("Failed to initialize %s language server: %v", language, rpcErr))
		return fmt.Errorf("failed to initialize language server: %w", rpcErr)
	}

	// Notify initialized
	err = client.Initialized(&lsp.InitializedParams{})

	spinner.Success(fmt.Sprintf("%s language server started successfully", language))

	// Update server state
	server.Client = client
	server.IsRunning = true

	return nil
}

// stopServer stops a language server
func (m *Manager) stopServer(language string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[language]
	if !ok || !server.IsRunning || server.Client == nil {
		return nil // Server not running
	}

	// Shutdown and exit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	respErr, err := server.Client.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed to shutdown language server: %w", err)
	}

	if respErr != nil {
		return fmt.Errorf("failed to shutdown language server: %v", respErr)
	}

	err = server.Client.Exit()
	if err != nil {
		return fmt.Errorf("failed to exit language server: %w", err)
	}

	server.Client = nil
	server.IsRunning = false

	return nil
}

// StopAllServers stops all running language servers
func (m *Manager) StopAllServers() {
	m.mu.RLock()
	languages := make([]string, 0, len(m.servers))
	for lang := range m.servers {
		languages = append(languages, lang)
	}
	m.mu.RUnlock()

	for _, lang := range languages {
		_ = m.stopServer(lang)
	}
}

// GetDefinition gets definition location of a symbol
func (m *Manager) GetDefinition(filePath string, line, character int) ([]lsp.Location, error) {
	// Ensure a server is running for this file
	language, err := m.ensureServerRunning(filePath)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	server := m.servers[language]
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert file path to URI
	fileURI := lsp.NewDocumentURI(filePath)

	// Create TextDocumentIdentifier
	textDocument := lsp.TextDocumentIdentifier{
		URI: fileURI,
	}

	// Create Position
	position := lsp.Position{
		Line:      line,
		Character: character,
	}

	// Request definition
	params := &lsp.DefinitionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: textDocument,
			Position:     position,
		},
	}

	locations, _, rpcError, err := server.Client.TextDocumentDefinition(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to get definition: %w", err)
	}

	if rpcError != nil {
		return nil, fmt.Errorf("failed to get definition: %v", rpcError)
	}

	return locations, nil
}

// GetReferences gets all references to a symbol
func (m *Manager) GetReferences(filePath string, line, character int, includeDeclaration bool) ([]lsp.Location, error) {
	// Ensure a server is running for this file
	language, err := m.ensureServerRunning(filePath)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	server := m.servers[language]
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert file path to URI
	fileURI := lsp.NewDocumentURI(filePath)

	// Create TextDocumentIdentifier
	textDocument := lsp.TextDocumentIdentifier{
		URI: fileURI,
	}

	// Create Position
	position := lsp.Position{
		Line:      (line),
		Character: (character),
	}

	// Request references
	references, rpcErr, err := server.Client.TextDocumentReferences(ctx, &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: textDocument,
			Position:     position,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}
	if rpcErr != nil {
		return nil, fmt.Errorf("failed to get references: %v", rpcErr)
	}

	return references, nil
}
