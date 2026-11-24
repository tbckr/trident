package httpclient_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
)

// These tests guard against traffic leaving the configured proxy: every TCP
// connection the origin server accepts must have arrived through a tunnel of
// the test proxy, checked connection by connection (the proxy records the local
// address of each upstream connection, the origin the remote address of each
// accepted one). Everything runs on loopback.

// testProxy is a minimal HTTP CONNECT or SOCKS5 proxy that counts tunnels and
// records the local address of every upstream connection it dials. It can kill
// tunnels to simulate a dropped proxy connection, freeze them to simulate a
// proxy that stops forwarding without closing anything, and stall handshakes.
type testProxy struct {
	ln      net.Listener
	scheme  string
	tunnels atomic.Int32

	mu       sync.Mutex
	conns    []net.Conn
	open     []*tunnel
	upstream []string
	stall    int
	stallTLS int
}

// tunnel is the relay state of one proxied connection.
type tunnel struct {
	frozen atomic.Bool
}

func newTestProxy(t *testing.T, scheme string) *testProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return newTestProxyOn(t, scheme, ln)
}

// newTestProxyOn starts the proxy on ln, for an address other than IPv4
// loopback.
func newTestProxyOn(t *testing.T, scheme string, ln net.Listener) *testProxy {
	t.Helper()
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
		if p.take(&p.stall) {
			// Never answered: no SOCKS greeting, no CONNECT reply. The
			// connection stays open until killTunnels.
			p.mu.Lock()
			p.conns = append(p.conns, c)
			p.mu.Unlock()
			continue
		}
		go p.handle(c)
	}
}

// take consumes one unit of a fault counter and reports whether there was one.
func (p *testProxy) take(counter *int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if *counter == 0 {
		return false
	}
	*counter--
	return true
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
	p.mu.Lock()
	p.upstream = append(p.upstream, up.LocalAddr().String())
	p.mu.Unlock()
	if !p.confirm(c) {
		_ = c.Close()
		_ = up.Close()
		return
	}
	p.tunnels.Add(1)
	tn := &tunnel{}
	tn.frozen.Store(p.take(&p.stallTLS))
	p.mu.Lock()
	p.conns = append(p.conns, c, up)
	p.open = append(p.open, tn)
	p.mu.Unlock()
	go func() {
		tn.relay(up, br)
		_ = up.Close()
	}()
	tn.relay(c, up)
	_ = c.Close()
}

// relay copies src to dst until src fails or dst cannot be written. Once the
// tunnel is frozen it keeps reading but discards what it reads, so both ends
// see a live connection that has gone silent. A peer that hangs up still ends
// the tunnel.
func (tn *tunnel) relay(dst io.Writer, src io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 && !tn.frozen.Load() {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// freezeTunnels makes every tunnel that is open now stop relaying in both
// directions while staying open. Tunnels opened later relay normally.
func (p *testProxy) freezeTunnels() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, tn := range p.open {
		tn.frozen.Store(true)
	}
}

// stallNext makes the proxy accept the next n client connections and never
// answer them: the SOCKS greeting or CONNECT reply is never sent.
func (p *testProxy) stallNext(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stall = n
}

// stallTLSNext makes the next n tunnels come up, so the client sees a
// successful CONNECT or SOCKS reply, and then relay nothing: a TLS handshake
// through them gets no answer.
func (p *testProxy) stallTLSNext(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stallTLS = n
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
// a new Tor circuit would. It also closes stalled connections.
func (p *testProxy) killTunnels() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		_ = c.Close()
	}
	p.conns = nil
	p.open = nil
}

// testOrigin is a TLS server that answers with the protocol of the request and
// records the remote address of every TCP connection it accepts.
type testOrigin struct {
	*httptest.Server

	mu       sync.Mutex
	addrs    []string
	accepted atomic.Int32
	active   atomic.Int32
}

// newTestOrigin starts a TLS origin that offers HTTP/2 if h2 is set and waits
// delay before it answers (or until the client has gone away).
func newTestOrigin(t *testing.T, h2 bool, delay time.Duration) *testOrigin {
	t.Helper()
	return newTestOriginWith(t, h2, delay, nil)
}

// newTestOriginWith is newTestOrigin with configure, if non-nil, applied to the
// server's http.Server before it starts (e.g. to set its HTTP2 config).
func newTestOriginWith(t *testing.T, h2 bool, delay time.Duration, configure func(*http.Server)) *testOrigin {
	t.Helper()
	o := &testOrigin{}
	o.Server = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.active.Add(1)
		defer o.active.Add(-1)
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		_, _ = io.WriteString(w, r.Proto)
	}))
	o.EnableHTTP2 = h2
	// Fault injection cuts connections in mid-handshake; the server would log
	// every one of those to the default logger.
	o.Config.ErrorLog = log.New(io.Discard, "", 0)
	o.Config.ConnState = func(c net.Conn, s http.ConnState) {
		if s != http.StateNew {
			return
		}
		o.mu.Lock()
		o.addrs = append(o.addrs, c.RemoteAddr().String())
		o.accepted.Add(1)
		o.mu.Unlock()
	}
	if configure != nil {
		configure(o.Config)
	}
	o.StartTLS()
	t.Cleanup(o.Close)
	return o
}

