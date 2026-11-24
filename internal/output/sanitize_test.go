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

func TestSanitizeTerminal(t *testing.T) {
	tests := []struct{ in, want string }{
		{"\x1b]0;owned\a\x1b[2J\x1b[31mFAKE: all good", "FAKE: all good"},
		{"osc with st \x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\", "osc with st link"},
		{"bell\a cr\r c1\u009b31m nul\x00", "bell cr c131m nul"},
		{"keeps\nnewline\tand tab", "keeps\nnewline\tand tab"},
		{"proxyconnect tcp: dial tcp 127.0.0.1:1: connect: connection refused", "proxyconnect tcp: dial tcp 127.0.0.1:1: connect: connection refused"},
		// CSI with private-mode parameter and a non-letter final byte
		{"\x1b[?25l\x1b[3~x", "x"},
		// other two-byte escape sequences
		{"a\x1b7b\x1bMc\x1b=d", "abcd"},
		// ESC before a newline: only the ESC goes
		{"a\x1b\nb", "a\nb"},
		// DEL and a raw (invalid UTF-8) 0x9b byte, which an 8-bit terminal reads as CSI
		{"a\x7fb\x9bc", "ab\uFFFDc"},
		// truncated sequences at the end of the string: no panic, no ESC left
		{"a\x1b]0;title", "a0;title"},
		{"a\x1b]0;title\x1b", "a0;title"},
		{"a\x1b[31", "a31"},
		{"a\x1b[", "a"},
		{"a\x1b", "a"},
		{"\x1b", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := output.SanitizeTerminal(tt.in)
		assert.Equal(t, tt.want, got, "%q", tt.in)
		for _, r := range got {
			assert.False(t, r < 0x20 && r != '\n' && r != '\t' || r == 0x7f || (r >= 0x80 && r <= 0x9f),
				"control rune %U left in output for %q", r, tt.in)
		}
	}
}
