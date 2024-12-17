package securitytrails

import (
	"context"
	"fmt"

	"github.com/tbckr/trident/pkg/opsec"
	plugin "github.com/tbckr/trident/pkg/plugins/securitytrails"
	"github.com/tbckr/trident/pkg/report"
)

type SecuritytrailsDescriber struct {
	client        *plugin.SecurityTrailsClient
	escapeDomains bool
}

func NewSecuritytrailsDescriber(client *plugin.SecurityTrailsClient, escapeDomains bool) *SecuritytrailsDescriber {
	// TODO only pass in, if the domains have to be bracketed
	return &SecuritytrailsDescriber{
		client:        client,
		escapeDomains: escapeDomains,
	}
}

func (s *SecuritytrailsDescriber) DescribeDomain(ctx context.Context, domain string) (report.DomainDescriptionReport, error) {
	// TODO move this into the cli

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
	hostReport := GenerateDomainReport(domainResponse, s.escapeDomains)

	// If the domain is a subdomain, we also need to get the apex domain details
	if domainResponse.ApexDomain != "" && domainResponse.ApexDomain != domain {
		domainDescriptionReport = report.DomainDescriptionReport{
			HostDomain: hostReport,
		}

		apexDomainResponse, err = s.client.GetDomainDetails(ctx, domainResponse.ApexDomain)
		if err != nil {
			return report.DomainDescriptionReport{}, err
		}
		apexReport := GenerateDomainReport(apexDomainResponse, s.escapeDomains)
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

func GenerateDomainReport(resp plugin.DomainDetailsResponse, escapeDomains bool) report.DomainReport {
	records := report.RecordReport{
		A:    make([]report.ARecord, len(resp.CurrentDNS.A.Values)),
		AAAA: make([]report.AaaaRecord, len(resp.CurrentDNS.AAAA.Values)),
		MX:   make([]report.MxRecord, len(resp.CurrentDNS.MX.Values)),
		NS:   make([]report.NsRecord, len(resp.CurrentDNS.NS.Values)),
		SOA:  make([]report.SoaRecord, len(resp.CurrentDNS.SOA.Values)),
		TXT:  make([]report.TxtRecord, len(resp.CurrentDNS.TXT.Values)),
	}

	aFirstSeen := resp.CurrentDNS.A.FirstSeen
	for i, val := range resp.CurrentDNS.A.Values {
		ip := val["ip"].(string)
		if escapeDomains {
			ip = opsec.BracketDomain(ip)
		}

		records.A[i].Type = "A"
		records.A[i].FirstSeen = aFirstSeen
		records.A[i].IP = ip
		records.A[i].Organization = val["ip_organization"].(string)
	}

	aaaaFirstSeen := resp.CurrentDNS.AAAA.FirstSeen
	for i, val := range resp.CurrentDNS.AAAA.Values {
		// TODO escape v6 addresses
		records.AAAA[i].Type = "AAAA"
		records.AAAA[i].FirstSeen = aaaaFirstSeen
		records.AAAA[i].IP = val["ip"].(string)
		records.AAAA[i].Organization = val["ip_organization"].(string)
	}

	mxFirstSeen := resp.CurrentDNS.MX.FirstSeen
	for i, val := range resp.CurrentDNS.MX.Values {
		records.MX[i].Type = "MX"
		records.MX[i].FirstSeen = mxFirstSeen
		records.MX[i].Hostname = val["hostname"].(string)
		records.MX[i].Priority = int(val["priority"].(float64))
		records.MX[i].Organization = val["hostname_organization"].(string)
	}

	nsFirstSeen := resp.CurrentDNS.NS.FirstSeen
	for i, val := range resp.CurrentDNS.NS.Values {
		records.NS[i].Type = "NS"
		records.NS[i].FirstSeen = nsFirstSeen
		records.NS[i].Nameserver = val["nameserver"].(string)
		records.NS[i].Organization = val["nameserver_organization"].(string)
	}

	soaFirstSeen := resp.CurrentDNS.SOA.FirstSeen
	for i, val := range resp.CurrentDNS.SOA.Values {
		records.SOA[i].Type = "SOA"
		records.SOA[i].FirstSeen = soaFirstSeen
		records.SOA[i].Email = val["email"].(string)
		records.SOA[i].Ttl = int(val["ttl"].(float64))
	}

	txtFirstSeen := resp.CurrentDNS.TXT.FirstSeen
	for i, val := range resp.CurrentDNS.TXT.Values {
		records.TXT[i].Type = "TXT"
		records.TXT[i].FirstSeen = txtFirstSeen
		records.TXT[i].Text = val["value"].(string)
	}

	r := report.DomainReport{
		Hostname:     resp.Hostname,
		ApexDomain:   resp.ApexDomain,
		AlexaRank:    resp.AlexaRank,
		RecordReport: records,
	}

	return r
}
