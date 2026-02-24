package detect

import (
	"io"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// MultiResult holds detection results for multiple domains.
type MultiResult struct {
	services.MultiResultBase[Result, *Result]
}

// WriteTable renders all results in a combined table grouped by domain.
// Columns: Domain / Type / Provider / Evidence.
func (m *MultiResult) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		for _, d := range sortDetections(r.Detections) {
			rows = append(rows, []string{r.Input, d.Type, d.Provider, d.Source + ": " + d.Evidence})
		}
	}
	table := output.NewGroupedWrappingTable(w, 20, 40)
	table.Header([]string{"Domain", "Type", "Provider", "Evidence"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
