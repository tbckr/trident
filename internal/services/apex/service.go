package apex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"sort"
	"strings"
	"sync"

	"codeberg.org/miekg/dns"
	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/doh"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	cymrusvc "github.com/tbckr/trident/internal/services/cymru"
)

// Compile-time interface check.
var _ services.AggregateService = (*Service)(nil)

const (
	// Name is the service identifier.
	Name = "apex"
	// MinPAP is the minimum PAP level required to produce any useful output.
	// Quad9 DoH (the mandatory core sub-service) is AMBER.
	MinPAP = pap.AMBER
	// PAP is the highest PAP level among all sub-services.
	// Currently MinPAP == PAP == AMBER because every sub-service (Quad9 DoH + Cymru ASN) is AMBER.
	// This equality means runAggregateCmd's MinPAP gate is sufficient: if the command runs at all,
	// all sub-services are permitted and no per-sub-service PAP check inside Run() is needed.
	// If a future sub-service at a higher PAP level is added, PAP must be raised and a per-sub-service
	// guard (with the service's papLimit stored in the struct) should be introduced at that point.
	PAP = pap.AMBER
)

// Service aggregates DNS reconnaissance for an apex domain via Quad9 DoH.
type Service struct {
	client   *req.Client
	txt      cymrusvc.TXTResolver
	logger   *slog.Logger
	detector *detect.Detector
}

// NewService creates a new Service with the given HTTP client, the resolver for
// the Team Cymru ASN lookups, logger, and patterns. Use [CymruResolver] to pick
// that resolver.
func NewService(client *req.Client, txt cymrusvc.TXTResolver, logger *slog.Logger, patterns detect.Patterns) *Service {
	return &Service{
		client:   client,
		txt:      txt,
		logger:   logger,
		detector: detect.NewDetector(patterns),
	}
}

// CymruResolver returns the resolver for the Team Cymru ASN lookups. The system
// resolver sends plain DNS over UDP/TCP and never goes through an HTTP proxy,
// so when the DoH queries are proxied (dohProxied) the Cymru lookups go through
// DoH on client as well; otherwise system is used.
func CymruResolver(client *req.Client, dohProxied bool, system cymrusvc.TXTResolver) cymrusvc.TXTResolver {
	if dohProxied {
		return doh.NewTXTResolver(client)
	}
	return system
}

// Name returns the service identifier.
func (s *Service) Name() string { return Name }

// PAP returns the PAP activity level for the apex service (Quad9 third-party API).
func (s *Service) PAP() pap.Level { return PAP }

