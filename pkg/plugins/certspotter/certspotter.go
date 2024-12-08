package certspotter

import (
	"context"
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/tbckr/trident/pkg/client"
	"github.com/tbckr/trident/pkg/utils"
	"log/slog"
)

const source = "certspotter"

type CertspotterClient struct {
	*req.Client
}

type CertspotterDomainResult struct {
	DNSNames []string `json:"dns_names"`
}

func NewCertspotterClient(reqClient *req.Client) *CertspotterClient {
	return &CertspotterClient{
		Client: reqClient,
	}
}

func (c *CertspotterClient) FetchDomains(ctx context.Context, domain string) ([]string, error) {
	// Params
	fetchURL := fmt.Sprintf("https://api.certspotter.com/v1/issuances?domain=%s&expand=dns_names", domain)
	// Log
	slog.Debug("Fetching domains",
		"source", source,
		"domain", domain,
		slog.Group("params",
			"url", fetchURL,
		),
	)

	// Request
	var results []CertspotterDomainResult
	resp, err := c.R().
		SetSuccessResult(&results).
		SetContext(ctx).
		Get(fetchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check response
	if !resp.IsSuccessState() {
		return []string{}, client.NewHTTPError(source, resp.StatusCode)
	}

	// Extract domains
	var output []string
	for _, w := range results {
		for _, d := range w.DNSNames {
			output = append(output, utils.CleanDomain(d))
		}
	}
	slog.Debug("Retrieved domains",
		"source", source,
		"domain", domain,
		"count", len(output),
	)
	return output, nil
}
