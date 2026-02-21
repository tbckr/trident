package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/config"
)

// newTestFlags registers all config flags on a fresh FlagSet, then parses extra args.
func newTestFlags(t *testing.T, cfgFile string, extra ...string) *pflag.FlagSet {
	t.Helper()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	config.RegisterFlags(flags)
	args := append([]string{"--config=" + cfgFile}, extra...)
	require.NoError(t, flags.Parse(args))
	return flags
}

func TestLoad_DefaultsWithTempDir(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(newTestFlags(t, cfgFile))
	require.NoError(t, err)
	assert.Equal(t, cfgFile, cfg.ConfigFile)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "text", cfg.Output)
	assert.Equal(t, "white", cfg.PAPLimit)
	assert.Equal(t, 10, cfg.Concurrency)
	assert.False(t, cfg.Defang)
	assert.False(t, cfg.NoDefang)

	// Config file should now exist with 0600 permissions.
	info, err := os.Stat(cfgFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestLoad_ExistingConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	// Pre-create the file; Load must not fail if it already exists.
	require.NoError(t, os.WriteFile(cfgFile, []byte{}, 0o600))

	cfg, err := config.Load(newTestFlags(t, cfgFile, "--verbose", "--output=json"))
	require.NoError(t, err)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, "json", cfg.Output)
}

func TestLoad_VerboseAndOutput(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(newTestFlags(t, cfgFile, "--verbose", "--output=json"))
	require.NoError(t, err)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, "json", cfg.Output)
}

func TestLoad_NewFields(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(newTestFlags(t, cfgFile,
		"--proxy=http://proxy:8080",
		"--user-agent=MyAgent/1.0",
		"--pap-limit=amber",
		"--defang",
		"--concurrency=5",
	))
	require.NoError(t, err)
	assert.Equal(t, "http://proxy:8080", cfg.Proxy)
	assert.Equal(t, "MyAgent/1.0", cfg.UserAgent)
	assert.Equal(t, "amber", cfg.PAPLimit)
	assert.True(t, cfg.Defang)
	assert.False(t, cfg.NoDefang)
	assert.Equal(t, 5, cfg.Concurrency)
}

func TestLoad_ConcurrencyDefault(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(newTestFlags(t, cfgFile))
	require.NoError(t, err)
	assert.Equal(t, 10, cfg.Concurrency)
}

func TestLoad_PAPLimitDefault(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(newTestFlags(t, cfgFile))
	require.NoError(t, err)
	assert.Equal(t, "white", cfg.PAPLimit)
}

func TestValidateKey(t *testing.T) {
	t.Run("valid_underscore", func(t *testing.T) {
		require.NoError(t, config.ValidateKey("pap_limit"))
	})
	t.Run("valid_hyphen", func(t *testing.T) {
		require.NoError(t, config.ValidateKey("pap-limit"))
	})
	t.Run("all_keys", func(t *testing.T) {
		for _, k := range config.ValidKeys() {
			require.NoError(t, config.ValidateKey(k), "key %q should be valid", k)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		err := config.ValidateKey("does_not_exist")
		require.Error(t, err)
		require.ErrorIs(t, err, config.ErrUnknownKey)
	})
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		want    any
		wantErr bool
	}{
		// bool
		{key: "verbose", value: "true", want: true},
		{key: "verbose", value: "false", want: false},
		{key: "defang", value: "1", want: true},
		{key: "verbose", value: "yes", wantErr: true},
		// int
		{key: "concurrency", value: "5", want: 5},
		{key: "concurrency", value: "0", wantErr: true},
		{key: "concurrency", value: "-1", wantErr: true},
		{key: "concurrency", value: "abc", wantErr: true},
		// enum string — output
		{key: "output", value: "json", want: "json"},
		{key: "output", value: "text", want: "text"},
		{key: "output", value: "plain", want: "plain"},
		{key: "output", value: "xml", wantErr: true},
		// enum string — pap_limit (hyphenated key)
		{key: "pap-limit", value: "amber", want: "amber"},
		{key: "pap_limit", value: "white", want: "white"},
		{key: "pap_limit", value: "invalid", wantErr: true},
		// free-form string
		{key: "proxy", value: "http://proxy:3128", want: "http://proxy:3128"},
		{key: "user_agent", value: "MyAgent/1.0", want: "MyAgent/1.0"},
	}
	for _, tc := range tests {
		t.Run(tc.key+"/"+tc.value, func(t *testing.T) {
			got, err := config.ParseValue(tc.key, tc.value)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseValue_UnknownKey(t *testing.T) {
	_, err := config.ParseValue("nonexistent", "value")
	require.ErrorIs(t, err, config.ErrUnknownKey)
}

func TestLoad_ConfigFileValues(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	// Write config file with explicit values to verify file-based precedence.
	yamlContent := "proxy: \"http://fileproxy:3128\"\nuser_agent: \"FileAgent/2.0\"\npap_limit: \"green\"\nconcurrency: 20\n"
	require.NoError(t, os.WriteFile(cfgFile, []byte(yamlContent), 0o600))

	// No CLI flags for these keys — viper should read them from the file.
	cfg, err := config.Load(newTestFlags(t, cfgFile))
	require.NoError(t, err)
	assert.Equal(t, "http://fileproxy:3128", cfg.Proxy)
	assert.Equal(t, "FileAgent/2.0", cfg.UserAgent)
	assert.Equal(t, "green", cfg.PAPLimit)
	assert.Equal(t, 20, cfg.Concurrency)
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := config.DefaultConfigPath()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.True(t, filepath.IsAbs(path), "expected absolute path, got %q", path)
	assert.Equal(t, "config.yaml", filepath.Base(path))
	assert.Equal(t, "trident", filepath.Base(filepath.Dir(path)))
}

func TestLoadAliases_FileNotFound(t *testing.T) {
	aliases, err := config.LoadAliases("/nonexistent/path/config.yaml")
	require.NoError(t, err)
	assert.NotNil(t, aliases)
	assert.Empty(t, aliases)
}

func TestLoadAliases_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte{}, 0o600))

	aliases, err := config.LoadAliases(cfgFile)
	require.NoError(t, err)
	assert.NotNil(t, aliases)
	assert.Empty(t, aliases)
}

func TestLoadAliases_NoAliasesKey(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("output: json\nverbose: true\n"), 0o600))

	aliases, err := config.LoadAliases(cfgFile)
	require.NoError(t, err)
	assert.NotNil(t, aliases)
	assert.Empty(t, aliases)
}

func TestLoadAliases_WithAliases(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("alias:\n  mydns: dns -o json\n  qs: crtsh\n"), 0o600))

	aliases, err := config.LoadAliases(cfgFile)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"mydns": "dns -o json",
		"qs":    "crtsh",
	}, aliases)
}
