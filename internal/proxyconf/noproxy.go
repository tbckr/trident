package proxyconf

import (
	"net"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

// noProxy is a parsed NO_PROXY list, matched with curl's rules rather than
// Go's: there is no implicit exemption for localhost or loopback addresses, a
// "*" among other entries is an error, and an IP or CIDR entry only ever
// matches an IP-literal host. The zero value matches nothing.
type noProxy struct {
	entries []noProxyEntry
}

// noProxyEntry is one NO_PROXY entry. Exactly one of domain, ip and cidr is
// set. An empty port matches every port.
type noProxyEntry struct {
	domain string
	ip     net.IP
	cidr   *net.IPNet
	port   string
}

// parseNoProxy parses a NO_PROXY value (a comma-separated list, never the
// whole-value "*", which the caller handles). Entries are trimmed and
// lowercased, empty ones are skipped. An entry that fits no interpretation
// (a CIDR such as 10.0.0.0/33, a label IDNA refuses) is no error: it is kept
// as a plain lowercase domain, which curl does too, and then never matches a
// real host. Only a "*" among the entries is an error: Go would bypass the
// proxy for everything, curl for nothing. source names the variable in it.
func parseNoProxy(raw, source string) (noProxy, error) {
	var n noProxy
	for part := range strings.SplitSeq(raw, ",") {
		e := strings.ToLower(strings.TrimSpace(part))
		if e == "" {
			continue
		}
		if e == "*" {
			return noProxy{}, configError(source, `"*" must stand alone; use NO_PROXY=* to disable the proxy`)
		}
		if entry, ok := parseNoProxyEntry(e); ok {
			n.entries = append(n.entries, entry)
		}
	}
	return n, nil
}

// parseNoProxyEntry interprets one trimmed, lowercased, non-empty entry. It
// reports false for an entry that is empty once its dots are stripped.
func parseNoProxyEntry(e string) (noProxyEntry, bool) {
	// "*.example.com" and ".example.com" both mean example.com and every
	// subdomain, which is what a plain "example.com" already means.
	e = strings.TrimPrefix(e, "*.")
	e = strings.TrimPrefix(e, ".")
	if e == "" {
		return noProxyEntry{}, false
	}

	if _, cidr, err := net.ParseCIDR(e); err == nil {
		return noProxyEntry{cidr: cidr}, true
	}

	host, port := e, ""
	if h, p, err := net.SplitHostPort(e); err == nil {
		host, port = h, p
	} else if inner, ok := strings.CutPrefix(e, "["); ok {
		// "[::1]" without a port is the address itself; a bare "::1" never
		// gets here, it has no bracket.
		if inner, ok = strings.CutSuffix(inner, "]"); ok {
			host = inner
		}
	}
	if port != "" {
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			// Not a port: the whole entry is just an odd domain string.
			return noProxyEntry{domain: asciiHost(e)}, true
		}
		port = strconv.FormatUint(n, 10)
	}

	if ip := net.ParseIP(host); ip != nil {
		return noProxyEntry{ip: ip, port: port}, true
	}
	domain := asciiHost(host)
	if domain == "" {
		return noProxyEntry{}, false
	}
	return noProxyEntry{domain: domain, port: port}, true
}

// asciiHost returns h lowercase, without one trailing dot, in its IDNA ASCII
// form. A name IDNA refuses stays the lowercase string it was: both the
// entries and the hosts go through here, so such a name still equals itself.
func asciiHost(h string) string {
	h = strings.ToLower(strings.TrimSuffix(h, "."))
	if a, err := idna.Lookup.ToASCII(h); err == nil {
		return a
	}
	return h
}

// matches reports whether a request to host and port must bypass the proxy.
// host comes without brackets, port is the request's effective port. An IP
// host is compared with the IP and CIDR entries only, any other host with the
// domain entries only: no name is ever resolved for matching.
func (n noProxy) matches(host, port string) bool {
	if num, err := strconv.ParseUint(port, 10, 16); err == nil {
		port = strconv.FormatUint(num, 10)
	}

	ip := net.ParseIP(strings.TrimSuffix(host, "."))
	var h string
	if ip == nil {
		h = asciiHost(host)
	}
	for _, e := range n.entries {
		if e.port != "" && e.port != port {
			continue
		}
		switch {
		case ip != nil:
			if (e.ip != nil && e.ip.Equal(ip)) || (e.cidr != nil && e.cidr.Contains(ip)) {
				return true
			}
		case e.domain != "":
			if h == e.domain || strings.HasSuffix(h, "."+e.domain) {
				return true
			}
		}
	}
	return false
}
