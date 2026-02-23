package detect

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// Detection holds a single provider detection result.
type Detection struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Evidence string `json:"evidence"`
}

// Result holds provider detections for a single domain.
type Result struct {
	Input      string      `json:"input"`
	Detections []Detection `json:"detections,omitempty"`
}

// IsEmpty reports whether no providers were detected.
func (r *Result) IsEmpty() bool {
	return len(r.Detections) == 0
}

// evidenceLabel returns the DNS record type label for a detection type.
func evidenceLabel(detType string) string {
	switch detType {
	case "CDN":
		return "cname"
	case "Email":
		return "mx"
	default:
		return "ns"
	}
}

// WriteText renders detections as plain text, one per line.
// Format: "TYPE Provider (label: evidence)"
func (r *Result) WriteText(w io.Writer) error {
	for _, d := range r.Detections {
		label := evidenceLabel(d.Type)
		if _, err := fmt.Fprintf(w, "%s %s (%s: %s)\n", d.Type, d.Provider, label, d.Evidence); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable renders detections as a grouped ASCII table with columns Type, Provider, Evidence.
func (r *Result) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, d := range r.Detections {
		rows = append(rows, []string{d.Type, d.Provider, d.Evidence})
	}
	table := output.NewGroupedWrappingTable(w, 20, 30)
	table.Header([]string{"Type", "Provider", "Evidence"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
