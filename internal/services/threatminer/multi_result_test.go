package threatminer_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/threatminer"
)

func TestMultiResult_IsEmpty(t *testing.T) {
	t.Run("empty when no results", func(t *testing.T) {
		m := &threatminer.MultiResult{}
		assert.True(t, m.IsEmpty())
	})

	t.Run("empty when all results empty", func(t *testing.T) {
		m := &threatminer.MultiResult{}
		m.Results = []*threatminer.Result{
			{Input: "example.com", InputType: "domain"},
			{Input: "1.2.3.4", InputType: "ip"},
		}
		assert.True(t, m.IsEmpty())
	})

	t.Run("not empty when one result has passive DNS", func(t *testing.T) {
		m := &threatminer.MultiResult{}
		m.Results = []*threatminer.Result{
			{Input: "example.com", InputType: "domain"},
			{Input: "1.2.3.4", InputType: "ip", PassiveDNS: []threatminer.PDNSEntry{
				{IP: "1.2.3.4", Domain: "example.com"},
			}},
		}
		assert.False(t, m.IsEmpty())
	})
}

func TestMultiResult_WriteTable_PassiveDNS(t *testing.T) {
	m := &threatminer.MultiResult{}
	m.Results = []*threatminer.Result{
		{
			Input:     "example.com",
			InputType: "domain",
			PassiveDNS: []threatminer.PDNSEntry{
				{IP: "1.2.3.4", Domain: "example.com", FirstSeen: "2020-01-01", LastSeen: "2021-01-01"},
			},
		},
		{
			Input:     "example.org",
			InputType: "domain",
			PassiveDNS: []threatminer.PDNSEntry{
				{IP: "5.6.7.8", Domain: "example.org", FirstSeen: "2021-06-01", LastSeen: "2022-06-01"},
			},
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "INPUT")
	assert.Contains(t, out, "IP")
	assert.Contains(t, out, "DOMAIN")
	assert.Contains(t, out, "FIRST SEEN")
	assert.Contains(t, out, "LAST SEEN")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "example.org")
	assert.Contains(t, out, "5.6.7.8")

	// example.com entries before example.org entries
	firstIdx := strings.Index(out, "1.2.3.4")
	secondIdx := strings.Index(out, "5.6.7.8")
	assert.Less(t, firstIdx, secondIdx)
}

func TestMultiResult_WriteTable_Subdomains(t *testing.T) {
	m := &threatminer.MultiResult{}
	m.Results = []*threatminer.Result{
		{
			Input:      "example.com",
			InputType:  "domain",
			Subdomains: []string{"mail.example.com", "www.example.com"},
		},
		{
			Input:      "example.org",
			InputType:  "domain",
			Subdomains: []string{"api.example.org"},
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "SUBDOMAIN")
	assert.Contains(t, out, "mail.example.com")
	assert.Contains(t, out, "api.example.org")
}

func TestMultiResult_WriteTable_HashInfo(t *testing.T) {
	m := &threatminer.MultiResult{}
	m.Results = []*threatminer.Result{
		{
			Input:     "aabbcc",
			InputType: "hash",
			HashInfo: &threatminer.HashMetadata{
				MD5:      "aabbcc",
				FileType: "PE32",
				FileName: "malware.exe",
			},
		},
		{
			Input:     "ddeeff",
			InputType: "hash",
			HashInfo: &threatminer.HashMetadata{
				MD5:      "ddeeff",
				FileType: "PDF",
				FileName: "document.pdf",
			},
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "FIELD")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "aabbcc")
	assert.Contains(t, out, "PE32")
	assert.Contains(t, out, "malware.exe")
	assert.Contains(t, out, "ddeeff")
	assert.Contains(t, out, "PDF")
	assert.Contains(t, out, "document.pdf")
}

func TestMultiResult_WriteTable_SkipsEmptySubtables(t *testing.T) {
	// Only PassiveDNS — Subdomains and HashInfo tables should not appear
	m := &threatminer.MultiResult{}
	m.Results = []*threatminer.Result{
		{
			Input:     "1.2.3.4",
			InputType: "ip",
			PassiveDNS: []threatminer.PDNSEntry{
				{IP: "1.2.3.4", Domain: "example.com"},
			},
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "1.2.3.4")
	// Subdomain header should not be present since there are no subdomains
	assert.NotContains(t, out, "SUBDOMAIN")
}

func TestMultiResult_WritePlain(t *testing.T) {
	m := &threatminer.MultiResult{}
	m.Results = []*threatminer.Result{
		{
			Input:     "example.com",
			InputType: "domain",
			PassiveDNS: []threatminer.PDNSEntry{
				{IP: "1.2.3.4", Domain: "example.com"},
			},
			Subdomains: []string{"www.example.com"},
		},
		{
			Input:     "aabbcc",
			InputType: "hash",
			HashInfo: &threatminer.HashMetadata{
				MD5:      "aabbcc",
				FileType: "PE32",
			},
		},
	}

	var buf bytes.Buffer
	err := m.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "example.com 1.2.3.4 example.com")
	assert.Contains(t, out, "example.com www.example.com")
	assert.Contains(t, out, "aabbcc MD5: aabbcc")
	assert.Contains(t, out, "aabbcc FileType: PE32")
}

func TestMultiResult_MarshalJSON(t *testing.T) {
	m := &threatminer.MultiResult{}
	m.Results = []*threatminer.Result{
		{Input: "example.com", InputType: "domain"},
		{Input: "1.2.3.4", InputType: "ip"},
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	// Should be a JSON array
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(data, &arr))
	assert.Len(t, arr, 2)
	assert.Equal(t, "example.com", arr[0]["input"])
	assert.Equal(t, "1.2.3.4", arr[1]["input"])
}
