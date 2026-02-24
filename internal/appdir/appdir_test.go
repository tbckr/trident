package appdir_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/appdir"
)

func TestConfigDir(t *testing.T) {
	dir, err := appdir.ConfigDir()
	require.NoError(t, err)
	assert.NotEmpty(t, dir)
	assert.True(t, filepath.IsAbs(dir), "expected absolute path, got %q", dir)
	assert.True(t, strings.HasSuffix(dir, "/trident") || strings.HasSuffix(dir, `\trident`),
		"expected path ending in /trident or \\trident, got %q", dir)
}

func TestEnsureFile_CreatesFileAndDir(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "subdir", "file.txt")

	err := appdir.EnsureFile(path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestEnsureFile_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")

	require.NoError(t, appdir.EnsureFile(path))
	require.NoError(t, appdir.EnsureFile(path)) // second call must not error
}
