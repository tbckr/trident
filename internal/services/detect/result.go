package detect

import (
	"fmt"
	"io"
	"sort"

	"github.com/tbckr/trident/internal/output"
)

// Detection holds a single provider detection result.
type Detection struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Evidence string `json:"evidence"`
	Source   string `json:"source"`
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

// WriteText renders detections as plain text, one per line.
// Format: "TYPE Provider (source: evidence)"
func (r *Result) WriteText(w io.Writer) error {
	for _, d := range r.Detections {
		if _, err := fmt.Fprintf(w, "%s %s (%s: %s)\n", d.Type, d.Provider, d.Source, d.Evidence); err != nil {
			return err
		}
	}
	return nil
}

// sortDetections returns a copy of detections sorted by (Type, Source, Provider).
// This groups same-type rows consecutively so MergeHierarchical works correctly.
func sortDetections(detections []Detection) []Detection {
	sorted := make([]Detection, len(detections))
	copy(sorted, detections)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Provider < b.Provider
	})
	return sorted
}

// WriteTable renders detections as a grouped ASCII table with columns Type, Provider, Evidence.
func (r *Result) WriteTable(w io.Writer) error {
	sorted := sortDetections(r.Detections)
	var rows [][]string
	for _, d := range sorted {
		rows = append(rows, []string{d.Type, d.Provider, d.Source + ": " + d.Evidence})
	}
	table := output.NewGroupedWrappingTable(w, 20, 30)
	table.Header([]string{"Type", "Provider", "Evidence"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
