package quad9_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/quad9"
)

func TestResolveMultiResult_IsEmpty(t *testing.T) {
	mr := &quad9.ResolveMultiResult{}
	assert.True(t, mr.IsEmpty())

	mr.Results = append(mr.Results, &quad9.ResolveResult{Input: "example.com"})
	assert.True(t, mr.IsEmpty(), "result with no records should still be empty")

	mr.Results = append(mr.Results, &quad9.ResolveResult{Input: "example.org", A: []string{"1.2.3.4"}})
	assert.False(t, mr.IsEmpty())
}

func TestResolveMultiResult_WriteText(t *testing.T) {
	mr := &quad9.ResolveMultiResult{}
	mr.Results = []*quad9.ResolveResult{
		{Input: "a.com", NS: []string{"ns1.a.com."}, A: []string{"1.1.1.1"}},
		{Input: "b.com", A: []string{"2.2.2.2"}},
	}

	var buf bytes.Buffer
	err := mr.WriteText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "DOMAIN")
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "a.com")
	assert.Contains(t, out, "NS")
	assert.Contains(t, out, "ns1.a.com.")
	assert.Contains(t, out, "1.1.1.1")
	assert.Contains(t, out, "b.com")
	assert.Contains(t, out, "2.2.2.2")
}
