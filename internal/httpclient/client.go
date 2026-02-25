package httpclient

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/version"
)

// DefaultUserAgent is the User-Agent sent when no explicit value is configured.
// It identifies trident honestly so server operators can recognise its traffic.
// var (not const) because version.Version is a link-time variable, not a compile-time constant.
var DefaultUserAgent = "trident/" + version.Version + " (+https://github.com/tbckr/trident)"

// impersonatePresets are the preset names for which req provides a full ImpersonateXxx()
// method that sets TLS fingerprint, HTTP/2 settings, header order, and User-Agent atomically.
var impersonatePresets = map[string]bool{
	"chrome":  true,
	"firefox": true,
	"safari":  true,
}

// tlsFingerprintPresets are all preset names accepted by --user-agent and --tls-fingerprint.
// Superset of impersonatePresets; edge/ios/android/randomized provide TLS fingerprint only.
var tlsFingerprintPresets = map[string]bool{
	"chrome": true, "firefox": true, "safari": true,
	"edge": true, "ios": true, "android": true, "randomized": true,
}

// PresetNames returns a sorted slice of all browser preset names.
// Suitable for shell completion functions.
func PresetNames() []string {
	names := make([]string, 0, len(tlsFingerprintPresets)-1) // exclude "randomized"
	for name := range tlsFingerprintPresets {
		if name != "randomized" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// ResolveUserAgent returns the User-Agent value for display in config show/get.
//
// Resolution order:
//  1. userAgent is an impersonate preset (chrome/firefox/safari) → return preset name;
//     req's ImpersonateXxx() manages the actual UA string.
//  2. userAgent is a non-empty custom string → use as-is
//  3. userAgent is empty and tlsFingerprint is an impersonate preset → return preset name
//  4. otherwise → DefaultUserAgent
func ResolveUserAgent(userAgent, tlsFingerprint string) string {
	if impersonatePresets[userAgent] {
		return userAgent
	}
	if tlsFingerprintPresets[userAgent] {
		// TLS-only preset (edge/ios/android/randomized): req sets no UA; use default.
		return DefaultUserAgent
	}
	if userAgent != "" {
		return userAgent
	}
	if impersonatePresets[tlsFingerprint] {
		return tlsFingerprint
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
	if tlsFingerprintPresets[userAgent] {
		return userAgent
	}
	return ""
}

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
	resolvedTLS := ResolveTLSFingerprint(userAgent, tlsFingerprint)

	// isCustomUA is true when the caller provided a non-preset User-Agent string.
	isCustomUA := userAgent != "" && !tlsFingerprintPresets[userAgent]

	client := req.NewClient()

	switch resolvedTLS {
	case "chrome":
		client.ImpersonateChrome()
	case "firefox":
		client.ImpersonateFirefox()
	case "safari":
		client.ImpersonateSafari()
	case "edge":
		client.SetTLSFingerprintEdge()
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

	// ImpersonateXxx sets the User-Agent from its built-in browser profile.
	// For non-impersonate cases set DefaultUserAgent; for any case a custom string overrides.
	if isCustomUA {
		client.SetUserAgent(userAgent)
	} else if !impersonatePresets[resolvedTLS] {
		client.SetUserAgent(DefaultUserAgent)
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
