package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ErrUnknownKey is returned when a config key is not recognised.
var ErrUnknownKey = errors.New("unknown config key")

// configKeyType describes the Go type a config key holds.
type configKeyType string

const (
	keyTypeBool   configKeyType = "bool"
	keyTypeInt    configKeyType = "int"
	keyTypeString configKeyType = "string"
)

// configKeyMeta bundles the type and optional allowed values for one config key.
type configKeyMeta struct {
	typ     configKeyType
	allowed []string // non-nil → enum; nil → free-form
}

// configKeys is the single source of truth for valid config keys.
// Keys use the viper/mapstructure naming convention (underscores, not hyphens).
var configKeys = map[string]configKeyMeta{
	"verbose":     {typ: keyTypeBool},
	"output":      {typ: keyTypeString, allowed: []string{"table", "json", "text"}},
	"proxy":       {typ: keyTypeString},
	"user_agent":  {typ: keyTypeString},
	"pap_limit":   {typ: keyTypeString, allowed: []string{"red", "amber", "green", "white"}},
	"defang":      {typ: keyTypeBool},
	"no_defang":   {typ: keyTypeBool},
	"concurrency": {typ: keyTypeInt},
}

// ValidKeys returns every recognised config key in sorted order.
func ValidKeys() []string {
	keys := make([]string, 0, len(configKeys))
	for k := range configKeys {
		keys = append(keys, k)
	}
	return keys
}

// KeyCompletions returns the allowed completions for the given key, or nil when
// the key accepts free-form input (string / int).
func KeyCompletions(key string) []string {
	if meta, ok := configKeys[key]; ok {
		if meta.typ == keyTypeBool {
			return []string{"true", "false"}
		}
		return meta.allowed // nil for free-form string/int
	}
	return nil
}

// ValidateKey returns ErrUnknownKey when key is not a recognised config key.
// Accepts both hyphenated (user-agent) and underscored (user_agent) forms.
func ValidateKey(key string) error {
	if _, ok := configKeys[normalizeKey(key)]; !ok {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	return nil
}

// ParseValue converts the raw string value to the correct Go type for key and
// validates enum constraints. Returns an error for type mismatches or invalid
// enum values.
func ParseValue(key, value string) (any, error) {
	meta, ok := configKeys[normalizeKey(key)]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	switch meta.typ {
	case keyTypeBool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid bool value for %q: %q (want true or false)", key, value)
		}
		return b, nil
	case keyTypeInt:
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("invalid integer value for %q: %q (want a positive integer)", key, value)
		}
		return n, nil
	default: // keyTypeString
		if len(meta.allowed) > 0 {
			if !slices.Contains(meta.allowed, value) {
				return nil, fmt.Errorf("invalid value for %q: %q (allowed: %v)", key, value, meta.allowed)
			}
			return value, nil
		}
		return value, nil
	}
}

// normalizeKey converts hyphenated flag names to their viper key equivalents.
func normalizeKey(key string) string {
	result := make([]byte, len(key))
	for i := range key {
		if key[i] == '-' {
			result[i] = '_'
		} else {
			result[i] = key[i]
		}
	}
	return string(result)
}

// Config holds the runtime settings resolved from flags, env vars, and config file.
type Config struct {
	ConfigFile  string            // set after Unmarshal — no mapstructure tag
	Verbose     bool              `mapstructure:"verbose"`
	Output      string            `mapstructure:"output"`      // table | json | text
	Proxy       string            `mapstructure:"proxy"`       // http://, https://, socks5://
	UserAgent   string            `mapstructure:"user_agent"`  // override or empty (→ rotation)
	PAPLimit    string            `mapstructure:"pap_limit"`   // "white" (default)
	Defang      bool              `mapstructure:"defang"`      // force defang
	NoDefang    bool              `mapstructure:"no_defang"`   // suppress defang
	Concurrency int               `mapstructure:"concurrency"` // default 10
	Aliases     map[string]string `mapstructure:"alias"`       // file-only; no flag/env binding
}

// RegisterFlags defines all persistent CLI flags on the given FlagSet.
// Call this on the root command's PersistentFlags().
func RegisterFlags(flags *pflag.FlagSet) {
	flags.String("config", "", "config file (default: $XDG_CONFIG_HOME/trident/config.yaml)")
	flags.BoolP("verbose", "v", false, "enable verbose (debug) logging")
	flags.StringP("output", "o", "table", "output format: table, json, or text")
	flags.String("proxy", "", "proxy URL (http://, https://, or socks5://)")
	flags.String("user-agent", "", "HTTP User-Agent (empty = random rotation)")
	flags.String("pap-limit", "white", "PAP limit: white, green, amber, or red")
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
	v.SetDefault("output", "table")
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
	_ = v.BindPFlag("pap_limit", flags.Lookup("pap-limit"))
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

// DefaultConfigPath returns the default config file path for trident.
// On Linux this is $XDG_CONFIG_HOME/trident/config.yaml.
func DefaultConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", fmt.Errorf("resolving config dir: %w", err)
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// LoadAliases reads only the alias section from the config file at path.
// Returns an empty (non-nil) map when the file is missing or has no alias key.
func LoadAliases(path string) (map[string]string, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) || os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	return v.GetStringMapString("alias"), nil
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
