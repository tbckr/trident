package apex_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/apex"
)

func TestMultiResult_IsEmpty(t *testing.T) {
	mr := &apex.MultiResult{}
	assert.True(t, mr.IsEmpty())

	mr.Results = append(mr.Results, &apex.Result{Input: "example.com"})
	assert.True(t, mr.IsEmpty(), "result with no records should still be empty")

	mr.Results = append(mr.Results, &apex.Result{
		Input:   "example.org",
		Records: []apex.Record{{Host: "example.org", Type: "A", Value: "1.2.3.4"}},
	})
	assert.False(t, mr.IsEmpty())
}

func TestMultiResult_WriteText(t *testing.T) {
	mr := &apex.MultiResult{}
	mr.Results = []*apex.Result{
		{
			Input:   "a.com",
			Records: []apex.Record{{Host: "a.com", Type: "A", Value: "1.1.1.1"}},
		},
		{
			Input:   "b.com",
			Records: []apex.Record{{Host: "b.com", Type: "NS", Value: "ns1.b.com."}},
		},
	}

	var buf bytes.Buffer
	err := mr.WriteText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "a.com A 1.1.1.1")
	assert.Contains(t, out, "b.com NS ns1.b.com.")
}

func TestMultiResult_MarshalJSON(t *testing.T) {
	mr := &apex.MultiResult{}
	mr.Results = []*apex.Result{
		{
			Input:   "a.com",
			Records: []apex.Record{{Host: "a.com", Type: "A", Value: "1.1.1.1"}},
		},
	}

	data, err := json.Marshal(mr)
	require.NoError(t, err)

	// MarshalJSON serializes as a JSON array of results.
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(data, &arr))
	require.Len(t, arr, 1)
	assert.Equal(t, "a.com", arr[0]["input"])
}

func TestMultiResult_WriteTable(t *testing.T) {
	mr := &apex.MultiResult{}
	mr.Results = []*apex.Result{
		{
			Input:   "a.com",
			Records: []apex.Record{{Host: "a.com", Type: "A", Value: "1.1.1.1"}},
		},
		{
			Input:   "b.com",
			Records: []apex.Record{{Host: "b.com", Type: "NS", Value: "ns1.b.com."}},
		},
	}

	var buf bytes.Buffer
	err := mr.WriteTable(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "APEX DOMAIN")
	assert.Contains(t, out, "HOST")
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "a.com")
	assert.Contains(t, out, "1.1.1.1")
	assert.Contains(t, out, "b.com")
	assert.Contains(t, out, "ns1.b.com.")
}
