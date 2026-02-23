package crtsh_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/crtsh"
)

func TestMultiResult_IsEmpty(t *testing.T) {
	t.Run("empty when no results", func(t *testing.T) {
		m := &crtsh.MultiResult{}
		assert.True(t, m.IsEmpty())
	})

	t.Run("empty when all results empty", func(t *testing.T) {
		m := &crtsh.MultiResult{}
		m.Results = []*crtsh.Result{
			{Input: "example.com"},
			{Input: "example.org"},
		}
		assert.True(t, m.IsEmpty())
	})

	t.Run("not empty when one result has data", func(t *testing.T) {
		m := &crtsh.MultiResult{}
		m.Results = []*crtsh.Result{
			{Input: "example.com"},
			{Input: "example.org", Subdomains: []string{"www.example.org"}},
		}
		assert.False(t, m.IsEmpty())
	})
}

func TestMultiResult_WriteTable(t *testing.T) {
	m := &crtsh.MultiResult{}
	m.Results = []*crtsh.Result{
		{
			Input:      "example.com",
			Subdomains: []string{"mail.example.com", "www.example.com"},
		},
		{
			Input:      "example.org",
			Subdomains: []string{"api.example.org"},
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "DOMAIN")
	assert.Contains(t, out, "SUBDOMAIN")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "mail.example.com")
	assert.Contains(t, out, "www.example.com")
	assert.Contains(t, out, "example.org")
	assert.Contains(t, out, "api.example.org")

	// example.com entries should appear before example.org
	comIdx := strings.Index(out, "mail.example.com")
	orgIdx := strings.Index(out, "api.example.org")
	assert.Less(t, comIdx, orgIdx)
}

func TestMultiResult_WritePlain(t *testing.T) {
	m := &crtsh.MultiResult{}
	m.Results = []*crtsh.Result{
		{
			Input:      "example.com",
			Subdomains: []string{"mail.example.com", "www.example.com"},
		},
		{
			Input:      "example.org",
			Subdomains: []string{"api.example.org"},
		},
	}

	var buf bytes.Buffer
	err := m.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "mail.example.com")
	assert.Contains(t, out, "www.example.com")
	assert.Contains(t, out, "api.example.org")
	// Plain mode: one subdomain per line, no domain prefix
	assert.NotContains(t, out, "DOMAIN")
}

func TestMultiResult_MarshalJSON(t *testing.T) {
	m := &crtsh.MultiResult{}
	m.Results = []*crtsh.Result{
		{Input: "example.com", Subdomains: []string{"www.example.com"}},
		{Input: "example.org", Subdomains: []string{"api.example.org"}},
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
