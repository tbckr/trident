package httpclient

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/version"
)

// DefaultUserAgent is the User-Agent sent when no explicit value is configured.
// It identifies trident honestly so server operators can recognise its traffic.
// var (not const) because version.Version is a link-time variable, not a compile-time constant.
var DefaultUserAgent = "trident/" + version.Version + " (+https://github.com/tbckr/trident)"

// UserAgentPresets maps browser preset names to their full User-Agent strings.
// Preset names correspond to TLS fingerprint identifiers so both can be derived from each other.
var UserAgentPresets = map[string]string{
	"chrome":  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"firefox": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
	"safari":  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
	"edge":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	"ios":     "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
	"android": "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
}

// PresetNames returns a sorted slice of all UA preset names.
// Suitable for shell completion functions.
func PresetNames() []string {
	names := make([]string, 0, len(UserAgentPresets))
	for name := range UserAgentPresets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveUserAgent returns the User-Agent string that will actually be sent.
//
// Resolution order:
//  1. userAgent is a known preset name → full browser UA string
//  2. userAgent is a non-empty custom string → use as-is
//  3. userAgent is empty and tlsFingerprint is a known preset (not "randomized") → matching browser UA
//  4. otherwise → DefaultUserAgent
func ResolveUserAgent(userAgent, tlsFingerprint string) string {
	if ua, ok := UserAgentPresets[userAgent]; ok {
		return ua
	}
	if userAgent != "" {
		return userAgent
	}
	if tlsFingerprint != "" && tlsFingerprint != "randomized" {
		if ua, ok := UserAgentPresets[tlsFingerprint]; ok {
			return ua
		}
	}
	return DefaultUserAgent
}

// ResolveTLSFingerprint returns the TLS fingerprint that will actually be used.
//
// Resolution order:
//  1. tlsFingerprint is non-empty → use as-is (explicit always wins)
//  2. userAgent is a known preset name → matching TLS fingerprint
//  3. otherwise → "" (Go default TLS)
func ResolveTLSFingerprint(userAgent, tlsFingerprint string) string {
	if tlsFingerprint != "" {
		return tlsFingerprint
	}
	if _, ok := UserAgentPresets[userAgent]; ok {
		return userAgent
	}
	return ""
}

// New builds a *req.Client with optional proxy, user-agent, and TLS fingerprint configuration.
// If userAgent is empty, DefaultUserAgent is used.
// proxy supports http://, https://, and socks5:// URLs via req's SetProxyURL.
// When proxy is empty, HTTP_PROXY / HTTPS_PROXY / NO_PROXY environment variables
// are honoured automatically via http.ProxyFromEnvironment.
// tlsFingerprint selects a uTLS client hello profile (chrome, firefox, edge, safari, ios,
// android, randomized). An empty string uses Go's default TLS implementation.
// When debug is true and logger is non-nil, an OnAfterResponse hook is attached
// that logs the HTTP method, URL, and status code at DEBUG level.
// Returns an error if the proxy URL is syntactically invalid or tlsFingerprint is unrecognised.
func New(proxy, userAgent, tlsFingerprint string, logger *slog.Logger, debug bool) (*req.Client, error) {
	resolvedUA := ResolveUserAgent(userAgent, tlsFingerprint)
	resolvedTLS := ResolveTLSFingerprint(userAgent, tlsFingerprint)

	client := req.NewClient().SetUserAgent(resolvedUA)

	switch resolvedTLS {
	case "chrome":
		client.SetTLSFingerprintChrome()
	case "firefox":
		client.SetTLSFingerprintFirefox()
	case "edge":
		client.SetTLSFingerprintEdge()
	case "safari":
		client.SetTLSFingerprintSafari()
	case "ios":
		client.SetTLSFingerprintIOS()
	case "android":
		client.SetTLSFingerprintAndroid()
	case "randomized":
		client.SetTLSFingerprintRandomized()
	case "": // default Go TLS — no-op
	default:
		return nil, fmt.Errorf("unknown TLS fingerprint %q", resolvedTLS)
	}

	if proxy != "" {
		if err := validateProxy(proxy); err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
		}
		// SetProxyURL with a socks5:// URL forwards hostnames (not pre-resolved IPs)
		// through the proxy via golang.org/x/net/proxy.SOCKS5, preventing DNS leaks
		// for HTTP-based services. DNS-based services (dns, cymru) use resolver.NewResolver instead.
		client.SetProxyURL(proxy)
	} else {
		client.SetProxy(http.ProxyFromEnvironment)
	}

	if debug && logger != nil {
		attachDebugHook(client, logger)
	}

	return client, nil
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
