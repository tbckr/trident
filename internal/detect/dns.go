package detect

import "strings"

var dnsPatterns = []pattern{
	{"ns.cloudflare.com", "Cloudflare DNS"},
	{"azure-dns.com", "Azure DNS"},
	{"azure-dns.net", "Azure DNS"},
	{"azure-dns.org", "Azure DNS"},
	{"azure-dns.info", "Azure DNS"},
	{"googledomains.com", "Google Cloud DNS"},
	{"nsone.net", "NS1"},
	{"dnsimple.com", "DNSimple"},
	{"ultradns.net", "UltraDNS"},
	{"ultradns.com", "UltraDNS"},
	{"dnsmadeeasy.com", "DNS Made Easy"},
	{"cloudns.net", "ClouDNS"},
	{"domaincontrol.com", "GoDaddy"},
	{"registrar-servers.com", "Namecheap"},
}

// DNSHost matches NS server hostnames against known DNS hosting provider
// patterns and returns one Detection per unique (provider, evidence) pair.
func DNSHost(nsHosts []string) []Detection {
	var detections []Detection
	seen := map[string]bool{}
	for _, host := range nsHosts {
		provider := ""
		// AWS Route 53 uses a contains match (e.g. ns-123.awsdns-45.com).
		if strings.Contains(strings.TrimSuffix(host, "."), "awsdns") {
			provider = "AWS Route 53"
		} else {
			for _, p := range dnsPatterns {
				if matchSuffix(host, p.suffix) {
					provider = p.provider
					break
				}
			}
		}
		if provider == "" {
			continue
		}
		key := provider + ":" + host
		if seen[key] {
			continue
		}
		seen[key] = true
		detections = append(detections, Detection{
			Type:     TypeDNS,
			Provider: provider,
			Evidence: host,
		})
	}
	return detections
}
