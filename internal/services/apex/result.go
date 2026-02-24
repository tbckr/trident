package apex

import (
	"fmt"
	"io"
	"sort"

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
	Skipped []string `json:"skipped,omitempty"`
}

// IsEmpty reports whether the result contains no records and no skipped sub-services.
func (r *Result) IsEmpty() bool {
	return len(r.Records) == 0 && len(r.Skipped) == 0
}

// WriteText renders each record as "HOST TYPE VALUE\n".
// Any skipped sub-services are listed at the end as "[skipped: <name>]".
func (r *Result) WriteText(w io.Writer) error {
	for _, rec := range r.Records {
		if _, err := fmt.Fprintf(w, "%s %s %s\n", rec.Host, rec.Type, rec.Value); err != nil {
			return err
		}
	}
	for _, name := range r.Skipped {
		if _, err := fmt.Fprintf(w, "[skipped: %s]\n", name); err != nil {
			return err
		}
	}
	return nil
}

// sortRecordsForDisplay returns a sorted copy of records for display purposes.
// The apex input domain is sorted first (prefix "0:"), other hosts alphabetically (prefix "1:"),
// sentinel detected rows (CDN/Email/DNS) next (prefix "2:"), and ASN rows last (prefix "3:").
// The original slice is not mutated.
func sortRecordsForDisplay(input string, records []Record) []Record {
	sorted := make([]Record, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		return recSortKey(input, sorted[i]) < recSortKey(input, sorted[j])
	})
	return sorted
}

func recSortKey(input string, rec Record) string {
	switch {
	case rec.Host == input:
		return "0:" + rec.Type + ":" + rec.Value
	case rec.Type == "ASN":
		return "3:" + rec.Type + ":" + rec.Value
	case rec.Host == "detected":
		return "2:" + rec.Host + ":" + rec.Type + ":" + rec.Value
	default:
		return "1:" + rec.Host + ":" + rec.Type + ":" + rec.Value
	}
}

// WriteTable renders the result as a 3-column table grouped by HOST.
// Skipped sub-services appear at the end with Host="skipped".
func (r *Result) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, rec := range sortRecordsForDisplay(r.Input, r.Records) {
		rows = append(rows, []string{rec.Host, rec.Type, rec.Value})
	}
	for _, name := range r.Skipped {
		rows = append(rows, []string{"skipped", name, ""})
	}
	// 3-col table: | host | type | value |
	// Fixed structure: 10 chars (4 borders + 6 padding spaces).
	// Type column (col 1) is left unconstrained — values are at most ~6 chars.
	// Host column (col 0) is capped proportionally to avoid long subdomain names
	// (e.g. "_sipfederationtls._tcp.example.com") from overflowing the terminal.
	// Value column (col 2) receives the remaining space.
	const structureAndType = 18 // 10 fixed + 8 generous type-col budget
	termWidth := output.TerminalWidth(w)
	available := termWidth - structureAndType
	hostMax := max(20, min(30, available*2/5))
	valueMax := max(20, available-hostMax)
	table := output.NewGroupedWrappingTablePerCol(w, map[int]int{0: hostMax, 2: valueMax})
	table.Header([]string{"Host", "Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
