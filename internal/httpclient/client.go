package httpclient

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/imroc/req/v3"
)

// defaultUserAgents is a pool of modern browser UA strings used for rotation.
// Selected randomly when no explicit user-agent is configured.
var defaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.7; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
}

// New builds a *req.Client with optional proxy and user-agent configuration.
// If userAgent is empty, a random UA from the built-in pool is selected.
// proxy supports http://, https://, and socks5:// URLs via req's SetProxyURL.
// When proxy is empty, HTTP_PROXY / HTTPS_PROXY / NO_PROXY environment variables
// are honoured automatically via http.ProxyFromEnvironment.
// When debug is true and logger is non-nil, an OnAfterResponse hook is attached
// that logs request timing and error body snippets at DEBUG level.
// Returns an error only if the proxy URL is syntactically invalid.
func New(proxy, userAgent string, logger *slog.Logger, debug bool) (*req.Client, error) {
	ua := userAgent
	if ua == "" {
		ua = defaultUserAgents[rand.IntN(len(defaultUserAgents))] //nolint:gosec // non-cryptographic random is fine for UA selection
	}

	client := req.NewClient().SetUserAgent(ua)

	if proxy != "" {
		if err := validateProxy(proxy); err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
		}
		client.SetProxyURL(proxy)
	} else {
		client.SetProxy(http.ProxyFromEnvironment)
	}

	if debug && logger != nil {
		attachDebugHook(client, logger)
	}

	return client, nil
}

// attachDebugHook enables req trace capture and registers an OnAfterResponse hook
// that logs HTTP timing and (on non-2xx) a body snippet at DEBUG level.
func attachDebugHook(client *req.Client, logger *slog.Logger) {
	client.EnableTraceAll()
	client.OnAfterResponse(func(_ *req.Client, resp *req.Response) error {
		if resp.Request == nil || resp.Request.RawRequest == nil {
			return nil
		}
		method := resp.Request.RawRequest.Method
		url := resp.Request.RawRequest.URL.String()
		ti := resp.TraceInfo()
		logger.Debug("http response",
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"total", ti.TotalTime.Round(time.Millisecond),
			"dns", ti.DNSLookupTime.Round(time.Millisecond),
			"tcp", ti.TCPConnectTime.Round(time.Millisecond),
			"tls", ti.TLSHandshakeTime.Round(time.Millisecond),
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
