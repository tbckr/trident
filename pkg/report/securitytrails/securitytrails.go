package securitytrails

import (
	"github.com/tbckr/trident/pkg/opsec"
	plugin "github.com/tbckr/trident/pkg/plugins/securitytrails"
	"github.com/tbckr/trident/pkg/report"
)

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
