package apex

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// Record holds a single DNS reconnaissance record for an apex domain.
type Record struct {
	Host  string `json:"host"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Result holds aggregated DNS reconnaissance results for an apex domain.
type Result struct {
	Input   string   `json:"input"`
	Records []Record `json:"records,omitempty"`
}

// IsEmpty reports whether the result contains no records.
func (r *Result) IsEmpty() bool {
	return len(r.Records) == 0
}

// WriteText renders each record as "HOST TYPE VALUE\n".
func (r *Result) WriteText(w io.Writer) error {
	for _, rec := range r.Records {
		if _, err := fmt.Fprintf(w, "%s %s %s\n", rec.Host, rec.Type, rec.Value); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable renders the result as a 3-column table grouped by HOST.
func (r *Result) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, rec := range r.Records {
		rows = append(rows, []string{rec.Host, rec.Type, rec.Value})
	}
	table := output.NewGroupedWrappingTable(w, 20, 30)
	table.Header([]string{"Host", "Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
