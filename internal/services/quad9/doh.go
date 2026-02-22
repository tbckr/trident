package quad9

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"codeberg.org/miekg/dns"
	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/services"
)

const (
	// dohURL is the Quad9 DNS-over-HTTPS endpoint.
	dohURL = "https://dns.quad9.net/dns-query"

	// DefaultRPS is the target request rate for the Quad9 service.
	DefaultRPS float64 = 5
	// DefaultBurst is the burst capacity above DefaultRPS.
	DefaultBurst = 10

	// DNS record type codes (RFC 1035 / RFC 3596).
	dnsTypeA    = 1
	dnsTypeNS   = 2
	dnsTypeMX   = 15
	dnsTypeAAAA = 28
	dnsTypeTXT  = 16

	// DoH status codes (RFC 8482 / RFC 1035).
	dohStatusNXDomain = 3
)

// dohResponse holds the parsed DNS wire-format response.
type dohResponse struct {
	Status       int
	HasAuthority bool
	Answer       []dohAnswer
}

// dohAnswer holds a single DNS resource record from a DoH response.
type dohAnswer struct {
	Name string
	Type int
	TTL  int
	Data string
}

// buildDNSQuery encodes a DNS query for the given domain and record type into wire format.
func buildDNSQuery(domain string, recordType int) ([]byte, error) {
	m := dns.NewMsg(domain, uint16(recordType))
	if m == nil {
		return nil, fmt.Errorf("unknown DNS record type: %d", recordType)
	}
	if err := m.Pack(); err != nil {
		return nil, err
	}
	return m.Data, nil
}

// parseDNSResponse decodes a DNS wire-format response into a dohResponse.
func parseDNSResponse(data []byte) (*dohResponse, error) {
	m := new(dns.Msg)
	m.Data = data
	if err := m.Unpack(); err != nil {
		return nil, fmt.Errorf("failed to parse DNS response: %w", err)
	}
	resp := &dohResponse{
		Status:       int(m.Rcode),
		HasAuthority: len(m.Ns) > 0,
	}
	for _, rr := range m.Answer {
		ans := dohAnswer{
			Name: rr.Header().Name,
			TTL:  int(rr.Header().TTL),
		}
		switch v := rr.(type) {
		case *dns.A:
			ans.Type = dnsTypeA
			ans.Data = v.Addr.String()
		case *dns.AAAA:
			ans.Type = dnsTypeAAAA
			ans.Data = v.Addr.String()
		case *dns.NS:
			ans.Type = dnsTypeNS
			ans.Data = v.Ns
		case *dns.MX:
			ans.Type = dnsTypeMX
			ans.Data = fmt.Sprintf("%d %s", v.Preference, v.Mx)
		case *dns.TXT:
			ans.Type = dnsTypeTXT
			ans.Data = strings.Join(v.Txt, "")
		default:
			continue
		}
		resp.Answer = append(resp.Answer, ans)
	}
	return resp, nil
}

// makeDoHRequest performs a DNS-over-HTTPS query using RFC 8484 wire format.
// It encodes the DNS query as base64url and sends it as the "dns" query parameter.
func makeDoHRequest(ctx context.Context, client *req.Client, domain string, recordType int) (*dohResponse, error) {
	query, err := buildDNSQuery(domain, recordType)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to build DNS query for %q type %d: %w", services.ErrRequestFailed, domain, recordType, err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(query)

	httpResp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/dns-message").
		SetQueryParam("dns", encoded).
		Get(dohURL)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: quad9 request error for %q type %d: %w", services.ErrRequestFailed, domain, recordType, err)
	}
	if !httpResp.IsSuccessState() {
		body := httpResp.String()
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return nil, fmt.Errorf("%w: quad9 returned HTTP %d for %q type %d: %q", services.ErrRequestFailed, httpResp.StatusCode, domain, recordType, body)
	}
	return parseDNSResponse(httpResp.Bytes())
}
