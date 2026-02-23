package quad9

import (
	"fmt"
	"io"

	"github.com/tbckr/trident/internal/output"
)

// Result holds the Quad9 threat-intelligence verdict for a single domain.
type Result struct {
	Input   string `json:"input"`
	Blocked bool   `json:"blocked"`
}

// IsEmpty reports whether the result is unpopulated (no input was set).
func (r *Result) IsEmpty() bool {
	return r.Input == ""
}

// WriteText renders the verdict as a single line: "blocked" or "not blocked".
func (r *Result) WriteText(w io.Writer) error {
	verdict := "not blocked"
	if r.Blocked {
		verdict = "blocked"
	}
	_, err := fmt.Fprintln(w, verdict)
	return err
}

// WriteTable renders the result as an ASCII table with Domain and Blocked columns.
func (r *Result) WriteTable(w io.Writer) error {
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