// conns returns the number of TCP connections the origin has accepted.
func (o *testOrigin) conns() int32 { return o.accepted.Load() }

// inFlight returns the number of requests the origin is handling right now.
func (o *testOrigin) inFlight() int32 { return o.active.Load() }

// remoteAddrs returns the remote address of every accepted connection.
func (o *testOrigin) remoteAddrs() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.addrs)
}

func newTestClient(t *testing.T, proxy string, origin *testOrigin) *req.Client {
	t.Helper()
	client, err := httpclient.New(proxy, "", nil, false)
	require.NoError(t, err)
	client.SetRootCertFromString(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: origin.Certificate().Raw})))
	return client
}

// untunnelledConns returns the remote address of every connection the origin
// accepted that is not the upstream connection of a tunnel of the proxy, that
// is, every connection that reached the origin without the proxy.
func untunnelledConns(p *testProxy, o *testOrigin) []string {
	p.mu.Lock()
	tunnelled := make(map[string]bool, len(p.upstream))
	for _, addr := range p.upstream {
		tunnelled[addr] = true
	}
	p.mu.Unlock()

	var direct []string
	for _, addr := range o.remoteAddrs() {
		if !tunnelled[addr] {
			direct = append(direct, addr)
		}
	}
	return direct
}

// assertNoDirectConns fails if the origin accepted a connection that did not
// come through the proxy, and names it.
func assertNoDirectConns(t *testing.T, p *testProxy, o *testOrigin) {
	t.Helper()
	assert.Empty(t, untunnelledConns(p, o), "origin accepted connections that did not come through the proxy: traffic bypassed it")
}

func TestNew_ProxyNotBypassedAfterTunnelDies(t *testing.T) {
	for _, scheme := range []string{"http", "socks5"} {
		t.Run(scheme, func(t *testing.T) {
			origin := newTestOrigin(t, true, 0)
			proxy := newTestProxy(t, scheme)
			client := newTestClient(t, proxy.url(), origin)

			resp, err := client.R().Get(origin.URL)
			require.NoError(t, err)
			assert.Equal(t, "HTTP/1.1", resp.String(), "behind a proxy the client must not use HTTP/2")
			require.EqualValues(t, 1, proxy.tunnels.Load(), "first request must be tunnelled")
			assertNoDirectConns(t, proxy, origin)

			proxy.killTunnels()

			// The next requests must go through a new tunnel, never directly
			// to the origin.
			for range 3 {
				_, err := client.R().Get(origin.URL)
				require.NoError(t, err)
				assertNoDirectConns(t, proxy, origin)
			}
			assert.GreaterOrEqual(t, proxy.tunnels.Load(), int32(2), "requests after the drop must use a new tunnel")
		})
	}
}

func TestNew_ProxyNotBypassedWithForcedHTTP2(t *testing.T) {
	origin := newTestOrigin(t, true, 0)
	proxy := newTestProxy(t, "http")
	client := newTestClient(t, proxy.url(), origin)
	// req's forced HTTP/2 ignores the proxy (imroc/req#206); the guard must
	// refuse the direct connection instead.
	client.EnableForceHTTP2()

	_, err := client.R().Get(origin.URL)
	require.ErrorIs(t, err, apperr.ErrProxyBypass)
	assert.Zero(t, origin.conns(), "the origin must not be reached directly")
	assertNoDirectConns(t, proxy, origin)
}

func TestNew_ProxyBypassIsNotRetried(t *testing.T) {
	origin := newTestOrigin(t, true, 0)
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
	origin := newTestOrigin(t, true, 0)
	client := newTestClient(t, "", origin)

	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", resp.String(), "without a proxy HTTP/2 is negotiated as before")
	assert.EqualValues(t, 1, origin.conns())
}

func clearProxyEnv(t *testing.T) {
	t.Helper()
	for _, env := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		t.Setenv(env, "")
	}
}

// getParallel sends n concurrent GETs and fails on the first error.
func getParallel(t *testing.T, client *req.Client, target string, n int) {
	t.Helper()
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			_, err := client.R().Get(target)
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
	origin := newTestOrigin(t, true, 0)
	client := newTestClient(t, "", origin)
	httpclient.ForceHTTP2(client, "")

	getParallel(t, client, origin.URL, 20)
	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", resp.String())
	assert.EqualValues(t, 1, origin.conns(), "forced HTTP/2 multiplexes all requests over one connection")
}

