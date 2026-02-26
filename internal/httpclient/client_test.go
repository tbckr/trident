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

func TestNew_DefaultUserAgent(t *testing.T) {
	client, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
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

func TestResolveProxy_ExplicitValue(t *testing.T) {
	got := httpclient.ResolveProxy("http://proxy.example.com:8080")
	assert.Equal(t, "http://proxy.example.com:8080", got)
}

func TestResolveProxy_EnvHTTPSProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://envproxy.example.com:8080")
	got := httpclient.ResolveProxy("")
	assert.Equal(t, "<from environment>", got)
}

func TestResolveProxy_EnvHTTPProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://envproxy.example.com:8080")
	got := httpclient.ResolveProxy("")
	assert.Equal(t, "<from environment>", got)
}

func TestResolveProxy_EnvALLProxy(t *testing.T) {
	t.Setenv("ALL_PROXY", "socks5://envproxy.example.com:1080")
	got := httpclient.ResolveProxy("")
	assert.Equal(t, "<from environment>", got)
}

func TestResolveProxy_EnvLowercase(t *testing.T) {
	t.Setenv("https_proxy", "http://envproxy.example.com:8080")
	got := httpclient.ResolveProxy("")
	assert.Equal(t, "<from environment>", got)
}

func TestResolveProxy_ExplicitWinsOverEnv(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://envproxy.example.com:8080")
	got := httpclient.ResolveProxy("http://explicit.example.com:8080")
	assert.Equal(t, "http://explicit.example.com:8080", got)
}

func TestResolveProxy_NoProxy(t *testing.T) {
	// Ensure no proxy env vars are set.
	for _, env := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		t.Setenv(env, "")
	}
	got := httpclient.ResolveProxy("")
	assert.Equal(t, "", got)
}
