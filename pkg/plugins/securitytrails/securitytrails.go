package securitytrails

import (
	"context"
	"fmt"
	"github.com/imroc/req/v3"
)

const (
	source = "securitytrails"

	baseURL = "https://api.securitytrails.com/v1"
)

type SecurityTrailsClient struct {
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

func NewSecurityTrailsClient(reqClient *req.Client, apiKey string) *SecurityTrailsClient {
	c := reqClient.Clone().
		SetBaseURL(baseURL).
		SetCommonContentType("application/json").
		SetCommonHeader("APIKEY", apiKey)
	return &SecurityTrailsClient{
		Client: c,
	}
}

func (c *SecurityTrailsClient) Ping() (bool, error) {
	resp, err := c.R().
		Get("/ping")
	if err != nil {
		return false, err
	}
	return resp.StatusCode == 200, nil
}

func (c *SecurityTrailsClient) Subdomains(ctx context.Context, domain string, subdomainsOnly, includeInactive bool) (resp SubdomainResponse, err error) {
	var r *req.Response
	r, err = c.R().
		SetContext(ctx).
		SetPathParam("hostname", domain).
		SetQueryParam("children_only", fmt.Sprintf("%t", subdomainsOnly)).
		SetQueryParam("include_inactive", fmt.Sprintf("%t", includeInactive)).
		SetSuccessResult(&resp).
		Get("/domain/{hostname}/subdomains")
	if err != nil {
		return
	}
	defer r.Body.Close()
	return
}

func (c *SecurityTrailsClient) DomainDetails(ctx context.Context, domain string) (resp DomainDetailsResponse, err error) {
	var r *req.Response
	r, err = c.R().
		SetContext(ctx).
		SetPathParam("hostname", domain).
		SetSuccessResult(&resp).
		Get("/domain/{hostname}")
	if err != nil {
		return
	}
	defer r.Body.Close()
	return
}
