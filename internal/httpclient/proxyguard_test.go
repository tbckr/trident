package httpclient_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/httpclient"
)

func proxyURL(t *testing.T, raw string) func(*http.Request) (*url.URL, error) {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return http.ProxyURL(u)
}

func TestCheckDirectDial(t *testing.T) {
	tests := []struct {
		name    string
		proxy   string
		addr    string
		bypass  bool
		comment string
	}{
		{"proxy address itself", "http://proxy.example:3128", "proxy.example:3128", false, ""},
		{"origin behind http proxy", "http://proxy.example:3128", "crt.sh:443", true, ""},
		{"origin behind socks5 proxy", "socks5://127.0.0.1:9050", "dns.quad9.net:443", true, ""},
		{"http default port", "http://proxy.example", "proxy.example:80", false, ""},
		{"https default port", "https://proxy.example", "proxy.example:443", false, ""},
		{"socks5 default port", "socks5://127.0.0.1", "127.0.0.1:1080", false, ""},
		{"host case differs", "http://Proxy.Example:3128", "proxy.example:3128", false, ""},
		{"ipv6 proxy", "socks5://[::1]:9050", "[::1]:9050", false, ""},
		{"same host other port", "socks5://127.0.0.1:9050", "127.0.0.1:443", true, ""},
		{"credentials are redacted", "http://user:secret@proxy.example:3128", "crt.sh:443", true, "secret"},
		{"idn proxy host", "http://bücher.example:3128", "xn--bcher-kva.example:3128", false, ""},
	}
	for _, tt := range tests {
		s := settings(tt.proxy)
		funcs := map[string]func(*http.Request) (*url.URL, error){
			"url":      proxyURL(t, tt.proxy),
			"settings": s.ProxyFunc(nil),
		}
		for kind, proxyFor := range funcs {
			t.Run(tt.name+"/"+kind, func(t *testing.T) {
				require.NoError(t, s.Err())
				err := httpclient.CheckDirectDial(proxyFor, tt.addr)
				if !tt.bypass {
					require.NoError(t, err)
					return
				}
				require.ErrorIs(t, err, apperr.ErrProxyBypass)
				assert.Contains(t, err.Error(), tt.addr)
				if tt.comment != "" {
					assert.NotContains(t, err.Error(), tt.comment)
				}
			})
		}
	}
}

func TestCheckDirectDial_NoProxyForHost(t *testing.T) {
	// A proxy function that exempts the host (NO_PROXY) allows the direct dial.
	none := func(*http.Request) (*url.URL, error) { return nil, nil }
	require.NoError(t, httpclient.CheckDirectDial(none, "crt.sh:443"))
}

func TestCheckDirectDial_ProxyFuncError(t *testing.T) {
	boom := errors.New("boom")
	failing := func(*http.Request) (*url.URL, error) { return nil, boom }
	err := httpclient.CheckDirectDial(failing, "crt.sh:443")
	require.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, apperr.ErrProxyBypass)
}

func TestCheckDirectDial_SeparateHTTPAndHTTPSProxies(t *testing.T) {
	proxyFor := settings("", "HTTP_PROXY", "http://proxy-a.corp:3128", "HTTPS_PROXY", "http://proxy-b.corp:3128").ProxyFunc(nil)
	// Dials to either proxy are the proxied connections themselves.
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "proxy-a.corp:3128"))
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "proxy-b.corp:3128"))
	require.ErrorIs(t, httpclient.CheckDirectDial(proxyFor, "crt.sh:443"), apperr.ErrProxyBypass)
}

func TestCheckDirectDial_OnlyHTTPSProxyRefusesPlainHTTPTargets(t *testing.T) {
	// The dialer cannot tell http:// from https://, so with a proxy for https
	// only, a direct dial for an http:// URL is refused too (fails closed).
	proxyFor := settings("", "HTTPS_PROXY", "http://proxy-b.corp:3128").ProxyFunc(nil)
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "proxy-b.corp:3128"))
	require.ErrorIs(t, httpclient.CheckDirectDial(proxyFor, "patterns.example:80"), apperr.ErrProxyBypass)
}

func TestCheckDirectDial_BrokenSettingsFailClosed(t *testing.T) {
	// Only HTTP_PROXY: https would go direct, so the run has a deferred error,
	// and the guard refuses every dial with it.
	proxyFor := settings("", "HTTP_PROXY", "http://proxy-a.corp:3128").ProxyFunc(nil)
	for _, addr := range []string{"crt.sh:443", "proxy-a.corp:3128"} {
		err := httpclient.CheckDirectDial(proxyFor, addr)
		require.ErrorIs(t, err, apperr.ErrProxyConfig, addr)
	}
}

func TestCheckDirectDial_NoProxyExemption(t *testing.T) {
	proxyFor := settings("", "HTTPS_PROXY", "http://proxy-b.corp:3128", "NO_PROXY", "internal.corp").ProxyFunc(nil)
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "patterns.internal.corp:443"))
	require.ErrorIs(t, httpclient.CheckDirectDial(proxyFor, "crt.sh:443"), apperr.ErrProxyBypass)
}

func TestCheckDirectDial_RedactsWholeUserinfo(t *testing.T) {
	err := httpclient.CheckDirectDial(settings("http://user:secret@proxy.example:3128").ProxyFunc(nil), "crt.sh:443")
	require.ErrorIs(t, err, apperr.ErrProxyBypass)
	assert.Contains(t, err.Error(), "http://***@proxy.example:3128")
	assert.NotContains(t, err.Error(), "user", "no part of the userinfo may be shown")
	assert.NotContains(t, err.Error(), "secret", "no part of the userinfo may be shown")
}
