package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/imroc/req/v3"
	"golang.org/x/net/idna"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/proxyconf"
)

// proxyFunc is the type of http.Transport.Proxy: the proxy for a request, or
// nil to connect directly.
type proxyFunc func(*http.Request) (*url.URL, error)

// dialFunc is the type of a transport's DialContext.
type dialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// guardProxyBypass makes the client fail closed instead of leaking around its
// proxy. req's HTTP/2 code paths dial the origin directly even when a proxy is
// configured (imroc/req#206, #541), and so would any future path that forgets
// the proxy. Every TCP connection req opens, proxied or not, goes through the
// transport's DialContext, with the proxy's address for a proxied connection,
// so the guard sits there: a dial to any other host that proxyFor would send
// through a proxy is refused with apperr.ErrProxyBypass, before any DNS lookup
// for that host.
func guardProxyBypass(client *req.Client, proxyFor proxyFunc) {
	next := client.GetTransport().DialContext
	if next == nil {
		// What req uses without a DialContext.
		var d net.Dialer
		next = d.DialContext
	}
	client.SetDial(guardedDial(proxyFor, next))
}

// guardedDial wraps next so that it refuses (apperr.ErrProxyBypass) every dial
// that checkDirectDial judges a proxy bypass. It is the dial guard of both
// req's transport (guardProxyBypass) and the DoH transport (newDoH).
func guardedDial(proxyFor proxyFunc, next dialFunc) dialFunc {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if err := checkDirectDial(proxyFor, addr); err != nil {
			return nil, err
		}
		return next(ctx, network, addr)
	}
}

// checkDirectDial returns apperr.ErrProxyBypass if a dial to addr
// ("host:port") would bypass the proxy. A dial to the address of the proxy
// that proxyFor picks for addr, for HTTPS or for plain HTTP (HTTPS_PROXY and
// HTTP_PROXY may differ), is the proxied connection itself. Any other dial is
// refused when proxyFor would send an HTTPS request to addr through a proxy.
// The dialer does not see the request's scheme, so it judges by HTTPS: every
// endpoint trident ships with is HTTPS, and req's direct-dial paths are all
// HTTP/2 over TLS. The one case this refuses wrongly fails closed: a plain
// http:// URL (a custom detect_patterns.url) with a proxy for https but none
// for http (HTTPS_PROXY without HTTP_PROXY or ALL_PROXY), which the proxy
// function sends directly.
func checkDirectDial(proxyFor proxyFunc, addr string) error {
	viaHTTPS, err := proxyFor(&http.Request{URL: &url.URL{Scheme: "https", Host: addr}})
	if err != nil {
		return fmt.Errorf("resolving the proxy for %s: %w", addr, err)
	}
	viaHTTP, err := proxyFor(&http.Request{URL: &url.URL{Scheme: "http", Host: addr}})
	if err != nil {
		return fmt.Errorf("resolving the proxy for %s: %w", addr, err)
	}
	for _, p := range []*url.URL{viaHTTPS, viaHTTP} {
		if p != nil && strings.EqualFold(proxyAddr(p), addr) {
			return nil
		}
	}
	if viaHTTPS == nil {
		return nil
	}
	return fmt.Errorf("%w: %s would not go through proxy %s", apperr.ErrProxyBypass, addr, proxyconf.Redact(viaHTTPS))
}

// proxyAddr is the "host:port" req dials for proxy p: the host in IDNA ASCII
// form, with the scheme's default port when p has none (as req's
// canonicalAddr).
func proxyAddr(p *url.URL) string {
	host := p.Hostname()
	if !isASCII(host) {
		if a, err := idna.Lookup.ToASCII(host); err == nil {
			host = a
		}
	}
	port := p.Port()
	if port == "" {
		switch strings.ToLower(p.Scheme) {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default: // socks5, socks5h, socks4, socks4a
			port = "1080"
		}
	}
	return net.JoinHostPort(host, port)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// refuseUnsupportedSchemes wraps a proxy function so that a proxy whose scheme
// Go's transport cannot speak fails with apperr.ErrUnsupportedProxy. Go's
// transport speaks http, https, socks5 and socks5h; for any other scheme
// (socks4, socks4a, ...) it would send an HTTP CONNECT to the proxy's port, or,
// with no port in the URL, the dial guard would report a misleading proxy
// bypass. The resolved proxy settings never hand out such a scheme; this is
// defence in depth for the DoH transport.
func refuseUnsupportedSchemes(proxyFor proxyFunc) proxyFunc {
	return func(r *http.Request) (*url.URL, error) {
		p, err := proxyFor(r)
		if err != nil || p == nil {
			return p, err
		}
		switch strings.ToLower(p.Scheme) {
		case "http", "https", "socks5", "socks5h":
			return p, nil
		}
		return nil, fmt.Errorf("%w: proxy %s uses %s://; DNS-over-HTTPS supports http://, https://, socks5:// and socks5h:// proxies",
			apperr.ErrUnsupportedProxy, proxyconf.Redact(p), p.Scheme)
	}
}
