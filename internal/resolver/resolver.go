package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/net/proxy"

	"github.com/tbckr/trident/internal/proxyconf"
)

// nameserver is the nameserver DNS queries go to through a SOCKS proxy. It is
// a variable only so that tests can point it at a loopback server.
var nameserver = proxyconf.SOCKSNameserver

// Resolver performs DNS lookups like a *net.Resolver, with the same method
// signatures. Behind a SOCKS proxy it roots every name it is asked for, so
// that no search domain from the system configuration is appended to it and
// sent through the proxy, and its errors name the pinned nameserver the query
// went to. It satisfies services.DNSResolverInterface and cymru.TXTResolver.
type Resolver struct {
	r      *net.Resolver
	rooted bool
	// server is the nameserver every query goes to through the proxy, or ""
	// for the system resolver.
	server string
}

// NewResolver returns the resolver for the run's proxy settings. It fails with
// the settings' deferred error before building anything, so a broken proxy
// configuration aborts DNS commands like every other network command.
//
// When DNS goes through a SOCKS5 proxy (socks5:// or socks5h://, see
// proxyconf.Settings.DNSProxy), every query goes over TCP through the proxy to
// the pinned nameserver proxyconf.SOCKSNameserver, never to the nameserver of
// the system configuration (a local stub resolver is the proxy's own loopback,
// and Tor refuses it), and names are queried rooted. Credentials in the proxy
// URL become SOCKS username/password authentication (which Tor uses for
// circuit isolation). Otherwise — no proxy, NO_PROXY=*, or an HTTP(S) proxy,
// which cannot carry DNS — it returns the system resolver, unrooted.
func NewResolver(settings *proxyconf.Settings) (*Resolver, error) {
	if err := settings.Err(); err != nil {
		return nil, err
	}
	u := settings.DNSProxy()
	if u == nil {
		return &Resolver{r: &net.Resolver{}}, nil
	}

	dialer, err := proxy.FromURL(u, proxy.Direct)
	if err != nil {
		// The error names only the scheme, never the URL.
		return nil, fmt.Errorf("creating SOCKS5 dialer for DNS: %w", err)
	}
	cd, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New("creating SOCKS5 dialer for DNS: dialer does not support contexts")
	}
	ns := nameserver
	return &Resolver{
		r: &net.Resolver{
			// Only Go's resolver uses Dial; the system's would query the
			// configured nameserver directly.
			PreferGo: true,
			// Dial ignores the network and the nameserver Go picked from the
			// system configuration: every query goes over TCP through the
			// proxy to the pinned nameserver.
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return cd.DialContext(ctx, "tcp", ns)
			},
		},
		rooted: true,
		server: ns,
	}, nil
}

// pin returns err with the server of its *net.DNSError replaced by the pinned
// nameserver. Go fills that field with the nameserver from the system
// configuration, which Dial ignored; shown in an error behind a proxy, it
// would read like a query that leaked to it. err itself is not changed: the
// Go resolver shares one error between concurrent lookups of the same name.
// It returns err as it is for the system resolver, for an error that names no
// server (the query never left) and for any other error.
func (r *Resolver) pin(err error) error {
	if r.server == "" {
		return err
	}
	// net.Resolver returns its *net.DNSError unwrapped.
	de, ok := err.(*net.DNSError)
	if !ok || de.Server == "" {
		return err
	}
	c := *de
	c.Server = r.server
	return &c
}

// root returns name with a trailing dot when the resolver roots names. Empty
// names, names that already end in a dot and IP literals are left as they are.
func (r *Resolver) root(name string) string {
	if !r.rooted || name == "" || strings.HasSuffix(name, ".") || net.ParseIP(name) != nil {
		return name
	}
	return name + "."
}

// LookupIPAddr looks up host's IPv4 and IPv6 addresses, like
// [net.Resolver.LookupIPAddr].
func (r *Resolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	addrs, err := r.r.LookupIPAddr(ctx, r.root(host))
	return addrs, r.pin(err)
}

// LookupMX looks up the MX records of name, like [net.Resolver.LookupMX].
func (r *Resolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	mx, err := r.r.LookupMX(ctx, r.root(name))
	return mx, r.pin(err)
}

// LookupNS looks up the NS records of name, like [net.Resolver.LookupNS].
func (r *Resolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	ns, err := r.r.LookupNS(ctx, r.root(name))
	return ns, r.pin(err)
}

// LookupTXT looks up the TXT records of name, like [net.Resolver.LookupTXT].
func (r *Resolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	txt, err := r.r.LookupTXT(ctx, r.root(name))
	return txt, r.pin(err)
}

// LookupAddr performs a reverse lookup of addr, like
// [net.Resolver.LookupAddr]. The resolver builds the rooted in-addr.arpa or
// ip6.arpa name itself.
func (r *Resolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	names, err := r.r.LookupAddr(ctx, addr)
	return names, r.pin(err)
}

// LookupCNAME returns the canonical name of host, like
// [net.Resolver.LookupCNAME].
func (r *Resolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	cname, err := r.r.LookupCNAME(ctx, r.root(host))
	return cname, r.pin(err)
}

// LookupSRV looks up the SRV records of _service._proto.name (or of name when
// service and proto are empty), like [net.Resolver.LookupSRV].
func (r *Resolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	cname, addrs, err := r.r.LookupSRV(ctx, service, proto, r.root(name))
	return cname, addrs, r.pin(err)
}
