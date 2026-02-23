package quad9

import (
	"io"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// MultiResult holds Quad9 threat-intelligence verdicts for multiple inputs.
type MultiResult struct {
	services.MultiResultBase[Result, *Result]
}

// WriteTable renders all verdicts in a single combined table.
// Columns: Domain / Blocked.
func (m *MultiResult) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		blocked := "false"
		if r.Blocked {
			blocked = "true"
		}
		rows = append(rows, []string{r.Input, blocked})
	}
	table := output.NewWrappingTable(w, 30, 6)
	table.Header([]string{"Domain", "Blocked"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
