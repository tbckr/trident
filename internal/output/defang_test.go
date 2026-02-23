package output_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
)

func TestDefangDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example[.]com"},
		{"www.example.com", "www[.]example[.]com"},
		{"sub.domain.co.uk", "sub[.]domain[.]co[.]uk"},
		{"nodots", "nodots"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, output.DefangDomain(tt.input))
		})
	}
}

func TestDefangIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.2.3.4", "1[.]2[.]3[.]4"},
		{"192.168.0.1", "192[.]168[.]0[.]1"},
		{"::1", "[::1]"},
		{"2001:db8::1", "[2001:db8::1]"},
		{"notanip", "notanip"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, output.DefangIP(tt.input))
		})
	}
}

func TestDefangURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://example.com", "hxxp://example[.]com"},
		{"https://example.com", "hxxps://example[.]com"},
		{"https://www.example.com/path?q=1", "hxxps://www[.]example[.]com/path?q=1"},
		{"HTTP://EXAMPLE.COM", "hxxp://EXAMPLE[.]COM"},
		{"ftp://example.com", "ftp://example[.]com"}, // only http/https defanged
		{"example.com", "example[.]com"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, output.DefangURL(tt.input))
		})
	}
}

func TestDefangWriter(t *testing.T) {
	var buf bytes.Buffer
	w := &output.DefangWriter{Inner: &buf}

	_, err := w.Write([]byte("http://example.com and 1.2.3.4 visited\n"))
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "hxxp://")
	assert.Contains(t, out, "example[.]com")
	assert.Contains(t, out, "1[.]2[.]3[.]4")
}

func TestDefangWriter_ReturnsOriginalLength(t *testing.T) {
	var buf bytes.Buffer
	w := &output.DefangWriter{Inner: &buf}
	data := []byte("example.com")
	n, err := w.Write(data)
	require.NoError(t, err)
	// Must return original length even though expanded bytes were written
	assert.Equal(t, len(data), n)
}

func TestResolveDefang(t *testing.T) {
	tests := []struct {
		name           string
		papLevel       pap.Level
		format         output.Format
		explicitDefang bool
		noDefang       bool
		want           bool
	}{
		{
			name:           "no-defang suppresses PAP trigger",
			papLevel:       pap.AMBER,
			format:         output.FormatTable,
			explicitDefang: false,
			noDefang:       true,
			want:           false,
		},
		{
			name:           "explicit defang, text",
			papLevel:       pap.WHITE,
			format:         output.FormatTable,
			explicitDefang: true,
			noDefang:       false,
			want:           true,
		},
		{
			name:           "explicit defang, JSON",
			papLevel:       pap.WHITE,
			format:         output.FormatJSON,
			explicitDefang: true,
			noDefang:       false,
			want:           true,
		},
		{
			name:           "PAP=amber, text, auto-trigger",
			papLevel:       pap.AMBER,
			format:         output.FormatTable,
			explicitDefang: false,
			noDefang:       false,
			want:           true,
		},
		{
			name:           "PAP=red, text, auto-trigger",
			papLevel:       pap.RED,
			format:         output.FormatText,
			explicitDefang: false,
			noDefang:       false,
			want:           true,
		},
		{
			name:           "PAP=amber, JSON, no auto-trigger",
			papLevel:       pap.AMBER,
			format:         output.FormatJSON,
			explicitDefang: false,
			noDefang:       false,
			want:           false,
		},
		{
			name:           "PAP=white default, text",
			papLevel:       pap.WHITE,
			format:         output.FormatTable,
			explicitDefang: false,
			noDefang:       false,
			want:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := output.ResolveDefang(tt.papLevel, tt.format, tt.explicitDefang, tt.noDefang)
			assert.Equal(t, tt.want, got)
		})
	}
}
