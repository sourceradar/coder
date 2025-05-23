package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Provider    ProviderConfig   `mapstructure:"provider"`
	UI          UIConfig         `mapstructure:"ui"`
	Permissions PermissionConfig `mapstructure:"permissions"`
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"model"`
	LiteModel string `mapstructure:"lite_model"`
}

// UIConfig holds UI-specific configuration
type UIConfig struct {
	ColorEnabled bool `mapstructure:"color_enabled"`
	ShowSpinner  bool `mapstructure:"show_spinner"`
	UseBubbleTea bool `mapstructure:"use_bubble_tea"`
}

// LoadConfig loads the configuration from file
func LoadConfig() (Config, error) {
	config := DefaultConfig()

	// Setup viper to look for config files
	viper.SetConfigName(".coder")
	viper.SetConfigType("yaml")

	// Add search paths: current directory and home directory
	viper.AddConfigPath(".")
	homeDir, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(homeDir)
	}

	viper.AutomaticEnv()

	// Read config (will use first found file)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Return error only if it's not a "config file not found" error
			return config, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found - continue with defaults
	}

	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unmarshaling config: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(config Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	configPath := filepath.Join(homeDir, ".coder.yaml")

	// Set config values
	viper.SetConfigFile(configPath)

	viper.Set("provider.api_key", config.Provider.APIKey)
	viper.Set("provider.model", config.Provider.Model)
	viper.Set("provider.endpoint", config.Provider.Endpoint)

	viper.Set("ui.color_enabled", config.UI.ColorEnabled)
	viper.Set("ui.show_spinner", config.UI.ShowSpinner)
	viper.Set("ui.use_bubble_tea", config.UI.UseBubbleTea)

	// Save permission settings
	for tool, autoApprove := range config.Permissions.AutoApprove {
		viper.Set(fmt.Sprintf("permissions.auto_approve.%s", tool), autoApprove)
	}

	return viper.WriteConfig()
}

// GetDataDir returns the data directory for the application
func GetDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".coder")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("creating data dir: %w", err)
	}

	return dataDir, nil
}

// getConfigDir returns the configuration directory
func getConfigDir() (string, error) {
	return GetDataDir()
}
