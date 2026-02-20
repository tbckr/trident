package pgp

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/tbckr/trident/internal/output"
)

// MultiResult holds PGP key search results for multiple inputs.
type MultiResult struct {
	Results []*Result
}

// IsEmpty reports whether all contained results have no keys.
func (m *MultiResult) IsEmpty() bool {
	for _, r := range m.Results {
		if !r.IsEmpty() {
			return false
		}
	}
	return true
}

// MarshalJSON serializes the multi-result as a JSON array of individual results.
func (m *MultiResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Results)
}

// WritePlain writes all results as plain text.
func (m *MultiResult) WritePlain(w io.Writer) error {
	for _, r := range m.Results {
		if err := r.WritePlain(w); err != nil {
			return err
		}
	}
	return nil
}

// WriteText renders all keys from all results in a single combined table.
// No Input column — UIDs already contain identity information.
func (m *MultiResult) WriteText(w io.Writer) error {
	tbl := output.NewWrappingTable(w, 20, 30)
	tbl.Header([]string{"Key ID", "UID", "Algorithm", "Bits", "Created", "Expires"})
	var rows [][]string
	for _, r := range m.Results {
		for _, k := range r.Keys {
			uid := strings.Join(k.UIDs, ", ")
			rows = append(rows, []string{k.KeyID, uid, k.Algorithm, strconv.Itoa(k.Bits), k.CreatedAt, k.ExpiresAt})
		}
	}
	if err := tbl.Bulk(rows); err != nil {
		return err
	}
	return tbl.Render()
}
