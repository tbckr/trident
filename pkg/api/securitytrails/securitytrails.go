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

package securitytrails

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/imroc/req/v3"
)

const (
	source = "securitytrails"

	userAgent = "secscan (https://github.com/tbckr/secscan)"

	baseURL        = "https://api.securitytrails.com/v1"
	requestTimeout = 5 * time.Second
)

type Client struct {
	*req.Client
}

type MetaResponse struct {
	LimitReached bool `json:"limit_reached"`
}

type SubdomainResponse struct {
	Endpoint       string       `json:"endpoint"`
	Meta           MetaResponse `json:"meta"`
	SubdomainCount int          `json:"subdomain_count"`
	Subdomains     []string     `json:"subdomains"`
}

type DomainDetailsResponse struct {
	AlexaRank      int         `json:"alexa_rank"`
	ApexDomain     string      `json:"apex_domain"`
	CurrentDNS     DNSResponse `json:"current_dns"`
	Endpoint       string      `json:"endpoint"`
	Hostname       string      `json:"hostname"`
	SubdomainCount int         `json:"subdomain_count"`
}

type DNSResponse struct {
	A    DNSRecordResponse `json:"a"`
	AAAA DNSRecordResponse `json:"aaaa"`
	MX   DNSRecordResponse `json:"mx"`
	NS   DNSRecordResponse `json:"ns"`
	SOA  DNSRecordResponse `json:"soa"`
	TXT  DNSRecordResponse `json:"txt"`
}

type DNSRecordResponse struct {
	FirstSeen string                   `json:"first_seen"`
	Values    []map[string]interface{} `json:"values"`
}

func NewClient(apiKey string) *Client {
	c := req.C().
		SetUserAgent(userAgent).
		SetTimeout(requestTimeout).
		SetCommonRetryCount(2).
		SetBaseURL(baseURL).
		SetCommonContentType("application/json").
		SetCommonHeader("APIKEY", apiKey).
		OnAfterResponse(func(client *req.Client, resp *req.Response) error {
			// There is an underlying error, e.g. network error or unmarshal error.
			if resp.Err != nil {
				return nil
			}
			// Neither a success response nor a error response, record details to help troubleshooting
			if !resp.IsSuccessState() {
				resp.Err = fmt.Errorf("bad status: %s\nraw content:\n%s", resp.Status, resp.Dump())
			}
			return nil
		})
	return &Client{
		Client: c,
	}
}

func (c *Client) Ping() (bool, error) {
	resp, err := c.R().
		Get("/ping")
	return resp.StatusCode == 200, err
}

// Subdomains returns a list of subdomains for a given domain.
// If subdomainsOnly is true, only subdomains are returned.
// If includeInactive is true, inactive subdomains are included.
func (c *Client) Subdomains(domain string, subdomainsOnly, includeInactive bool) (resp SubdomainResponse, err error) {
	_, err = c.R().
		SetPathParam("hostname", domain).
		SetQueryParam("children_only", fmt.Sprintf("%t", subdomainsOnly)).
		SetQueryParam("include_inactive", fmt.Sprintf("%t", includeInactive)).
		SetSuccessResult(&resp).
		Get("/domain/{hostname}/subdomains")
	return
}

func (c *Client) DomainDetails(domain string) (resp DomainDetailsResponse, err error) {
	_, err = c.R().
		SetPathParam("hostname", domain).
		SetSuccessResult(&resp).
		Get("/domain/{hostname}")
	return
}

func FetchDomains(domain string) ([]string, error) {
	slog.Debug("Fetching domains",
		"source", source,
		"domain", domain,
	)

	c := NewClient(os.Getenv("SECURITYTRAILS_API_KEY"))
	resp, err := c.Subdomains(domain, true, false)
	if err != nil {
		return nil, err
	}

	// Add the domain itself because SecurityTrails doesn't return it
	fullDomain := make([]string, 0)
	for _, subdomain := range resp.Subdomains {
		fullDomain = append(fullDomain, subdomain+"."+domain)
	}

	slog.Debug("Retrieved domains",
		"source", source,
		"domain", domain,
		"count", len(fullDomain),
	)

	return fullDomain, nil
}
