package httpclient

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/version"
)

// defaultUserAgent is the User-Agent sent when no explicit value is configured.
// It identifies trident honestly so server operators can recognise its traffic.
// var (not const) because version.Version is a link-time variable, not a compile-time constant.
var defaultUserAgent = "trident/" + version.Version + " (+https://github.com/tbckr/trident)"

// New builds a *req.Client with optional proxy and user-agent configuration.
// If userAgent is empty, defaultUserAgent is used.
// proxy supports http://, https://, and socks5:// URLs via req's SetProxyURL.
// When proxy is empty, HTTP_PROXY / HTTPS_PROXY / NO_PROXY environment variables
// are honoured automatically via http.ProxyFromEnvironment.
// When debug is true and logger is non-nil, an OnAfterResponse hook is attached
// that logs request timing and error body snippets at DEBUG level.
// Returns an error only if the proxy URL is syntactically invalid.
func New(proxy, userAgent string, logger *slog.Logger, debug bool) (*req.Client, error) {
	ua := userAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	client := req.NewClient().SetUserAgent(ua)

	if proxy != "" {
		if err := validateProxy(proxy); err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
		}
		// SetProxyURL with a socks5:// URL forwards hostnames (not pre-resolved IPs)
		// through the proxy via golang.org/x/net/proxy.SOCKS5, preventing DNS leaks
		// for HTTP-based services. DNS-based services (dns, asn) use resolver.NewResolver instead.
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
