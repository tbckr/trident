package resolver

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/dns/dnsmessage"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/proxyconf"
)

// env returns a getenv function over m, for proxyconf.Resolve.
func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestNewResolver_NoProxy(t *testing.T) {
	r, err := NewResolver(proxyconf.Resolve("", nil))
	require.NoError(t, err)
	assert.Nil(t, r.r.Dial, "no proxy: the system resolver")
	assert.False(t, r.rooted, "the system resolver keeps the search domains")
}

func TestNewResolver_HTTPProxyUsesSystemResolver(t *testing.T) {
	for _, p := range []string{"http://127.0.0.1:3128", "https://127.0.0.1:3128", "127.0.0.1:3128"} {
		t.Run(p, func(t *testing.T) {
			r, err := NewResolver(proxyconf.Resolve(p, nil))
			require.NoError(t, err)
			assert.Nil(t, r.r.Dial, "DNS cannot go through an HTTP proxy: the system resolver")
			assert.False(t, r.rooted)
		})
	}
}

func TestNewResolver_NoProxyStar(t *testing.T) {
	s := proxyconf.Resolve("", env(map[string]string{
		"ALL_PROXY": "socks5h://127.0.0.1:9050",
		"NO_PROXY":  "*",
	}))
	r, err := NewResolver(s)
	require.NoError(t, err)
	assert.Nil(t, r.r.Dial, "NO_PROXY=* connects directly: the system resolver")
	assert.False(t, r.rooted)
}

func TestNewResolver_BrokenSettingsFail(t *testing.T) {
	r, err := NewResolver(proxyconf.Resolve("socks4://h:1", nil))
	require.ErrorIs(t, err, apperr.ErrProxyConfig)
	assert.Nil(t, r)
}

func TestNewResolver_SOCKSProxyFromEnvironment(t *testing.T) {
	// ALL_PROXY alone supplies the https proxy, and DNS follows it.
	r, err := NewResolver(proxyconf.Resolve("", env(map[string]string{"ALL_PROXY": "socks5h://127.0.0.1:9050"})))
	require.NoError(t, err)
	assert.NotNil(t, r.r.Dial, "a SOCKS proxy carries DNS")
	assert.True(t, r.r.PreferGo, "only the Go resolver uses Dial")
	assert.True(t, r.rooted, "names through the proxy are queried rooted")
}

func TestNewResolver_SOCKSQueriesPinnedNameserver(t *testing.T) {
	ns := newFakeDNS(t)
	saved := nameserver
	nameserver = ns.addr()
	t.Cleanup(func() { nameserver = saved })
	socks := newFakeSOCKS(t, ns.addr())

	tests := []struct {
		name       string
		proxy      string
		user, pass string
	}{
		{name: "credentials become SOCKS auth", proxy: "socks5://alice:p%40ss@" + socks.addr(), user: "alice", pass: "p@ss"},
		{name: "socks5h", proxy: "socks5h://" + socks.addr()},
		{name: "uppercase scheme and trailing slash", proxy: "SOCKS5H://" + socks.addr() + "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socks.reset()
			ns.reset()
			r, err := NewResolver(proxyconf.Resolve(tt.proxy, nil))
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			addrs, err := r.LookupIPAddr(ctx, "example.com")
			require.NoError(t, err)
			var ips []string
			for _, a := range addrs {
				ips = append(ips, a.IP.String())
			}
			assert.Contains(t, ips, "192.0.2.1")

			reqs := socks.requests()
			require.NotEmpty(t, reqs, "the queries must go through the SOCKS proxy")
			for _, rq := range reqs {
				assert.Equal(t, ns.addr(), rq.target, "every query goes to the pinned nameserver")
				assert.Equal(t, tt.user, rq.user)
				assert.Equal(t, tt.pass, rq.pass)
			}

			// A single-label name is the one a search domain would be appended
			// to; rooted, it leaves through the proxy as typed.
			ns.reset()
			_, err = r.LookupIPAddr(ctx, "intranet")
			require.Error(t, err)
			seen := ns.seen()
			require.NotEmpty(t, seen)
			for _, name := range seen {
				assert.Equal(t, "intranet.", name, "no search domain may be appended")
			}
		})
	}
}

