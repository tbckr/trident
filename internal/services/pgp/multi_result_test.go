package pgp_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/pgp"
)

func TestMultiResult_IsEmpty(t *testing.T) {
	t.Run("empty when no results", func(t *testing.T) {
		m := &pgp.MultiResult{}
		assert.True(t, m.IsEmpty())
	})

	t.Run("empty when all results have no keys", func(t *testing.T) {
		m := &pgp.MultiResult{Results: []*pgp.Result{
			{Input: "alice@example.com"},
			{Input: "bob@example.com"},
		}}
		assert.True(t, m.IsEmpty())
	})

	t.Run("not empty when one result has keys", func(t *testing.T) {
		m := &pgp.MultiResult{Results: []*pgp.Result{
			{Input: "alice@example.com"},
			{Input: "bob@example.com", Keys: []pgp.Key{
				{KeyID: "0x1234ABCD", Algorithm: "RSA", Bits: 4096},
			}},
		}}
		assert.False(t, m.IsEmpty())
	})
}

func TestMultiResult_WriteText(t *testing.T) {
	m := &pgp.MultiResult{Results: []*pgp.Result{
		{
			Input: "alice@example.com",
			Keys: []pgp.Key{
				{
					KeyID:     "0x1234ABCD",
					Algorithm: "RSA",
					Bits:      4096,
					CreatedAt: "2021-01-01",
					UIDs:      []string{"Alice <alice@example.com>"},
				},
			},
		},
		{
			Input: "bob@example.com",
			Keys: []pgp.Key{
				{
					KeyID:     "0xDEADBEEF",
					Algorithm: "EdDSA",
					Bits:      256,
					CreatedAt: "2022-06-15",
					UIDs:      []string{"Bob <bob@example.com>"},
				},
			},
		},
	}}

	var buf bytes.Buffer
	err := m.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "KEY ID")
	assert.Contains(t, out, "UID")
	assert.Contains(t, out, "ALGORITHM")
	assert.Contains(t, out, "0x1234ABCD")
	assert.Contains(t, out, "Alice <alice@example.com>")
	assert.Contains(t, out, "RSA")
	assert.Contains(t, out, "0xDEADBEEF")
	assert.Contains(t, out, "Bob <bob@example.com>")
	assert.Contains(t, out, "EdDSA")
}

func TestMultiResult_WritePlain(t *testing.T) {
	m := &pgp.MultiResult{Results: []*pgp.Result{
		{
			Input: "alice@example.com",
			Keys: []pgp.Key{
				{KeyID: "0x1234ABCD", UIDs: []string{"Alice <alice@example.com>"}},
			},
		},
		{
			Input: "bob@example.com",
			Keys: []pgp.Key{
				{KeyID: "0xDEADBEEF", UIDs: []string{"Bob <bob@example.com>"}},
			},
		},
	}}

	var buf bytes.Buffer
	err := m.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "0x1234ABCD")
	assert.Contains(t, out, "Alice <alice@example.com>")
	assert.Contains(t, out, "0xDEADBEEF")
	assert.Contains(t, out, "Bob <bob@example.com>")
}

func TestMultiResult_MarshalJSON(t *testing.T) {
	m := &pgp.MultiResult{Results: []*pgp.Result{
		{Input: "alice@example.com", Keys: []pgp.Key{{KeyID: "0x1234ABCD"}}},
		{Input: "bob@example.com", Keys: []pgp.Key{{KeyID: "0xDEADBEEF"}}},
	}}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	// Should be a JSON array
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(data, &arr))
	assert.Len(t, arr, 2)
	assert.Equal(t, "alice@example.com", arr[0]["input"])
	assert.Equal(t, "bob@example.com", arr[1]["input"])
}
