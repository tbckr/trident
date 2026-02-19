package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Format is the output format requested by the user.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// TextFormattable results know how to render themselves as an ASCII table.
type TextFormattable interface {
	WriteText(w io.Writer) error
}

// Write dispatches a service result to the appropriate formatter.
// JSON uses json.Encoder with indentation. Text requires the result to implement TextFormattable.
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
	default:
		return fmt.Errorf("unsupported output format: %q", format)
	}
}
