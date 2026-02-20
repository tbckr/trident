package dns_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/dns"
)

func TestResult_IsEmpty(t *testing.T) {
	assert.True(t, (&dns.Result{}).IsEmpty())
	assert.False(t, (&dns.Result{A: []string{"1.2.3.4"}}).IsEmpty())
}

func TestResult_WriteText(t *testing.T) {
	result := &dns.Result{
		Input: "example.com",
		NS:    []string{"ns1.example.com."},
		A:     []string{"1.2.3.4"},
		MX:    []string{"mail.example.com."},
		TXT:   []string{"v=spf1 -all"},
	}

	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "mail.example.com.")

	// Assert canonical ordering: NS before A before MX
	nsIdx := strings.Index(out, "ns1.example.com.")
	aIdx := strings.Index(out, "1.2.3.4")
	mxIdx := strings.Index(out, "mail.example.com.")
	assert.Less(t, nsIdx, aIdx, "NS records should appear before A records")
	assert.Less(t, aIdx, mxIdx, "A records should appear before MX records")
}

func TestResult_WriteText_PTR(t *testing.T) {
	result := &dns.Result{
		Input: "8.8.8.8",
		PTR:   []string{"dns.google."},
	}

	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "PTR")
	assert.Contains(t, out, "dns.google.")
}

func TestResult_WritePlain(t *testing.T) {
	result := &dns.Result{
		Input: "example.com",
		NS:    []string{"ns1.example.com."},
		A:     []string{"1.2.3.4"},
		AAAA:  []string{"::1"},
		MX:    []string{"mail.example.com."},
		TXT:   []string{"v=spf1 -all"},
		PTR:   []string{"host.example.com."},
	}

	var buf bytes.Buffer
	err := result.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "NS ns1.example.com.")
	assert.Contains(t, out, "A 1.2.3.4")
	assert.Contains(t, out, "AAAA ::1")
	assert.Contains(t, out, "MX mail.example.com.")
	assert.Contains(t, out, "TXT v=spf1 -all")
	assert.Contains(t, out, "PTR host.example.com.")

	// Canonical order: NS before A before AAAA before MX before TXT before PTR
	nsIdx := strings.Index(out, "NS ")
	aIdx := strings.Index(out, "A ")
	mxIdx := strings.Index(out, "MX ")
	assert.Less(t, nsIdx, aIdx)
	assert.Less(t, aIdx, mxIdx)
}
