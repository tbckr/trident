package services

import (
	"encoding/json"
	"io"
)

// multiItem constrains the element type stored in MultiResultBase.
type multiItem[T any] interface {
	*T
	IsEmpty() bool
	WritePlain(w io.Writer) error
}

// MultiResultBase provides the three identical MultiResult methods shared by every
// service. Embed it and add WriteText to complete the output interfaces.
type MultiResultBase[T any, PT multiItem[T]] struct {
	Results []PT
}

// IsEmpty reports whether all contained results are empty.
func (m *MultiResultBase[T, PT]) IsEmpty() bool {
	for _, r := range m.Results {
		if !r.IsEmpty() {
			return false
		}
	}
	return true
}

// MarshalJSON serializes the multi-result as a JSON array of individual results.
func (m *MultiResultBase[T, PT]) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Results)
}

// WritePlain writes all results as plain text (one record per line).
func (m *MultiResultBase[T, PT]) WritePlain(w io.Writer) error {
	for _, r := range m.Results {
		if err := r.WritePlain(w); err != nil {
			return err
		}
	}
	return nil
}
