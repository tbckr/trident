package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the runtime settings resolved from flags, env vars, and config file.
type Config struct {
	ConfigFile  string // set after Unmarshal — no mapstructure tag
	Verbose     bool   `mapstructure:"verbose"`
	Output      string `mapstructure:"output"`      // text | json | plain
	Proxy       string `mapstructure:"proxy"`       // http://, https://, socks5://
	UserAgent   string `mapstructure:"user_agent"`  // override or empty (→ rotation)
	PAPLimit    string `mapstructure:"pap_limit"`   // "white" (default)
	Defang      bool   `mapstructure:"defang"`      // force defang
	NoDefang    bool   `mapstructure:"no_defang"`   // suppress defang
	Concurrency int    `mapstructure:"concurrency"` // default 10
}

// RegisterFlags defines all persistent CLI flags on the given FlagSet.
// Call this on the root command's PersistentFlags().
func RegisterFlags(flags *pflag.FlagSet) {
	flags.String("config", "", "config file (default: $XDG_CONFIG_HOME/trident/config.yaml)")
	flags.BoolP("verbose", "v", false, "enable verbose (debug) logging")
	flags.StringP("output", "o", "text", "output format: text, json, or plain")
	flags.String("proxy", "", "proxy URL (http://, https://, or socks5://)")
	flags.String("user-agent", "", "HTTP User-Agent (empty = random rotation)")
	flags.String("pap", "white", "PAP limit: white, green, amber, or red")
	flags.Bool("defang", false, "defang text/plain output (dots → [.], http → hxxp)")
	flags.Bool("no-defang", false, "disable defanging even if enabled in config")
	flags.IntP("concurrency", "c", 10, "parallel workers for bulk stdin input")
}

// Load initializes Viper with the full precedence chain:
//
//	CLI flag (changed) > TRIDENT_* env var > config.yaml > viper SetDefault
//
// Creates the config file with 0600 permissions if it does not exist.
// A missing config file is not an error.
func Load(flags *pflag.FlagSet) (*Config, error) {
	v := viper.New()

	// Defaults — only used when nothing else provides the value.
	v.SetDefault("output", "text")
	v.SetDefault("pap_limit", "white")
	v.SetDefault("concurrency", 10)

	// Env vars: TRIDENT_VERBOSE, TRIDENT_OUTPUT, TRIDENT_USER_AGENT, etc.
	v.SetEnvPrefix("TRIDENT")
	v.AutomaticEnv()

	// Bind cobra flags → viper keys (flag name ≠ viper key for 3 flags).
	_ = v.BindPFlag("verbose", flags.Lookup("verbose"))
	_ = v.BindPFlag("output", flags.Lookup("output"))
	_ = v.BindPFlag("proxy", flags.Lookup("proxy"))
	_ = v.BindPFlag("user_agent", flags.Lookup("user-agent"))
	_ = v.BindPFlag("pap_limit", flags.Lookup("pap"))
	_ = v.BindPFlag("defang", flags.Lookup("defang"))
	_ = v.BindPFlag("no_defang", flags.Lookup("no-defang"))
	_ = v.BindPFlag("concurrency", flags.Lookup("concurrency"))

	// Config file resolution.
	var resolvedPath string
	configFile, _ := flags.GetString("config")
	if configFile != "" {
		resolvedPath = configFile
		v.SetConfigFile(configFile)
	} else {
		dir, err := configDir()
		if err != nil {
			return nil, fmt.Errorf("resolving config dir: %w", err)
		}
		resolvedPath = filepath.Join(dir, "config.yaml")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(dir)
	}

	if err := ensureConfigFile(resolvedPath); err != nil {
		return nil, fmt.Errorf("ensuring config file: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	// Unmarshal → Config (mapstructure tags drive field assignment).
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}
	cfg.ConfigFile = resolvedPath
	return &cfg, nil
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
