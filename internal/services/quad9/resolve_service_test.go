package quad9_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/netip"
	"testing"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/quad9"
	"github.com/tbckr/trident/internal/testutil"
)

func newTestClient(t *testing.T) *req.Client {
	t.Helper()
	client := req.NewClient()
	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)
	return client
}

// dohURL is the Quad9 DoH endpoint, used to register mock responders.
const dohURL = "https://dns.quad9.net/dns-query"

// buildWireResponse packs a DNS response message into wire format.
func buildWireResponse(t *testing.T, rcode int, answers []dns.RR, authority []dns.RR) []byte {
	t.Helper()
	m := new(dns.Msg)
	m.Rcode = uint16(rcode)
	m.Response = true
	m.Answer = answers
	m.Ns = authority
	require.NoError(t, m.Pack())
	return m.Data
}

// wireResponder returns an httpmock.Responder that decodes the ?dns= base64url query param,
// unpacks the DNS query, and delegates to handler based on the question type.
func wireResponder(t *testing.T, handler func(qtype uint16) []byte) httpmock.Responder {
	t.Helper()
	return func(r *http.Request) (*http.Response, error) {
		encoded := r.URL.Query().Get("dns")
		data, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode base64url: %w", err)
		}
		m := new(dns.Msg)
		m.Data = data
		if err := m.Unpack(); err != nil {
			return nil, fmt.Errorf("unpack DNS query: %w", err)
		}
		var qtype uint16
		if len(m.Question) > 0 {
			qtype = dns.RRToType(m.Question[0])
		}
		return httpmock.NewBytesResponse(http.StatusOK, handler(qtype)), nil
	}
}

func registerAllRecordTypes(t *testing.T) {
	t.Helper()

	aRR := &dns.A{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300}}
	aRR.Addr = netip.MustParseAddr("93.184.216.34")

	aaaaRR := &dns.AAAA{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300}}
	aaaaRR.Addr = netip.MustParseAddr("2606:2800:220:1:248:1893:25c8:1946")

	ns1RR := &dns.NS{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 3600}, NS: rdata.NS{Ns: "a.iana-servers.net."}}
	ns2RR := &dns.NS{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 3600}, NS: rdata.NS{Ns: "b.iana-servers.net."}}

	mxRR := &dns.MX{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 3600}, MX: rdata.MX{Preference: 0, Mx: "."}}

	txtRR := &dns.TXT{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 3600}, TXT: rdata.TXT{Txt: []string{"v=spf1 -all"}}}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, wireResponder(t, func(qtype uint16) []byte {
		switch qtype {
		case dns.TypeA:
			return buildWireResponse(t, 0, []dns.RR{aRR}, nil)
		case dns.TypeAAAA:
			return buildWireResponse(t, 0, []dns.RR{aaaaRR}, nil)
		case dns.TypeNS:
			return buildWireResponse(t, 0, []dns.RR{ns1RR, ns2RR}, nil)
		case dns.TypeMX:
			return buildWireResponse(t, 0, []dns.RR{mxRR}, nil)
		case dns.TypeTXT:
			return buildWireResponse(t, 0, []dns.RR{txtRR}, nil)
		default:
			return buildWireResponse(t, 0, nil, nil)
		}
	}))
}

func TestResolveService_Run_ValidDomain(t *testing.T) {
	client := newTestClient(t)
	registerAllRecordTypes(t)

	svc := quad9.NewResolveService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*quad9.ResolveResult)
	require.True(t, ok, "expected *quad9.ResolveResult")

	assert.Equal(t, "example.com", result.Input)
	assert.Equal(t, []string{"93.184.216.34"}, result.A)
	assert.Equal(t, []string{"2606:2800:220:1:248:1893:25c8:1946"}, result.AAAA)
	assert.Contains(t, result.NS, "a.iana-servers.net.")
	assert.Contains(t, result.NS, "b.iana-servers.net.")
	assert.Equal(t, []string{"0 ."}, result.MX)
	assert.Equal(t, []string{"v=spf1 -all"}, result.TXT)
}

func TestResolveService_Run_EmptyDomain(t *testing.T) {
	client := newTestClient(t)
	svc := quad9.NewResolveService(client, testutil.NopLogger())

	_, err := svc.Run(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrInvalidInput)
}

func TestResolveService_Run_InvalidDomain(t *testing.T) {
	client := newTestClient(t)
	svc := quad9.NewResolveService(client, testutil.NopLogger())

	for _, bad := range []string{"not_a_domain", "has space.com", "$(injection)"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestResolveService_Run_HTTPError(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet, `=~^`+dohURL,
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	svc := quad9.NewResolveService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
	assert.Nil(t, raw)
}

func TestResolveService_Run_NetworkError(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet, `=~^`+dohURL,
		httpmock.NewErrorResponder(fmt.Errorf("connection refused")))

	svc := quad9.NewResolveService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
	assert.Nil(t, raw)
}

func TestResolveService_Run_ContextCanceled(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, wireResponder(t, func(_ uint16) []byte {
		return buildWireResponse(t, 0, nil, nil)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := quad9.NewResolveService(client, testutil.NopLogger())
	raw, err := svc.Run(ctx, "example.com")
	// Context cancelled — partial result returned, no error
	require.NoError(t, err)
	result, ok := raw.(*quad9.ResolveResult)
	require.True(t, ok, "expected *quad9.ResolveResult")
	assert.Equal(t, "example.com", result.Input)
}

func TestResolveService_AggregateResults(t *testing.T) {
	client := req.NewClient()
	svc := quad9.NewResolveService(client, testutil.NopLogger())

	r1 := &quad9.ResolveResult{Input: "a.com", A: []string{"1.2.3.4"}}
	r2 := &quad9.ResolveResult{Input: "b.com", A: []string{"5.6.7.8"}}

	agg := svc.AggregateResults([]services.Result{r1, r2})
	mr, ok := agg.(*quad9.ResolveMultiResult)
	require.True(t, ok, "expected *quad9.ResolveMultiResult")
	assert.Len(t, mr.Results, 2)
	assert.Equal(t, "a.com", mr.Results[0].Input)
	assert.Equal(t, "b.com", mr.Results[1].Input)
}

func TestResolveService_PAP(t *testing.T) {
	client := req.NewClient()
	svc := quad9.NewResolveService(client, testutil.NopLogger())
	assert.Equal(t, "amber", svc.PAP().String())
}
