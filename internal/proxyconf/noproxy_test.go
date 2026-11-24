package proxyconf

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
)

func TestNoProxyMatches(t *testing.T) {
	tests := []struct {
		list string
		host string
		port string
		want bool
	}{
		{list: ".corp,localhost", host: "patterns.corp", port: "443", want: true},
		{list: ".corp", host: "corp", port: "443", want: true},
		{list: "example.com", host: "a.example.com", port: "443", want: true},
		{list: "example.com", host: "badexample.com", port: "443", want: false},
		{list: "*.crt.sh", host: "crt.sh", port: "443", want: true},
		{list: "*.crt.sh", host: "x.crt.sh", port: "443", want: true},
		// No implicit exemption for localhost or loopback addresses.
		{list: "localhost", host: "127.0.0.1", port: "443", want: false},
		{list: "", host: "localhost", port: "443", want: false},
		{list: "", host: "127.0.0.1", port: "443", want: false},
		{list: "10.0.0.0/8", host: "10.1.2.3", port: "443", want: true},
		{list: "10.0.0.0/8", host: "11.0.0.1", port: "443", want: false},
		// A CIDR entry matches IP-literal hosts only, no DNS lookup.
		{list: "10.0.0.0/8", host: "ten.example", port: "443", want: false},
		{list: "fd00::/8", host: "fd12::1", port: "443", want: true},
		{list: "::1", host: "::1", port: "443", want: true},
		{list: "[::1]:8080", host: "::1", port: "8080", want: true},
		{list: "[::1]:8080", host: "::1", port: "443", want: false},
		{list: "example.com:8443", host: "example.com", port: "8443", want: true},
		{list: "example.com:8443", host: "example.com", port: "443", want: false},
		{list: "bücher.example", host: "xn--bcher-kva.example", port: "443", want: true},
		{list: "EXAMPLE.com", host: "example.COM", port: "443", want: true},
		{list: "example.com", host: "example.com.", port: "443", want: true},
		{list: " localhost , .corp ", host: "x.corp", port: "443", want: true},

		// An entry that is neither a valid CIDR nor anything else parseable is
		// kept as a plain domain string: it never errors and never matches.
		{list: "10.0.0.0/33", host: "10.0.0.1", port: "443", want: false},
		{list: "10.0.0.0/33", host: "example.com", port: "443", want: false},
		{list: "10.0.0.0/33,example.com", host: "example.com", port: "443", want: true},
		// The unicode host is compared in IDNA ASCII form on both sides.
		{list: "xn--bcher-kva.example", host: "bücher.example", port: "443", want: true},
		{list: "bücher.example", host: "x.bücher.example", port: "443", want: true},
		// IP entries match IP-literal hosts only, and the other way around.
		{list: "127.0.0.1", host: "127.0.0.1", port: "443", want: true},
		{list: "127.0.0.1", host: "127.0.0.2", port: "443", want: false},
		{list: "127.0.0.1:8080", host: "127.0.0.1", port: "8080", want: true},
		{list: "127.0.0.1:8080", host: "127.0.0.1", port: "443", want: false},
		{list: "1.2.3", host: "0.1.2.3", port: "443", want: false},
		{list: "example.com", host: "::1", port: "443", want: false},
		{list: "FD00::/8", host: "FD12::1", port: "443", want: true},
		// A trailing dot makes a host a DNS name (resolved, then dialled
		// directly), not an IP literal: no IP or CIDR entry matches it.
		{list: "10.0.0.0/8", host: "10.1.2.3.", port: "443", want: false},
		{list: "10.1.2.3", host: "10.1.2.3.", port: "443", want: false},
		{list: "::1", host: "::1.", port: "443", want: false},
		{list: "10.1.2.3.", host: "10.1.2.3.", port: "443", want: true},
		{list: "10.1.2.3.", host: "10.1.2.3", port: "443", want: false},
		// Go dials an IPv4-mapped IPv6 literal as IPv4, so it matches the
		// IPv4 entries of its address.
		{list: "10.0.0.0/8", host: "::ffff:10.1.2.3", port: "443", want: true},
		{list: "10.1.2.3", host: "::ffff:10.1.2.3", port: "443", want: true},
		// A bracketed IPv6 literal without a port is the address itself.
		{list: "[::1]", host: "::1", port: "443", want: true},
		// An empty port restricts nothing; a port is compared as a number.
		{list: "example.com:", host: "example.com", port: "443", want: true},
		{list: "example.com:08443", host: "example.com", port: "8443", want: true},
		{list: "example.com:notaport", host: "example.com", port: "443", want: false},
		{list: ".corp:8443", host: "x.corp", port: "8443", want: true},
		{list: "*.corp:8443", host: "x.corp", port: "443", want: false},
		// A trailing dot on the entry means the same as without.
		{list: "example.com.", host: "example.com", port: "443", want: true},
		// Empty entries and lone dots match nothing.
		{list: ",, ,.,*.", host: "example.com", port: "443", want: false},
		{list: ",, ,.,*.", host: "", port: "443", want: false},
		// Only whole labels match.
		{list: "example.com", host: "example.com.evil", port: "443", want: false},
		{list: "example.com", host: "com", port: "443", want: false},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q vs %s:%s", tc.list, tc.host, tc.port), func(t *testing.T) {
			n, err := parseNoProxy(tc.list, "NO_PROXY")
			require.NoError(t, err)
			assert.Equal(t, tc.want, n.matches(tc.host, tc.port))
		})
	}
}

func TestNoProxyMatches_ZeroValueMatchesNothing(t *testing.T) {
	var n noProxy
	assert.False(t, n.matches("example.com", "443"))
	assert.False(t, n.matches("127.0.0.1", "443"))
}

func TestParseNoProxy_StarAmongEntries(t *testing.T) {
	for _, raw := range []string{"localhost,*", "*,localhost", " * ,example.com", "example.com,*"} {
		t.Run(raw, func(t *testing.T) {
			_, err := parseNoProxy(raw, "NO_PROXY")
			require.ErrorIs(t, err, apperr.ErrProxyConfig)
			assert.ErrorContains(t, err, `NO_PROXY: "*" must stand alone; use NO_PROXY=* to disable the proxy`)
		})
	}
}

func TestParseNoProxy_NamesSource(t *testing.T) {
	_, err := parseNoProxy("a,*", "no_proxy")
	require.ErrorIs(t, err, apperr.ErrProxyConfig)
	assert.ErrorContains(t, err, "no_proxy: ")
}

func TestParseNoProxy_InvalidEntriesAreNotErrors(t *testing.T) {
	for _, raw := range []string{"10.0.0.0/33", "a_b.example", "example.com:99999", "[::1", "::1/129", "a:b:c"} {
		t.Run(raw, func(t *testing.T) {
			_, err := parseNoProxy(raw, "NO_PROXY")
			assert.NoError(t, err)
		})
	}
}
