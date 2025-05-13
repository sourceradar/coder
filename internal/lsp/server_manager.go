package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/pterm/pterm"
	"github.com/recrsn/coder/internal/platform"
)

// ServerConfig represents the configuration for a language server
type ServerConfig struct {
	// Command is the command to run the language server
	Command string `json:"command"`
	// Args are the arguments to pass to the command
	Args []string `json:"args"`
	// FileExtensions are the file extensions supported by this language server
	FileExtensions []string `json:"file_extensions"`
	// DownloadInfo contains information about how to download and install the language server
	DownloadInfo *DownloadInfo `json:"download_info,omitempty"`
}

// DownloadInfo represents the information needed to download and install a language server
type DownloadInfo struct {
	// Platforms contains download information for each supported platform
	Platforms map[string]PlatformDownloadInfo `json:"platforms"`
	// Dependencies are the dependencies required by the language server
	Dependencies []*Dependency `json:"dependencies,omitempty"`
}

// PlatformDownloadInfo represents platform-specific download information
type PlatformDownloadInfo struct {
	// URL is the download URL for the language server
	URL string `json:"url"`
	// Type is the archive type (e.g., "zip", "tar.gz")
	Type string `json:"type"`
	// Binary is the path to the binary within the extracted archive
	Binary string `json:"binary,omitempty"`
	// Setup contains the setup commands to run after downloading
	Setup []string `json:"setup,omitempty"`
}

// Dependency represents a dependency required by a language server
type Dependency struct {
	// Name is the name of the dependency
	Name string `json:"name"`
	// CheckCommand is the command to check if the dependency is installed
	CheckCommand []string `json:"check_command"`
	// InstallInstructions is the instruction to show the user for installing the dependency
	InstallInstructions string `json:"install_instructions"`
}

// ServerManager manages the language server configurations and installations
type ServerManager struct {
	// configs is a map of language to server configuration
	configs map[string]ServerConfig
	// servers is a map of language to running server
	servers map[string]*LanguageServer
	// installedServers is a map of language to the path of the installed server
	installedServers map[string]string
	// directories holds the platform-specific directories
	directories *platform.Directories
	// mu is a mutex to protect the maps
	mu sync.RWMutex
}

// NewServerManager creates a new server manager
func NewServerManager() (*ServerManager, error) {
	dirs, err := platform.GetDirectories("coder")
	if err != nil {
		return nil, fmt.Errorf("failed to get application directories: %w", err)
	}

	// Create the LSP servers directory if it doesn't exist
	serverDir := dirs.GetLSPServersDir()
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create language server directory: %w", err)
	}

	// Create the LSP config directory if it doesn't exist
	configDir := dirs.GetLSPConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create language server config directory: %w", err)
	}

	return &ServerManager{
		configs:         make(map[string]ServerConfig),
		servers:         make(map[string]*LanguageServer),
		installedServers: make(map[string]string),
		directories:     dirs,
	}, nil
}

