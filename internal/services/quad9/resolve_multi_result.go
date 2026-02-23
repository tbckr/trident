package quad9

import (
	"io"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// ResolveMultiResult holds Quad9 DoH DNS lookup results for multiple inputs.
type ResolveMultiResult struct {
	services.MultiResultBase[ResolveResult, *ResolveResult]
}

// WriteTable renders all results in a single combined table grouped by domain.
// Columns: Domain / Type / Value. Domain and Type cells are merged hierarchically.
// Canonical type order per domain: NS → SOA → CNAME → A → AAAA → MX → SRV → TXT → CAA → DNSKEY → HTTPS → SSHFP.
func (m *ResolveMultiResult) WriteTable(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		for _, v := range r.NS {
			rows = append(rows, []string{r.Input, "NS", v})
		}
		for _, v := range r.SOA {
			rows = append(rows, []string{r.Input, "SOA", v})
		}
		for _, v := range r.CNAME {
			rows = append(rows, []string{r.Input, "CNAME", v})
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
		for _, v := range r.SRV {
			rows = append(rows, []string{r.Input, "SRV", v})
		}
		for _, v := range r.TXT {
			rows = append(rows, []string{r.Input, "TXT", v})
		}
		for _, v := range r.CAA {
			rows = append(rows, []string{r.Input, "CAA", v})
		}
		for _, v := range r.DNSKEY {
			rows = append(rows, []string{r.Input, "DNSKEY", v})
		}
		for _, v := range r.HTTPS {
			rows = append(rows, []string{r.Input, "HTTPS", v})
		}
		for _, v := range r.SSHFP {
			rows = append(rows, []string{r.Input, "SSHFP", v})
		}
	}
	table := output.NewGroupedWrappingTable(w, 20, 30)
	table.Header([]string{"Domain", "Type", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}
