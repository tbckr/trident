package pgp_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/pgp"
)

func TestResult_IsEmpty(t *testing.T) {
	assert.True(t, (&pgp.Result{}).IsEmpty())
	assert.False(t, (&pgp.Result{Keys: []pgp.Key{{KeyID: "0xABCD"}}}).IsEmpty())
}

func TestResult_WriteText(t *testing.T) {
	result := &pgp.Result{
		Input: "alice@example.com",
		Keys: []pgp.Key{
			{
				KeyID:     "0x1234",
				Algorithm: "RSA",
				Bits:      4096,
				CreatedAt: "2021-01-01",
				UIDs:      []string{"Alice <alice@example.com>"},
			},
		},
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "0x1234")
	assert.Contains(t, out, "alice@example.com")
}

func TestResult_WritePlain(t *testing.T) {
	result := &pgp.Result{
		Input: "alice@example.com",
		Keys: []pgp.Key{
			{KeyID: "0xAAAA", UIDs: []string{"Alice <alice@example.com>", "Alice Work <alice@work.com>"}},
			{KeyID: "0xBBBB", UIDs: []string{"Bob <bob@example.com>"}},
		},
	}
	var buf bytes.Buffer
	err := result.WritePlain(&buf)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "0xAAAA")
	assert.Contains(t, lines[0], "Alice <alice@example.com>")
	assert.Contains(t, lines[1], "0xBBBB")
}
