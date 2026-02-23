package apex_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/apex"
	"github.com/tbckr/trident/internal/testutil"
)

const dohURL = "https://dns.quad9.net/dns-query"

func newTestClient(t *testing.T) *req.Client {
	t.Helper()
	client := req.NewClient()
	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)
	return client
}

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

// apexWireResponder decodes the DNS wire query and dispatches based on hostname and type.
func apexWireResponder(t *testing.T, handler func(qname string, qtype uint16) []byte) httpmock.Responder {
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
		var qname string
		var qtype uint16
		if len(m.Question) > 0 {
			qname = strings.TrimSuffix(m.Question[0].Header().Name, ".")
			qtype = dns.RRToType(m.Question[0])
		}
		return httpmock.NewBytesResponse(http.StatusOK, handler(qname, qtype)), nil
	}
}

func TestApexService_Name(t *testing.T) {
	svc := apex.NewService(req.NewClient(), testutil.NopLogger())
	assert.Equal(t, "apex", svc.Name())
}

func TestApexService_PAP(t *testing.T) {
	svc := apex.NewService(req.NewClient(), testutil.NopLogger())
	assert.Equal(t, "amber", svc.PAP().String())
}

func TestApexService_Run_ValidDomain(t *testing.T) {
	client := newTestClient(t)

	aRR := &dns.A{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300}}
	aRR.Addr = netip.MustParseAddr("93.184.216.34")

	ns1RR := &dns.NS{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 3600}, NS: rdata.NS{Ns: "a.iana-servers.net."}}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			if qname == "example.com" {
				switch qtype {
				case dns.TypeA:
					return buildWireResponse(t, 0, []dns.RR{aRR}, nil)
				case dns.TypeNS:
					return buildWireResponse(t, 0, []dns.RR{ns1RR}, nil)
				}
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	assert.Equal(t, "example.com", result.Input)
	assert.False(t, result.IsEmpty())

	var aRecords, nsRecords []apex.Record
	for _, rec := range result.Records {
		switch rec.Type {
		case "A":
			aRecords = append(aRecords, rec)
		case "NS":
			nsRecords = append(nsRecords, rec)
		}
	}
	require.NotEmpty(t, aRecords)
	assert.Equal(t, "example.com", aRecords[0].Host)
	assert.Equal(t, "93.184.216.34", aRecords[0].Value)

	require.NotEmpty(t, nsRecords)
	assert.Equal(t, "example.com", nsRecords[0].Host)
	assert.Equal(t, "a.iana-servers.net.", nsRecords[0].Value)
}

func TestApexService_Run_CDNDetection(t *testing.T) {
	client := newTestClient(t)

	cnameRR := &dns.CNAME{
		Hdr:   dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300},
		CNAME: rdata.CNAME{Target: "abc.cloudfront.net."},
	}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			// Only return a CNAME for the apex domain CNAME query.
			// All other hostnames return empty responses.
			if qtype == dns.TypeCNAME && qname == "example.com" {
				return buildWireResponse(t, 0, []dns.RR{cnameRR}, nil)
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	var cdnRecords []apex.Record
	for _, rec := range result.Records {
		if rec.Type == "CDN" {
			cdnRecords = append(cdnRecords, rec)
		}
	}
	require.Len(t, cdnRecords, 1)
	assert.Contains(t, cdnRecords[0].Value, "AWS CloudFront")
	assert.Contains(t, cdnRecords[0].Value, "abc.cloudfront.net.")
}

func TestApexService_Run_EmailProviderDetection(t *testing.T) {
	client := newTestClient(t)

	mxRR := &dns.MX{
		Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300},
		MX:  rdata.MX{Preference: 10, Mx: "aspmx.l.google.com."},
	}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			if qname == "example.com" && qtype == dns.TypeMX {
				return buildWireResponse(t, 0, []dns.RR{mxRR}, nil)
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	var emailRecords []apex.Record
	for _, rec := range result.Records {
		if rec.Host == "email" {
			emailRecords = append(emailRecords, rec)
		}
	}
	require.Len(t, emailRecords, 1)
	assert.Equal(t, "Email", emailRecords[0].Type)
	assert.Contains(t, emailRecords[0].Value, "Google Workspace")
}

func TestApexService_Run_DNSHostDetection(t *testing.T) {
	client := newTestClient(t)

	nsRR := &dns.NS{
		Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 3600},
		NS:  rdata.NS{Ns: "liz.ns.cloudflare.com."},
	}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			if qname == "example.com" && qtype == dns.TypeNS {
				return buildWireResponse(t, 0, []dns.RR{nsRR}, nil)
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	var dnsRecords []apex.Record
	for _, rec := range result.Records {
		if rec.Host == "dns" {
			dnsRecords = append(dnsRecords, rec)
		}
	}
	require.Len(t, dnsRecords, 1)
	assert.Equal(t, "DNS", dnsRecords[0].Type)
	assert.Contains(t, dnsRecords[0].Value, "Cloudflare DNS")
}

func TestApexService_Run_InvalidInput(t *testing.T) {
	client := newTestClient(t)
	svc := apex.NewService(client, testutil.NopLogger())

	for _, bad := range []string{"", "not_a_domain", "has space.com", "$(injection)"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestApexService_Run_ContextCanceled(t *testing.T) {
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(_ string, _ uint16) []byte {
			return buildWireResponse(t, 0, nil, nil)
		}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(ctx, "example.com")
	// Context cancelled — partial result returned, no error
	require.NoError(t, err)
	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")
	assert.Equal(t, "example.com", result.Input)
}

func TestApexService_Run_HTTPError(t *testing.T) {
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	svc := apex.NewService(client, testutil.NopLogger())
	// HTTP errors are logged and skipped — service still returns a result (possibly empty)
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)
	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")
	assert.Equal(t, "example.com", result.Input)
}

func TestApexService_Run_NetworkError(t *testing.T) {
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		httpmock.NewErrorResponder(fmt.Errorf("connection refused")))

	svc := apex.NewService(client, testutil.NopLogger())
	// Network errors are logged and skipped — service still returns a result (possibly empty)
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)
	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")
	assert.Equal(t, "example.com", result.Input)
}

