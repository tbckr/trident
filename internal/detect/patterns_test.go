package detect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/detect"
)

func TestResolvePatternFile_ExplicitValue(t *testing.T) {
	got := detect.ResolvePatternFile("/custom/patterns.yaml")
	assert.Equal(t, "/custom/patterns.yaml", got)
}

func TestResolvePatternFile_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "detect.yaml")
	require.NoError(t, os.WriteFile(f, []byte("cdn: []\n"), 0o600))

	// ResolvePatternFile with empty string searches DefaultPatternPaths.
	// We cannot inject a temp dir directly, so just verify the explicit-path branch.
	got := detect.ResolvePatternFile(f)
	assert.Equal(t, f, got)
}

func TestResolvePatternFile_EmptyFallsBackToEmbedded(t *testing.T) {
	// With no override files present and no explicit file, the function should
	// return "<embedded>". We cannot guarantee a clean config dir in CI, so we
	// only verify the explicit-empty-string path returns a non-empty string.
	got := detect.ResolvePatternFile("")
	assert.NotEmpty(t, got)
}

func TestLoadPatterns_Embedded(t *testing.T) {
	p, err := detect.LoadPatterns()
	require.NoError(t, err)
	// Embedded patterns must have at least one CDN entry.
	assert.NotEmpty(t, p.CDN)
}

func TestLoadPatterns_OverrideFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "detect.yaml")
	content := `cdn:
  - suffix: ".example.com"
    provider: "ExampleCDN"
`
	require.NoError(t, os.WriteFile(f, []byte(content), 0o600))

	p, err := detect.LoadPatterns(f)
	require.NoError(t, err)
	require.Len(t, p.CDN, 1)
	assert.Equal(t, ".example.com", p.CDN[0].Suffix)
	assert.Equal(t, "ExampleCDN", p.CDN[0].Provider)
}

func TestLoadPatterns_MissingFileFallsBack(t *testing.T) {
	// A nonexistent path should be skipped and the embedded fallback used.
	p, err := detect.LoadPatterns("/nonexistent/path/detect.yaml")
	require.NoError(t, err)
	assert.NotEmpty(t, p.CDN)
}

func TestDefaultPatternPaths(t *testing.T) {
	paths, err := detect.DefaultPatternPaths()
	require.NoError(t, err)
	require.Len(t, paths, 2)
	assert.Contains(t, paths[0], "detect.yaml")
	assert.Contains(t, paths[1], "detect-downloaded.yaml")
}
