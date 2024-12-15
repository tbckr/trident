package securitytrails

import (
	"github.com/tbckr/trident/pkg/opsec"
	plugin "github.com/tbckr/trident/pkg/plugins/securitytrails"
	"github.com/tbckr/trident/pkg/report"
)

func GenerateDomainReport(resp plugin.DomainDetailsResponse, escapeDomains bool) report.DomainReport {
	records := report.RecordReport{
		A: make([]report.ARecord, len(resp.CurrentDNS.A.Values)),
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

	r := report.DomainReport{
		Hostname:     resp.Hostname,
		ApexDomain:   resp.ApexDomain,
		AlexaRank:    resp.AlexaRank,
		RecordReport: records,
	}

	return r
}
