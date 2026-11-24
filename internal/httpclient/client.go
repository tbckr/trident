package httpclient

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/imroc/req/v3"

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

// ResolveProxy returns the proxy value that will actually be used.
// If proxy is explicitly configured, it is returned as-is.
// Otherwise the standard proxy env vars are checked
// (HTTPS_PROXY, HTTP_PROXY, ALL_PROXY and their lowercase variants);
// if any are set "<from environment>" is returned.
// If none are set, an empty string is returned.
func ResolveProxy(proxy string) string {
	if proxy != "" {
		return proxy
	}
	for _, env := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		if os.Getenv(env) != "" {
			return "<from environment>"
		}
	}
	return ""
}

// New builds a *req.Client with optional proxy and user-agent configuration.
// If userAgent is empty, DefaultUserAgent is used.
// proxy supports http://, https://, and socks5:// URLs.
// When proxy is empty, HTTP_PROXY / HTTPS_PROXY / NO_PROXY environment variables
// are honoured automatically via http.ProxyFromEnvironment.
// Whenever a proxy is configured, the client speaks HTTP/1.1 only, and it
// refuses (apperr.ErrProxyBypass) to connect around the proxy to a host the
// proxy is meant to reach (see guardProxyBypass), or to use a socks4:// proxy
// from the environment (see refuseLocalDNSProxies). For HTTP/2 behind a proxy
// use NewDoH, not client.EnableForceHTTP2.
// When debug is true and logger is non-nil, an OnAfterResponse hook is attached
// that logs the HTTP method, URL, and status code at DEBUG level.
// Returns an error if the proxy URL is syntactically invalid.
func New(proxy, userAgent string, logger *slog.Logger, debug bool) (*req.Client, error) {
	proxyFor, err := proxyFuncFor(proxy)
	if err != nil {
		return nil, err
	}
	return newClient(proxyFor, ResolveProxy(proxy) != "", userAgent, logger, debug), nil
}

// proxyFuncFor returns the proxy function for proxy as passed to New: the
// proxy environment variables when it is empty (a socks4:// proxy there is
// refused, see refuseLocalDNSProxies), else proxy itself for every request.
// It returns an error if proxy is not a valid http://, https:// or socks5://
// URL.
func proxyFuncFor(proxy string) (proxyFunc, error) {
	if proxy == "" {
		return refuseLocalDNSProxies(http.ProxyFromEnvironment), nil
	}
	if err := validateProxy(proxy); err != nil {
		return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
	}
	// With a socks5:// URL, req forwards hostnames (not pre-resolved IPs)
	// through the proxy with its own SOCKS5 dialer, preventing DNS leaks for
	// HTTP-based services. DNS-based services (dns, cymru) use
	// resolver.NewResolver instead.
	return http.ProxyURL(u), nil
}

// newClient builds the client New describes, with proxyFor as its proxy
// function. forceH1 makes it speak HTTP/1.1 only; it must be set whenever
// proxyFor can return a proxy.
func newClient(proxyFor proxyFunc, forceH1 bool, userAgent string, logger *slog.Logger, debug bool) *req.Client {
	client := req.NewClient()

	if userAgent != "" {
		client.SetUserAgent(userAgent)
	} else {
		client.SetUserAgent(DefaultUserAgent)
	}

	client.SetProxy(proxyFor)

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
	guardProxyBypass(client, proxyFor)

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

// validateProxy performs a basic check that the proxy URL has a recognised scheme.
func validateProxy(proxy string) error {
	// req.SetProxyURL will log a warning on empty strings but we already gate on that.
	// Just verify scheme is one we expect.
	for _, scheme := range []string{"http://", "https://", "socks5://"} {
		if len(proxy) >= len(scheme) && proxy[:len(scheme)] == scheme {
			return nil
		}
	}
	return fmt.Errorf("proxy scheme must be http://, https://, or socks5://")
}
