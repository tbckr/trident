package httpclient_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
)

// dohTransport returns the Go transport a DoH client sends its requests through.
func dohTransport(t *testing.T, client *req.Client) *http.Transport {
	t.Helper()
	tr, ok := client.GetClient().Transport.(*http.Transport)
	require.True(t, ok, "the DoH client must run on a *http.Transport, got %T", client.GetClient().Transport)
	return tr
}

// trustOrigin makes both transports of a DoH client trust the origin's
// certificate: req's own (the fallback for Clone and RoundTrip) and the Go
// transport. It must run before the first request.
func trustOrigin(t *testing.T, client *req.Client, o *testOrigin) {
	t.Helper()
	client.SetRootCertFromString(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: o.Certificate().Raw})))
	pool := x509.NewCertPool()
	pool.AddCert(o.Certificate())
	tr := dohTransport(t, client)
	tr.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	t.Cleanup(tr.CloseIdleConnections)
}

// newDoHTestClient builds a DoH client for the origin through proxy (the
// --proxy flag value, or "" for none) and trusts the origin.
func newDoHTestClient(t *testing.T, proxy string, o *testOrigin) (*req.Client, bool) {
	t.Helper()
	client, proxied, err := httpclient.NewDoH(proxy, o.URL, "", nil, false)
	require.NoError(t, err)
	trustOrigin(t, client, o)
	return client, proxied
}

// newDoHSeamClient builds a DoH client for the origin with the proxy function
// proxyFor, as HTTPS_PROXY would set it, and trusts the origin.
func newDoHSeamClient(t *testing.T, proxyFor func(*http.Request) (*url.URL, error), o *testOrigin) (*req.Client, bool) {
	t.Helper()
	client, proxied, err := httpclient.NewDoHWithProxyFunc(proxyFor, true, o.URL, "", nil, false)
	require.NoError(t, err)
	trustOrigin(t, client, o)
	return client, proxied
}

func TestNewDoH_HTTP2ThroughProxy(t *testing.T) {
	cases := []struct {
		name string
		// scheme is the protocol the test proxy speaks.
		scheme string
		client func(t *testing.T, p *testProxy, o *testOrigin) (*req.Client, bool)
	}{
		{"http", "http", func(t *testing.T, p *testProxy, o *testOrigin) (*req.Client, bool) {
			return newDoHTestClient(t, p.url(), o)
		}},
		{"socks5", "socks5", func(t *testing.T, p *testProxy, o *testOrigin) (*req.Client, bool) {
			return newDoHTestClient(t, p.url(), o)
		}},
		{"socks5h", "socks5", func(t *testing.T, p *testProxy, o *testOrigin) (*req.Client, bool) {
			return newDoHSeamClient(t, proxyURL(t, "socks5h://"+p.ln.Addr().String()), o)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origin := newTestOrigin(t, true, 0)
			proxy := newTestProxy(t, tc.scheme)
			client, proxied := tc.client(t, proxy, origin)
			assert.True(t, proxied, "a proxy applies to the endpoint")

			resp, err := client.R().Get(origin.URL)
			require.NoError(t, err)
			assert.Equal(t, "HTTP/2.0", resp.String(), "DoH must speak HTTP/2 through the proxy")
			assert.EqualValues(t, 1, proxy.tunnels.Load())
			assertNoDirectConns(t, proxy, origin)
		})
	}
}

func TestNewDoH_NoProxyUsesOneConnection(t *testing.T) {
	clearProxyEnv(t)
	origin := newTestOrigin(t, true, 0)
	client, proxied := newDoHTestClient(t, "", origin)
	assert.False(t, proxied)

	getParallel(t, client, origin.URL, 20)
	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", resp.String())
	assert.EqualValues(t, 1, origin.conns(), "without a proxy all requests share one HTTP/2 connection")
}

