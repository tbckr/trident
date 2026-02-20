package output

import (
	"io"
	"net"
	"regexp"
	"strings"

	"github.com/tbckr/trident/internal/pap"
)

// defangDotRe matches dots in domain names and IPv4 addresses.
var defangDotRe = regexp.MustCompile(`\.`)

// defangSchemeRe matches http:// and https:// scheme prefixes.
var defangSchemeRe = regexp.MustCompile(`(?i)^(https?)://`)

// DefangDomain replaces all dots in a domain name with [.] to prevent
// clickable links in reports. Example: "example.com" → "example[.]com".
func DefangDomain(s string) string {
	return defangDotRe.ReplaceAllString(s, "[.]")
}

// DefangIP defangs an IP address string.
// IPv4: replaces dots with [.]. Example: "1.2.3.4" → "1[.]2[.]3[.]4".
// IPv6: wraps the address in brackets. Example: "::1" → "[::1]".
// Non-IP strings are returned unchanged.
func DefangIP(s string) string {
	ip := net.ParseIP(s)
	if ip == nil {
		return s
	}
	if ip.To4() != nil {
		return defangDotRe.ReplaceAllString(s, "[.]")
	}
	return "[" + s + "]"
}

// DefangURL defangs a URL by replacing the scheme and dots in the host.
// Example: "http://example.com/path" → "hxxp://example[.]com/path".
// "https://foo.bar" → "hxxps://foo[.]bar".
// Non-URL strings (no "://") have all dots replaced.
func DefangURL(s string) string {
	// Replace scheme: http → hxxp, https → hxxps
	s = defangSchemeRe.ReplaceAllStringFunc(s, func(match string) string {
		lower := strings.ToLower(match)
		lower = strings.Replace(lower, "http", "hxxp", 1)
		return lower
	})
	// Find host portion: it starts after "://" and ends at the next "/" (or end of string).
	schemeEnd := strings.Index(s, "://")
	if schemeEnd < 0 {
		// No authority — defang dots in the whole string.
		return defangDotRe.ReplaceAllString(s, "[.]")
	}
	hostStart := schemeEnd + 3 // skip "://"
	after := s[hostStart:]
	slashIdx := strings.Index(after, "/")
	if slashIdx < 0 {
		// No path — defang dots in host only.
		return s[:hostStart] + defangDotRe.ReplaceAllString(after, "[.]")
	}
	host := after[:slashIdx]
	rest := after[slashIdx:]
	return s[:hostStart] + defangDotRe.ReplaceAllString(host, "[.]") + rest
}

// ResolveDefang determines whether output should be defanged given the current
// flags and PAP level.
//
// Rules:
//   - --no-defang (noDefang=true): always returns false regardless of other flags.
//   - --defang (explicitDefang=true): always returns true for every format,
//     including JSON.
//   - PAP=AMBER or PAP=RED without --no-defang: returns true for text/plain
//     formats; JSON stays raw because downstream consumers want unmodified data.
//   - Default (PAP=WHITE, no flags): returns false.
//
// noDefang and explicitDefang are mutually exclusive; callers must validate
// this before calling ResolveDefang.
func ResolveDefang(papLevel pap.Level, format Format, explicitDefang, noDefang bool) bool {
	if noDefang {
		return false
	}
	isPAPTriggered := papLevel == pap.AMBER || papLevel == pap.RED
	return explicitDefang || (isPAPTriggered && format != FormatJSON)
}

// DefangWriter wraps an io.Writer and applies defanging transforms on every Write call.
// It replaces dots in domain-like and IP-like patterns, and defangs http/https schemes.
type DefangWriter struct {
	Inner io.Writer
}

// Write implements io.Writer; it defangs p before forwarding to the inner writer.
func (d *DefangWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	// Defang scheme prefixes
	s = defangSchemeRe.ReplaceAllStringFunc(s, func(match string) string {
		lower := strings.ToLower(match)
		return strings.Replace(lower, "http", "hxxp", 1)
	})
	// Defang dots
	s = defangDotRe.ReplaceAllString(s, "[.]")
	written, err := d.Inner.Write([]byte(s))
	// Return the original length so callers don't think a short write occurred due to expansion.
	if err != nil {
		return written, err
	}
	return len(p), nil
}
