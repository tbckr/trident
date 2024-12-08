package hackertarget

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/tbckr/trident/pkg/client"
	"github.com/tbckr/trident/pkg/utils"
	"log/slog"
	"strings"
)

const source = "hackertarget"

type HackerTargetClient struct {
	*req.Client
}

func NewHackerTargetClient(reqClient *req.Client) *HackerTargetClient {
	return &HackerTargetClient{
		Client: reqClient,
	}
}

func (c *HackerTargetClient) FetchDomains(ctx context.Context, domain string) ([]string, error) {
	// Params
	fetchURL := fmt.Sprintf("https://api.hackertarget.com/hostsearch/?q=%s", domain)
	// Log
	slog.Debug("Fetching domains",
		"source", source,
		"domain", domain,
		slog.Group("params",
			"url", fetchURL,
		),
	)

	// Request
	resp, err := c.R().
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
	sc := bufio.NewScanner(bytes.NewReader(resp.Bytes()))
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ",", 2)
		if len(parts) != 2 {
			continue
		}
		output = append(output, utils.CleanDomain(parts[0]))
	}
	slog.Debug("Retrieved domains",
		"source", source,
		"domain", domain,
		"count", len(output),
	)
	return output, nil
}
