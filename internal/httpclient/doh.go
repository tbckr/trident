package httpclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/proxyconf"
)

// Timeouts of the DoH client; variables so that tests can shorten them.
var (
	// dohRequestTimeout bounds a whole DoH request, including the wait for a
	// proxy handshake that stalls (Go runs the handshake on a context detached
	// from the request; a stalled SOCKS greeting has no other timeout).
	dohRequestTimeout = 30 * time.Second
	// dohPingInterval is how long an HTTP/2 connection may stay silent before
	// the client pings it, and dohPingTimeout how long it waits for the answer
	// before it closes the connection: without the health check, a tunnel that
	// freezes without FIN or RST (a dead Tor circuit) takes every later request
	// down with it.
	dohPingInterval = 15 * time.Second
	dohPingTimeout  = 15 * time.Second
)

// NewDoH builds the client for DNS-over-HTTPS requests to endpoint. It is the
// client New builds (same proxy settings, User-Agent, NO_PROXY warning, debug
// hook, and req's transport stays proxied, guarded and on HTTP/1.1 behind a
// proxy), except that its requests go through Go's own transport, which
// speaks HTTP/2 through a proxy without leaving it: Quad9 answers DoH over
// HTTP/1.1 with 505, and req's HTTP/2 ignores the proxy (imroc/req#206, #541).
//
// The Go transport keeps the dial guard (apperr.ErrProxyBypass), refuses proxy
// schemes it cannot speak (apperr.ErrUnsupportedProxy: socks4, socks4a, ...),
// health-checks its HTTP/2 connections, and bounds a request at 30 s. A
// request that times out is retried while its caller still waits for it (the
// retry count is AttachRateLimit's). Without a proxy for endpoint it
// multiplexes all requests over one connection to the host (Go opens another
// only when that one is at its stream limit).
//
// proxied reports whether a proxy applies to endpoint. Clone and RoundTrip on
// the client use req's transport, not Go's. Like New, NewDoH returns proxy's
// deferred error before building anything; it also fails for an endpoint that
// is not a URL.
func NewDoH(proxy *proxyconf.Settings, endpoint, userAgent string, logger *slog.Logger, debug bool) (client *req.Client, proxied bool, err error) {
	if err = proxy.Err(); err != nil {
		return nil, false, err
	}
	return newDoH(proxy.ProxyFunc(nil), proxy.ProxyFunc(exemptionWarner(logger)), proxy.Configured(), endpoint, userAgent, logger, debug)
}

// newDoH is NewDoH with the proxy functions and forceH1 (see newClient) passed
// in, so that tests can hand it proxy functions the resolved settings never
// produce: proxy schemes that validation rejects (socks4, socks4a, ...) and
// functions that fail.
func newDoH(guardProxy, transportProxy proxyFunc, forceH1 bool, endpoint, userAgent string, logger *slog.Logger, debug bool) (*req.Client, bool, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, false, fmt.Errorf("invalid DoH endpoint %q: %w", endpoint, err)
	}
	// Asked through the guard's function: building the client contacts
	// nothing, so it must not warn about a NO_PROXY exemption.
	p, err := guardProxy(&http.Request{URL: u})
	if err != nil {
		return nil, false, fmt.Errorf("resolving the proxy for %s: %w", u.Host, err)
	}
	proxied := p != nil

	// req's transport stays as New sets it up: the fallback for Clone and
	// RoundTrip, proxied and guarded.
	client := newClient(guardProxy, transportProxy, forceH1, userAgent, logger, debug)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = refuseUnsupportedSchemes(transportProxy)
	// Every TCP connection, to the proxy or (without one) to the origin, goes
	// through DialContext. Never set DialTLS or DialTLSContext: for an https://
	// first hop they replace DialContext, and the guard would never run.
	tr.DialContext = guardedDial(guardProxy, (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext)
	tr.ForceAttemptHTTP2 = true
	tr.HTTP2 = &http.HTTP2Config{SendPingTimeout: dohPingInterval, PingTimeout: dohPingTimeout}
	if !proxied {
		// One multiplexed HTTP/2 connection for all requests. Behind a proxy
		// there is no cap: Go runs a proxy handshake on a context detached from
		// the request, with no timeout of its own for SOCKS, so one stalled
		// handshake would starve the host for the whole run.
		tr.MaxConnsPerHost = 1
	}
	client.GetClient().Transport = tr
	client.SetTimeout(dohRequestTimeout)
	client.AddCommonRetryCondition(retryTimeoutWhileCallerWaits)

	return client, proxied, nil
}

// retryTimeoutWhileCallerWaits is a retry condition: it retries a request that
// timed out while the caller's own context is still alive. Go runs a proxy
// handshake on a context detached from the request, and a stalled one has no
// timeout of its own (SOCKS) or Go's fixed 1 min (HTTP CONNECT), so it ends
// the request in the client timeout. That error, like the CONNECT timeout,
// matches context.DeadlineExceeded as the caller's own deadline does, and
// AttachRateLimit never retries it; yet a retry dials a new tunnel (behind a
// proxy there is no MaxConnsPerHost cap) and gets an answer. A caller that
// cancelled, or whose own deadline passed, gets no retry.
func retryTimeoutWhileCallerWaits(resp *req.Response, err error) bool {
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return resp != nil && resp.Request != nil && resp.Request.Context().Err() == nil
}
