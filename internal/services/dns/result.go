package dns

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// Result holds the DNS lookup results for a single domain or IP input.
type Result struct {
	Input string   `json:"input"`
	NS    []string `json:"ns,omitempty"`
	CNAME []string `json:"cname,omitempty"`
	A     []string `json:"a,omitempty"`
	AAAA  []string `json:"aaaa,omitempty"`
	MX    []string `json:"mx,omitempty"`
	SRV   []string `json:"srv,omitempty"`
	TXT   []string `json:"txt,omitempty"`
	PTR   []string `json:"ptr,omitempty"`
}

// IsEmpty reports whether the result contains no DNS records.
func (r *Result) IsEmpty() bool {
	return len(r.NS) == 0 && len(r.CNAME) == 0 &&
		len(r.A) == 0 && len(r.AAAA) == 0 &&
		len(r.MX) == 0 && len(r.SRV) == 0 &&
		len(r.TXT) == 0 && len(r.PTR) == 0
}

// WritePlain renders the result as plain text with one record per line.
// Each line has the format: "TYPE value" (e.g. "NS ns1.example.com").
func (r *Result) WritePlain(w io.Writer) error {
	for _, v := range r.NS {
		if _, err := fmt.Fprintf(w, "NS %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.CNAME {
		if _, err := fmt.Fprintf(w, "CNAME %s\n", v); err != nil {
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
	for _, v := range r.SRV {
		if _, err := fmt.Fprintf(w, "SRV %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.TXT {
		if _, err := fmt.Fprintf(w, "TXT %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.PTR {
		if _, err := fmt.Fprintf(w, "PTR %s\n", v); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable renders the result as an ASCII table, sorted and grouped by record type.
func (r *Result) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, v := range r.NS {
		rows = append(rows, []string{"NS", v})
	}
	for _, v := range r.CNAME {
		rows = append(rows, []string{"CNAME", v})
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
	for _, v := range r.SRV {
		rows = append(rows, []string{"SRV", v})
	}
	for _, v := range r.TXT {
		rows = append(rows, []string{"TXT", v})
	}
	for _, v := range r.PTR {
		rows = append(rows, []string{"PTR", v})
	}
	table := output.NewGroupedWrappingTable(w, 20, 20)
	table.Header([]string{"Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
