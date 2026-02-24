package detect

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

const (
	// Name is the service identifier.
	Name = "detect"
	// PAP is the PAP activity level for the detect service.
	PAP = pap.GREEN
)

// Service detects CDN, email, and DNS hosting providers from DNS records.
type Service struct {
	resolver services.DNSResolverInterface
	logger   *slog.Logger
	detector *providers.Detector
}

// NewService creates a new detect service with the given resolver, logger, and patterns.
func NewService(resolver services.DNSResolverInterface, logger *slog.Logger, patterns providers.Patterns) *Service {
	return &Service{
		resolver: resolver,
		logger:   logger,
		detector: providers.NewDetector(patterns),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return Name }

// PAP returns the PAP activity level for the detect service.
func (s *Service) PAP() pap.Level { return PAP }

// AggregateResults combines multiple detect results into a MultiResult.
func (s *Service) AggregateResults(results []services.Result) services.Result {
	mr := &MultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*Result))
	}
	return mr
}

// Run detects cloud service providers from DNS records for the given domain.
func (s *Service) Run(ctx context.Context, input string) (services.Result, error) {
	clean := output.StripANSI(input)
	if !services.IsDomain(clean) {
		return nil, fmt.Errorf("%w: must be a valid domain name: %q", services.ErrInvalidInput, input)
	}

	result := &Result{Input: clean}

	// Collect CNAME targets (apex and www).
	var cnames []string
	if canonical, err := s.resolver.LookupCNAME(ctx, clean); err != nil {
		s.logger.Debug("CNAME lookup failed", "domain", clean, "error", err)
	} else if canonical != "" && strings.TrimSuffix(canonical, ".") != clean {
		cnames = append(cnames, canonical)
	}
	www := "www." + clean
	if canonical, err := s.resolver.LookupCNAME(ctx, www); err != nil {
		s.logger.Debug("CNAME lookup failed", "domain", www, "error", err)
	} else if canonical != "" && strings.TrimSuffix(canonical, ".") != www {
		cnames = append(cnames, canonical)
	}

	// Collect MX hosts.
	var mxHosts []string
	if mxs, err := s.resolver.LookupMX(ctx, clean); err != nil {
		s.logger.Debug("MX lookup failed", "domain", clean, "error", err)
	} else {
		for _, mx := range mxs {
			mxHosts = append(mxHosts, output.StripANSI(mx.Host))
		}
	}

	// Collect NS hosts.
	var nsHosts []string
	if nss, err := s.resolver.LookupNS(ctx, clean); err != nil {
		s.logger.Debug("NS lookup failed", "domain", clean, "error", err)
	} else {
		for _, ns := range nss {
			nsHosts = append(nsHosts, output.StripANSI(ns.Host))
		}
	}

	// Collect TXT records.
	var txtRecords []string
	if txts, err := s.resolver.LookupTXT(ctx, clean); err != nil {
		s.logger.Debug("TXT lookup failed", "domain", clean, "error", err)
	} else {
		for _, txt := range txts {
			txtRecords = append(txtRecords, output.StripANSI(txt))
		}
	}

	// Run detections.
	for _, d := range s.detector.CDN(cnames) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
			Source:   d.Source,
		})
	}
	for _, d := range s.detector.EmailProvider(mxHosts) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
			Source:   d.Source,
		})
	}
	for _, d := range s.detector.DNSHost(nsHosts) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
			Source:   d.Source,
		})
	}
	for _, d := range s.detector.TXTRecord(txtRecords) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
			Source:   d.Source,
		})
	}

	return result, nil
}
