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
		"--pap=amber",
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