func TestResolver_EveryLookupGoesThroughTheProxyRooted(t *testing.T) {
	ns := newFakeDNS(t)
	saved := nameserver
	nameserver = ns.addr()
	t.Cleanup(func() { nameserver = saved })
	socks := newFakeSOCKS(t, ns.addr())

	r, err := NewResolver(proxyconf.Resolve("socks5h://"+socks.addr(), nil))
	require.NoError(t, err)

	lookups := []struct {
		name   string
		lookup func(ctx context.Context) error
		want   string
	}{
		{"MX", func(ctx context.Context) error { _, err := r.LookupMX(ctx, "intranet"); return err }, "intranet."},
		{"NS", func(ctx context.Context) error { _, err := r.LookupNS(ctx, "intranet"); return err }, "intranet."},
		{"TXT", func(ctx context.Context) error { _, err := r.LookupTXT(ctx, "intranet"); return err }, "intranet."},
		{"CNAME", func(ctx context.Context) error { _, err := r.LookupCNAME(ctx, "intranet"); return err }, "intranet."},
		{"SRV", func(ctx context.Context) error { _, _, err := r.LookupSRV(ctx, "sip", "tcp", "intranet"); return err }, "_sip._tcp.intranet."},
		{"PTR", func(ctx context.Context) error { _, err := r.LookupAddr(ctx, "192.0.2.1"); return err }, "1.2.0.192.in-addr.arpa."},
	}
	for _, tt := range lookups {
		t.Run(tt.name, func(t *testing.T) {
			socks.reset()
			ns.reset()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			require.Error(t, tt.lookup(ctx), "the fake nameserver answers NXDOMAIN")
			seen := ns.seen()
			require.NotEmpty(t, seen)
			for _, name := range seen {
				assert.Equal(t, tt.want, name)
			}
			reqs := socks.requests()
			require.NotEmpty(t, reqs)
			for _, rq := range reqs {
				assert.Equal(t, ns.addr(), rq.target)
			}
		})
	}
}

func TestResolver_Root(t *testing.T) {
	rooted := &Resolver{rooted: true}
	unrooted := &Resolver{}
	tests := []struct{ in, want string }{
		{"example.com", "example.com."},
		{"example.com.", "example.com."},
		{"192.0.2.1", "192.0.2.1"},
		{"2001:db8::1", "2001:db8::1"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, rooted.root(tt.in), "rooted %q", tt.in)
		assert.Equal(t, tt.in, unrooted.root(tt.in), "unrooted %q", tt.in)
	}
}

// fakeDNS is a DNS-over-TCP server on loopback. It answers A 192.0.2.1 for
// example.com., an empty NOERROR answer for every other type of that name and
// NXDOMAIN for any other name, and records the question name of every query.
type fakeDNS struct {
	ln net.Listener

	mu    sync.Mutex
	names []string
}

func newFakeDNS(t *testing.T) *fakeDNS {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &fakeDNS{ln: ln}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *fakeDNS) addr() string { return s.ln.Addr().String() }

func (s *fakeDNS) seen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.names)
}

func (s *fakeDNS) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.names = nil
}

func (s *fakeDNS) serve() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(c)
	}
}

// handle answers length-prefixed queries (RFC 1035 §4.2.2) until the client
// hangs up.
func (s *fakeDNS) handle(c net.Conn) {
	defer func() { _ = c.Close() }()
	for {
		var size [2]byte
		if _, err := io.ReadFull(c, size[:]); err != nil {
			return
		}
		query := make([]byte, binary.BigEndian.Uint16(size[:]))
		if _, err := io.ReadFull(c, query); err != nil {
			return
		}
		resp, err := s.answer(query)
		if err != nil {
			return
		}
		if _, err := c.Write(append(binary.BigEndian.AppendUint16(nil, uint16(len(resp))), resp...)); err != nil {
			return
		}
	}
}

func (s *fakeDNS) answer(query []byte) ([]byte, error) {
	var p dnsmessage.Parser
	h, err := p.Start(query)
	if err != nil {
		return nil, err
	}
	q, err := p.Question()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.names = append(s.names, q.Name.String())
	s.mu.Unlock()

	known := strings.EqualFold(q.Name.String(), "example.com.")
	rh := dnsmessage.Header{ID: h.ID, Response: true, RecursionDesired: h.RecursionDesired, RecursionAvailable: true}
	if !known {
		rh.RCode = dnsmessage.RCodeNameError
	}
	b := dnsmessage.NewBuilder(nil, rh)
	if err := b.StartQuestions(); err != nil {
		return nil, err
	}
	if err := b.Question(q); err != nil {
		return nil, err
	}
	if known && q.Type == dnsmessage.TypeA {
		if err := b.StartAnswers(); err != nil {
			return nil, err
		}
		rr := dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 60}
		if err := b.AResource(rr, dnsmessage.AResource{A: [4]byte{192, 0, 2, 1}}); err != nil {
			return nil, err
		}
	}
	return b.Finish()
}

