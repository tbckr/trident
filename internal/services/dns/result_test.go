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
	assert.False(t, (&dns.Result{CNAME: []string{"alias.example.com."}}).IsEmpty())
	assert.False(t, (&dns.Result{SRV: []string{"10 20 443 web.example.com."}}).IsEmpty())
}

func TestResult_WriteText(t *testing.T) {
	result := &dns.Result{
		Input: "example.com",
		NS:    []string{"ns1.example.com."},
		CNAME: []string{"alias.example.com."},
		A:     []string{"1.2.3.4"},
		MX:    []string{"mail.example.com."},
		SRV:   []string{"10 20 443 web.example.com."},
		TXT:   []string{"v=spf1 -all"},
	}

	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "mail.example.com.")
	assert.Contains(t, out, "alias.example.com.")
	assert.Contains(t, out, "web.example.com.")

	// Assert canonical ordering: NS → CNAME → A → MX → SRV
	nsIdx := strings.Index(out, "ns1.example.com.")
	cnameIdx := strings.Index(out, "alias.example.com.")
	aIdx := strings.Index(out, "1.2.3.4")
	mxIdx := strings.Index(out, "mail.example.com.")
	srvIdx := strings.Index(out, "web.example.com.")
	assert.Less(t, nsIdx, cnameIdx, "NS should appear before CNAME")
	assert.Less(t, cnameIdx, aIdx, "CNAME should appear before A")
	assert.Less(t, aIdx, mxIdx, "A records should appear before MX records")
	assert.Less(t, mxIdx, srvIdx, "MX should appear before SRV")
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
		CNAME: []string{"alias.example.com."},
		A:     []string{"1.2.3.4"},
		AAAA:  []string{"::1"},
		MX:    []string{"mail.example.com."},
		SRV:   []string{"10 20 443 web.example.com."},
		TXT:   []string{"v=spf1 -all"},
		PTR:   []string{"host.example.com."},
	}

	var buf bytes.Buffer
	err := result.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "NS ns1.example.com.")
	assert.Contains(t, out, "CNAME alias.example.com.")
	assert.Contains(t, out, "A 1.2.3.4")
	assert.Contains(t, out, "AAAA ::1")
	assert.Contains(t, out, "MX mail.example.com.")
	assert.Contains(t, out, "SRV 10 20 443 web.example.com.")
	assert.Contains(t, out, "TXT v=spf1 -all")
	assert.Contains(t, out, "PTR host.example.com.")

	// Canonical order: NS → CNAME → A → AAAA → MX → SRV → TXT → PTR
	nsIdx := strings.Index(out, "NS ")
	cnameIdx := strings.Index(out, "CNAME ")
	aIdx := strings.Index(out, "A ")
	mxIdx := strings.Index(out, "MX ")
	srvIdx := strings.Index(out, "SRV ")
	assert.Less(t, nsIdx, cnameIdx)
	assert.Less(t, cnameIdx, aIdx)
	assert.Less(t, aIdx, mxIdx)
	assert.Less(t, mxIdx, srvIdx)
}
