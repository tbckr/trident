package asn

import (
	"encoding/json"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// MultiResult holds ASN lookup results for multiple inputs.
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

// WritePlain writes all results as plain text (pipe-delimited lines).
func (m *MultiResult) WritePlain(w io.Writer) error {
	for _, r := range m.Results {
		if err := r.WritePlain(w); err != nil {
			return err
		}
	}
	return nil
}

// WriteText renders all results in a single combined table grouped by input.
// Columns: Input / Field / Value. Input cells are merged hierarchically.
func (m *MultiResult) WriteText(w io.Writer) error {
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
