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

package crtsh

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/imroc/req/v3"
	"github.com/tbckr/trident/pkg/client"
	"github.com/tbckr/trident/pkg/utils"
)

const source = "crt.sh"

type CrtShClient struct {
	*req.Client
}

type CrtShDomainResult struct {
	Name string `json:"name_value"`
}

func NewCrtShClient(reqClient *req.Client) *CrtShClient {
	return &CrtShClient{
		Client: reqClient,
	}
}

func (c *CrtShClient) FetchDomains(ctx context.Context, domain string) ([]string, error) {
	// Params
	outputType := "json"
	fetchURL := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=%s", domain, outputType)

	// Log
	slog.Debug("Fetching domains",
		"source", source,
		"domain", domain,
		slog.Group("params",
			"url", fetchURL,
			"output", outputType,
		),
	)

	// Request
	var results []CrtShDomainResult
	resp, err := c.R().
		SetContext(ctx).
		SetSuccessResult(&results).
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
	for _, res := range results {
		split := strings.Split(res.Name, "\n")
		for _, elem := range split {
			output = append(output, utils.CleanDomain(elem))
		}
	}
	slog.Debug("Retrieved domains",
		"source", source,
		"domain", domain,
		"count", len(output),
	)
	return output, nil
}
