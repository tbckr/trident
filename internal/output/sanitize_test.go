package output_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tbckr/trident/internal/output"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"clean string", "hello world", "hello world"},
		{"red color", "\x1b[31mred\x1b[0m", "red"},
		{"bold", "\x1b[1mbold\x1b[0m", "bold"},
		{"multiple sequences", "\x1b[1m\x1b[31merror\x1b[0m", "error"},
		{"empty", "", ""},
		{"no escape", "plain text 123", "plain text 123"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, output.StripANSI(tc.input))
		})
	}
}
