package apex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"codeberg.org/miekg/dns"
	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/doh"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

// Service aggregates DNS reconnaissance for an apex domain via Quad9 DoH.
type Service struct {
	client *req.Client
	logger *slog.Logger
}

// NewService creates a new Service with the given HTTP client and logger.
func NewService(client *req.Client, logger *slog.Logger) *Service {
	return &Service{client: client, logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "apex" }

// PAP returns the PAP activity level for the apex service (Quad9 third-party API).
func (s *Service) PAP() pap.Level { return pap.AMBER }

// AggregateResults combines multiple apex results into an MultiResult.
func (s *Service) AggregateResults(results []services.Result) services.Result {
	mr := &MultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*Result))
	}
	return mr
}

const maxCNAMEHops = 20

// resolveCNAMEChain follows CNAME records for domain and returns each target in order.
// It stops when no new CNAME is found, a cycle is detected, or maxCNAMEHops is reached.
// Context cancellation causes an early return of the partial chain without error.
func (s *Service) resolveCNAMEChain(ctx context.Context, domain string) ([]string, error) {
	seen := map[string]bool{domain: true}
	if len(domain) == 0 || domain[len(domain)-1] != '.' {
		seen[domain+"."] = true
	}
	current := domain
	var chain []string
	for range maxCNAMEHops {
		resp, err := doh.MakeDoHRequest(ctx, s.client, current, dns.TypeCNAME)
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

// directQuery represents a single (hostname, DNS record type) lookup.
type directQuery struct {
	host     string
	typeCode uint16
	typeName string
}

// Run performs parallel DNS reconnaissance for the given apex domain.
// Partial results are returned when context is cancelled mid-query.
func (s *Service) Run(ctx context.Context, domain string) (services.Result, error) {
	domain = output.StripANSI(domain)
	if !services.IsDomain(domain) {
		return nil, fmt.Errorf("%w: must be a valid domain name: %q", services.ErrInvalidInput, domain)
	}

	result := &Result{Input: domain}

	directQueries := []directQuery{
		// Apex domain — core types
		{domain, dns.TypeA, "A"},
		{domain, dns.TypeAAAA, "AAAA"},
		{domain, dns.TypeCAA, "CAA"},
		{domain, dns.TypeDNSKEY, "DNSKEY"},
		{domain, dns.TypeHTTPS, "HTTPS"},
		{domain, dns.TypeMX, "MX"},
		{domain, dns.TypeNS, "NS"},
		{domain, dns.TypeSOA, "SOA"},
		{domain, dns.TypeSSHFP, "SSHFP"},
		{domain, dns.TypeTXT, "TXT"},
		// SRV services
		{"_sip._tls." + domain, dns.TypeSRV, "SRV"},
		{"_sipfederationtls._tcp." + domain, dns.TypeSRV, "SRV"},
		{"_xmpp-client._tcp." + domain, dns.TypeSRV, "SRV"},
		{"_xmpp-server._tcp." + domain, dns.TypeSRV, "SRV"},
		// www subdomain
		{"www." + domain, dns.TypeA, "A"},
		{"www." + domain, dns.TypeAAAA, "AAAA"},
		{"www." + domain, dns.TypeHTTPS, "HTTPS"},
		// autodiscover subdomain
		{"autodiscover." + domain, dns.TypeA, "A"},
		{"autodiscover." + domain, dns.TypeAAAA, "AAAA"},
		// Email security subdomains (TXT then CNAME for each)
		{"_dmarc." + domain, dns.TypeTXT, "TXT"},
		{"_dmarc." + domain, dns.TypeCNAME, "CNAME"},
		{"_domainkey." + domain, dns.TypeTXT, "TXT"},
		{"_domainkey." + domain, dns.TypeCNAME, "CNAME"},
		{"_mta-sts." + domain, dns.TypeTXT, "TXT"},
		{"_mta-sts." + domain, dns.TypeCNAME, "CNAME"},
		{"_smtp._tls." + domain, dns.TypeTXT, "TXT"},
		{"_smtp._tls." + domain, dns.TypeCNAME, "CNAME"},
		// BIMI + DKIM selectors
		{"default._bimi." + domain, dns.TypeTXT, "TXT"},
		{"google._domainkey." + domain, dns.TypeTXT, "TXT"},
		{"selector1._domainkey." + domain, dns.TypeTXT, "TXT"},
		{"selector2._domainkey." + domain, dns.TypeTXT, "TXT"},
	}

	cnameHosts := []string{
		domain,
		"www." + domain,
		"autodiscover." + domain,
	}

	results := make([][]Record, len(directQueries))
	cnameResults := make([][]Record, len(cnameHosts))
	var wg sync.WaitGroup

	for i, q := range directQueries {
		wg.Go(func() {
			resp, err := doh.MakeDoHRequest(ctx, s.client, q.host, q.typeCode)
			if err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					s.logger.Debug("apex: query failed", "host", q.host, "type", q.typeName, "error", err)
				}
				return
			}
			var recs []Record
			for _, ans := range resp.Answer {
				if ans.Type != q.typeCode {
					continue
				}
				recs = append(recs, Record{Host: q.host, Type: q.typeName, Value: output.StripANSI(ans.Data)})
			}
			results[i] = recs
		})
	}

	for i, h := range cnameHosts {
		wg.Go(func() {
			chain, err := s.resolveCNAMEChain(ctx, h)
			if err != nil {
				s.logger.Debug("apex: CNAME chain failed", "host", h, "error", err)
				return
			}
			if len(chain) == 0 {
				return
			}
			var recs []Record
			for _, target := range chain {
				recs = append(recs, Record{Host: h, Type: "CNAME", Value: output.StripANSI(target)})
			}
			cnameResults[i] = recs
		})
	}

	wg.Wait()

	// Flatten direct query results in slice order (deterministic).
	for _, recs := range results {
		result.Records = append(result.Records, recs...)
	}

	// Flatten CNAME chains in slice order; collect targets for CDN detection.
	var allCNAMEs []string
	for _, recs := range cnameResults {
		for _, rec := range recs {
			allCNAMEs = append(allCNAMEs, rec.Value)
		}
		result.Records = append(result.Records, recs...)
	}

	// CDN detection from CNAME chains
	if len(allCNAMEs) > 0 {
		for _, d := range detect.CDN(allCNAMEs) {
			result.Records = append(result.Records, Record{
				Host:  "detected",
				Type:  string(d.Type),
				Value: d.Provider + " (cname: " + d.Evidence + ")",
			})
		}
	}

	// Email provider detection from MX records
	var mxHosts []string
	for _, rec := range result.Records {
		if rec.Type == "MX" {
			// MX value format: "10 aspmx.l.google.com." — take last token
			parts := strings.Fields(rec.Value)
			if len(parts) >= 2 {
				mxHosts = append(mxHosts, parts[len(parts)-1])
			}
		}
	}
	if len(mxHosts) > 0 {
		for _, d := range detect.EmailProvider(mxHosts) {
			result.Records = append(result.Records, Record{
				Host:  "detected",
				Type:  string(d.Type),
				Value: d.Provider + " (mx: " + d.Evidence + ")",
			})
		}
	}

	// DNS hosting detection from NS records
	var nsHosts []string
	for _, rec := range result.Records {
		if rec.Type == "NS" {
			nsHosts = append(nsHosts, rec.Value)
		}
	}
	if len(nsHosts) > 0 {
		for _, d := range detect.DNSHost(nsHosts) {
			result.Records = append(result.Records, Record{
				Host:  "detected",
				Type:  string(d.Type),
				Value: d.Provider + " (ns: " + d.Evidence + ")",
			})
		}
	}

	return result, nil
}
