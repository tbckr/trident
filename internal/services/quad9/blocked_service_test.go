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

// blockedWireResponder returns a responder that always replies with the given wire data.
func blockedWireResponder(data []byte) httpmock.Responder {
	return func(r *http.Request) (*http.Response, error) {
		encoded := r.URL.Query().Get("dns")
		if _, err := base64.RawURLEncoding.DecodeString(encoded); err != nil {
			return nil, fmt.Errorf("decode base64url: %w", err)
		}
		return httpmock.NewBytesResponse(http.StatusOK, data), nil
	}
}

func TestBlockedService_Run_Blocked(t *testing.T) {
	client := newTestClient(t)

	// Quad9 blocked: NXDOMAIN with empty authority section.
	blockedData := buildWireResponse(t, dns.RcodeNameError, nil, nil)
	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, blockedWireResponder(blockedData))

	svc := quad9.NewBlockedService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "malicious.example")
	require.NoError(t, err)

	result, ok := raw.(*quad9.BlockedResult)
	require.True(t, ok, "expected *quad9.BlockedResult")
	assert.Equal(t, "malicious.example", result.Input)
	assert.True(t, result.Blocked)
}

func TestBlockedService_Run_NotBlocked(t *testing.T) {
	client := newTestClient(t)

	aRR := &dns.A{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300}}
	aRR.Addr = netip.MustParseAddr("93.184.216.34")
	notBlockedData := buildWireResponse(t, dns.RcodeSuccess, []dns.RR{aRR}, nil)
	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, blockedWireResponder(notBlockedData))

	svc := quad9.NewBlockedService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*quad9.BlockedResult)
	require.True(t, ok, "expected *quad9.BlockedResult")
	assert.Equal(t, "example.com", result.Input)
	assert.False(t, result.Blocked)
}

func TestBlockedService_Run_NXDOMAIN_NoComment(t *testing.T) {
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
	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL, blockedWireResponder(nxdomainData))

	svc := quad9.NewBlockedService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "nonexistent.example")
	require.NoError(t, err)

	result, ok := raw.(*quad9.BlockedResult)
	require.True(t, ok, "expected *quad9.BlockedResult")
	assert.False(t, result.Blocked, "NXDOMAIN with authority section should not be flagged as blocked")
}

func TestBlockedService_Run_InvalidInput(t *testing.T) {
	client := newTestClient(t)
	svc := quad9.NewBlockedService(client, testutil.NopLogger())

	for _, bad := range []string{"", "not_a_domain", "has space.com"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestBlockedService_Run_HTTPError(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet, `=~^`+dohURL,
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	svc := quad9.NewBlockedService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
	assert.Nil(t, raw)
}

func TestBlockedService_AggregateResults(t *testing.T) {
	client := req.NewClient()
	svc := quad9.NewBlockedService(client, testutil.NopLogger())

	r1 := &quad9.BlockedResult{Input: "a.com", Blocked: true}
	r2 := &quad9.BlockedResult{Input: "b.com", Blocked: false}

	agg := svc.AggregateResults([]services.Result{r1, r2})
	mr, ok := agg.(*quad9.BlockedMultiResult)
	require.True(t, ok, "expected *quad9.BlockedMultiResult")
	assert.Len(t, mr.Results, 2)
	assert.True(t, mr.Results[0].Blocked)
	assert.False(t, mr.Results[1].Blocked)
}

func TestBlockedService_PAP(t *testing.T) {
	client := req.NewClient()
	svc := quad9.NewBlockedService(client, testutil.NopLogger())
	assert.Equal(t, "amber", svc.PAP().String())
}
