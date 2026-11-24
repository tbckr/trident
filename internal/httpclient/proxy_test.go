package httpclient_test

import (
	"bufio"
	"encoding/binary"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
)

// These tests guard against traffic leaving the configured proxy: every TCP
// connection the origin server accepts must have arrived through a tunnel of
// the test proxy. Everything runs on loopback.

// testProxy is a minimal HTTP CONNECT or SOCKS5 proxy that counts tunnels and
// can kill them to simulate a dropped proxy connection.
type testProxy struct {
	ln      net.Listener
	scheme  string
	tunnels atomic.Int32

	mu    sync.Mutex
	conns []net.Conn
}

func newTestProxy(t *testing.T, scheme string) *testProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &testProxy{ln: ln, scheme: scheme}
	go p.serve()
	t.Cleanup(func() {
		_ = ln.Close()
		p.killTunnels()
	})
	return p
}

func (p *testProxy) url() string { return p.scheme + "://" + p.ln.Addr().String() }

func (p *testProxy) serve() {
	for {
		c, err := p.ln.Accept()
		if err != nil {
			return
		}
		go p.handle(c)
	}
}

func (p *testProxy) handle(c net.Conn) {
	br := bufio.NewReader(c)
	var target string
	var ok bool
	if p.scheme == "socks5" {
		target, ok = socks5Handshake(br, c)
	} else {
		target, ok = connectHandshake(br, c)
	}
	if !ok {
		_ = c.Close()
		return
	}
	up, err := net.Dial("tcp", target)
	if err != nil {
		_ = c.Close()
		return
	}
	if !p.confirm(c) {
		_ = c.Close()
		_ = up.Close()
		return
	}
	p.tunnels.Add(1)
	p.mu.Lock()
	p.conns = append(p.conns, c, up)
	p.mu.Unlock()
	go func() {
		_, _ = io.Copy(up, br)
		_ = up.Close()
	}()
	_, _ = io.Copy(c, up)
	_ = c.Close()
}

// connectHandshake reads an HTTP CONNECT request and returns its target.
func connectHandshake(br *bufio.Reader, _ net.Conn) (string, bool) {
	r, err := http.ReadRequest(br)
	if err != nil || r.Method != http.MethodConnect {
		return "", false
	}
	return r.Host, true
}

// socks5Handshake reads a no-auth SOCKS5 CONNECT request (RFC 1928) and
// returns its target.
func socks5Handshake(br *bufio.Reader, c net.Conn) (string, bool) {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(br, hdr); err != nil || hdr[0] != 5 {
		return "", false
	}
	if _, err := io.ReadFull(br, make([]byte, hdr[1])); err != nil {
		return "", false
	}
	if _, err := c.Write([]byte{5, 0}); err != nil {
		return "", false
	}
	req := make([]byte, 4)
	if _, err := io.ReadFull(br, req); err != nil || req[1] != 1 {
		return "", false
	}
	var host string
	switch req[3] {
	case 1, 4:
		ip := make([]byte, map[byte]int{1: 4, 4: 16}[req[3]])
		if _, err := io.ReadFull(br, ip); err != nil {
			return "", false
		}
		host = net.IP(ip).String()
	case 3:
		n, err := br.ReadByte()
		if err != nil {
			return "", false
		}
		name := make([]byte, n)
		if _, err := io.ReadFull(br, name); err != nil {
			return "", false
		}
		host = string(name)
	default:
		return "", false
	}
	port := make([]byte, 2)
	if _, err := io.ReadFull(br, port); err != nil {
		return "", false
	}
	return net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(port)))), true
}

// confirm tells the client that the tunnel is up.
func (p *testProxy) confirm(c net.Conn) bool {
	var reply []byte
	if p.scheme == "socks5" {
		reply = []byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	} else {
		reply = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")
	}
	_, err := c.Write(reply)
	return err == nil
}

// killTunnels closes every open tunnel, as a proxy restart, an idle timeout or
// a new Tor circuit would.
func (p *testProxy) killTunnels() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		_ = c.Close()
	}
	p.conns = nil
}

// newH2Origin starts a TLS server that offers HTTP/2, answers with the
// protocol of the request and counts the TCP connections it accepts.
func newH2Origin(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var conns atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Proto)
	}))
	srv.EnableHTTP2 = true
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv, &conns
}

func newTestClient(t *testing.T, proxy string, origin *httptest.Server) *req.Client {
	t.Helper()
	client, err := httpclient.New(proxy, "", nil, false)
	require.NoError(t, err)
	client.SetRootCertFromString(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: origin.Certificate().Raw})))
	return client
}

// assertNoDirectConns fails if the origin accepted a connection that did not
// come through the proxy.
func assertNoDirectConns(t *testing.T, proxy *testProxy, originConns *atomic.Int32) {
	t.Helper()
	assert.LessOrEqual(t, originConns.Load(), proxy.tunnels.Load(),
		"origin accepted %d connections but the proxy tunnelled only %d: traffic bypassed the proxy",
		originConns.Load(), proxy.tunnels.Load())
}

