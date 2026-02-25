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
	client, err := httpclient.New("", "", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithUserAgent(t *testing.T) {
	client, err := httpclient.New("", "MyBot/1.0", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithHTTPProxy(t *testing.T) {
	client, err := httpclient.New("http://proxy.example.com:8080", "", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithHTTPSProxy(t *testing.T) {
	client, err := httpclient.New("https://proxy.example.com:8080", "", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithSocks5Proxy(t *testing.T) {
	client, err := httpclient.New("socks5://127.0.0.1:9050", "", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_InvalidProxyScheme(t *testing.T) {
	_, err := httpclient.New("ftp://proxy.example.com:8080", "", "", nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proxy scheme")
}

func TestNew_DefaultUserAgent(t *testing.T) {
	client, err := httpclient.New("", "", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithDebugLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := httpclient.New("", "", "", logger, true)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithNilLoggerDebugFalse(t *testing.T) {
	client, err := httpclient.New("", "", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithEnvProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	client, err := httpclient.New("", "", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithTLSFingerprint(t *testing.T) {
	fingerprints := []string{"chrome", "firefox", "edge", "safari", "ios", "android", "randomized"}
	for _, fp := range fingerprints {
		t.Run(fp, func(t *testing.T) {
			client, err := httpclient.New("", "", fp, nil, false)
			require.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}

func TestNew_InvalidTLSFingerprint(t *testing.T) {
	_, err := httpclient.New("", "", "ie6", nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown TLS fingerprint")
}

func TestNew_PresetUADerivesTLS(t *testing.T) {
	// Passing a preset UA name with no explicit TLS should succeed (derived TLS is applied internally).
	client, err := httpclient.New("", "chrome", "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestPresetNames(t *testing.T) {
	names := httpclient.PresetNames()
	assert.Equal(t, []string{"android", "chrome", "edge", "firefox", "ios", "safari"}, names)
}

func TestResolveUserAgent_PresetName(t *testing.T) {
	for name, want := range httpclient.UserAgentPresets {
		t.Run(name, func(t *testing.T) {
			got := httpclient.ResolveUserAgent(name, "")
			assert.Equal(t, want, got)
		})
	}
}

func TestResolveUserAgent_CustomString(t *testing.T) {
	custom := "MyBot/2.0 (+https://example.com)"
	got := httpclient.ResolveUserAgent(custom, "")
	assert.Equal(t, custom, got)
}

func TestResolveUserAgent_TLSDerived(t *testing.T) {
	// Empty UA + TLS preset → matching browser UA.
	got := httpclient.ResolveUserAgent("", "firefox")
	assert.Equal(t, httpclient.UserAgentPresets["firefox"], got)
}

func TestResolveUserAgent_RandomizedNoDerive(t *testing.T) {
	// "randomized" TLS should NOT derive a UA.
	got := httpclient.ResolveUserAgent("", "randomized")
	assert.Equal(t, httpclient.DefaultUserAgent, got)
}

func TestResolveUserAgent_ExplicitWinsOverTLS(t *testing.T) {
	// Preset UA overrides TLS-derived UA.
	got := httpclient.ResolveUserAgent("chrome", "firefox")
	assert.Equal(t, httpclient.UserAgentPresets["chrome"], got)
}

func TestResolveUserAgent_CustomWinsOverTLS(t *testing.T) {
	// Custom string overrides TLS-derived UA.
	custom := "CustomAgent/1.0"
	got := httpclient.ResolveUserAgent(custom, "chrome")
	assert.Equal(t, custom, got)
}

func TestResolveUserAgent_Default(t *testing.T) {
	got := httpclient.ResolveUserAgent("", "")
	assert.Equal(t, httpclient.DefaultUserAgent, got)
}

func TestResolveTLSFingerprint_ExplicitValue(t *testing.T) {
	got := httpclient.ResolveTLSFingerprint("", "firefox")
	assert.Equal(t, "firefox", got)
}

func TestResolveTLSFingerprint_DerivedFromPreset(t *testing.T) {
	got := httpclient.ResolveTLSFingerprint("safari", "")
	assert.Equal(t, "safari", got)
}

func TestResolveTLSFingerprint_CustomUANoDerive(t *testing.T) {
	got := httpclient.ResolveTLSFingerprint("MyBot/1.0", "")
	assert.Equal(t, "", got)
}

func TestResolveTLSFingerprint_ExplicitWinsOverPreset(t *testing.T) {
	// Explicit TLS fingerprint wins over UA-derived.
	got := httpclient.ResolveTLSFingerprint("chrome", "firefox")
	assert.Equal(t, "firefox", got)
}

func TestResolveTLSFingerprint_Default(t *testing.T) {
	got := httpclient.ResolveTLSFingerprint("", "")
	assert.Equal(t, "", got)
}
