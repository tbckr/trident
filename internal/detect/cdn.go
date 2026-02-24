package detect

import "strings"

var cdnPatterns = []pattern{
	{"cloudfront.net", "AWS CloudFront"},
	{"akamaiedge.net", "Akamai"},
	{"edgekey.net", "Akamai"},
	{"edgesuite.net", "Akamai"},
	{"fastly.net", "Fastly"},
	{"cloudflare.net", "Cloudflare"},
	{"azureedge.net", "Azure CDN"},
	{"azurefd.net", "Azure Front Door"},
	{"googleplex.com", "Google Cloud CDN"},
	{"l.google.com", "Google Cloud CDN"},
	{"b-cdn.net", "Bunny CDN"},
	{"incapdns.net", "Imperva"},
	{"sucuri.net", "Sucuri"},
	{"stackpathcdn.com", "StackPath"},
	{"netdna-cdn.com", "StackPath"},
	{"llnwd.net", "Edgio"},
	{"edgio.net", "Edgio"},
	{"cdn77.org", "CDN77"},
	{"kxcdn.com", "KeyCDN"},
	{"edgecastcdn.net", "Verizon EdgeCast"},
	{"cachefly.net", "CacheFly"},
	{"gcdn.co", "G-Core"},
	{"alikunlun.com", "Alibaba Cloud CDN"},
}

// CDN matches CNAME targets against known CDN provider patterns and returns
// one Detection per unique (provider, evidence) pair.
func CDN(cnames []string) []Detection {
	var detections []Detection
	seen := map[string]bool{}
	for _, cname := range cnames {
		target := strings.TrimSuffix(cname, ".")
		for _, p := range cdnPatterns {
			if matchSuffix(target, p.suffix) {
				key := p.provider + ":" + cname
				if seen[key] {
					continue
				}
				seen[key] = true
				detections = append(detections, Detection{
					Type:     TypeCDN,
					Provider: p.provider,
					Evidence: cname,
					Source:   "cname",
				})
			}
		}
	}
	return detections
}
