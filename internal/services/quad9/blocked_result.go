package quad9

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// BlockedResult holds the Quad9 threat-intelligence verdict for a single domain.
type BlockedResult struct {
	Input   string `json:"input"`
	Blocked bool   `json:"blocked"`
}

// IsEmpty reports whether the result is unpopulated (no input was set).
func (r *BlockedResult) IsEmpty() bool {
	return r.Input == ""
}

// WritePlain renders the verdict as a single line: "blocked" or "not blocked".
func (r *BlockedResult) WritePlain(w io.Writer) error {
	verdict := "not blocked"
	if r.Blocked {
		verdict = "blocked"
	}
	_, err := fmt.Fprintln(w, verdict)
	return err
}

// WriteTable renders the result as an ASCII table with Domain and Blocked columns.
func (r *BlockedResult) WriteTable(w io.Writer) error {
	blocked := "false"
	if r.Blocked {
		blocked = "true"
	}
	table := output.NewWrappingTable(w, 30, 6)
	table.Header([]string{"Domain", "Blocked"})
	if err := table.Bulk([][]string{{r.Input, blocked}}); err != nil {
		return err
	}
	return table.Render()
}
