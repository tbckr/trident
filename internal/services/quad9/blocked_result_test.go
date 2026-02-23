package quad9_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/quad9"
)

func TestBlockedResult_IsEmpty(t *testing.T) {
	empty := &quad9.BlockedResult{}
	assert.True(t, empty.IsEmpty(), "result with no Input should be empty")

	populated := &quad9.BlockedResult{Input: "example.com", Blocked: false}
	assert.False(t, populated.IsEmpty(), "result with Input set should not be empty")

	blocked := &quad9.BlockedResult{Input: "malicious.example", Blocked: true}
	assert.False(t, blocked.IsEmpty())
}

func TestBlockedResult_WriteText_Blocked(t *testing.T) {
	r := &quad9.BlockedResult{Input: "malicious.example", Blocked: true}
	var buf bytes.Buffer
	err := r.WriteText(&buf)
	require.NoError(t, err)
	assert.Equal(t, "blocked\n", buf.String())
}

func TestBlockedResult_WriteText_NotBlocked(t *testing.T) {
	r := &quad9.BlockedResult{Input: "example.com", Blocked: false}
	var buf bytes.Buffer
	err := r.WriteText(&buf)
	require.NoError(t, err)
	assert.Equal(t, "not blocked\n", buf.String())
}

func TestBlockedResult_WriteTable(t *testing.T) {
	r := &quad9.BlockedResult{Input: "example.com", Blocked: true}
	var buf bytes.Buffer
	err := r.WriteTable(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "DOMAIN")
	assert.Contains(t, out, "BLOCKED")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "true")
}

func TestBlockedResult_WriteTable_NotBlocked(t *testing.T) {
	r := &quad9.BlockedResult{Input: "example.com", Blocked: false}
	var buf bytes.Buffer
	err := r.WriteTable(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "false")
}
