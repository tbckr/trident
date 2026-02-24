package dns

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

const (
	// Name is the service identifier.
	Name = "dns"
	// PAP is the PAP activity level for the DNS service.
	PAP = pap.GREEN
)

// Service performs DNS lookups using the injected resolver.
type Service struct {
	resolver services.DNSResolverInterface
	logger   *slog.Logger
}

// NewService creates a new DNS service with the given resolver and logger.
func NewService(resolver services.DNSResolverInterface, logger *slog.Logger) *Service {
	return &Service{resolver: resolver, logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return Name }

// PAP returns the PAP activity level for the DNS service (passive lookup).
func (s *Service) PAP() pap.Level { return PAP }

// AggregateResults combines multiple DNS results into a MultiResult.
func (s *Service) AggregateResults(results []services.Result) services.Result {
	mr := &MultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*Result))
	}
	return mr
}

// Run executes DNS lookups for the given domain or IP address.
// For domain input: resolves A, AAAA, MX, NS, TXT records.
// For IP input: performs a reverse lookup (PTR records).
// Partial results are returned when individual record type lookups fail.
func (s *Service) Run(ctx context.Context, input string) (services.Result, error) {
	result := &Result{Input: output.StripANSI(input)}

	if ip := net.ParseIP(input); ip != nil {
		return s.runReverse(ctx, result, ip.String())
	}
	if !services.IsDomain(input) {
		return nil, fmt.Errorf("%w: must be a valid domain name or IP address: %q", services.ErrInvalidInput, input)
	}
	return s.runForward(ctx, result, input)
}

// runReverse performs a PTR lookup for the given IP address.
func (s *Service) runReverse(ctx context.Context, result *Result, ip string) (*Result, error) {
	ptrs, err := s.resolver.LookupAddr(ctx, ip)
	if err != nil {
		s.logger.Debug("PTR lookup failed", "ip", ip, "error", err)
	}
	for _, ptr := range ptrs {
		result.PTR = append(result.PTR, output.StripANSI(ptr))
	}
	return result, nil
}

// runForward resolves NS, CNAME, A, AAAA, MX, SRV, and TXT records for the given domain.
func (s *Service) runForward(ctx context.Context, result *Result, domain string) (*Result, error) {
	nss, err := s.resolver.LookupNS(ctx, domain)
	if err != nil {
		s.logger.Debug("NS lookup failed", "domain", domain, "error", err)
	}
	for _, ns := range nss {
		result.NS = append(result.NS, output.StripANSI(ns.Host))
	}

	canonical, err := s.resolver.LookupCNAME(ctx, domain)
	if err != nil {
		s.logger.Debug("CNAME lookup failed", "domain", domain, "error", err)
	} else if canonical != "" && strings.TrimSuffix(canonical, ".") != domain {
		result.CNAME = append(result.CNAME, output.StripANSI(canonical))
	}

	addrs, err := s.resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		s.logger.Debug("A/AAAA lookup failed", "domain", domain, "error", err)
	}
	for _, addr := range addrs {
		clean := output.StripANSI(addr.IP.String())
		if addr.IP.To4() != nil {
			result.A = append(result.A, clean)
		} else {
			result.AAAA = append(result.AAAA, clean)
		}
	}

	mxs, err := s.resolver.LookupMX(ctx, domain)
	if err != nil {
		s.logger.Debug("MX lookup failed", "domain", domain, "error", err)
	}
	for _, mx := range mxs {
		result.MX = append(result.MX, output.StripANSI(mx.Host))
	}

	_, srvs, err := s.resolver.LookupSRV(ctx, "", "", domain)
	if err != nil {
		s.logger.Debug("SRV lookup failed", "domain", domain, "error", err)
	}
	for _, srv := range srvs {
		result.SRV = append(result.SRV, output.StripANSI(
			fmt.Sprintf("%d %d %d %s", srv.Priority, srv.Weight, srv.Port, srv.Target)))
	}

	txts, err := s.resolver.LookupTXT(ctx, domain)
	if err != nil {
		s.logger.Debug("TXT lookup failed", "domain", domain, "error", err)
	}
	for _, txt := range txts {
		result.TXT = append(result.TXT, output.StripANSI(txt))
	}

	return result, nil
}
