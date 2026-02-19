// Package config handles loading and validation of the trident configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the runtime settings resolved from flags, env vars, and config file.
type Config struct {
	ConfigFile string
	Verbose    bool
	Output     string
}

// Load initializes Viper with TRIDENT_* env var overrides and an XDG-compliant config
// file path. Creates the config file with 0600 permissions if it does not exist.
// A missing config file is not an error.
func Load(configFile string, verbose bool, output string) (*Config, error) {
	cfg := &Config{
		Verbose: verbose,
		Output:  output,
	}

	v := viper.New()
	v.SetEnvPrefix("TRIDENT")
	v.AutomaticEnv()

	if configFile != "" {
		cfg.ConfigFile = configFile
		v.SetConfigFile(configFile)
	} else {
		dir, err := configDir()
		if err != nil {
			return nil, fmt.Errorf("resolving config dir: %w", err)
		}
		cfg.ConfigFile = filepath.Join(dir, "config.yaml")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(dir)
	}

	if err := ensureConfigFile(cfg.ConfigFile); err != nil {
		return nil, fmt.Errorf("ensuring config file: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			// Tolerate "not found" by path when using SetConfigFile
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("reading config: %w", err)
			}
		}
	}

	return cfg, nil
}

// configDir returns the OS-appropriate config directory for trident.
// Uses os.UserConfigDir() which returns XDG_CONFIG_HOME on Linux,
// ~/Library/Application Support on macOS, and %AppData% on Windows.
func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config dir: %w", err)
	}
	return filepath.Join(base, "trident"), nil
}

// ensureConfigFile creates the config file (and its parent directory) if they do
// not already exist. The file is created with 0600 permissions (owner read/write only).
func ensureConfigFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("creating config file: %w", err)
	}
	// Nothing is written; the file is a zero-byte placeholder whose presence
	// confirms the path is initialised with the correct permissions (0600).
	return f.Close()
}
