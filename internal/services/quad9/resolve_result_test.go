package quad9_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/quad9"
)

func TestResolveResult_IsEmpty(t *testing.T) {
	empty := &quad9.ResolveResult{Input: "example.com"}
	assert.True(t, empty.IsEmpty())

	withA := &quad9.ResolveResult{Input: "example.com", A: []string{"1.2.3.4"}}
	assert.False(t, withA.IsEmpty())

	withAAAA := &quad9.ResolveResult{Input: "example.com", AAAA: []string{"::1"}}
	assert.False(t, withAAAA.IsEmpty())

	withNS := &quad9.ResolveResult{Input: "example.com", NS: []string{"ns1.example.com."}}
	assert.False(t, withNS.IsEmpty())

	withMX := &quad9.ResolveResult{Input: "example.com", MX: []string{"0 mail.example.com."}}
	assert.False(t, withMX.IsEmpty())

	withTXT := &quad9.ResolveResult{Input: "example.com", TXT: []string{"v=spf1 -all"}}
	assert.False(t, withTXT.IsEmpty())
}

func TestResolveResult_WritePlain(t *testing.T) {
	r := &quad9.ResolveResult{
		Input: "example.com",
		NS:    []string{"ns1.example.com."},
		A:     []string{"93.184.216.34"},
		AAAA:  []string{"2606:2800::1"},
		MX:    []string{"0 ."},
		TXT:   []string{"v=spf1 -all"},
	}

	var buf bytes.Buffer
	err := r.WritePlain(&buf)
	require.NoError(t, err)

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// Canonical order: NS → A → AAAA → MX → TXT
	assert.Equal(t, "NS ns1.example.com.", lines[0])
	assert.Equal(t, "A 93.184.216.34", lines[1])
	assert.Equal(t, "AAAA 2606:2800::1", lines[2])
	assert.Equal(t, "MX 0 .", lines[3])
	assert.Equal(t, "TXT v=spf1 -all", lines[4])
}

func TestResolveResult_WriteText(t *testing.T) {
	r := &quad9.ResolveResult{
		Input: "example.com",
		NS:    []string{"ns1.example.com."},
		A:     []string{"93.184.216.34"},
	}

	var buf bytes.Buffer
	err := r.WriteText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "NS")
	assert.Contains(t, out, "ns1.example.com.")
	assert.Contains(t, out, "A")
	assert.Contains(t, out, "93.184.216.34")
}
