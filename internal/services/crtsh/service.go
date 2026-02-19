// Package crtsh queries the crt.sh certificate transparency log for subdomain enumeration.
package crtsh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/output"
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

// Result holds the unique subdomains found in the crt.sh certificate log.
type Result struct {
	Input      string   `json:"input"`
	Subdomains []string `json:"subdomains,omitempty"`
}

// WriteText renders the result as an ASCII table.
func (r *Result) WriteText(w io.Writer) error {
	var rows [][]string
	for _, sub := range r.Subdomains {
		rows = append(rows, []string{sub})
	}
	table := output.NewWrappingTable(w, 30, 6)
	table.Header([]string{"Subdomain"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}

// Run queries crt.sh for subdomains of the given domain.
func (s *Service) Run(ctx context.Context, domain string) (any, error) {
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
		s.logger.Debug("crt.sh request failed", "domain", domain, "error", err)
		return result, nil
	}
	if !resp.IsSuccessState() {
		s.logger.Debug("crt.sh non-success response", "domain", domain, "status", resp.StatusCode)
		return result, nil
	}

	seen := make(map[string]struct{})
	for _, entry := range entries {
		for _, name := range []string{entry.CommonName, entry.NameValue} {
			for sub := range strings.SplitSeq(name, "\n") {
				sub = output.StripANSI(strings.TrimSpace(sub))
				if sub == "" {
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
