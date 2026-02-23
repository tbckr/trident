package asn_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/asn"
)

func TestMultiResult_IsEmpty(t *testing.T) {
	t.Run("empty when no results", func(t *testing.T) {
		m := &asn.MultiResult{}
		assert.True(t, m.IsEmpty())
	})

	t.Run("empty when all results empty", func(t *testing.T) {
		m := &asn.MultiResult{}
		m.Results = []*asn.Result{
			{Input: "8.8.8.8"},
			{Input: "1.1.1.1"},
		}
		assert.True(t, m.IsEmpty())
	})

	t.Run("not empty when one result has data", func(t *testing.T) {
		m := &asn.MultiResult{}
		m.Results = []*asn.Result{
			{Input: "8.8.8.8"},
			{Input: "1.1.1.1", ASN: "AS15169", Description: "GOOGLE, US"},
		}
		assert.False(t, m.IsEmpty())
	})
}

func TestMultiResult_WriteTable(t *testing.T) {
	m := &asn.MultiResult{}
	m.Results = []*asn.Result{
		{
			Input:       "8.8.8.8",
			ASN:         "AS15169",
			Prefix:      "8.8.8.0/24",
			Country:     "US",
			Registry:    "arin",
			Description: "GOOGLE, US",
		},
		{
			Input:       "1.1.1.1",
			ASN:         "AS13335",
			Prefix:      "1.1.1.0/24",
			Country:     "AU",
			Registry:    "apnic",
			Description: "CLOUDFLARENET",
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "INPUT")
	assert.Contains(t, out, "FIELD")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "8.8.8.8")
	assert.Contains(t, out, "AS15169")
	assert.Contains(t, out, "GOOGLE, US")
	assert.Contains(t, out, "1.1.1.1")
	assert.Contains(t, out, "AS13335")
	assert.Contains(t, out, "CLOUDFLARENET")

	// 8.8.8.8 should appear before 1.1.1.1
	googleIdx := strings.Index(out, "8.8.8.8")
	cfIdx := strings.Index(out, "1.1.1.1")
	assert.Less(t, googleIdx, cfIdx)
}

func TestMultiResult_WritePlain(t *testing.T) {
	m := &asn.MultiResult{}
	m.Results = []*asn.Result{
		{
			Input:       "8.8.8.8",
			ASN:         "AS15169",
			Prefix:      "8.8.8.0/24",
			Country:     "US",
			Registry:    "arin",
			Description: "GOOGLE, US",
		},
		{
			Input:       "1.1.1.1",
			ASN:         "AS13335",
			Prefix:      "1.1.1.0/24",
			Country:     "AU",
			Registry:    "apnic",
			Description: "CLOUDFLARENET",
		},
	}

	var buf bytes.Buffer
	err := m.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "AS15169")
	assert.Contains(t, out, "GOOGLE, US")
	assert.Contains(t, out, "AS13335")
	assert.Contains(t, out, "CLOUDFLARENET")
}

func TestMultiResult_MarshalJSON(t *testing.T) {
	m := &asn.MultiResult{}
	m.Results = []*asn.Result{
		{Input: "8.8.8.8", ASN: "AS15169"},
		{Input: "1.1.1.1", ASN: "AS13335"},
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	// Should be a JSON array
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(data, &arr))
	assert.Len(t, arr, 2)
	assert.Equal(t, "8.8.8.8", arr[0]["input"])
	assert.Equal(t, "1.1.1.1", arr[1]["input"])
}
