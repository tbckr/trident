package detect

// EmailProvider matches MX exchange hostnames against known email provider
// patterns and returns one Detection per unique (provider, evidence) pair.
func (d *Detector) EmailProvider(mxHosts []string) []Detection {
	var detections []Detection
	seen := map[string]bool{}
	for _, host := range mxHosts {
		for _, p := range d.patterns.Email {
			if matchSuffix(host, p.Suffix) {
				key := p.Provider + ":" + host
				if seen[key] {
					continue
				}
				seen[key] = true
				detections = append(detections, Detection{
					Type:     TypeEmail,
					Provider: p.Provider,
					Evidence: host,
					Source:   "mx",
				})
			}
		}
	}
	return detections
}
