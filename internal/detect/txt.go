package detect

import "strings"

// TXTRecord matches TXT record values against known provider patterns.
// Each txt value is checked against all patterns using substring matching.
// Duplicate detections (same provider+txt) are suppressed.
func (d *Detector) TXTRecord(txts []string) []Detection {
	seen := map[string]bool{}
	var detections []Detection
	for _, txt := range txts {
		for _, p := range d.patterns.TXT {
			if !strings.Contains(txt, p.Substring) {
				continue
			}
			key := p.Provider + ":" + txt
			if seen[key] {
				continue
			}
			seen[key] = true
			detections = append(detections, Detection{
				Type:     p.Type,
				Provider: p.Provider,
				Evidence: txt,
				Source:   "txt",
			})
		}
	}
	return detections
}
