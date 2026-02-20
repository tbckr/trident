package crtsh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/validate"
)

// crtshBaseURL is the crt.sh API endpoint base for certificate transparency log searches.
// The query uses the `%.domain` wildcard form to find all subdomains.
const crtshBaseURL = "https://crt.sh/?q=%%.%s&output=json"

// crtshEntry represents a single record returned by the crt.sh JSON API.
type crtshEntry struct {
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"`
}

// Service queries the crt.sh certificate transparency log API.
type Service struct {
	client *req.Client
	logger *slog.Logger
}

// NewService creates a new crt.sh service with the given HTTP client and logger.
func NewService(client *req.Client, logger *slog.Logger) *Service {
	return &Service{client: client, logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "crtsh" }

// PAP returns the PAP activity level for the crt.sh service (external API query).
func (s *Service) PAP() pap.Level { return pap.AMBER }

// AggregateResults combines multiple crt.sh results into a MultiResult.
func (s *Service) AggregateResults(results []services.Result) services.Result {
	mr := &MultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*Result))
	}
	return mr
}

// Run queries crt.sh for subdomains of the given domain.
func (s *Service) Run(ctx context.Context, domain string) (services.Result, error) {
	domain = output.StripANSI(domain)
	if !validate.IsDomain(domain) {
		return nil, fmt.Errorf("%w: must be a valid domain name: %q", services.ErrInvalidInput, domain)
	}

	result := &Result{Input: domain}

	url := fmt.Sprintf(crtshBaseURL, domain)
	var entries []crtshEntry
	resp, err := s.client.R().
		SetContext(ctx).
		SetSuccessResult(&entries).
		Get(url)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return result, nil
		}
		return nil, fmt.Errorf("%w: crt.sh request error for %q: %w", services.ErrRequestFailed, domain, err)
	}
	if !resp.IsSuccessState() {
		body := resp.String()
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return nil, fmt.Errorf("%w: crt.sh returned HTTP %d for %q: %q", services.ErrRequestFailed, resp.StatusCode, domain, body)
	}

	seen := make(map[string]struct{})
	for _, entry := range entries {
		for _, name := range []string{entry.CommonName, entry.NameValue} {
			for sub := range strings.SplitSeq(name, "\n") {
				sub = output.StripANSI(strings.TrimSpace(sub))
				if sub == "" {
					continue
				}
				if !s.isValidSubdomain(sub, domain) {
					continue
				}
				if _, ok := seen[sub]; !ok {
					seen[sub] = struct{}{}
					result.Subdomains = append(result.Subdomains, sub)
				}
			}
		}
	}
	sort.Strings(result.Subdomains)
	return result, nil
}

func (s *Service) isValidSubdomain(sub, domain string) bool {
	if strings.HasPrefix(sub, "*") {
		s.logger.Debug("crt.sh: skipping wildcard", "sub", sub, "domain", domain)
		return false
	}
	if sub == domain {
		s.logger.Debug("crt.sh: skipping root domain", "sub", sub, "domain", domain)
		return false
	}
	if !strings.HasSuffix(sub, "."+domain) {
		s.logger.Debug("crt.sh: skipping foreign domain", "sub", sub, "domain", domain)
		return false
	}
	if !validate.IsDomain(sub) {
		s.logger.Debug("crt.sh: skipping invalid format", "sub", sub, "domain", domain)
		return false
	}
	return true
}
