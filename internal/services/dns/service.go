// Package dns implements DNS lookups using the Go standard library resolver.
package dns

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/validate"
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
func (s *Service) Name() string { return "dns" }

// PAP returns the PAP activity level for the DNS service (passive lookup).
func (s *Service) PAP() pap.Level { return pap.GREEN }

// Result holds the DNS lookup results for a single domain or IP input.
type Result struct {
	Input string   `json:"input"`
	A     []string `json:"a,omitempty"`
	AAAA  []string `json:"aaaa,omitempty"`
	MX    []string `json:"mx,omitempty"`
	NS    []string `json:"ns,omitempty"`
	TXT   []string `json:"txt,omitempty"`
	PTR   []string `json:"ptr,omitempty"`
}

// IsEmpty reports whether the result contains no DNS records.
func (r *Result) IsEmpty() bool {
	return len(r.A) == 0 && len(r.AAAA) == 0 &&
		len(r.MX) == 0 && len(r.NS) == 0 &&
		len(r.TXT) == 0 && len(r.PTR) == 0
}

// WritePlain renders the result as plain text with one record per line.
// Each line has the format: "TYPE value" (e.g. "NS ns1.example.com").
func (r *Result) WritePlain(w io.Writer) error {
	for _, v := range r.NS {
		if _, err := fmt.Fprintf(w, "NS %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.A {
		if _, err := fmt.Fprintf(w, "A %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.AAAA {
		if _, err := fmt.Fprintf(w, "AAAA %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.MX {
		if _, err := fmt.Fprintf(w, "MX %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.TXT {
		if _, err := fmt.Fprintf(w, "TXT %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.PTR {
		if _, err := fmt.Fprintf(w, "PTR %s\n", v); err != nil {
			return err
		}
	}
	return nil
}

// WriteText renders the result as an ASCII table, sorted and grouped by record type.
func (r *Result) WriteText(w io.Writer) error {
	var rows [][]string
	for _, v := range r.NS {
		rows = append(rows, []string{"NS", v})
	}
	for _, v := range r.A {
		rows = append(rows, []string{"A", v})
	}
	for _, v := range r.AAAA {
		rows = append(rows, []string{"AAAA", v})
	}
	for _, v := range r.MX {
		rows = append(rows, []string{"MX", v})
	}
	for _, v := range r.TXT {
		rows = append(rows, []string{"TXT", v})
	}
	for _, v := range r.PTR {
		rows = append(rows, []string{"PTR", v})
	}
	table := output.NewGroupedWrappingTable(w, 20, 20)
	table.Header([]string{"Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}

// Run executes DNS lookups for the given domain or IP address.
// For domain input: resolves A, AAAA, MX, NS, TXT records.
// For IP input: performs a reverse lookup (PTR records).
// Partial results are returned when individual record type lookups fail.
func (s *Service) Run(ctx context.Context, input string) (any, error) {
	result := &Result{Input: output.StripANSI(input)}

	if ip := net.ParseIP(input); ip != nil {
		return s.runReverse(ctx, result, ip.String())
	}
	if !validate.IsDomain(input) {
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

// runForward resolves A, AAAA, MX, NS, and TXT records for the given domain.
func (s *Service) runForward(ctx context.Context, result *Result, domain string) (*Result, error) {
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

	nss, err := s.resolver.LookupNS(ctx, domain)
	if err != nil {
		s.logger.Debug("NS lookup failed", "domain", domain, "error", err)
	}
	for _, ns := range nss {
		result.NS = append(result.NS, output.StripANSI(ns.Host))
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