func TestNewDoH_StreamLimitDoesNotDeadlock(t *testing.T) {
	clearProxyEnv(t)
	origin := newTestOriginWith(t, true, 300*time.Millisecond, func(s *http.Server) {
		s.HTTP2 = &http.HTTP2Config{MaxConcurrentStreams: 2}
	})
	client, proxied := newDoHTestClient(t, "", origin)
	require.False(t, proxied)
	require.Equal(t, 1, dohTransport(t, client).MaxConnsPerHost)

	// The cap of one connection must not make requests wait for a stream slot
	// that never frees up. (Go opens another connection when the first is at
	// its stream limit, cap or not: about 5 here.)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errs := make(chan error, 10)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_, err := client.R().SetContext(ctx).Get(origin.URL)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err, "10 requests against a server allowing 2 streams must finish within 5 s")
	}
}

func TestNewDoH_ProxiedFollowsProxyFunc(t *testing.T) {
	const endpoint = "https://dns.example/dns-query"
	socks, err := url.Parse("socks5://127.0.0.1:9050")
	require.NoError(t, err)

	// As NO_PROXY=dns.example would: every host but the endpoint is proxied.
	exempt := func(r *http.Request) (*url.URL, error) {
		if r.URL.Hostname() == "dns.example" {
			return nil, nil
		}
		return socks, nil
	}
	client, proxied, err := httpclient.NewDoHWithProxyFunc(exempt, true, endpoint, "", nil, false)
	require.NoError(t, err)
	assert.False(t, proxied, "no proxy applies to the endpoint")
	assert.Equal(t, 1, dohTransport(t, client).MaxConnsPerHost, "direct: one multiplexed connection")

	client, proxied, err = httpclient.NewDoHWithProxyFunc(http.ProxyURL(socks), true, endpoint, "", nil, false)
	require.NoError(t, err)
	assert.True(t, proxied)
	assert.Zero(t, dohTransport(t, client).MaxConnsPerHost, "behind a proxy a stalled handshake must not starve the host")
}

func TestNewDoH_ProxyFuncErrorFailsConstruction(t *testing.T) {
	// As HTTPS_PROXY=socks4://... would: whether the endpoint is proxied is
	// unknown, so there is no client to fall back on.
	socks4 := httpclient.RefuseLocalDNSProxies(proxyURL(t, "socks4://user:secret@127.0.0.1:1080"))
	client, proxied, err := httpclient.NewDoHWithProxyFunc(socks4, true, "https://dns.example/dns-query", "", nil, false)
	require.ErrorIs(t, err, apperr.ErrProxyBypass)
	assert.NotContains(t, err.Error(), "secret", "proxy credentials must not appear in errors")
	assert.Nil(t, client)
	assert.False(t, proxied)
}

func TestNewDoH_TransportConfig(t *testing.T) {
	client, _, err := httpclient.NewDoH("socks5://127.0.0.1:9050", "https://dns.example/dns-query", "", nil, false)
	require.NoError(t, err)
	tr := dohTransport(t, client)

	assert.Equal(t, 30*time.Second, client.GetClient().Timeout)
	require.NotNil(t, tr.HTTP2)
	assert.Equal(t, 15*time.Second, tr.HTTP2.SendPingTimeout)
	assert.Equal(t, 15*time.Second, tr.HTTP2.PingTimeout)
	assert.Equal(t, 10*time.Second, tr.TLSHandshakeTimeout, "the stdlib defaults must carry over")
	assert.Equal(t, 90*time.Second, tr.IdleConnTimeout, "the stdlib defaults must carry over")
	assert.True(t, tr.ForceAttemptHTTP2)
	assert.NotNil(t, tr.Proxy)
	assert.NotNil(t, tr.DialContext)
	assert.Nil(t, tr.DialTLSContext, "a TLS dialer would bypass DialContext and the guard")

	httpclient.SetDoHTimeouts(t, time.Second, 2*time.Second, 3*time.Second)
	client, _, err = httpclient.NewDoH("socks5://127.0.0.1:9050", "https://dns.example/dns-query", "", nil, false)
	require.NoError(t, err)
	tr = dohTransport(t, client)
	assert.Equal(t, time.Second, client.GetClient().Timeout)
	assert.Equal(t, 2*time.Second, tr.HTTP2.SendPingTimeout)
	assert.Equal(t, 3*time.Second, tr.HTTP2.PingTimeout)
}

