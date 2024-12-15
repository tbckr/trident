package securitytrails

import (
	"context"
	"fmt"
	"github.com/tbckr/trident/pkg/opsec"

	plugin "github.com/tbckr/trident/pkg/plugins/securitytrails"
	"github.com/tbckr/trident/pkg/report"
	reporter "github.com/tbckr/trident/pkg/report/securitytrails"
)

type SecuritytrailsStrategy struct {
	client        *plugin.SecurityTrailsClient
	escapeDomains bool
}

func NewSecuritytrailsStrategy(client *plugin.SecurityTrailsClient, escapeDomains bool) *SecuritytrailsStrategy {
	// TODO only pass in, if the domains have to be bracketed
	return &SecuritytrailsStrategy{
		client:        client,
		escapeDomains: escapeDomains,
	}
}

func (s *SecuritytrailsStrategy) DescribeDomain(ctx context.Context, domain string) (report.DomainDescriptionReport, error) {
	var (
		err                                error
		domainDescriptionReport            report.DomainDescriptionReport
		subdomains                         []string
		domainResponse, apexDomainResponse plugin.DomainDetailsResponse
		subdomainResp                      plugin.SubdomainResponse
	)

	// Get the domain details
	domainResponse, err = s.client.GetDomainDetails(ctx, domain)
	if err != nil {
		return report.DomainDescriptionReport{}, err
	}
	hostReport := reporter.GenerateDomainReport(domainResponse, s.escapeDomains)

	// If the domain is a subdomain, we also need to get the apex domain details
	if domainResponse.ApexDomain != "" && domainResponse.ApexDomain != domain {
		domainDescriptionReport = report.DomainDescriptionReport{
			HostDomain: hostReport,
		}

		apexDomainResponse, err = s.client.GetDomainDetails(ctx, domainResponse.ApexDomain)
		if err != nil {
			return report.DomainDescriptionReport{}, err
		}
		apexReport := reporter.GenerateDomainReport(apexDomainResponse, s.escapeDomains)
		domainDescriptionReport.ApexDomain = apexReport
	} else {
		domainDescriptionReport = report.DomainDescriptionReport{
			ApexDomain: hostReport,
		}
	}

	// Get subdomains
	subdomainResp, err = s.client.GetSubdomains(ctx, domain, true, true)
	if err != nil {
		return report.DomainDescriptionReport{}, err
	}
	subdomains = make([]string, subdomainResp.SubdomainCount)
	for i, d := range subdomainResp.Subdomains {
		d = fmt.Sprintf("%s.%s", d, domain)
		if s.escapeDomains {
			d = opsec.BracketDomain(d)
		}
		subdomains[i] = d
	}
	domainDescriptionReport.Subdomains = subdomains

	return domainDescriptionReport, nil
}
