package asn_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/asn"
)

func TestResult_IsEmpty(t *testing.T) {
	assert.True(t, (&asn.Result{}).IsEmpty())
	assert.False(t, (&asn.Result{ASN: "AS15169"}).IsEmpty())
}

func TestResult_WriteTable(t *testing.T) {
	result := &asn.Result{
		Input:       "8.8.8.8",
		ASN:         "AS15169",
		Prefix:      "8.8.8.0/24",
		Country:     "US",
		Registry:    "arin",
		Description: "GOOGLE, US",
	}
	var buf bytes.Buffer
	err := result.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "AS15169")
	assert.Contains(t, out, "GOOGLE, US")
}

func TestResult_WriteText(t *testing.T) {
	result := &asn.Result{
		Input:       "8.8.8.8",
		ASN:         "AS15169",
		Prefix:      "8.8.8.0/24",
		Country:     "US",
		Registry:    "arin",
		Description: "GOOGLE, US",
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "AS15169")
	assert.Contains(t, out, "8.8.8.0/24")
	assert.Contains(t, out, "US")
	assert.Contains(t, out, "arin")
	assert.Contains(t, out, "GOOGLE, US")
}
