// Copyright (c) 2023 Tim <tbckr>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// SPDX-License-Identifier: MIT

package certspotter

import (
	"fmt"
	"log/slog"

	"github.com/tbckr/trident/pkg/api"
	"github.com/tbckr/trident/pkg/domainutils"
)

const source = "certspotter"

type certspotterResult struct {
	DNSNames []string `json:"dns_names"`
}

func FetchDomains(domain string) ([]string, error) {
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
	var results []certspotterResult
	resp, err := api.Request().
		SetSuccessResult(&results).
		Get(fetchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check response
	if !resp.IsSuccessState() {
		return []string{}, api.NewHTTPError(source, resp.StatusCode)
	}

	// Extract domains
	var output []string
	for _, w := range results {
		for _, d := range w.DNSNames {
			output = append(output, domainutils.Clean(d))
		}
	}
	slog.Debug("Retrieved domains",
		"source", source,
		"domain", domain,
		"count", len(output),
	)
	return output, nil
}
