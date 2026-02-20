package httpclient_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/httpclient"
)

func TestNew_NoProxy(t *testing.T) {
	client, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithUserAgent(t *testing.T) {
	client, err := httpclient.New("", "MyBot/1.0", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithHTTPProxy(t *testing.T) {
	client, err := httpclient.New("http://proxy.example.com:8080", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithHTTPSProxy(t *testing.T) {
	client, err := httpclient.New("https://proxy.example.com:8080", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithSocks5Proxy(t *testing.T) {
	client, err := httpclient.New("socks5://127.0.0.1:9050", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_InvalidProxyScheme(t *testing.T) {
	_, err := httpclient.New("ftp://proxy.example.com:8080", "", nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proxy scheme")
}

func TestNew_RotatesUA(t *testing.T) {
	// Call multiple times and verify we get non-nil clients (UA rotation doesn't error)
	for i := range 5 {
		client, err := httpclient.New("", "", nil, false)
		require.NoError(t, err, "iteration %d", i)
		assert.NotNil(t, client)
	}
}

func TestNew_WithDebugLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := httpclient.New("", "", logger, true)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithNilLoggerDebugFalse(t *testing.T) {
	client, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithEnvProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	client, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}
