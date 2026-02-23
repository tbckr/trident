package asn

import (
	"io"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// MultiResult holds ASN lookup results for multiple inputs.
type MultiResult struct {
	services.MultiResultBase[Result, *Result]
}

// WriteTable renders all results in a single combined table grouped by input.
// Columns: Input / Field / Value. Input cells are merged hierarchically.
func (m *MultiResult) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		rows = append(rows,
			[]string{r.Input, "ASN", r.ASN},
			[]string{r.Input, "Prefix", r.Prefix},
			[]string{r.Input, "Country", r.Country},
			[]string{r.Input, "Registry", r.Registry},
			[]string{r.Input, "Description", r.Description},
		)
	}
	table := output.NewGroupedWrappingTable(w, 20, 30)
	table.Header([]string{"Input", "Field", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
