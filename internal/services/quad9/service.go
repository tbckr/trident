package quad9

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"codeberg.org/miekg/dns"
	"github.com/imroc/req/v3"

	dohpkg "github.com/tbckr/trident/internal/doh"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

const (
	// Name is the service identifier.
	Name = "quad9"
	// PAP is the PAP activity level for the Quad9 service.
	PAP = pap.AMBER
)

// Service queries Quad9 DNS-over-HTTPS to detect whether a domain is blocked.
// Quad9 returns NXDOMAIN (Status=3) with a "blocked" comment for known-malicious domains.
type Service struct {
	client *req.Client
	logger *slog.Logger
}

// NewService creates a new Quad9 blocked service with the given HTTP client and logger.
func NewService(client *req.Client, logger *slog.Logger) *Service {
	return &Service{client: client, logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return Name }

// PAP returns the PAP activity level for the Quad9 blocked service (external API query).
func (s *Service) PAP() pap.Level { return PAP }

// AggregateResults combines multiple Quad9 blocked results into a MultiResult.
func (s *Service) AggregateResults(results []services.Result) services.Result {
	mr := &MultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*Result))
	}
	return mr
}

// Run queries Quad9 DoH with an A record request to determine whether the domain is blocked.
// A domain is considered blocked when Quad9 returns NXDOMAIN (Status=3) with an empty authority
// section, indicating a Quad9 threat-intelligence verdict. Genuine NXDOMAIN responses include
// a SOA record in the authority section.
func (s *Service) Run(ctx context.Context, domain string) (services.Result, error) {
	domain = output.StripANSI(domain)
	if !services.IsDomain(domain) {
		return nil, fmt.Errorf("%w: must be a valid domain name: %q", services.ErrInvalidInput, domain)
	}

	result := &Result{Input: domain}

	resp, err := dohpkg.MakeDoHRequest(ctx, s.client, domain, dns.TypeA)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			s.logger.Debug("quad9 blocked: context cancelled", "domain", domain)
			return result, nil
		}
		return nil, err
	}

	if resp.Status == dns.RcodeNameError && !resp.HasAuthority {
		result.Blocked = true
	}

	return result, nil
}