// LoadDefaultConfigs loads the default language server configurations
func (sm *ServerManager) LoadDefaultConfigs() error {
	// Define default configurations for common language servers
	configs := map[string]ServerConfig{
		"gopls": {
			Command: "gopls",
			Args:    []string{"serve", "-rpc.trace"},
			FileExtensions: []string{".go"},
			DownloadInfo: &DownloadInfo{
				Dependencies: []*Dependency{
					{
						Name:                "Go",
						CheckCommand:        []string{"go", "version"},
						InstallInstructions: "Install Go from https://golang.org/dl/",
					},
				},
				Platforms: map[string]PlatformDownloadInfo{
					"all": {
						Setup: []string{"go", "install", "golang.org/x/tools/gopls@latest"},
					},
				},
			},
		},
		"typescript": {
			Command: "typescript-language-server",
			Args:    []string{"--stdio"},
			FileExtensions: []string{".ts", ".tsx", ".js", ".jsx"},
			DownloadInfo: &DownloadInfo{
				Dependencies: []*Dependency{
					{
						Name:                "Node.js",
						CheckCommand:        []string{"node", "--version"},
						InstallInstructions: "Install Node.js from https://nodejs.org/",
					},
					{
						Name:                "npm",
						CheckCommand:        []string{"npm", "--version"},
						InstallInstructions: "npm is included with Node.js",
					},
				},
				Platforms: map[string]PlatformDownloadInfo{
					"all": {
						Setup: []string{"npm", "install", "-g", "typescript-language-server", "typescript"},
					},
				},
			},
		},
		"pyright": {
			Command: "pyright-langserver",
			Args:    []string{"--stdio"},
			FileExtensions: []string{".py"},
			DownloadInfo: &DownloadInfo{
				Dependencies: []*Dependency{
					{
						Name:                "Node.js",
						CheckCommand:        []string{"node", "--version"},
						InstallInstructions: "Install Node.js from https://nodejs.org/",
					},
					{
						Name:                "npm",
						CheckCommand:        []string{"npm", "--version"},
						InstallInstructions: "npm is included with Node.js",
					},
				},
				Platforms: map[string]PlatformDownloadInfo{
					"all": {
						Setup: []string{"npm", "install", "-g", "pyright"},
					},
				},
			},
		},
		"rust-analyzer": {
			Command: "rust-analyzer",
			FileExtensions: []string{".rs"},
			DownloadInfo: &DownloadInfo{
				Platforms: map[string]PlatformDownloadInfo{
					"darwin-amd64": {
						URL:    "https://github.com/rust-analyzer/rust-analyzer/releases/latest/download/rust-analyzer-aarch64-apple-darwin.gz",
						Type:   "gz",
						Binary: "rust-analyzer",
					},
					"darwin-arm64": {
						URL:    "https://github.com/rust-analyzer/rust-analyzer/releases/latest/download/rust-analyzer-aarch64-apple-darwin.gz",
						Type:   "gz",
						Binary: "rust-analyzer",
					},
					"linux-amd64": {
						URL:    "https://github.com/rust-analyzer/rust-analyzer/releases/latest/download/rust-analyzer-x86_64-unknown-linux-gnu.gz",
						Type:   "gz",
						Binary: "rust-analyzer",
					},
					"linux-arm64": {
						URL:    "https://github.com/rust-analyzer/rust-analyzer/releases/latest/download/rust-analyzer-aarch64-unknown-linux-gnu.gz",
						Type:   "gz",
						Binary: "rust-analyzer",
					},
					"windows-amd64": {
						URL:    "https://github.com/rust-analyzer/rust-analyzer/releases/latest/download/rust-analyzer-x86_64-pc-windows-msvc.gz",
						Type:   "gz",
						Binary: "rust-analyzer.exe",
					},
				},
			},
		},
		"clangd": {
			Command: "clangd",
			FileExtensions: []string{".c", ".cpp", ".h", ".hpp"},
			DownloadInfo: &DownloadInfo{
				Platforms: map[string]PlatformDownloadInfo{
					"darwin-amd64": {
						URL:    "https://github.com/clangd/clangd/releases/latest/download/clangd-mac-amd64.zip",
						Type:   "zip",
						Binary: "clangd_16.0.0/bin/clangd",
					},
					"darwin-arm64": {
						URL:    "https://github.com/clangd/clangd/releases/latest/download/clangd-mac-arm64.zip",
						Type:   "zip",
						Binary: "clangd_16.0.0/bin/clangd",
					},
					"linux-amd64": {
						URL:    "https://github.com/clangd/clangd/releases/latest/download/clangd-linux-amd64.zip",
						Type:   "zip",
						Binary: "clangd_16.0.0/bin/clangd",
					},
					"windows-amd64": {
						URL:    "https://github.com/clangd/clangd/releases/latest/download/clangd-windows-amd64.zip",
						Type:   "zip",
						Binary: "clangd_16.0.0/bin/clangd.exe",
					},
				},
			},
		},
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for lang, config := range configs {
		sm.configs[lang] = config
	}

	return nil
}

// GetConfigForLanguage returns the server configuration for the specified language
func (sm *ServerManager) GetConfigForLanguage(language string) (ServerConfig, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	config, exists := sm.configs[language]
	return config, exists
}

// GetConfigForFileExtension returns the server configuration for the specified file extension
func (sm *ServerManager) GetConfigForFileExtension(fileExt string) (string, ServerConfig, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for lang, config := range sm.configs {
		for _, ext := range config.FileExtensions {
			if ext == fileExt {
				return lang, config, true
			}
		}
	}

	return "", ServerConfig{}, false
}

// EnsureServerInstalled ensures that a language server is installed for the given language
// It installs the server if not already installed
func (sm *ServerManager) EnsureServerInstalled(language string) (string, error) {
	// Check if the server is already installed
	sm.mu.RLock()
	serverPath, exists := sm.installedServers[language]
	sm.mu.RUnlock()

	if exists {
		// Verify the path still exists
		if _, err := os.Stat(serverPath); err == nil {
			return serverPath, nil
		}

		// If path doesn't exist, remove from installedServers
		sm.mu.Lock()
		delete(sm.installedServers, language)
		sm.mu.Unlock()
	}

	// Get the server configuration
	_, exists = sm.GetConfigForLanguage(language)
	if !exists {
		return "", fmt.Errorf("no configuration found for language %s", language)
	}

	// Create a downloader and install the server
	downloader := NewDownloader(sm)

	pterm.Info.Printfln("Installing %s language server...", language)
	serverPath, err := downloader.DownloadAndInstallServer(language)
	if err != nil {
		pterm.Error.Printfln("Failed to install %s language server: %v", language, err)
		return "", err
	}

	return serverPath, nil
}

// GetLanguageForFile determines the language from a file path
func (sm *ServerManager) GetLanguageForFile(filePath string) (string, error) {
	fileExt := filepath.Ext(filePath)
	if fileExt == "" {
		return "", fmt.Errorf("file has no extension")
	}

	language, _, exists := sm.GetConfigForFileExtension(fileExt)
	if !exists {
		return "", fmt.Errorf("no language server configured for file extension %s", fileExt)
	}

	return language, nil
}

// EnsureServerInstalledForFile ensures a language server is installed for the given file
func (sm *ServerManager) EnsureServerInstalledForFile(filePath string) (string, string, error) {
	language, err := sm.GetLanguageForFile(filePath)
	if err != nil {
		return "", "", err
	}

	serverPath, err := sm.EnsureServerInstalled(language)
	if err != nil {
		return "", "", err
	}

	return language, serverPath, nil
}

// SaveConfigs saves the server configurations to disk
func (sm *ServerManager) SaveConfigs() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	configDir := sm.directories.GetLSPConfigDir()
	configFile := filepath.Join(configDir, "language-servers.json")

	data, err := json.MarshalIndent(sm.configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfigs loads the server configurations from disk
func (sm *ServerManager) LoadConfigs() error {
	configDir := sm.directories.GetLSPConfigDir()
	configFile := filepath.Join(configDir, "language-servers.json")

	// If the config file doesn't exist, create it with default configs
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := sm.LoadDefaultConfigs(); err != nil {
			return fmt.Errorf("failed to load default configs: %w", err)
		}
		return sm.SaveConfigs()
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configs map[string]ServerConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to unmarshal configs: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.configs = configs
	return nil
}

// CheckDependency checks if a dependency is installed
func (sm *ServerManager) CheckDependency(dep *Dependency) (bool, error) {
	if len(dep.CheckCommand) == 0 {
		return false, fmt.Errorf("no check command specified for dependency %s", dep.Name)
	}

	cmd := exec.Command(dep.CheckCommand[0], dep.CheckCommand[1:]...)
	err := cmd.Run()
	return err == nil, nil
}

// GetPlatformKey returns the platform key for the current system
func GetPlatformKey() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map GOARCH to more common architecture names
	archMap := map[string]string{
		"amd64": "amd64",
		"386":   "386",
		"arm64": "arm64",
		"arm":   "arm",
	}

	mappedArch, ok := archMap[arch]
	if !ok {
		mappedArch = arch
	}

	return fmt.Sprintf("%s-%s", os, mappedArch)
}

// ShallowMergeConfigs merges src into dst without overriding existing values
func ShallowMergeConfigs(dst, src map[string]ServerConfig) {
	for lang, config := range src {
		if _, exists := dst[lang]; !exists {
			dst[lang] = config
		}
	}
}