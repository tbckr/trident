package detect

import "strings"

// DNSHost matches NS server hostnames against known DNS hosting provider
// patterns and returns one Detection per unique (provider, evidence) pair.
func (d *Detector) DNSHost(nsHosts []string) []Detection {
	var detections []Detection
	seen := map[string]bool{}
	for _, host := range nsHosts {
		provider := ""
		for _, p := range d.patterns.DNS {
			if p.Contains != "" {
				if strings.Contains(strings.TrimSuffix(host, "."), p.Contains) {
					provider = p.Provider
					break
				}
			} else if matchSuffix(host, p.Suffix) {
				provider = p.Provider
				break
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
			Source:   "ns",
		})
	}
	return detections
}
