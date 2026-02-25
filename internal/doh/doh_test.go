package doh

import (
	"net/netip"
	"testing"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildWire packs a DNS response into wire format for use in tests.
func buildWire(t *testing.T, rcode int, answers []dns.RR, authority []dns.RR) []byte {
	t.Helper()
	m := new(dns.Msg)
	m.Rcode = uint16(rcode)
	m.Response = true
	m.Answer = answers
	m.Ns = authority
	require.NoError(t, m.Pack())
	return m.Data
}

func TestParseDNSResponse_ARecord(t *testing.T) {
	aRR := &dns.A{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300}}
	aRR.Addr = netip.MustParseAddr("93.184.216.34")

	data := buildWire(t, dns.RcodeSuccess, []dns.RR{aRR}, nil)
	resp, err := parseDNSResponse(data)
	require.NoError(t, err)

	assert.Equal(t, uint16(dns.RcodeSuccess), resp.Status)
	assert.False(t, resp.HasAuthority)
	require.Len(t, resp.Answer, 1)
	assert.Equal(t, uint16(dns.TypeA), resp.Answer[0].Type)
	assert.Equal(t, "93.184.216.34", resp.Answer[0].Data)
}

func TestParseDNSResponse_TXTRecord(t *testing.T) {
	txtRR := &dns.TXT{
		Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300},
		TXT: rdata.TXT{Txt: []string{"v=spf1 -all"}},
	}

	data := buildWire(t, dns.RcodeSuccess, []dns.RR{txtRR}, nil)
	resp, err := parseDNSResponse(data)
	require.NoError(t, err)

	require.Len(t, resp.Answer, 1)
	assert.Equal(t, uint16(dns.TypeTXT), resp.Answer[0].Type)
	assert.Equal(t, "v=spf1 -all", resp.Answer[0].Data)
}

func TestParseDNSResponse_HasAuthority(t *testing.T) {
	soaRR := &dns.SOA{
		Hdr: dns.Header{Name: "example.", Class: dns.ClassINET, TTL: 3600},
		SOA: rdata.SOA{
			Ns:      "ns1.example.",
			Mbox:    "hostmaster.example.",
			Serial:  1,
			Refresh: 3600,
			Retry:   900,
			Expire:  604800,
			Minttl:  300,
		},
	}

	data := buildWire(t, dns.RcodeNameError, nil, []dns.RR{soaRR})
	resp, err := parseDNSResponse(data)
	require.NoError(t, err)

	assert.Equal(t, uint16(dns.RcodeNameError), resp.Status)
	assert.True(t, resp.HasAuthority)
	assert.Empty(t, resp.Answer)
}

func TestParseDNSResponse_MalformedBytes(t *testing.T) {
	_, err := parseDNSResponse([]byte("garbage bytes"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse DNS response")
}

func TestParseDNSResponse_EmptyResponse(t *testing.T) {
	data := buildWire(t, dns.RcodeSuccess, nil, nil)
	resp, err := parseDNSResponse(data)
	require.NoError(t, err)

	assert.Equal(t, uint16(dns.RcodeSuccess), resp.Status)
	assert.False(t, resp.HasAuthority)
	assert.Empty(t, resp.Answer)
}
