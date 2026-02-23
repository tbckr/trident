package quad9

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// ResolveResult holds Quad9 DoH DNS lookup results for a single domain.
type ResolveResult struct {
	Input  string   `json:"input"`
	NS     []string `json:"ns,omitempty"`
	SOA    []string `json:"soa,omitempty"`
	CNAME  []string `json:"cname,omitempty"`
	A      []string `json:"a,omitempty"`
	AAAA   []string `json:"aaaa,omitempty"`
	MX     []string `json:"mx,omitempty"`
	SRV    []string `json:"srv,omitempty"`
	TXT    []string `json:"txt,omitempty"`
	CAA    []string `json:"caa,omitempty"`
	DNSKEY []string `json:"dnskey,omitempty"`
	HTTPS  []string `json:"https,omitempty"`
	SSHFP  []string `json:"sshfp,omitempty"`
}

// IsEmpty reports whether the result contains no DNS records.
func (r *ResolveResult) IsEmpty() bool {
	return len(r.NS) == 0 && len(r.SOA) == 0 && len(r.CNAME) == 0 &&
		len(r.A) == 0 && len(r.AAAA) == 0 && len(r.MX) == 0 && len(r.SRV) == 0 &&
		len(r.TXT) == 0 && len(r.CAA) == 0 && len(r.DNSKEY) == 0 &&
		len(r.HTTPS) == 0 && len(r.SSHFP) == 0
}

// WriteText renders the result as plain text with one record per line.
// Each line has the format: "TYPE value" (e.g. "NS ns1.example.com").
// Canonical order: NS → SOA → CNAME → A → AAAA → MX → SRV → TXT → CAA → DNSKEY → HTTPS → SSHFP.
func (r *ResolveResult) WriteText(w io.Writer) error {
	for _, v := range r.NS {
		if _, err := fmt.Fprintf(w, "NS %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.SOA {
		if _, err := fmt.Fprintf(w, "SOA %s\n", v); err != nil {
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
	for _, v := range r.CAA {
		if _, err := fmt.Fprintf(w, "CAA %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.DNSKEY {
		if _, err := fmt.Fprintf(w, "DNSKEY %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.HTTPS {
		if _, err := fmt.Fprintf(w, "HTTPS %s\n", v); err != nil {
			return err
		}
	}
	for _, v := range r.SSHFP {
		if _, err := fmt.Fprintf(w, "SSHFP %s\n", v); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable renders the result as an ASCII table, grouped by record type.
// Canonical order: NS → SOA → CNAME → A → AAAA → MX → SRV → TXT → CAA → DNSKEY → HTTPS → SSHFP.
func (r *ResolveResult) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, v := range r.NS {
		rows = append(rows, []string{"NS", v})
	}
	for _, v := range r.SOA {
		rows = append(rows, []string{"SOA", v})
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
	for _, v := range r.CAA {
		rows = append(rows, []string{"CAA", v})
	}
	for _, v := range r.DNSKEY {
		rows = append(rows, []string{"DNSKEY", v})
	}
	for _, v := range r.HTTPS {
		rows = append(rows, []string{"HTTPS", v})
	}
	for _, v := range r.SSHFP {
		rows = append(rows, []string{"SSHFP", v})
	}
	table := output.NewGroupedWrappingTable(w, 20, 20)
	table.Header([]string{"Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