func TestNewDoH_GuardRefusesDesyncedTransport(t *testing.T) {
	origin := newTestOrigin(t, true, 0)
	proxy := newTestProxy(t, "socks5")
	client, _ := newDoHTestClient(t, proxy.url(), origin)
	httpclient.AttachRateLimit(client, ratelimit.New(100, 10))
	// A transport whose proxy function forgets the proxy must fail closed.
	dohTransport(t, client).Proxy = func(*http.Request) (*url.URL, error) { return nil, nil }

	r := client.R()
	_, err := r.Get(origin.URL)
	require.ErrorIs(t, err, apperr.ErrProxyBypass)
	assert.Zero(t, r.RetryAttempt, "a refused proxy bypass must not be retried")
	assert.Zero(t, origin.conns(), "the origin must not be reached directly")
}

func TestNewDoH_FallbackTransportStaysProxied(t *testing.T) {
	for _, scheme := range []string{"http", "socks5"} {
		t.Run(scheme, func(t *testing.T) {
			origin := newTestOrigin(t, true, 0)
			proxy := newTestProxy(t, scheme)
			client, _ := newDoHTestClient(t, proxy.url(), origin)

			// Clone rebuilds the client on req's own transport.
			resp, err := client.Clone().R().Get(origin.URL)
			require.NoError(t, err)
			assert.Equal(t, "HTTP/1.1", resp.String(), "req's transport stays on HTTP/1.1 behind a proxy")

			// RoundTrip uses req's transport directly.
			httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, origin.URL, nil)
			require.NoError(t, err)
			raw, err := client.RoundTrip(httpReq)
			require.NoError(t, err)
			body, err := io.ReadAll(raw.Body)
			require.NoError(t, raw.Body.Close())
			require.NoError(t, err)
			assert.Equal(t, "HTTP/1.1", string(body))

			assert.GreaterOrEqual(t, proxy.tunnels.Load(), int32(1))
			assertNoDirectConns(t, proxy, origin)
		})
	}
}

func TestNewDoH_RefusesUnsupportedSchemes(t *testing.T) {
	tests := []struct {
		proxy  string
		scheme string
	}{
		{"socks4://127.0.0.1:1", "socks4"},
		{"socks4a://127.0.0.1:1", "socks4a"},
		{"ftp://127.0.0.1:1", "ftp"},
		{"socks4a://user:secret@127.0.0.1:1", "socks4a"},
	}
	for _, tt := range tests {
		t.Run(tt.proxy, func(t *testing.T) {
			origin := newTestOrigin(t, true, 0)
			client, _ := newDoHSeamClient(t, proxyURL(t, tt.proxy), origin)
			httpclient.AttachRateLimit(client, ratelimit.New(100, 10))

			r := client.R()
			_, err := r.Get(origin.URL)
			require.ErrorIs(t, err, apperr.ErrUnsupportedProxy)
			assert.Contains(t, err.Error(), "uses "+tt.scheme+"://")
			assert.NotContains(t, err.Error(), "secret", "proxy credentials must not appear in errors")
			assert.Zero(t, r.RetryAttempt, "an unsupported proxy must not be retried")
			assert.Zero(t, origin.conns())
		})
	}
}

func TestNewDoH_IPv6Proxy(t *testing.T) {
	ln, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	origin := newTestOrigin(t, true, 0)
	proxy := newTestProxyOn(t, "socks5", ln)
	client, proxied := newDoHTestClient(t, proxy.url(), origin)
	require.True(t, proxied)

	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", resp.String())
	assert.EqualValues(t, 1, proxy.tunnels.Load())
	assertNoDirectConns(t, proxy, origin)
}
