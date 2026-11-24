package proxyconf

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tbckr/trident/internal/apperr"
)

// supportedSchemes is the list named in the "unsupported scheme" message.
const supportedSchemes = "http://, https://, socks5:// or socks5h://"

// schemeToken matches a scheme that is safe to print back in an error message.
var schemeToken = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)

// configError builds the one error shape this package returns for a bad proxy
// value: "invalid proxy configuration: <source>: <reason>". source names the
// setting (proxy, HTTPS_PROXY, ...) and reason is a fixed phrase. Neither the
// value nor url.Parse's error (which quotes it) may ever reach the message.
func configError(source, reason string) error {
	return fmt.Errorf("%w: %s: %s", apperr.ErrProxyConfig, source, reason)
}

// textProblem returns the reason s must be refused, or "" if it is fine to
// parse and show: no whitespace or control characters, and valid UTF-8. Format
// characters (zero-width space, bidi overrides, BOM) are invisible or reorder
// what is displayed, so they count as control here.
func textProblem(s string) string {
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return "contains whitespace or control characters"
		}
	}
	if !utf8.ValidString(s) {
		return "not a valid URL"
	}
	return ""
}

// parseProxyURL validates a proxy value and returns it as a URL with a
// lowercase scheme and an empty path. A value without "://" means http://
// plus the value. The value is never trimmed or repaired: anything the rules
// below do not accept is an error naming source and the reason, never any
// part of raw, because a proxy value that is silently misread would send
// traffic around the proxy.
func parseProxyURL(raw, source string) (*url.URL, error) {
	// Whitespace and control characters first, before anything else could
	// misread them: a value with a leading space would otherwise have gone
	// direct.
	if reason := textProblem(raw); reason != "" {
		return nil, configError(source, reason)
	}

	// Detect the scheme by "://", not by url.Parse: "localhost:9050" parses
	// with the scheme "localhost".
	scheme, _, found := strings.Cut(raw, "://")
	if !found {
		raw = "http://" + raw
		scheme = "http"
	}
	scheme = strings.ToLower(scheme)
	switch scheme {
	case "http", "https", "socks5", "socks5h":
	case "socks4", "socks4a":
		return nil, configError(source, scheme+":// is not supported; use socks5h://")
	case "socks":
		return nil, configError(source, "socks:// is ambiguous; use socks5h://")
	default:
		// The text before "://" is only echoed when it is a plain token: it
		// can be a mangled value that holds a credential.
		if schemeToken.MatchString(scheme) {
			return nil, configError(source, fmt.Sprintf("unsupported scheme %q; use %s", scheme, supportedSchemes))
		}
		return nil, configError(source, "unsupported scheme; use "+supportedSchemes)
	}

	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" {
		// err quotes the whole value, credentials included: drop it.
		return nil, configError(source, "not a valid URL")
	}
	// url.Parse decodes %XX in the host ("%C2%9B" becomes a C1 control) and
	// Redact prints the host as is, so the decoded host gets the same check
	// as the raw value.
	if reason := textProblem(u.Host); reason != "" {
		return nil, configError(source, reason)
	}
	// ":9050" has a non-empty Host but no host.
	if u.Hostname() == "" {
		return nil, configError(source, "missing host")
	}
	// An IPv6 literal must be bracketed; net/url would otherwise split "::1"
	// into the host "::" and the port 1.
	if !strings.HasPrefix(u.Host, "[") && strings.Count(u.Host, ":") > 1 {
		return nil, configError(source, "not a valid URL")
	}
	// net/url only checks that a port is made of digits.
	if port := u.Port(); port != "" {
		if n, err := strconv.ParseUint(port, 10, 16); err != nil || n == 0 {
			return nil, configError(source, "port must be a number from 1 to 65535")
		}
	}
	// "socks5h://localhost/run/tor/socks" must not silently become
	// localhost:1080, and "socks5:/127.0.0.1:9050" is a typo. An empty
	// fragment leaves no trace in url.URL, hence the search for "#".
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || strings.Contains(raw, "#") {
		return nil, configError(source, "must not contain a path, query or fragment")
	}

	u.Path, u.RawPath = "", ""
	// "host:" has an empty port: the same as no port, without the colon.
	u.Host = strings.TrimSuffix(u.Host, ":")
	return u, nil
}

// Validate reports whether raw is a usable proxy URL, for callers that only
// need the verdict (such as "config set proxy"). The error wraps
// apperr.ErrProxyConfig, names the setting as "proxy" and never contains any
// part of raw.
func Validate(raw string) error {
	_, err := parseProxyURL(raw, "proxy")
	return err
}
