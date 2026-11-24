package doh

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"codeberg.org/miekg/dns"
	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/apperr"
)

const (
	// Endpoint is the Quad9 DNS-over-HTTPS endpoint.
	Endpoint = "https://dns.quad9.net/dns-query"

	// DefaultRPS is the target request rate for the Quad9 service.
	DefaultRPS float64 = 5
	// DefaultBurst is the burst capacity above DefaultRPS.
	DefaultBurst = 10
)

// Response holds the parsed DNS wire-format response.
type Response struct {
	Status       uint16
	HasAuthority bool
	Answer       []Answer
}

// Answer holds a single DNS resource record from a DoH response.
type Answer struct {
	Name string
	Type uint16
	TTL  int
	Data string
}

// buildDNSQuery encodes a DNS query for the given domain and record type into wire format.
func buildDNSQuery(domain string, recordType uint16) ([]byte, error) {
	m := dns.NewMsg(domain, recordType)
	if m == nil {
		return nil, fmt.Errorf("unknown DNS record type: %d", recordType)
	}
	if err := m.Pack(); err != nil {
		return nil, err
	}
	return m.Data, nil
}

// parseDNSResponse decodes a DNS wire-format response into a Response.
func parseDNSResponse(data []byte) (*Response, error) {
	m := new(dns.Msg)
	m.Data = data
	if err := m.Unpack(); err != nil {
		return nil, fmt.Errorf("failed to parse DNS response: %w", err)
	}
	resp := &Response{
		Status:       m.Rcode,
		HasAuthority: len(m.Ns) > 0,
	}
	for _, rr := range m.Answer {
		ans := Answer{
			Name: rr.Header().Name,
			TTL:  int(rr.Header().TTL),
		}
		switch v := rr.(type) {
		case *dns.A:
			ans.Type = dns.TypeA
			ans.Data = v.Addr.String()
		case *dns.AAAA:
			ans.Type = dns.TypeAAAA
			ans.Data = v.Addr.String()
		case *dns.NS:
			ans.Type = dns.TypeNS
			ans.Data = v.Ns
		case *dns.MX:
			ans.Type = dns.TypeMX
			ans.Data = fmt.Sprintf("%d %s", v.Preference, v.Mx)
		case *dns.CNAME:
			ans.Type = dns.TypeCNAME
			ans.Data = v.Target
		case *dns.SOA:
			ans.Type = dns.TypeSOA
			ans.Data = fmt.Sprintf("%s %s %d %d %d %d %d", v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
		case *dns.SRV:
			ans.Type = dns.TypeSRV
			ans.Data = fmt.Sprintf("%d %d %d %s", v.Priority, v.Weight, v.Port, v.Target)
		case *dns.TXT:
			ans.Type = dns.TypeTXT
			ans.Data = strings.Join(v.Txt, "")
		case *dns.CAA:
			ans.Type = dns.TypeCAA
			ans.Data = fmt.Sprintf("%d %s %q", v.Flag, v.Tag, v.Value)
		case *dns.DNSKEY:
			ans.Type = dns.TypeDNSKEY
			ans.Data = fmt.Sprintf("%d %d %d %s", v.Flags, v.Protocol, v.Algorithm, v.PublicKey)
		case *dns.HTTPS:
			ans.Type = dns.TypeHTTPS
			ans.Data = fmt.Sprintf("%d %s", v.Priority, v.Target)
		case *dns.SSHFP:
			ans.Type = dns.TypeSSHFP
			ans.Data = fmt.Sprintf("%d %d %s", v.Algorithm, v.Type, v.FingerPrint)
		default:
			continue
		}
		resp.Answer = append(resp.Answer, ans)
	}
	return resp, nil
}

// MakeDoHRequest performs a DNS-over-HTTPS query using RFC 8484 wire format.
// It encodes the DNS query as base64url and sends it as the "dns" query parameter.
func MakeDoHRequest(ctx context.Context, client *req.Client, domain string, recordType uint16) (*Response, error) {
	data, err := exchange(ctx, client, domain, recordType)
	if err != nil {
		return nil, err
	}
	return parseDNSResponse(data)
}

// exchange sends the DoH query and returns the raw wire-format response body.
//
// Errors are classified for the callers: a done caller context is returned
// unchanged (the caller cancelled or timed out), an [apperr.ErrProxyBypass]
// stays detectable so a refused direct connection is not mistaken for a plain
// network failure, and everything else is an [apperr.ErrRequestFailed]. Only
// the caller's context counts as a context error: an http.Client timeout also
// matches [context.DeadlineExceeded] while the caller is still waiting, and
// that is a failed request.
func exchange(ctx context.Context, client *req.Client, domain string, recordType uint16) ([]byte, error) {
	query, err := buildDNSQuery(domain, recordType)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to build DNS query for %q type %d: %v", apperr.ErrRequestFailed, domain, recordType, err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(query)

	httpResp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/dns-message").
		SetQueryParam("dns", encoded).
		Get(Endpoint)
	if err != nil {
		if ctx.Err() != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: quad9 request error for %q type %d: %v", apperr.RequestFailure(err), domain, recordType, err)
	}
	if !httpResp.IsSuccessState() {
		body := httpResp.String()
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return nil, fmt.Errorf("%w: quad9 returned HTTP %d for %q type %d: %q", apperr.ErrRequestFailed, httpResp.StatusCode, domain, recordType, body)
	}
	return httpResp.Bytes(), nil
}
