package quad9

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// ResolveResult holds Quad9 DoH DNS lookup results for a single domain.
type ResolveResult struct {
	Input string   `json:"input"`
	A     []string `json:"a,omitempty"`
	AAAA  []string `json:"aaaa,omitempty"`
	NS    []string `json:"ns,omitempty"`
	MX    []string `json:"mx,omitempty"`
	TXT   []string `json:"txt,omitempty"`
}

// IsEmpty reports whether the result contains no DNS records.
func (r *ResolveResult) IsEmpty() bool {
	return len(r.A) == 0 && len(r.AAAA) == 0 &&
		len(r.NS) == 0 && len(r.MX) == 0 && len(r.TXT) == 0
}

// WritePlain renders the result as plain text with one record per line.
// Each line has the format: "TYPE value" (e.g. "NS ns1.example.com").
// Canonical order: NS → A → AAAA → MX → TXT.
func (r *ResolveResult) WritePlain(w io.Writer) error {
	for _, v := range r.NS {
		if _, err := fmt.Fprintf(w, "NS %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.A {
		if _, err := fmt.Fprintf(w, "A %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.AAAA {
		if _, err := fmt.Fprintf(w, "AAAA %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.MX {
		if _, err := fmt.Fprintf(w, "MX %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.TXT {
		if _, err := fmt.Fprintf(w, "TXT %s\n", v); err != nil {
			return err
		}
	}
	return nil
}

// WriteText renders the result as an ASCII table, grouped by record type.
// Canonical order: NS → A → AAAA → MX → TXT.
func (r *ResolveResult) WriteText(w io.Writer) error {
	var rows [][]string
	for _, v := range r.NS {
		rows = append(rows, []string{"NS", v})
	}
	for _, v := range r.A {
		rows = append(rows, []string{"A", v})
	}
	for _, v := range r.AAAA {
		rows = append(rows, []string{"AAAA", v})
	}
	for _, v := range r.MX {
		rows = append(rows, []string{"MX", v})
	}
	for _, v := range r.TXT {
		rows = append(rows, []string{"TXT", v})
	}
	table := output.NewGroupedWrappingTable(w, 20, 20)
	table.Header([]string{"Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
