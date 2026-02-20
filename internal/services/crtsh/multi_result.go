package crtsh

import (
	"io"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// MultiResult holds crt.sh subdomain results for multiple inputs.
type MultiResult struct {
	services.MultiResultBase[Result, *Result]
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