func TestApexService_Run_DNSKEY(t *testing.T) {
	client := newTestClient(t)

	dnskeyRR := &dns.DNSKEY{
		Hdr:    dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 3600},
		DNSKEY: rdata.DNSKEY{Flags: 257, Protocol: 3, Algorithm: 8, PublicKey: "AwEAAagAIKlVZrpC"},
	}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			if qname == "example.com" && qtype == dns.TypeDNSKEY {
				return buildWireResponse(t, 0, []dns.RR{dnskeyRR}, nil)
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	var found []apex.Record
	for _, rec := range result.Records {
		if rec.Type == "DNSKEY" {
			found = append(found, rec)
		}
	}
	require.NotEmpty(t, found)
	assert.Equal(t, "example.com", found[0].Host)
}

func TestApexService_Run_SRVServices(t *testing.T) {
	client := newTestClient(t)

	srvRR := &dns.SRV{
		Hdr: dns.Header{Name: "_sip._tls.example.com.", Class: dns.ClassINET, TTL: 300},
		SRV: rdata.SRV{Priority: 10, Weight: 20, Port: 5061, Target: "sip.example.com."},
	}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			if qname == "_sip._tls.example.com" && qtype == dns.TypeSRV {
				return buildWireResponse(t, 0, []dns.RR{srvRR}, nil)
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	var found []apex.Record
	for _, rec := range result.Records {
		if rec.Type == "SRV" && rec.Host == "_sip._tls.example.com" {
			found = append(found, rec)
		}
	}
	require.NotEmpty(t, found)
	assert.Equal(t, "_sip._tls.example.com", found[0].Host)
}

func TestApexService_Run_EmailCNAME(t *testing.T) {
	client := newTestClient(t)

	cnameRR := &dns.CNAME{
		Hdr:   dns.Header{Name: "_dmarc.example.com.", Class: dns.ClassINET, TTL: 300},
		CNAME: rdata.CNAME{Target: "dmarc.provider.com."},
	}

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			if qname == "_dmarc.example.com" && qtype == dns.TypeCNAME {
				return buildWireResponse(t, 0, []dns.RR{cnameRR}, nil)
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	var found []apex.Record
	for _, rec := range result.Records {
		if rec.Type == "CNAME" && rec.Host == "_dmarc.example.com" {
			found = append(found, rec)
		}
	}
	require.NotEmpty(t, found)
	assert.Equal(t, "_dmarc.example.com", found[0].Host)
}

func TestApexService_Run_OutputOrder(t *testing.T) {
	client := newTestClient(t)

	apexARR := &dns.A{Hdr: dns.Header{Name: "example.com.", Class: dns.ClassINET, TTL: 300}}
	apexARR.Addr = netip.MustParseAddr("93.184.216.34")

	wwwARR := &dns.A{Hdr: dns.Header{Name: "www.example.com.", Class: dns.ClassINET, TTL: 300}}
	wwwARR.Addr = netip.MustParseAddr("93.184.216.35")

	httpmock.RegisterResponder(http.MethodGet, "=~^"+dohURL,
		apexWireResponder(t, func(qname string, qtype uint16) []byte {
			switch {
			case qname == "example.com" && qtype == dns.TypeA:
				return buildWireResponse(t, 0, []dns.RR{apexARR}, nil)
			case qname == "www.example.com" && qtype == dns.TypeA:
				return buildWireResponse(t, 0, []dns.RR{wwwARR}, nil)
			}
			return buildWireResponse(t, 0, nil, nil)
		}))

	svc := apex.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*apex.Result)
	require.True(t, ok, "expected *apex.Result")

	var apexIdx, wwwIdx int
	apexIdx, wwwIdx = -1, -1
	for i, rec := range result.Records {
		if rec.Type == "A" && rec.Host == "example.com" {
			apexIdx = i
		}
		if rec.Type == "A" && rec.Host == "www.example.com" {
			wwwIdx = i
		}
	}
	require.NotEqual(t, -1, apexIdx, "example.com A record not found")
	require.NotEqual(t, -1, wwwIdx, "www.example.com A record not found")
	assert.Less(t, apexIdx, wwwIdx, "example.com A must appear before www.example.com A")
}

func TestApexService_AggregateResults(t *testing.T) {
	client := req.NewClient()
	svc := apex.NewService(client, testutil.NopLogger())

	r1 := &apex.Result{
		Input:   "a.com",
		Records: []apex.Record{{Host: "a.com", Type: "A", Value: "1.2.3.4"}},
	}
	r2 := &apex.Result{
		Input:   "b.com",
		Records: []apex.Record{{Host: "b.com", Type: "A", Value: "5.6.7.8"}},
	}

	agg := svc.AggregateResults([]services.Result{r1, r2})
	mr, ok := agg.(*apex.MultiResult)
	require.True(t, ok, "expected *apex.MultiResult")
	assert.Len(t, mr.Results, 2)
	assert.Equal(t, "a.com", mr.Results[0].Input)
	assert.Equal(t, "b.com", mr.Results[1].Input)
}
