package proxyconf

import (
	"net/url"
	"regexp"
	"strings"
)

// userinfoPattern matches the scheme and userinfo of a URL in free text. There
// is deliberately no word boundary before the scheme, so a scheme glued to a
// word character ("X_" in front of it) is masked too. The userinfo ends at the
// last "@" before the first "/" or whitespace, so a password that holds an "@"
// is masked whole and "https://example.com/a@b" is left alone.
var userinfoPattern = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^\s/]+@`)

// proxyFlagPattern matches a --proxy flag and its value token: "--proxy v",
// "--proxy=v", with an optional opening quote kept out of the value.
var proxyFlagPattern = regexp.MustCompile(`(--proxy(?:=|\s+)["']?)(\S+)`)

// schemePrefix matches a value that starts with a URL scheme.
var schemePrefix = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://`)

// Redact returns u for display with its credentials masked: scheme, "***@"
// when u has userinfo (the whole userinfo, user name included) and host[:port].
// Path, query and fragment are not shown. It returns "" for nil. Every proxy
// URL trident shows goes through Redact.
func Redact(u *url.URL) string {
	if u == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(u.Scheme)
	b.WriteString("://")
	if u.User != nil {
		b.WriteString("***@")
	}
	b.WriteString(u.Host)
	return b.String()
}

// RedactText masks the userinfo of every scheme://userinfo@ in free text, such
// as a command line or an error message that quotes one. It also masks the
// userinfo of a --proxy value that has no scheme (a user:password@host:port
// value, which trident reads as an http:// URL); the generic pattern is not
// widened to scheme-less text, as that would mask e-mail addresses. A password
// that itself holds an unescaped "/" ends the authority early, so that URL is
// not recognised and not masked; such a value is not a valid proxy URL anyway
// (parseProxyURL never echoes it).
func RedactText(s string) string {
	s = userinfoPattern.ReplaceAllString(s, "${1}***@")
	return proxyFlagPattern.ReplaceAllStringFunc(s, maskProxyFlag)
}

// maskProxyFlag masks everything up to and including the last "@" of the value
// in a --proxy match, unless the value starts with a scheme (the generic
// pattern has handled those) or has no "@".
func maskProxyFlag(match string) string {
	m := proxyFlagPattern.FindStringSubmatch(match)
	if m == nil {
		return match
	}
	flag, value := m[1], m[2]
	if schemePrefix.MatchString(value) {
		return match
	}
	i := strings.LastIndex(value, "@")
	if i < 0 {
		return match
	}
	return flag + "***" + value[i:]
}
