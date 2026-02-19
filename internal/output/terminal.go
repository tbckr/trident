package output

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"golang.org/x/term"
)

const defaultTermWidth = 80

// TerminalWidth returns the terminal width for w, or defaultTermWidth if w is
// not a terminal or the width cannot be determined.
func TerminalWidth(w io.Writer) int {
	type fder interface{ Fd() uintptr }
	if f, ok := w.(fder); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 { //nolint:gosec // uintptr→int is safe for file descriptors; they fit in int on all supported platforms
			return width
		}
	}
	return defaultTermWidth
}

// NewWrappingTable returns a tablewriter that auto-wraps cell content to fit
// the terminal. minWidth is the floor for the computed column max width;
// overhead is the characters consumed by borders, padding, and fixed columns.
func NewWrappingTable(w io.Writer, minWidth, overhead int) *tablewriter.Table {
	maxColWidth := max(minWidth, TerminalWidth(w)-overhead)
	return tablewriter.NewTable(w,
		tablewriter.WithConfig(tablewriter.Config{
			Row: tw.CellConfig{
				Formatting:   tw.CellFormatting{AutoWrap: tw.WrapNormal},
				ColMaxWidths: tw.CellWidth{Global: maxColWidth},
			},
		}),
	)
}
