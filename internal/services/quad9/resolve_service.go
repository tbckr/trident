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

// Run queries Quad9 DoH for A, AAAA, NS, MX, and TXT records for the given domain.
// Partial results are returned when context is cancelled mid-query.
func (s *ResolveService) Run(ctx context.Context, domain string) (services.Result, error) {
	domain = output.StripANSI(domain)
	if !services.IsDomain(domain) {
		return nil, fmt.Errorf("%w: must be a valid domain name: %q", services.ErrInvalidInput, domain)
	}

	result := &ResolveResult{Input: domain}

	recordTypes := []struct {
		typeCode uint16
		name     string
	}{
		{dns.TypeA, "A"},
		{dns.TypeAAAA, "AAAA"},
		{dns.TypeNS, "NS"},
		{dns.TypeMX, "MX"},
		{dns.TypeTXT, "TXT"},
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
			case dns.TypeA:
				result.A = append(result.A, val)
			case dns.TypeAAAA:
				result.AAAA = append(result.AAAA, val)
			case dns.TypeNS:
				result.NS = append(result.NS, val)
			case dns.TypeMX:
				result.MX = append(result.MX, val)
			case dns.TypeTXT:
				result.TXT = append(result.TXT, val)
			}
		}
	}

	return result, nil
}
