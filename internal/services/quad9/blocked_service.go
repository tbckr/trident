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

// BlockedService queries Quad9 DNS-over-HTTPS to detect whether a domain is blocked.
// Quad9 returns NXDOMAIN (Status=3) with a "blocked" comment for known-malicious domains.
type BlockedService struct {
	client *req.Client
	logger *slog.Logger
}

// NewBlockedService creates a new Quad9 blocked service with the given HTTP client and logger.
func NewBlockedService(client *req.Client, logger *slog.Logger) *BlockedService {
	return &BlockedService{client: client, logger: logger}
}

// Name returns the service identifier.
func (s *BlockedService) Name() string { return "quad9-blocked" }

// PAP returns the PAP activity level for the Quad9 blocked service (external API query).
func (s *BlockedService) PAP() pap.Level { return pap.AMBER }

// AggregateResults combines multiple Quad9 blocked results into a MultiResult.
func (s *BlockedService) AggregateResults(results []services.Result) services.Result {
	mr := &BlockedMultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*BlockedResult))
	}
	return mr
}

// Run queries Quad9 DoH with an A record request to determine whether the domain is blocked.
// A domain is considered blocked when Quad9 returns NXDOMAIN (Status=3) with an empty authority
// section, indicating a Quad9 threat-intelligence verdict. Genuine NXDOMAIN responses include
// a SOA record in the authority section.
func (s *BlockedService) Run(ctx context.Context, domain string) (services.Result, error) {
	domain = output.StripANSI(domain)
	if !services.IsDomain(domain) {
		return nil, fmt.Errorf("%w: must be a valid domain name: %q", services.ErrInvalidInput, domain)
	}

	result := &BlockedResult{Input: domain}

	resp, err := makeDoHRequest(ctx, s.client, domain, dns.TypeA)
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