func TestForceHTTP2_BehindProxyStaysOnHTTP1(t *testing.T) {
	origin := newTestOrigin(t, true, 0)
	proxy := newTestProxy(t, "socks5")
	client := newTestClient(t, proxy.url(), origin)
	httpclient.ForceHTTP2(client, proxy.url())

	resp, err := client.R().Get(origin.URL)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/1.1", resp.String())
	assertNoDirectConns(t, proxy, origin)
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

func TestHarness_DetectsDirectConnection(t *testing.T) {
	clearProxyEnv(t)
	o := newTestOrigin(t, true, 0)
	p := newTestProxy(t, "http")

	proxied := newTestClient(t, p.url(), o)
	_, err := proxied.R().Get(o.URL)
	require.NoError(t, err)
	assert.Empty(t, untunnelledConns(p, o), "a request through the proxy is not a leak")

	require.NoError(t, getWithin(newStdClient(t, o, ""), o.URL, 10*time.Second))

	assert.Len(t, untunnelledConns(p, o), 1, "the direct connection must be reported, and only it")
}

// newStdClient returns a plain net/http client that trusts the origin and
// reaches it through proxy, or directly if proxy is empty.
func newStdClient(t *testing.T, o *testOrigin, proxy string) *http.Client {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(o.Certificate())
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		require.NoError(t, err)
		tr.Proxy = http.ProxyURL(proxyURL)
	}
	t.Cleanup(tr.CloseIdleConnections)
	return &http.Client{Transport: tr}
}

// getWithin GETs target and reads the body, failing if that takes longer than d.
func getWithin(c *http.Client, target string, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(r)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

func TestHarness_OriginDelaysAndCountsInFlightRequests(t *testing.T) {
	const delay = 300 * time.Millisecond
	o := newTestOrigin(t, true, delay)
	c := newStdClient(t, o, "")

	start := time.Now()
	done := make(chan error, 1)
	go func() { done <- getWithin(c, o.URL, 10*time.Second) }()

	require.Eventually(t, func() bool { return o.inFlight() == 1 }, 5*time.Second, 5*time.Millisecond)
	require.NoError(t, <-done)
	assert.GreaterOrEqual(t, time.Since(start), delay, "the origin must wait before it answers")
	assert.Eventually(t, func() bool { return o.inFlight() == 0 }, 5*time.Second, 5*time.Millisecond)
	assert.EqualValues(t, 1, o.conns())
}

func TestHarness_StalledHandshakes(t *testing.T) {
	cases := []struct {
		name  string
		stall func(p *testProxy)
		// tunnels is how many tunnels the proxy has confirmed when the first
		// request gives up.
		tunnels int32
	}{
		{"before the proxy reply", func(p *testProxy) { p.stallNext(1) }, 0},
		{"after the proxy reply", func(p *testProxy) { p.stallTLSNext(1) }, 1},
	}
	for _, scheme := range []string{"http", "socks5"} {
		for _, tc := range cases {
			t.Run(scheme+"/"+tc.name, func(t *testing.T) {
				o := newTestOrigin(t, true, 0)
				p := newTestProxy(t, scheme)
				c := newStdClient(t, o, p.url())
				tc.stall(p)

				require.Error(t, getWithin(c, o.URL, 300*time.Millisecond), "the stalled connection must not answer")
				assert.Equal(t, tc.tunnels, p.tunnels.Load())

				require.NoError(t, getWithin(c, o.URL, 10*time.Second), "only the next n connections stall")
				assert.Equal(t, tc.tunnels+1, p.tunnels.Load())
				assertNoDirectConns(t, p, o)
			})
		}
	}
}

func TestHarness_FrozenTunnel(t *testing.T) {
	for _, scheme := range []string{"http", "socks5"} {
		t.Run(scheme, func(t *testing.T) {
			o := newTestOrigin(t, true, 0)
			p := newTestProxy(t, scheme)
			c := newStdClient(t, o, p.url())

			require.NoError(t, getWithin(c, o.URL, 10*time.Second))
			require.EqualValues(t, 1, p.tunnels.Load())
			p.freezeTunnels()

			// The kept-alive connection is reused and goes nowhere.
			require.Error(t, getWithin(c, o.URL, 300*time.Millisecond), "a frozen tunnel must not answer")
			assert.EqualValues(t, 1, p.tunnels.Load(), "the request must have used the open tunnel")

			require.NoError(t, getWithin(c, o.URL, 10*time.Second), "tunnels opened after the freeze relay")
			assert.EqualValues(t, 2, p.tunnels.Load())
			assertNoDirectConns(t, p, o)
		})
	}
}
