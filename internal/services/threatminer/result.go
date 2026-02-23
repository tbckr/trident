package threatminer

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// PDNSEntry represents a single passive DNS record.
type PDNSEntry struct {
	IP        string `json:"ip"`
	Domain    string `json:"domain"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

// HashMetadata holds metadata returned for a file hash query.
type HashMetadata struct {
	MD5      string `json:"md5"`
	SHA1     string `json:"sha1"`
	SHA256   string `json:"sha256"`
	FileType string `json:"file_type"`
	FileName string `json:"file_name"`
	FileSize string `json:"file_size"`
}

// Result holds the output of a ThreatMiner query.
type Result struct {
	Input      string      `json:"input"`
	InputType  string      `json:"input_type"`
	PassiveDNS []PDNSEntry `json:"passive_dns,omitempty"`
	Subdomains []string    `json:"subdomains,omitempty"`
	// Hash-specific fields — non-nil only for hash queries
	HashInfo *HashMetadata `json:"hash_info,omitempty"`
}

// IsEmpty returns true when the result contains no data.
func (r *Result) IsEmpty() bool {
	return len(r.PassiveDNS) == 0 && len(r.Subdomains) == 0 && r.HashInfo == nil
}

// WriteTable writes a human-readable table to w.
func (r *Result) WriteTable(w io.Writer) error {
	if r.InputType == string(inputHash) && r.HashInfo != nil {
		h := r.HashInfo
		tbl := output.NewWrappingTable(w, 20, 20)
		tbl.Header([]string{"Field", "Value"})
		rows := [][]string{
			{"MD5", h.MD5},
			{"SHA1", h.SHA1},
			{"SHA256", h.SHA256},
			{"File Type", h.FileType},
			{"File Name", h.FileName},
			{"File Size", h.FileSize},
		}
		if err := tbl.Bulk(rows); err != nil {
			return err
		}
		return tbl.Render()
	}

	if len(r.PassiveDNS) > 0 {
		tbl := output.NewWrappingTable(w, 20, 30)
		tbl.Header([]string{"IP", "Domain", "First Seen", "Last Seen"})
		rows := make([][]string, 0, len(r.PassiveDNS))
		for _, e := range r.PassiveDNS {
			rows = append(rows, []string{e.IP, e.Domain, e.FirstSeen, e.LastSeen})
		}
		if err := tbl.Bulk(rows); err != nil {
			return err
		}
		if err := tbl.Render(); err != nil {
			return err
		}
	}

	if len(r.Subdomains) > 0 {
		tbl := output.NewWrappingTable(w, 30, 6)
		tbl.Header([]string{"Subdomain"})
		rows := make([][]string, 0, len(r.Subdomains))
		for _, s := range r.Subdomains {
			rows = append(rows, []string{s})
		}
		if err := tbl.Bulk(rows); err != nil {
			return err
		}
		if err := tbl.Render(); err != nil {
			return err
		}
	}

	return nil
}

// WritePlain writes one record per line to w.
//
// For passive DNS: "<ip> <domain>" per entry.
// For subdomains: one subdomain per line.
// For hashes: "<field>: <value>" per field.
func (r *Result) WritePlain(w io.Writer) error {
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
			if _, err := fmt.Fprintf(w, "%s: %s\n", f[0], f[1]); err != nil {
				return err
			}
		}
		return nil
	}

	for _, e := range r.PassiveDNS {
		if _, err := fmt.Fprintf(w, "%s %s\n", e.IP, e.Domain); err != nil {
			return err
		}
	}
	for _, s := range r.Subdomains {
		if _, err := fmt.Fprintln(w, s); err != nil {
			return err
		}
	}
	return nil
}
