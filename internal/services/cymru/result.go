package cymru

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// Result holds the ASN lookup result for a single IP or ASN input.
type Result struct {
	Input       string `json:"input"`
	ASN         string `json:"asn,omitempty"`
	Prefix      string `json:"prefix,omitempty"`
	Country     string `json:"country,omitempty"`
	Registry    string `json:"registry,omitempty"`
	Description string `json:"description,omitempty"`
}

// IsEmpty reports whether the result contains no ASN data.
func (r *Result) IsEmpty() bool {
	return r.ASN == "" && r.Prefix == "" && r.Country == "" &&
		r.Registry == "" && r.Description == ""
}

// WriteText renders the result as a single pipe-delimited line.
// Format: "ASN / Prefix / Country / Registry / Description"
func (r *Result) WriteText(w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s / %s / %s / %s / %s\n",
		r.ASN, r.Prefix, r.Country, r.Registry, r.Description)
	return err
}

// WriteTable renders the result as an ASCII table.
func (r *Result) WriteTable(w io.Writer) error {
	rows := [][]string{
		{"ASN", r.ASN},
		{"Prefix", r.Prefix},
		{"Country", r.Country},
		{"Registry", r.Registry},
		{"Description", r.Description},
	}
	table := output.NewWrappingTable(w, 20, 20)
	table.Header([]string{"Field", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
