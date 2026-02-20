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
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
)

// TextFormattable results know how to render themselves as an ASCII table.
type TextFormattable interface {
	WriteText(w io.Writer) error
}

// PlainFormattable results know how to render themselves as plain text (one record per line).
// Used for piping output to other tools.
type PlainFormattable interface {
	WritePlain(w io.Writer) error
}

// Write dispatches a service result to the appropriate formatter.
// JSON uses json.Encoder with indentation. Text requires the result to implement TextFormattable.
// Plain requires the result to implement PlainFormattable.
func Write(w io.Writer, format Format, result any) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case FormatText:
		tf, ok := result.(TextFormattable)
		if !ok {
			return fmt.Errorf("result type %T does not support text output", result)
		}
		return tf.WriteText(w)
	case FormatPlain:
		pf, ok := result.(PlainFormattable)
		if !ok {
			return fmt.Errorf("result type %T does not support plain output", result)
		}
		return pf.WritePlain(w)
	default:
		return fmt.Errorf("unsupported output format: %q", format)
	}
}
