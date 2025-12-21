package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// GetDefaultConfigPath returns the OS-appropriate default config file path.
// Accepts userConfigDir for dependency injection (testability).
// Ensures the app-specific config directory exists.
func GetDefaultConfigPath(userConfigDir func() (string, error)) (string, error) {
	// Get OS-appropriate config directory
	// - Windows: %AppData%
	// - macOS: $HOME/Library/Application Support
	// - Linux: $XDG_CONFIG_HOME or $HOME/.config
	configDir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	// App-specific directory
	appConfigDir := filepath.Join(configDir, "trident")

	// Ensure directory exists
	if err := os.MkdirAll(appConfigDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(appConfigDir, "config.yaml"), nil
}

// Load loads the configuration from the specified path or default location.
// If configPath is empty, it uses the OS-appropriate default path.
// If the config file doesn't exist, it returns a default configuration.
// Accepts userConfigDir for dependency injection (testability).
func Load(configPath string, userConfigDir func() (string, error)) (*Config, error) {
	// Determine config file path
	if configPath == "" {
		var err error
		configPath, err = GetDefaultConfigPath(userConfigDir)
		if err != nil {
			return nil, err
		}
	}

	// Initialize Viper
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set default values
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// If file not found, return default config (not an error)
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return NewDefaultConfig(), nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// setDefaults configures Viper default values matching NewDefaultConfig.
func setDefaults(v *viper.Viper) {
	v.SetDefault("global.output", "text")
	v.SetDefault("global.concurrency", 10)
	v.SetDefault("global.pap_limit", "white")
	v.SetDefault("global.proxy", "")
	v.SetDefault("global.user_agent", "")
	v.SetDefault("global.defang", false)
	v.SetDefault("global.no_defang", false)
}
