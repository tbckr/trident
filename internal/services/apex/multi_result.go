package apex

import (
	"io"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// MultiResult holds aggregated apex results for multiple inputs.
type MultiResult struct {
	services.MultiResultBase[Result, *Result]
}

// WriteTable renders all results as a 4-column table grouped by apex domain.
// Columns: Apex Domain / Host / Type / Value.
func (m *MultiResult) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		for _, rec := range sortRecordsForDisplay(r.Input, r.Records) {
			rows = append(rows, []string{r.Input, rec.Host, rec.Type, rec.Value})
		}
	}
	table := output.NewGroupedWrappingTable(w, 20, 40)
	table.Header([]string{"Apex Domain", "Host", "Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
