package threatminer

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// MultiResult holds ThreatMiner query results for multiple inputs.
type MultiResult struct {
	services.MultiResultBase[Result, *Result]
}

// WritePlain overrides the base: prefixes each record with the originating input.
func (m *MultiResult) WritePlain(w io.Writer) error {
	for _, r := range m.Results {
		if r.InputType == string(inputHash) && r.HashInfo != nil {
			h := r.HashInfo
			fields := [][2]string{
				{"MD5", h.MD5},
				{"SHA1", h.SHA1},
				{"SHA256", h.SHA256},
				{"FileType", h.FileType},
				{"FileName", h.FileName},
				{"FileSize", h.FileSize},
			}
			for _, f := range fields {
				if _, err := fmt.Fprintf(w, "%s %s: %s\n", r.Input, f[0], f[1]); err != nil {
					return err
				}
			}
			continue
		}
		for _, e := range r.PassiveDNS {
			if _, err := fmt.Fprintf(w, "%s %s %s\n", r.Input, e.IP, e.Domain); err != nil {
				return err
			}
		}
		for _, s := range r.Subdomains {
			if _, err := fmt.Fprintf(w, "%s %s\n", r.Input, s); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteText renders all results in combined sub-tables grouped by input.
// Each sub-table (PassiveDNS, Subdomains, HashInfo) is rendered only when
// at least one result contains data for that sub-table.
func (m *MultiResult) WriteText(w io.Writer) error {
	if err := m.writePassiveDNS(w); err != nil {
		return err
	}
	if err := m.writeSubdomains(w); err != nil {
		return err
	}
	return m.writeHashInfo(w)
}

func (m *MultiResult) writePassiveDNS(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		for _, e := range r.PassiveDNS {
			rows = append(rows, []string{r.Input, e.IP, e.Domain, e.FirstSeen, e.LastSeen})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	tbl := output.NewGroupedWrappingTable(w, 20, 40)
	tbl.Header([]string{"Input", "IP", "Domain", "First Seen", "Last Seen"})
	if err := tbl.Bulk(rows); err != nil {
		return err
	}
	return tbl.Render()
}

func (m *MultiResult) writeSubdomains(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		for _, s := range r.Subdomains {
			rows = append(rows, []string{r.Input, s})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	tbl := output.NewGroupedWrappingTable(w, 30, 20)
	tbl.Header([]string{"Input", "Subdomain"})
	if err := tbl.Bulk(rows); err != nil {
		return err
	}
	return tbl.Render()
}

func (m *MultiResult) writeHashInfo(w io.Writer) error {
	var rows [][]string
	for _, r := range m.Results {
		if r.HashInfo == nil {
			continue
		}
		h := r.HashInfo
		fields := [][2]string{
			{"MD5", h.MD5},
			{"SHA1", h.SHA1},
			{"SHA256", h.SHA256},
			{"File Type", h.FileType},
			{"File Name", h.FileName},
			{"File Size", h.FileSize},
		}
		for _, f := range fields {
			rows = append(rows, []string{r.Input, f[0], f[1]})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	tbl := output.NewGroupedWrappingTable(w, 20, 30)
	tbl.Header([]string{"Input", "Field", "Value"})
	if err := tbl.Bulk(rows); err != nil {
		return err
	}
	return tbl.Render()
}
