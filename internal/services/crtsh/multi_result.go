package crtsh

import (
	"encoding/json"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// MultiResult holds crt.sh subdomain results for multiple inputs.
type MultiResult struct {
	Results []*Result
}

// IsEmpty reports whether all contained results are empty.
func (m *MultiResult) IsEmpty() bool {
	for _, r := range m.Results {
		if !r.IsEmpty() {
			return false
		}
	}
	return true
}

// MarshalJSON serializes the multi-result as a JSON array of individual results.
func (m *MultiResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Results)
}

// WritePlain writes all results as plain text (one subdomain per line).
func (m *MultiResult) WritePlain(w io.Writer) error {
	for _, r := range m.Results {
		if err := r.WritePlain(w); err != nil {
			return err
		}
	}
	return nil
}

// WriteText renders all results in a single combined table grouped by domain.
// Columns: Domain / Subdomain. Domain cells are merged hierarchically.
func (m *MultiResult) WriteText(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		for _, sub := range r.Subdomains {
			rows = append(rows, []string{r.Input, sub})
		}
	}
	table := output.NewGroupedWrappingTable(w, 30, 20)
	table.Header([]string{"Domain", "Subdomain"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
