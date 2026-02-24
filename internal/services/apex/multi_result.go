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
	// 4-col table: | apex domain | host | type | value |
	// Fixed structure: 13 chars (5 borders + 8 padding spaces).
	// Type column (col 2) is unconstrained — values are at most ~6 chars.
	// Apex domain (col 0) is capped — root domains are usually short.
	// Host (col 1) is capped proportionally for long subdomain names.
	// Value (col 3) receives the remaining space.
	const structureAndType = 19 // 13 fixed + 6 generous type-col budget
	termWidth := output.TerminalWidth(w)
	available := termWidth - structureAndType
	apexMax := max(15, min(20, available/4))
	hostMax := max(15, min(25, (available-apexMax)*2/5))
	valueMax := max(15, available-apexMax-hostMax)
	table := output.NewGroupedWrappingTablePerCol(w, map[int]int{0: apexMax, 1: hostMax, 3: valueMax})
	table.Header([]string{"Apex Domain", "Host", "Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
