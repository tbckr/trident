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

// wireResponder returns a responder that always replies with the given wire data.
func wireResponder(data []byte) httpmock.Responder {
	return func(r *http.Request) (*http.Response, error) {
		encoded := r.URL.Query().Get("dns")
		if _, err := base64.RawURLEncoding.DecodeString(encoded); err != nil {
			return nil, fmt.Errorf("decode base64url: %w", err)
		}
		return httpmock.NewBytesResponse(http.StatusOK, data), nil
	}
}

func TestService_Run_Blocked(t *testing.T) {
	client := newTestClient(t)

	// Quad9 blocked: NXDOMAIN with empty authority section.
	blockedData := buildWireResponse(t, dns.RcodeNameError, nil, nil)
	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, wireResponder(blockedData))

	svc := quad9.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "malicious.example")
	require.NoError(t, err)

	result, ok := raw.(*quad9.Result)
	require.True(t, ok, "expected *quad9.Result")
	assert.Equal(t, "malicious.example", result.Input)
	assert.True(t, result.Blocked)
}

func TestService_Run_NotBlocked(t *testing.T) {
	client := newTestClient(t)

	aRR := &dns.A{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300}}
	aRR.Addr = netip.MustParseAddr("93.184.216.34")
	notBlockedData := buildWireResponse(t, dns.RcodeSuccess, []dns.RR{aRR}, nil)
	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, wireResponder(notBlockedData))

	svc := quad9.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*quad9.Result)
	require.True(t, ok, "expected *quad9.Result")
	assert.Equal(t, "example.com", result.Input)
	assert.False(t, result.Blocked)
}

func TestService_Run_NXDOMAIN_NoComment(t *testing.T) {
	client := newTestClient(t)

	// Genuine NXDOMAIN: Rcode=3 with SOA in authority section → HasAuthority=true → not blocked.
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
	nxdomainData := buildWireResponse(t, dns.RcodeNameError, nil, []dns.RR{soaRR})
	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, wireResponder(nxdomainData))

	svc := quad9.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "nonexistent.example")
	require.NoError(t, err)

	result, ok := raw.(*quad9.Result)
	require.True(t, ok, "expected *quad9.Result")
	assert.False(t, result.Blocked, "NXDOMAIN with authority section should not be flagged as blocked")
}

func TestService_Run_InvalidInput(t *testing.T) {
	client := newTestClient(t)
	svc := quad9.NewService(client, testutil.NopLogger())

	for _, bad := range []string{"", "not_a_domain", "has space.com"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestService_Run_HTTPError(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet, `=~^`+dohURL,
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	svc := quad9.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
	assert.Nil(t, raw)
}

func TestService_AggregateResults(t *testing.T) {
	client := req.NewClient()
	svc := quad9.NewService(client, testutil.NopLogger())

	r1 := &quad9.Result{Input: "a.com", Blocked: true}
	r2 := &quad9.Result{Input: "b.com", Blocked: false}

	agg := svc.AggregateResults([]services.Result{r1, r2})
	mr, ok := agg.(*quad9.MultiResult)
	require.True(t, ok, "expected *quad9.MultiResult")
	assert.Len(t, mr.Results, 2)
	assert.True(t, mr.Results[0].Blocked)
	assert.False(t, mr.Results[1].Blocked)
}

func TestService_PAP(t *testing.T) {
	client := req.NewClient()
	svc := quad9.NewService(client, testutil.NopLogger())
	assert.Equal(t, "amber", svc.PAP().String())
}
