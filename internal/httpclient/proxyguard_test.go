package httpclient_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http/httpproxy"

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
		t.Run(tt.name, func(t *testing.T) {
			err := httpclient.CheckDirectDial(proxyURL(t, tt.proxy), tt.addr)
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

// envProxy is the proxy function of an environment, with the semantics of
// http.ProxyFromEnvironment but without its process-wide cache.
func envProxy(cfg httpproxy.Config) func(*http.Request) (*url.URL, error) {
	pf := cfg.ProxyFunc()
	return func(r *http.Request) (*url.URL, error) { return pf(r.URL) }
}

func TestCheckDirectDial_SeparateHTTPAndHTTPSProxies(t *testing.T) {
	proxyFor := envProxy(httpproxy.Config{HTTPProxy: "http://proxy-a.corp:3128", HTTPSProxy: "http://proxy-b.corp:3128"})
	// Dials to either proxy are the proxied connections themselves.
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "proxy-a.corp:3128"))
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "proxy-b.corp:3128"))
	require.ErrorIs(t, httpclient.CheckDirectDial(proxyFor, "crt.sh:443"), apperr.ErrProxyBypass)
}

func TestCheckDirectDial_OnlyHTTPProxy(t *testing.T) {
	// Go sends HTTPS directly when only HTTP_PROXY is set; that is no bypass.
	proxyFor := envProxy(httpproxy.Config{HTTPProxy: "http://proxy-a.corp:3128"})
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "crt.sh:443"))
}

func TestCheckDirectDial_NoProxyExemption(t *testing.T) {
	proxyFor := envProxy(httpproxy.Config{HTTPSProxy: "http://proxy-b.corp:3128", NoProxy: "internal.corp"})
	require.NoError(t, httpclient.CheckDirectDial(proxyFor, "patterns.internal.corp:443"))
	require.ErrorIs(t, httpclient.CheckDirectDial(proxyFor, "crt.sh:443"), apperr.ErrProxyBypass)
}

func TestRefuseLocalDNSProxies(t *testing.T) {
	tests := []struct {
		proxy  string
		refuse bool
	}{
		{"socks4://proxy.corp:1080", true},
		{"SOCKS4://proxy.corp:1080", true},
		{"socks4a://proxy.corp:1080", false},
		{"socks5://proxy.corp:1080", false},
		{"http://proxy.corp:3128", false},
	}
	target := &http.Request{URL: &url.URL{Scheme: "https", Host: "crt.sh"}}
	for _, tt := range tests {
		t.Run(tt.proxy, func(t *testing.T) {
			p, err := httpclient.RefuseLocalDNSProxies(proxyURL(t, tt.proxy))(target)
			if tt.refuse {
				require.ErrorIs(t, err, apperr.ErrProxyBypass)
				assert.Nil(t, p)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.proxy, p.String())
		})
	}

	none := func(*http.Request) (*url.URL, error) { return nil, nil }
	p, err := httpclient.RefuseLocalDNSProxies(none)(target)
	require.NoError(t, err)
	assert.Nil(t, p)
}
