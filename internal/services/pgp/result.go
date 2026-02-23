package pgp

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tbckr/trident/internal/output"
)

// Key represents a single PGP key from a keyserver query.
type Key struct {
	KeyID     string   `json:"key_id"`
	Algorithm string   `json:"algorithm"`
	Bits      int      `json:"bits"`
	CreatedAt string   `json:"created_at"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	UIDs      []string `json:"uids"`
}

// Result holds the output of a PGP key search.
type Result struct {
	Input string `json:"input"`
	Keys  []Key  `json:"keys"`
}

// IsEmpty returns true when no keys were found.
func (r *Result) IsEmpty() bool {
	return len(r.Keys) == 0
}

// WriteTable writes a human-readable table to w.
// Each key is rendered with its UIDs on separate rows.
func (r *Result) WriteTable(w io.Writer) error {
	tbl := output.NewWrappingTable(w, 20, 30)
	tbl.Header([]string{"Key ID", "UID", "Algorithm", "Bits", "Created", "Expires"})
	rows := make([][]string, 0, len(r.Keys))
	for _, k := range r.Keys {
		uid := strings.Join(k.UIDs, ", ")
		rows = append(rows, []string{k.KeyID, uid, k.Algorithm, strconv.Itoa(k.Bits), k.CreatedAt, k.ExpiresAt})
	}
	if err := tbl.Bulk(rows); err != nil {
		return err
	}
	return tbl.Render()
}

// WritePlain writes one line per key: "<keyid> <first_uid>" to w.
func (r *Result) WritePlain(w io.Writer) error {
	for _, k := range r.Keys {
		uid := ""
		if len(k.UIDs) > 0 {
			uid = k.UIDs[0]
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", k.KeyID, uid); err != nil {
			return err
		}
	}
	return nil
}
