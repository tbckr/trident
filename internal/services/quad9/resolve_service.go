package quad9

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"codeberg.org/miekg/dns"
	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

// ResolveService queries Quad9 DNS-over-HTTPS for DNS records.
type ResolveService struct {
	client *req.Client
	logger *slog.Logger
}

// NewResolveService creates a new Quad9 resolve service with the given HTTP client and logger.
func NewResolveService(client *req.Client, logger *slog.Logger) *ResolveService {
	return &ResolveService{client: client, logger: logger}
}

// Name returns the service identifier.
func (s *ResolveService) Name() string { return "quad9-resolve" }

// PAP returns the PAP activity level for the Quad9 resolve service (external API query).
func (s *ResolveService) PAP() pap.Level { return pap.AMBER }

// AggregateResults combines multiple Quad9 resolve results into a MultiResult.
func (s *ResolveService) AggregateResults(results []services.Result) services.Result {
	mr := &ResolveMultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*ResolveResult))
	}
	return mr
}

// maxCNAMEHops is the maximum number of CNAME hops to follow before stopping.
const maxCNAMEHops = 20

// resolveCNAMEChain follows CNAME records for domain and returns each target in order.
// It stops when no new CNAME is found, a cycle is detected, or maxCNAMEHops is reached.
// Context cancellation causes an early return of the partial chain without error.
func (s *ResolveService) resolveCNAMEChain(ctx context.Context, domain string) ([]string, error) {
	seen := map[string]bool{domain: true}
	if len(domain) == 0 || domain[len(domain)-1] != '.' {
		seen[domain+"."] = true
	}
	current := domain
	var chain []string
	for range maxCNAMEHops {
		resp, err := makeDoHRequest(ctx, s.client, current, dns.TypeCNAME)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return chain, nil
			}
			return nil, err
		}
		found := false
		for _, ans := range resp.Answer {
			if ans.Type != dns.TypeCNAME {
				continue
			}
			target := ans.Data
			if seen[target] {
				return chain, nil
			}
			chain = append(chain, target)
			seen[target] = true
			current = target
			found = true
			break
		}
		if !found {
			break
		}
	}
	return chain, nil
}

// Run queries Quad9 DoH for NS, SOA, CNAME, A, AAAA, MX, SRV, TXT, CAA, DNSKEY, HTTPS, and SSHFP
// records for the given domain. CNAME is resolved as a chain via repeated lookups.
// Partial results are returned when context is cancelled mid-query.
func (s *ResolveService) Run(ctx context.Context, domain string) (services.Result, error) {
	domain = output.StripANSI(domain)
	if !services.IsDomain(domain) {
		return nil, fmt.Errorf("%w: must be a valid domain name: %q", services.ErrInvalidInput, domain)
	}

	result := &ResolveResult{Input: domain}

	chain, err := s.resolveCNAMEChain(ctx, domain)
	if err != nil {
		return nil, err
	}
	result.CNAME = chain

	recordTypes := []struct {
		typeCode uint16
		name     string
	}{
		{dns.TypeNS, "NS"},
		{dns.TypeSOA, "SOA"},
		{dns.TypeA, "A"},
		{dns.TypeAAAA, "AAAA"},
		{dns.TypeMX, "MX"},
		{dns.TypeSRV, "SRV"},
		{dns.TypeTXT, "TXT"},
		{dns.TypeCAA, "CAA"},
		{dns.TypeDNSKEY, "DNSKEY"},
		{dns.TypeHTTPS, "HTTPS"},
		{dns.TypeSSHFP, "SSHFP"},
	}

	for _, rt := range recordTypes {
		resp, err := makeDoHRequest(ctx, s.client, domain, rt.typeCode)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				s.logger.Debug("quad9 resolve: context cancelled", "domain", domain, "type", rt.name)
				return result, nil
			}
			return nil, err
		}
		for _, ans := range resp.Answer {
			if ans.Type != rt.typeCode {
				continue
			}
			val := output.StripANSI(ans.Data)
			switch rt.typeCode {
			case dns.TypeNS:
				result.NS = append(result.NS, val)
			case dns.TypeSOA:
				result.SOA = append(result.SOA, val)
			case dns.TypeA:
				result.A = append(result.A, val)
			case dns.TypeAAAA:
				result.AAAA = append(result.AAAA, val)
			case dns.TypeMX:
				result.MX = append(result.MX, val)
			case dns.TypeSRV:
				result.SRV = append(result.SRV, val)
			case dns.TypeTXT:
				result.TXT = append(result.TXT, val)
			case dns.TypeCAA:
				result.CAA = append(result.CAA, val)
			case dns.TypeDNSKEY:
				result.DNSKEY = append(result.DNSKEY, val)
			case dns.TypeHTTPS:
				result.HTTPS = append(result.HTTPS, val)
			case dns.TypeSSHFP:
				result.SSHFP = append(result.SSHFP, val)
			}
		}
	}

	return result, nil
}
