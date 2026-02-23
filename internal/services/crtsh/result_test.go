package crtsh_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/crtsh"
)

func TestResult_IsEmpty(t *testing.T) {
	assert.True(t, (&crtsh.Result{}).IsEmpty())
	assert.False(t, (&crtsh.Result{Subdomains: []string{"www.example.com"}}).IsEmpty())
}

func TestResult_WriteTable(t *testing.T) {
	result := &crtsh.Result{
		Input:      "example.com",
		Subdomains: []string{"example.com", "www.example.com"},
	}
	var buf bytes.Buffer
	err := result.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "www.example.com")
}

func TestResult_WriteText(t *testing.T) {
	result := &crtsh.Result{
		Input:      "example.com",
		Subdomains: []string{"api.example.com", "www.example.com"},
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Equal(t, []string{"api.example.com", "www.example.com"}, lines)
}
