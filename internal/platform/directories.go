package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Directories holds common system directories
type Directories struct {
	// Home is the user's home directory
	Home string
	// Config is the directory for storing application configuration
	Config string
	// Data is the directory for storing application data
	Data string
	// Cache is the directory for storing application cache
	Cache string
}

// GetDirectories returns the appropriate directories for the current platform
func GetDirectories(appName string) (*Directories, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	var dirs Directories
	dirs.Home = homeDir

	switch runtime.GOOS {
	case "darwin":
		// macOS
		dirs.Config = filepath.Join(homeDir, "Library", "Application Support", appName)
		dirs.Data = dirs.Config
		dirs.Cache = filepath.Join(homeDir, "Library", "Caches", appName)

	case "windows":
		// Windows
		appDataDir := os.Getenv("APPDATA")
		if appDataDir == "" {
			appDataDir = filepath.Join(homeDir, "AppData", "Roaming")
		}
		
		localAppDataDir := os.Getenv("LOCALAPPDATA")
		if localAppDataDir == "" {
			localAppDataDir = filepath.Join(homeDir, "AppData", "Local")
		}
		
		dirs.Config = filepath.Join(appDataDir, appName)
		dirs.Data = dirs.Config
		dirs.Cache = filepath.Join(localAppDataDir, appName, "Cache")

	default:
		// Linux/Unix/BSD
		// Follow XDG Base Directory Specification
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(homeDir, ".config")
		}
		
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome == "" {
			xdgDataHome = filepath.Join(homeDir, ".local", "share")
		}
		
		xdgCacheHome := os.Getenv("XDG_CACHE_HOME")
		if xdgCacheHome == "" {
			xdgCacheHome = filepath.Join(homeDir, ".cache")
		}
		
		dirs.Config = filepath.Join(xdgConfigHome, appName)
		dirs.Data = filepath.Join(xdgDataHome, appName)
		dirs.Cache = filepath.Join(xdgCacheHome, appName)
	}

	// Ensure directories exist
	for _, dir := range []string{dirs.Config, dirs.Data, dirs.Cache} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &dirs, nil
}

// GetLSPServersDir returns the directory for storing LSP servers
func (d *Directories) GetLSPServersDir() string {
	return filepath.Join(d.Data, "lsp-servers")
}

// GetLSPConfigDir returns the directory for storing LSP configuration
func (d *Directories) GetLSPConfigDir() string {
	return filepath.Join(d.Config, "lsp")
}

// GetLSPCacheDir returns the directory for storing LSP cache
func (d *Directories) GetLSPCacheDir() string {
	return filepath.Join(d.Cache, "lsp")
}