package threatminer_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/threatminer"
)

func TestResult_IsEmpty(t *testing.T) {
	assert.True(t, (&threatminer.Result{}).IsEmpty())
	assert.False(t, (&threatminer.Result{Subdomains: []string{"www.example.com"}}).IsEmpty())
}

func TestResult_WriteText_Domain(t *testing.T) {
	result := &threatminer.Result{
		Input:     "example.com",
		InputType: "domain",
		PassiveDNS: []threatminer.PDNSEntry{
			{IP: "1.2.3.4", Domain: "example.com", FirstSeen: "2021-01-01", LastSeen: "2024-01-01"},
		},
		Subdomains: []string{"www.example.com"},
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "www.example.com")
}

func TestResult_WriteText_Hash(t *testing.T) {
	result := &threatminer.Result{
		Input:     "d41d8cd98f00b204e9800998ecf8427e",
		InputType: "hash",
		HashInfo: &threatminer.HashMetadata{
			MD5:      "d41d8cd98f00b204e9800998ecf8427e",
			FileType: "PE32",
		},
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "d41d8cd98f00b204e9800998ecf8427e")
	assert.Contains(t, out, "PE32")
}

func TestResult_WritePlain_Domain(t *testing.T) {
	result := &threatminer.Result{
		Input:     "example.com",
		InputType: "domain",
		PassiveDNS: []threatminer.PDNSEntry{
			{IP: "1.2.3.4", Domain: "example.com"},
		},
		Subdomains: []string{"www.example.com"},
	}
	var buf bytes.Buffer
	err := result.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3.4 example.com")
	assert.Contains(t, out, "www.example.com")
}

func TestResult_WritePlain_Hash(t *testing.T) {
	result := &threatminer.Result{
		Input:     "d41d8cd98f00b204e9800998ecf8427e",
		InputType: "hash",
		HashInfo: &threatminer.HashMetadata{
			MD5:      "d41d8cd98f00b204e9800998ecf8427e",
			FileType: "PE32",
		},
	}
	var buf bytes.Buffer
	err := result.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "MD5: d41d8cd98f00b204e9800998ecf8427e")
	assert.Contains(t, out, "FileType: PE32")
}
