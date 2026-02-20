package pgp

import (
	"io"
	"strconv"
	"strings"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// MultiResult holds PGP key search results for multiple inputs.
type MultiResult struct {
	services.MultiResultBase[Result, *Result]
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
