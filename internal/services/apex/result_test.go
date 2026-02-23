package apex_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/apex"
)

func TestResult_IsEmpty(t *testing.T) {
	r := &apex.Result{}
	assert.True(t, r.IsEmpty())

	r.Records = append(r.Records, apex.Record{Host: "example.com", Type: "A", Value: "1.2.3.4"})
	assert.False(t, r.IsEmpty())
}

func TestResult_WriteText(t *testing.T) {
	r := &apex.Result{
		Input: "example.com",
		Records: []apex.Record{
			{Host: "example.com", Type: "A", Value: "1.2.3.4"},
			{Host: "example.com", Type: "NS", Value: "ns1.example.com."},
		},
	}

	var buf bytes.Buffer
	err := r.WriteText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "example.com A 1.2.3.4")
	assert.Contains(t, out, "example.com NS ns1.example.com.")
}

func TestResult_WriteText_Empty(t *testing.T) {
	r := &apex.Result{Input: "example.com"}
	var buf bytes.Buffer
	err := r.WriteText(&buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestResult_WriteTable(t *testing.T) {
	r := &apex.Result{
		Input: "example.com",
		Records: []apex.Record{
			{Host: "example.com", Type: "A", Value: "1.2.3.4"},
			{Host: "example.com", Type: "NS", Value: "ns1.example.com."},
		},
	}

	var buf bytes.Buffer
	err := r.WriteTable(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "HOST")
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "A")
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "NS")
	assert.Contains(t, out, "ns1.example.com.")
}
