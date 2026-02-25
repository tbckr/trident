package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/tbckr/trident/internal/appdir"
)

// DefaultPatternsURL is the built-in URL used by `download detect` when no
// custom URL is configured.
const DefaultPatternsURL = "https://raw.githubusercontent.com/tbckr/trident/refs/heads/main/internal/detect/patterns.yaml"

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
	"verbose":              {typ: keyTypeBool},
	"output":               {typ: keyTypeString, allowed: []string{"table", "json", "text"}},
	"proxy":                {typ: keyTypeString},
	"user_agent":           {typ: keyTypeString},
	"pap_limit":            {typ: keyTypeString, allowed: []string{"red", "amber", "green", "white"}},
	"defang":               {typ: keyTypeBool},
	"no_defang":            {typ: keyTypeBool},
	"concurrency":          {typ: keyTypeInt},
	"detect_patterns.url":  {typ: keyTypeString},
	"detect_patterns.file": {typ: keyTypeString},
	"tls_fingerprint":      {typ: keyTypeString, allowed: []string{"chrome", "firefox", "edge", "safari", "ios", "android", "randomized"}},
}

// ValidKeys returns every recognised config key in sorted order.
// The returned slice is always sorted; callers must not sort it again.
func ValidKeys() []string {
	keys := make([]string, 0, len(configKeys))
	for k := range configKeys {
		keys = append(keys, k)
	}
	slices.Sort(keys)
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
	if _, ok := configKeys[NormalizeKey(key)]; !ok {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	return nil
}

// ParseValue converts the raw string value to the correct Go type for key and
// validates enum constraints. Returns an error for type mismatches or invalid
// enum values.
func ParseValue(key, value string) (any, error) {
	meta, ok := configKeys[NormalizeKey(key)]
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

// NormalizeKey converts hyphenated flag names to their viper key equivalents
// (e.g. "pap-limit" → "pap_limit").
func NormalizeKey(key string) string {
	return strings.ReplaceAll(key, "-", "_")
}

// DetectPatternsConfig holds configuration for the detect patterns system.
type DetectPatternsConfig struct {
	URL  string `mapstructure:"url"`  // custom download URL; empty = built-in default
	File string `mapstructure:"file"` // custom patterns file; empty = use DefaultPatternPaths
}

// Config holds the runtime settings resolved from flags, env vars, and config file.
type Config struct {
	ConfigFile     string               // set after Unmarshal — no mapstructure tag
	Verbose        bool                 `mapstructure:"verbose"`
	Output         string               `mapstructure:"output"`          // table | json | text
	Proxy          string               `mapstructure:"proxy"`           // http://, https://, socks5://
	UserAgent      string               `mapstructure:"user_agent"`      // override or empty (→ rotation)
	PAPLimit       string               `mapstructure:"pap_limit"`       // "white" (default)
	Defang         bool                 `mapstructure:"defang"`          // force defang
	NoDefang       bool                 `mapstructure:"no_defang"`       // suppress defang
	Concurrency    int                  `mapstructure:"concurrency"`     // default 10
	Aliases        map[string]string    `mapstructure:"alias"`           // file-only; no flag/env binding
	DetectPatterns DetectPatternsConfig `mapstructure:"detect_patterns"` // detect patterns configuration
	TLSFingerprint string               `mapstructure:"tls_fingerprint"` // uTLS fingerprint (chrome, firefox, …)
}

// RegisterFlags defines all persistent CLI flags on the given FlagSet.
// Call this on the root command's PersistentFlags().
func RegisterFlags(flags *pflag.FlagSet) {
	flags.String("config", "", "config file (default: $XDG_CONFIG_HOME/trident/config.yaml)")
	flags.BoolP("verbose", "v", false, "enable verbose (debug) logging")
	flags.StringP("output", "o", "table", "output format: table, json, or text")
	flags.String("proxy", "", "proxy URL (http://, https://, or socks5://)")
	flags.String("user-agent", "", "HTTP User-Agent: preset name (chrome, firefox, safari, edge, ios, android) or custom string (default: trident/<version>)")
	flags.String("pap-limit", "white", "PAP limit: white, green, amber, or red")
	flags.Bool("defang", false, "defang text/plain output (dots → [.], http → hxxp)")
	flags.Bool("no-defang", false, "disable defanging even if enabled in config")
	flags.IntP("concurrency", "c", 10, "parallel workers for bulk stdin input")
	flags.String("patterns-file", "", "custom detect patterns file (overrides detect.yaml search)")
	flags.String("tls-fingerprint", "", "TLS client hello fingerprint (chrome, firefox, edge, safari, ios, android, randomized)")
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
	v.SetDefault("detect_patterns.url", DefaultPatternsURL)

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
	_ = v.BindPFlag("detect_patterns.file", flags.Lookup("patterns-file"))
	_ = v.BindPFlag("tls_fingerprint", flags.Lookup("tls-fingerprint"))

	// Config file resolution.
	configFile, _ := flags.GetString("config")
	useDefault := configFile == ""
	if useDefault {
		var err error
		configFile, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}
	v.SetConfigFile(configFile)

	if useDefault {
		if err := appdir.EnsureFile(configFile); err != nil {
			return nil, fmt.Errorf("ensuring config file: %w", err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		isNotFound := errors.As(err, &notFound) || os.IsNotExist(err)
		if !isNotFound || !useDefault {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	// Unmarshal → Config (mapstructure tags drive field assignment).
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}
	// ConfigFileUsed returns the path set via SetConfigFile even when
	// ReadInConfig failed with not-found, so this is always populated.
	cfg.ConfigFile = v.ConfigFileUsed()
	return &cfg, nil
}

// DefaultConfigPath returns the default config file path for trident.
// On Linux this is $XDG_CONFIG_HOME/trident/config.yaml.
func DefaultConfigPath() (string, error) {
	dir, err := appdir.ConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving config dir: %w", err)
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// WarnInsecurePermissions checks that the config file has safe permissions (0600).
// Returns a non-empty warning string when permissions are too open.
// Returns empty string when the file does not exist or cannot be stat'd.
func WarnInsecurePermissions(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	perm := info.Mode().Perm()
	if perm&0o077 != 0 {
		return fmt.Sprintf("config file %s has permissions %04o, want 0600; run: chmod 0600 %s", path, perm, path)
	}
	return ""
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
