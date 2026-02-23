package dns_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/dns"
)

func TestMultiResult_IsEmpty(t *testing.T) {
	t.Run("empty when no results", func(t *testing.T) {
		m := &dns.MultiResult{}
		assert.True(t, m.IsEmpty())
	})

	t.Run("empty when all results empty", func(t *testing.T) {
		m := &dns.MultiResult{}
		m.Results = []*dns.Result{
			{Input: "example.com"},
			{Input: "example.org"},
		}
		assert.True(t, m.IsEmpty())
	})

	t.Run("not empty when one result has data", func(t *testing.T) {
		m := &dns.MultiResult{}
		m.Results = []*dns.Result{
			{Input: "example.com"},
			{Input: "example.org", A: []string{"1.2.3.4"}},
		}
		assert.False(t, m.IsEmpty())
	})
}

func TestMultiResult_WriteTable(t *testing.T) {
	m := &dns.MultiResult{}
	m.Results = []*dns.Result{
		{
			Input: "example.com",
			NS:    []string{"ns1.example.com."},
			CNAME: []string{"alias.example.com."},
			A:     []string{"1.2.3.4"},
			SRV:   []string{"10 20 443 web.example.com."},
		},
		{
			Input: "example.org",
			A:     []string{"5.6.7.8"},
			TXT:   []string{"v=spf1 -all"},
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "DOMAIN")
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "example.org")
	assert.Contains(t, out, "ns1.example.com.")
	assert.Contains(t, out, "alias.example.com.")
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "web.example.com.")
	assert.Contains(t, out, "5.6.7.8")
	assert.Contains(t, out, "v=spf1 -all")

	// example.com should appear before example.org
	comIdx := strings.Index(out, "example.com")
	orgIdx := strings.Index(out, "example.org")
	assert.Less(t, comIdx, orgIdx)
}

func TestMultiResult_WriteText(t *testing.T) {
	m := &dns.MultiResult{}
	m.Results = []*dns.Result{
		{
			Input: "example.com",
			NS:    []string{"ns1.example.com."},
			A:     []string{"1.2.3.4"},
		},
		{
			Input: "example.org",
			A:     []string{"5.6.7.8"},
		},
	}

	var buf bytes.Buffer
	err := m.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "NS ns1.example.com.")
	assert.Contains(t, out, "A 1.2.3.4")
	assert.Contains(t, out, "A 5.6.7.8")
}

func TestMultiResult_MarshalJSON(t *testing.T) {
	m := &dns.MultiResult{}
	m.Results = []*dns.Result{
		{Input: "example.com", A: []string{"1.2.3.4"}},
		{Input: "example.org", A: []string{"5.6.7.8"}},
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	// Should be a JSON array
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(data, &arr))
	assert.Len(t, arr, 2)
	assert.Equal(t, "example.com", arr[0]["input"])
	assert.Equal(t, "example.org", arr[1]["input"])
}