// socksRequest is what the fake SOCKS5 proxy saw on one connection.
type socksRequest struct {
	user, pass string
	target     string
}

// fakeSOCKS is a SOCKS5 proxy on loopback (RFC 1928, CONNECT only) that
// accepts no authentication and username/password authentication (RFC 1929),
// records the credentials and target of every request, and connects only to
// allow: every other target is refused, so a test never leaves loopback.
type fakeSOCKS struct {
	ln    net.Listener
	allow string

	mu   sync.Mutex
	reqs []socksRequest
}

func newFakeSOCKS(t *testing.T, allow string) *fakeSOCKS {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &fakeSOCKS{ln: ln, allow: allow}
	go p.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return p
}

func (p *fakeSOCKS) addr() string { return p.ln.Addr().String() }

func (p *fakeSOCKS) requests() []socksRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.reqs)
}

func (p *fakeSOCKS) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reqs = nil
}

func (p *fakeSOCKS) serve() {
	for {
		c, err := p.ln.Accept()
		if err != nil {
			return
		}
		go p.handle(c)
	}
}

func (p *fakeSOCKS) handle(c net.Conn) {
	defer func() { _ = c.Close() }()
	br := bufio.NewReader(c)

	// Method selection: VER NMETHODS METHODS.
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(br, hdr); err != nil || hdr[0] != 5 {
		return
	}
	methods := make([]byte, hdr[1])
	if _, err := io.ReadFull(br, methods); err != nil {
		return
	}
	var rec socksRequest
	switch {
	case slices.Contains(methods, 2):
		// Username/password: VER(1) ULEN UNAME PLEN PASSWD, answered with
		// VER(1) STATUS(0).
		if _, err := c.Write([]byte{5, 2}); err != nil {
			return
		}
		if v, err := br.ReadByte(); err != nil || v != 1 {
			return
		}
		user, ok := readField(br)
		if !ok {
			return
		}
		pass, ok := readField(br)
		if !ok {
			return
		}
		rec.user, rec.pass = user, pass
		if _, err := c.Write([]byte{1, 0}); err != nil {
			return
		}
	case slices.Contains(methods, 0):
		if _, err := c.Write([]byte{5, 0}); err != nil {
			return
		}
	default:
		_, _ = c.Write([]byte{5, 0xff})
		return
	}

	// Request: VER CMD RSV ATYP DST.ADDR DST.PORT.
	req := make([]byte, 4)
	if _, err := io.ReadFull(br, req); err != nil || req[0] != 5 || req[1] != 1 {
		return
	}
	var host string
	switch req[3] {
	case 1, 4:
		ip := make([]byte, map[byte]int{1: 4, 4: 16}[req[3]])
		if _, err := io.ReadFull(br, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	case 3:
		name, ok := readField(br)
		if !ok {
			return
		}
		host = name
	default:
		return
	}
	port := make([]byte, 2)
	if _, err := io.ReadFull(br, port); err != nil {
		return
	}
	rec.target = net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(port))))
	p.mu.Lock()
	p.reqs = append(p.reqs, rec)
	p.mu.Unlock()

	refused := []byte{5, 2, 0, 1, 0, 0, 0, 0, 0, 0} // connection not allowed by ruleset
	if rec.target != p.allow {
		_, _ = c.Write(refused)
		return
	}
	up, err := net.Dial("tcp", rec.target)
	if err != nil {
		_, _ = c.Write(refused)
		return
	}
	defer func() { _ = up.Close() }()
	if _, err := c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	go func() {
		_, _ = io.Copy(up, br)
		_ = up.Close()
	}()
	_, _ = io.Copy(c, up)
}

// readField reads a length-prefixed byte string (one length byte).
func readField(br *bufio.Reader) (string, bool) {
	n, err := br.ReadByte()
	if err != nil {
		return "", false
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(br, b); err != nil {
		return "", false
	}
	return string(b), true
}
