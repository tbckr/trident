package quad9_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/quad9"
)

func TestBlockedMultiResult_IsEmpty(t *testing.T) {
	mr := &quad9.BlockedMultiResult{}
	assert.True(t, mr.IsEmpty())

	mr.Results = append(mr.Results, &quad9.BlockedResult{Input: "example.com"})
	assert.False(t, mr.IsEmpty(), "result with Input set is never empty")
}

func TestBlockedMultiResult_WriteText(t *testing.T) {
	mr := &quad9.BlockedMultiResult{}
	mr.Results = []*quad9.BlockedResult{
		{Input: "malicious.example", Blocked: true},
		{Input: "example.com", Blocked: false},
	}

	var buf bytes.Buffer
	err := mr.WriteText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "DOMAIN")
	assert.Contains(t, out, "BLOCKED")
	assert.Contains(t, out, "malicious.example")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "false")
}
