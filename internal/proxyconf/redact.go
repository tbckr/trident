package proxyconf

import (
	"net/url"
	"regexp"
	"strings"
)

// userinfoPattern matches the scheme and userinfo of a URL in free text. The
// userinfo ends at the last "@" before the first "/" or whitespace, so a
// password that holds an "@" is masked whole and "https://example.com/a@b" is
// left alone.
var userinfoPattern = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^\s/]+@`)

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
// as a command line or an error message that quotes one. A password that
// itself holds an unescaped "/" ends the authority early, so that URL is not
// recognised and not masked; such a value is not a valid proxy URL anyway
// (parseProxyURL never echoes it).
func RedactText(s string) string {
	return userinfoPattern.ReplaceAllString(s, "${1}***@")
}
