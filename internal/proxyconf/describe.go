package proxyconf

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Row is one line of the "config show" proxy report.
type Row struct {
	Key   string
	Value string
}

// httpRefused is the proxy_http row when only https has a proxy: the proxy
// function has none for http, but the dial guard refuses a direct connection
// to any host that https would proxy, so http:// URLs fail instead of going
// direct (a NO_PROXY exemption still goes direct, as on every path).
const httpRefused = "refused (no http proxy; https is proxied)"

// Describe returns the rows "config show" prints for the proxy, in this order:
// proxy (trident's own setting), proxy_dns, proxy_http, proxy_https and
// proxy_no_proxy. Proxy URLs are redacted and a value that failed validation is
// never shown. With a deferred error the three path rows show "error: " and the
// message; proxy shows an error only when trident's own setting is the invalid
// value, and proxy_no_proxy only when NO_PROXY itself is.
func (s *Settings) Describe() []Row {
	httpRow := s.describePath(s.http)
	if s.err == nil && !s.disabled && s.http.url == nil && s.https.url != nil {
		httpRow = httpRefused
	}
	return []Row{
		{Key: "proxy", Value: s.describeSetting()},
		{Key: "proxy_dns", Value: s.describeDNS()},
		{Key: "proxy_http", Value: httpRow},
		{Key: "proxy_https", Value: s.describePath(s.https)},
		{Key: "proxy_no_proxy", Value: s.describeNoProxy()},
	}
}

func (s *Settings) errorText() string {
	return "error: " + s.err.Error()
}

func (s *Settings) describeSetting() string {
	if !s.viaSetting {
		return ""
	}
	if s.err != nil {
		return s.errorText()
	}
	return Redact(s.https.url)
}

func (s *Settings) describePath(r route) string {
	switch {
	case s.err != nil:
		return s.errorText()
	case s.disabled:
		return "direct (NO_PROXY=*)"
	case r.url == nil:
		return "direct"
	}
	return Redact(r.url) + " (" + r.source + ")"
}

func (s *Settings) describeDNS() string {
	switch {
	case s.err != nil:
		return s.errorText()
	case s.disabled:
		return "system resolver (NO_PROXY=*)"
	case s.https.url == nil:
		return "system resolver"
	}
	switch s.https.url.Scheme {
	case "socks5", "socks5h":
		return Redact(s.https.url) + " (" + s.https.source + "), nameserver " + SOCKSNameserver
	}
	return "system resolver, not proxied (" + s.https.source + " is not a SOCKS5 proxy)"
}

func (s *Settings) describeNoProxy() string {
	if s.noProxyErr != nil {
		return "error: " + s.noProxyErr.Error()
	}
	return printable(s.noProxyValue)
}

// printable replaces control and format characters (escape sequences, bidi
// overrides) and invalid UTF-8 in text that comes from the environment, so
// that it can be shown on a terminal.
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return '?'
		}
		return r
	}, s)
}
