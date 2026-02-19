package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/config"
)

func TestLoad_DefaultsWithTempDir(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(cfgFile, false, "text")
	require.NoError(t, err)
	assert.Equal(t, cfgFile, cfg.ConfigFile)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "text", cfg.Output)

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

	cfg, err := config.Load(cfgFile, true, "json")
	require.NoError(t, err)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, "json", cfg.Output)
}

func TestLoad_VerboseAndOutput(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(cfgFile, true, "json")
	require.NoError(t, err)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, "json", cfg.Output)
}
