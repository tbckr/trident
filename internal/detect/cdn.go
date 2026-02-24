package detect

import "strings"

// CDN matches CNAME targets against known CDN provider patterns and returns
// one Detection per unique (provider, evidence) pair.
func (d *Detector) CDN(cnames []string) []Detection {
	var detections []Detection
	seen := map[string]bool{}
	for _, cname := range cnames {
		target := strings.TrimSuffix(cname, ".")
		for _, p := range d.patterns.CDN {
			if matchSuffix(target, p.Suffix) {
				key := p.Provider + ":" + cname
				if seen[key] {
					continue
				}
				seen[key] = true
				detections = append(detections, Detection{
					Type:     TypeCDN,
					Provider: p.Provider,
					Evidence: cname,
					Source:   "cname",
				})
			}
		}
	}
	return detections
}
