package dns

import (
	"encoding/json"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// MultiResult holds DNS lookup results for multiple inputs.
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

// WritePlain writes all results as plain text (one record per line).
func (m *MultiResult) WritePlain(w io.Writer) error {
	for _, r := range m.Results {
		if err := r.WritePlain(w); err != nil {
			return err
		}
	}
	return nil
}

// WriteText renders all results in a single combined table grouped by domain.
// Columns: Domain / Type / Value. Domain and Type cells are merged hierarchically.
func (m *MultiResult) WriteText(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		for _, v := range r.NS {
			rows = append(rows, []string{r.Input, "NS", v})
		}
		for _, v := range r.A {
			rows = append(rows, []string{r.Input, "A", v})
		}
		for _, v := range r.AAAA {
			rows = append(rows, []string{r.Input, "AAAA", v})
		}
		for _, v := range r.MX {
			rows = append(rows, []string{r.Input, "MX", v})
		}
		for _, v := range r.TXT {
			rows = append(rows, []string{r.Input, "TXT", v})
		}
		for _, v := range r.PTR {
			rows = append(rows, []string{r.Input, "PTR", v})
		}
	}
	table := output.NewGroupedWrappingTable(w, 20, 30)
	table.Header([]string{"Domain", "Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
