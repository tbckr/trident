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

// aliasSpace lists the runes unicode.IsSpace reports, which strings.Fields
// splits an alias expansion on. RE2's \s covers ASCII whitespace only, so an
// alias written with a no-break space or a vertical tab after --proxy would
// work but escape the masking.
const aliasSpace = `\t\n\v\f\r \x{85}\x{A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}`

// proxyFlagPattern matches a --proxy flag and its value: "--proxy v",
// "--proxy=v", where v is a quoted string (which may hold spaces) and whatever
// is glued to its end, or else a token of non-space characters. Space means
// every aliasSpace rune.
var proxyFlagPattern = regexp.MustCompile(
	`(--proxy(?:=|[` + aliasSpace + `]+))` +
		`("[^"]*"[^` + aliasSpace + `]*|'[^']*'[^` + aliasSpace + `]*|[^` + aliasSpace + `]+)`)

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
// as a command line or an error message that quotes one. A --proxy value is
// masked up to its last "@" in addition, with or without a scheme: a value
// without one (user:password@host:port, which trident reads as an http:// URL)
// is not caught by the generic pattern, which is not widened to scheme-less
// text because that would mask e-mail addresses, and neither is a password
// holding a "/" (which ends the generic pattern's authority) or, in a quoted
// value, a space. Outside a --proxy value such a URL is not recognised and not
// masked; it is not a valid proxy URL either, and parseProxyURL never echoes
// a value.
func RedactText(s string) string {
	s = userinfoPattern.ReplaceAllString(s, "${1}***@")
	return proxyFlagPattern.ReplaceAllStringFunc(s, maskProxyFlag)
}

// maskProxyFlag masks the value in a --proxy match from after its opening
// quote and scheme (each if present) up to and including the last "@"; a value
// with no "@" stays as it is. Masking up to the last "@" can hide part of a
// path holding an "@" as well, never too little.
func maskProxyFlag(match string) string {
	m := proxyFlagPattern.FindStringSubmatch(match)
	if m == nil {
		return match
	}
	flag, value := m[1], m[2]
	if q := value[:1]; q == `"` || q == "'" {
		flag, value = flag+q, value[1:]
	}
	i := strings.LastIndex(value, "@")
	if i < 0 {
		return match
	}
	return flag + schemePrefix.FindString(value) + "***" + value[i:]
}
