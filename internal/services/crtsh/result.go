package crtsh

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// Result holds the unique subdomains found in the crt.sh certificate log.
type Result struct {
	Input      string   `json:"input"`
	Subdomains []string `json:"subdomains,omitempty"`
}

// IsEmpty reports whether no subdomains were found.
func (r *Result) IsEmpty() bool {
	return len(r.Subdomains) == 0
}

// WritePlain renders the result as plain text with one subdomain per line.
func (r *Result) WritePlain(w io.Writer) error {
	for _, sub := range r.Subdomains {
		if _, err := fmt.Fprintln(w, sub); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable renders the result as an ASCII table.
func (r *Result) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, sub := range r.Subdomains {
		rows = append(rows, []string{sub})
	}
	table := output.NewWrappingTable(w, 30, 6)
	table.Header([]string{"Subdomain"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
