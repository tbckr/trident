package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Proxy       string `mapstructure:"proxy"`
	UserAgent   string `mapstructure:"user_agent"`
	PAPLimit    string `mapstructure:"pap_limit"`
	Concurrency int    `mapstructure:"concurrency"`
	Defang      bool   `mapstructure:"defang"`
}

func LoadConfig(configPath string, getenv func(string) string) (*Config, error) {
	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		// Respect XDG_CONFIG_HOME
		xdgConfigHome := getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			v.AddConfigPath(filepath.Join(xdgConfigHome, "trident"))
		} else {
			v.AddConfigPath(filepath.Join(home, ".config", "trident"))
		}
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("TRIDENT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// Default values
	v.SetDefault("pap_limit", "white")
	v.SetDefault("concurrency", 10)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found is OK
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Log(logger *slog.Logger) {
	logger.Debug("configuration loaded",
		"proxy", c.Proxy,
		"user_agent", c.UserAgent,
		"pap_limit", c.PAPLimit,
		"concurrency", c.Concurrency,
	)
}
