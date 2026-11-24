package httpclient

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/proxyconf"
	"github.com/tbckr/trident/internal/version"
)

// proxiedIdleConnsPerHost is how many idle connections per host a client keeps
// behind a proxy, where it speaks HTTP/1.1 only: more than the largest
// rate-limit burst (doh.DefaultBurst, 10) and the default concurrency (10).
const proxiedIdleConnsPerHost = 16

// DefaultUserAgent is the User-Agent sent when no explicit value is configured.
// It identifies trident honestly so server operators can recognise its traffic.
// var (not const) because version.Version is a link-time variable, not a compile-time constant.
var DefaultUserAgent = "trident/" + version.Version + " (+https://github.com/tbckr/trident)"

// New builds a *req.Client that sends its requests as proxy (the run's one
// proxy resolution, from proxyconf.Resolve; never nil) says, with the
// User-Agent userAgent (DefaultUserAgent if empty). If proxy has a deferred
// error, New returns it before building anything: a proxy setting trident
// cannot honour never turns into a direct connection.
//
// The transport's proxy function is always proxy's, also when no proxy
// applies. Whenever a proxy is configured, the client speaks HTTP/1.1 only,
// and it refuses (apperr.ErrProxyBypass) to connect around the proxy to a host
// the proxy is meant to reach (see guardProxyBypass). For HTTP/2 behind a proxy
// use NewDoH, not client.EnableForceHTTP2. With a logger, a request that
// NO_PROXY exempts from the proxy is warned about once per host (see
// exemptionWarner).
// When debug is true and logger is non-nil, an OnAfterResponse hook is attached
// that logs the HTTP method, URL, and status code at DEBUG level.
func New(proxy *proxyconf.Settings, userAgent string, logger *slog.Logger, debug bool) (*req.Client, error) {
	if err := proxy.Err(); err != nil {
		return nil, err
	}
	return newClient(proxy.ProxyFunc(nil), proxy.ProxyFunc(exemptionWarner(logger)), proxy.Configured(), userAgent, logger, debug), nil
}

// exemptionMessage is the warning exemptionWarner logs.
const exemptionMessage = "NO_PROXY exempts this host from the proxy; connecting directly"

// exemptionWarner returns the notify function for proxyconf's ProxyFunc on a
// client's transports: it logs at Warn level that NO_PROXY exempts a host the
// client is about to contact from the configured proxy, once per host. The
// host is normalised first (lowercase, one trailing dot removed), so "A.CORP."
// and "a.corp" warn once. It is safe for concurrent use, as services send
// requests in parallel. Every call returns a warner with a memory of its own;
// a client hands the same one to all its transports. The dial guard must not
// get one: its lookups contact nothing. It returns nil for a nil logger.
func exemptionWarner(logger *slog.Logger) func(host string) {
	if logger == nil {
		return nil
	}
	var warned sync.Map
	return func(host string) {
		host = strings.TrimSuffix(strings.ToLower(host), ".")
		if _, seen := warned.LoadOrStore(host, struct{}{}); seen {
			return
		}
		logger.Warn(exemptionMessage, "host", host)
	}
}

// newClient builds the client New describes. Its transport sends requests
// through transportProxy, and its dial guard judges every dial by guardProxy;
// both must describe the same proxies (guardProxy is the one without the
// exemption warning). forceH1 makes the client speak HTTP/1.1 only; it must be
// set whenever the proxy functions can return a proxy.
func newClient(guardProxy, transportProxy proxyFunc, forceH1 bool, userAgent string, logger *slog.Logger, debug bool) *req.Client {
	client := req.NewClient()

	if userAgent != "" {
		client.SetUserAgent(userAgent)
	} else {
		client.SetUserAgent(DefaultUserAgent)
	}

	client.SetProxy(transportProxy)

	if forceH1 {
		// req ignores the proxy on its HTTP/2 paths: forced HTTP/2 never uses it
		// (imroc/req#206), and once a proxied HTTP/2 connection dies, req dials
		// the origin directly (imroc/req#541). HTTP/1.1 takes neither path.
		// Revisit once a req release contains the fix of imroc/req#541.
		client.EnableForceHTTP1()
		// Without HTTP/2 multiplexing, each concurrent request needs its own
		// connection; keep a burst's worth idle so that the next burst reuses
		// them instead of opening new proxy tunnels (slow through Tor).
		client.GetTransport().MaxIdleConnsPerHost = proxiedIdleConnsPerHost
	}
	guardProxyBypass(client, guardProxy)

	if debug && logger != nil {
		attachDebugHook(client, logger)
	}

	return client
}

// attachDebugHook registers an OnAfterResponse hook that logs the HTTP method,
// URL, and status code at DEBUG level, and logs a body snippet on non-2xx responses.
func attachDebugHook(client *req.Client, logger *slog.Logger) {
	client.OnAfterResponse(func(_ *req.Client, resp *req.Response) error {
		if resp.Request == nil || resp.Request.RawRequest == nil {
			return nil
		}
		// The hook also runs for transport errors, where the embedded
		// *http.Response is nil and StatusCode, IsSuccessState and String would
		// panic on it. The error itself reaches the caller through the request.
		if resp.Response == nil {
			return nil
		}
		logger.Debug("http response",
			"method", resp.Request.RawRequest.Method,
			"url", resp.Request.RawRequest.URL.String(),
			"status", resp.StatusCode,
		)
		if !resp.IsSuccessState() {
			body := resp.String()
			if len(body) > 512 {
				body = body[:512]
			}
			logger.Debug("http error body",
				"status", resp.StatusCode,
				"body", body,
			)
		}
		return nil
	})
}