// MinPAP returns the minimum PAP level required to produce any useful results.
func (s *Service) MinPAP() pap.Level { return MinPAP }

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
// The caller's context ending causes an early return of the partial chain without error;
// any other failed query (including an http.Client timeout) is returned as an error.
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
			if ctx.Err() != nil {
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
//
// A failed query is one that errors while the caller's context is still live.
// A refused proxy bypass, in a DoH query or in a Cymru ASN lookup, fails the
// run, and so does every DoH query failing; when only some queries or ASN
// lookups fail the result is partial and one warning says so.
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
		// mail subdomain — outbound mail server; may carry a separate SPF record
		{"mail." + domain, dns.TypeA, "A"},
		{"mail." + domain, dns.TypeAAAA, "AAAA"},
		{"mail." + domain, dns.TypeTXT, "TXT"},
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
	// One error slot per query keeps the reported first error deterministic.
	directErrs := make([]error, len(directQueries))
	chainErrs := make([]error, len(cnameHosts))
	var wg sync.WaitGroup

	for i, q := range directQueries {
		wg.Go(func() {
			resp, err := doh.MakeDoHRequest(ctx, s.client, q.host, q.typeCode)
			if err != nil {
				// Only the caller's own context ending is a cancelled run; an
				// http.Client timeout also matches context.DeadlineExceeded
				// but is a failed query.
				if ctx.Err() == nil {
					s.logger.Debug("apex: query failed", "host", q.host, "type", q.typeName, "error", err)
					directErrs[i] = err
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
				// resolveCNAMEChain returns an error only while ctx is live.
				s.logger.Debug("apex: CNAME chain failed", "host", h, "error", err)
				chainErrs[i] = err
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

	if err := s.checkQueryFailures(domain, slices.Concat(directErrs, chainErrs)); err != nil {
		return nil, err
	}

	// Flatten direct query results in slice order (deterministic).
	for _, recs := range results {
		result.Records = append(result.Records, recs...)
	}

	// Flatten CNAME chains in slice order.
	for _, recs := range cnameResults {
		result.Records = append(result.Records, recs...)
	}

	// CDN detection from all CNAME records (chain traversals + direct queries
	// for email security subdomains such as _dmarc, _mta-sts, _domainkey, _smtp._tls).
	var allCNAMEs []string
	for _, rec := range result.Records {
		if rec.Type == "CNAME" {
			allCNAMEs = append(allCNAMEs, rec.Value)
		}
	}
	if len(allCNAMEs) > 0 {
		for _, d := range s.detector.CDN(allCNAMEs) {
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
		for _, d := range s.detector.EmailProvider(mxHosts) {
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
		for _, d := range s.detector.DNSHost(nsHosts) {
			result.Records = append(result.Records, Record{
				Host:  "detected",
				Type:  string(d.Type),
				Value: d.Provider + " (ns: " + d.Evidence + ")",
			})
		}
	}

	// TXT record detection from all TXT records (apex + subdomains such as _dmarc, _mta-sts, etc.).
	var allTXTs []string
	for _, rec := range result.Records {
		if rec.Type == "TXT" {
			allTXTs = append(allTXTs, rec.Value)
		}
	}
	if len(allTXTs) > 0 {
		for _, d := range s.detector.TXTRecord(allTXTs) {
			result.Records = append(result.Records, Record{
				Host:  "detected",
				Type:  string(d.Type),
				Value: d.Provider + " (txt: " + d.Evidence + ")",
			})
		}
	}

	// ASN lookup for all unique IPs discovered via A and AAAA records.
	ipSet := map[string]bool{}
	for _, rec := range result.Records {
		if rec.Type == "A" || rec.Type == "AAAA" {
			ipSet[rec.Value] = true
		}
	}
	if len(ipSet) > 0 {
		ips := make([]string, 0, len(ipSet))
		for ip := range ipSet {
			ips = append(ips, ip)
		}
		sort.Strings(ips)
		// The Cymru service logs a failed lookup at Debug and returns an empty
		// result, so the failures are counted on the resolver underneath it.
		counter := &countingResolver{inner: s.txt}
		cymruSvc := cymrusvc.NewService(counter, s.logger)
		asnResults := make([]*cymrusvc.Result, len(ips))
		var asnWg sync.WaitGroup
		for i, ip := range ips {
			asnWg.Go(func() {
				raw, err := cymruSvc.Run(ctx, ip)
				if err != nil {
					s.logger.Debug("apex: ASN lookup failed", "ip", ip, "error", err)
					return
				}
				if cr, ok := raw.(*cymrusvc.Result); ok && !cr.IsEmpty() {
					asnResults[i] = cr
				}
			})
		}
		asnWg.Wait()
		// The Cymru service swallows lookup errors, so a refused bypass is only
		// visible on the resolver underneath it. It fails the run, like the DoH
		// queries, instead of ending up in the partial-failure warning.
		if err := counter.bypassErr(); err != nil {
			return nil, err
		}
		if failed, total, firstErr := counter.failures(); failed > 0 {
			s.logger.Warn("apex: some ASN lookups failed; ASN data is incomplete",
				"domain", domain, "failed", failed, "total", total, "error", firstErr)
		}
		seenASN := map[string]bool{}
		for _, cr := range asnResults {
			if cr == nil {
				continue
			}
			value := fmt.Sprintf("%s / %s / %s / %s / %s",
				cr.ASN, cr.Prefix, cr.Country, cr.Registry, cr.Description)
			if seenASN[value] {
				continue
			}
			seenASN[value] = true
			result.Records = append(result.Records, Record{
				Host:  "detected",
				Type:  "ASN",
				Value: value,
			})
		}
	}

	return result, nil
}

// checkQueryFailures applies the failure policy to the DoH query outcomes (one
// entry per query, nil for a query that succeeded or was cut short by the
// caller's context). A refused proxy bypass is returned as is. When every query
// failed the run is an [services.ErrRequestFailed]; when only some did, one
// warning reports that the result is incomplete.
func (s *Service) checkQueryFailures(domain string, errs []error) error {
	var failed int
	var firstErr error
	for _, err := range errs {
		if err == nil {
			continue
		}
		if errors.Is(err, apperr.ErrProxyBypass) {
			return err
		}
		failed++
		if firstErr == nil {
			firstErr = err
		}
	}
	switch failed {
	case 0:
		return nil
	case len(errs):
		return fmt.Errorf("%w: apex: all %d DoH queries for %q failed: %v", services.ErrRequestFailed, len(errs), domain, firstErr)
	}
	s.logger.Warn("apex: some queries failed; results are incomplete",
		"domain", domain, "failed", failed, "total", len(errs), "error", firstErr)
	return nil
}

// countingResolver wraps a Cymru TXT resolver and counts the lookups that
// failed. A failure is an error while the caller's context is still live that
// is not the resolver's "no such record" answer. It also remembers the first
// lookup that was refused as a proxy bypass.
type countingResolver struct {
	inner cymrusvc.TXTResolver

	mu       sync.Mutex
	total    int
	failed   int
	firstErr error
	bypass   error
}

// LookupTXT implements [cymrusvc.TXTResolver].
func (c *countingResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	txts, err := c.inner.LookupTXT(ctx, name)
	failed := isLookupFailure(ctx, err)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.total++
	if c.bypass == nil && errors.Is(err, apperr.ErrProxyBypass) {
		c.bypass = err
	}
	if failed {
		c.failed++
		if c.firstErr == nil {
			c.firstErr = err
		}
	}
	return txts, err
}

// failures returns the number of failed lookups, the number of lookups made and
// the first failure.
func (c *countingResolver) failures() (failed, total int, firstErr error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failed, c.total, c.firstErr
}

// bypassErr returns the first lookup error that is an [apperr.ErrProxyBypass],
// or nil when no lookup was refused as a proxy bypass.
func (c *countingResolver) bypassErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bypass
}

// isLookupFailure reports whether err from a TXT lookup made under ctx is a
// real failure. Not found is an answer (the name has no record), and an error
// after the caller's context ended is a cancelled run.
func isLookupFailure(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return false
	}
	return true
}
