package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Format is the output format requested by the user.
type Format string

// Output format constants supported by the --output flag.
const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatText  Format = "text"
)

// TableFormattable results know how to render themselves as an ASCII table.
type TableFormattable interface {
	WriteTable(w io.Writer) error
}

// TextFormattable results know how to render themselves as plain text (one record per line).
// Used for piping output to other tools.
type TextFormattable interface {
	WriteText(w io.Writer) error
}

// Write dispatches a service result to the appropriate formatter.
// JSON uses json.Encoder with indentation. Table requires the result to implement TableFormattable.
// Text requires the result to implement TextFormattable.
func Write(w io.Writer, format Format, result any) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case FormatTable:
		tf, ok := result.(TableFormattable)
		if !ok {
			return fmt.Errorf("result type %T does not support table output", result)
		}
		return tf.WriteTable(w)
	case FormatText:
		pf, ok := result.(TextFormattable)
		if !ok {
			return fmt.Errorf("result type %T does not support text output", result)
		}
		return pf.WriteText(w)
	default:
		return fmt.Errorf("unsupported output format: %q", format)
	}
}