func TestNew_ProxyNotBypassedAfterTunnelDies(t *testing.T) {
	for _, scheme := range []string{"http", "socks5"} {
		t.Run(scheme, func(t *testing.T) {
			origin, originConns := newH2Origin(t)
			proxy := newTestProxy(t, scheme)
			client := newTestClient(t, proxy.url(), origin)

			resp, err := client.R().Get(origin.URL)
			require.NoError(t, err)
			assert.Equal(t, "HTTP/1.1", resp.String(), "behind a proxy the client must not use HTTP/2")
			require.EqualValues(t, 1, proxy.tunnels.Load(), "first request must be tunnelled")
			assertNoDirectConns(t, proxy, originConns)

			proxy.killTunnels()

			// The next requests must go through a new tunnel, never directly
			// to the origin.
			for range 3 {
				_, err := client.R().Get(origin.URL)
				require.NoError(t, err)
				assertNoDirectConns(t, proxy, originConns)
			}
			assert.GreaterOrEqual(t, proxy.tunnels.Load(), int32(2), "requests after the drop must use a new tunnel")
		})
	}
}

func TestNew_ProxyNotBypassedWithForcedHTTP2(t *testing.T) {
	origin, originConns := newH2Origin(t)
	proxy := newTestProxy(t, "http")
	client := newTestClient(t, proxy.url(), origin)
	// req's forced HTTP/2 ignores the proxy (imroc/req#206); the guard must
	// refuse the direct connection instead.
	client.EnableForceHTTP2()

	_, err := client.R().Get(origin.URL)
	require.ErrorIs(t, err, apperr.ErrProxyBypass)
	assert.Zero(t, originConns.Load(), "the origin must not be reached directly")
	assertNoDirectConns(t, proxy, originConns)
}

func TestNew_ProxyBypassIsNotRetried(t *testing.T) {
	origin, _ := newH2Origin(t)
	proxy := newTestProxy(t, "http")
	client := newTestClient(t, proxy.url(), origin)
	client.EnableForceHTTP2()
	httpclient.AttachRateLimit(client, ratelimit.New(100, 10))

	r := client.R()
	_, err := r.Get(origin.URL)
	require.ErrorIs(t, err, apperr.ErrProxyBypass)
	assert.Zero(t, r.RetryAttempt, "a refused proxy bypass must not be retried")
}

func TestNew_NoProxyKeepsHTTP2(t *testing.T) {
	for _, env := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		t.Setenv(env, "")
	}
	origin, originConns := newH2Origin(t)
	client := newTestClient(t, "", origin)

	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", resp.String(), "without a proxy HTTP/2 is negotiated as before")
	assert.EqualValues(t, 1, originConns.Load())
}

func clearProxyEnv(t *testing.T) {
	t.Helper()
	for _, env := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		t.Setenv(env, "")
	}
}

// getParallel sends n concurrent GETs and fails on the first error.
func getParallel(t *testing.T, client *req.Client, url string, n int) {
	t.Helper()
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			_, err := client.R().Get(url)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

func TestForceHTTP2_NoProxyUsesOneConnection(t *testing.T) {
	clearProxyEnv(t)
	origin, originConns := newH2Origin(t)
	client := newTestClient(t, "", origin)
	httpclient.ForceHTTP2(client, "")

	getParallel(t, client, origin.URL, 20)
	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", resp.String())
	assert.EqualValues(t, 1, originConns.Load(), "forced HTTP/2 multiplexes all requests over one connection")
}

func TestForceHTTP2_BehindProxyStaysOnHTTP1(t *testing.T) {
	origin, originConns := newH2Origin(t)
	proxy := newTestProxy(t, "socks5")
	client := newTestClient(t, proxy.url(), origin)
	httpclient.ForceHTTP2(client, proxy.url())

	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/1.1", resp.String())
	assertNoDirectConns(t, proxy, originConns)
}

func TestNew_ProxiedClientKeepsABurstOfIdleConns(t *testing.T) {
	// Behind a proxy the client speaks HTTP/1.1, so a burst of 10 concurrent
	// requests needs 10 connections; with the default of 2 idle connections
	// per host, every burst opens about 8 new tunnels. (Asserted on the
	// setting: when a connection returns to the idle pool is up to req's
	// read loop, so counting tunnels across back-to-back bursts is racy.)
	proxied, err := httpclient.New("socks5://127.0.0.1:9050", "", nil, false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, proxied.GetTransport().MaxIdleConnsPerHost, 10)

	clearProxyEnv(t)
	direct, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)
	assert.Zero(t, direct.GetTransport().MaxIdleConnsPerHost, "without a proxy req's default stays")
}
