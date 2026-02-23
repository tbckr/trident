package quad9_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/quad9"
)

func TestResolveResult_IsEmpty(t *testing.T) {
	empty := &quad9.ResolveResult{Input: "example.com"}
	assert.True(t, empty.IsEmpty())

	withA := &quad9.ResolveResult{Input: "example.com", A: []string{"1.2.3.4"}}
	assert.False(t, withA.IsEmpty())

	withAAAA := &quad9.ResolveResult{Input: "example.com", AAAA: []string{"::1"}}
	assert.False(t, withAAAA.IsEmpty())

	withNS := &quad9.ResolveResult{Input: "example.com", NS: []string{"ns1.example.com."}}
	assert.False(t, withNS.IsEmpty())

	withMX := &quad9.ResolveResult{Input: "example.com", MX: []string{"0 mail.example.com."}}
	assert.False(t, withMX.IsEmpty())

	withTXT := &quad9.ResolveResult{Input: "example.com", TXT: []string{"v=spf1 -all"}}
	assert.False(t, withTXT.IsEmpty())

	withSOA := &quad9.ResolveResult{Input: "example.com", SOA: []string{"ns1. admin. 2024 3600 900 604800 300"}}
	assert.False(t, withSOA.IsEmpty())

	withCNAME := &quad9.ResolveResult{Input: "example.com", CNAME: []string{"alias.example.com."}}
	assert.False(t, withCNAME.IsEmpty())

	withSRV := &quad9.ResolveResult{Input: "example.com", SRV: []string{"10 20 5060 sip.example.com."}}
	assert.False(t, withSRV.IsEmpty())

	withCAA := &quad9.ResolveResult{Input: "example.com", CAA: []string{`0 issue "letsencrypt.org"`}}
	assert.False(t, withCAA.IsEmpty())

	withDNSKEY := &quad9.ResolveResult{Input: "example.com", DNSKEY: []string{"257 3 13 abc123=="}}
	assert.False(t, withDNSKEY.IsEmpty())

	withHTTPS := &quad9.ResolveResult{Input: "example.com", HTTPS: []string{"1 h3pool.example.com."}}
	assert.False(t, withHTTPS.IsEmpty())

	withSSHFP := &quad9.ResolveResult{Input: "example.com", SSHFP: []string{"4 2 abc123"}}
	assert.False(t, withSSHFP.IsEmpty())
}

func TestResolveResult_WriteText(t *testing.T) {
	r := &quad9.ResolveResult{
		Input:  "example.com",
		NS:     []string{"ns1.example.com."},
		SOA:    []string{"ns1.example.com. admin.example.com. 2024010100 3600 900 604800 300"},
		CNAME:  []string{"alias.example.com."},
		A:      []string{"93.184.216.34"},
		AAAA:   []string{"2606:2800::1"},
		MX:     []string{"0 ."},
		SRV:    []string{"10 20 5060 sip.example.com."},
		TXT:    []string{"v=spf1 -all"},
		CAA:    []string{`0 issue "letsencrypt.org"`},
		DNSKEY: []string{"257 3 13 abc123=="},
		HTTPS:  []string{"1 h3pool.example.com."},
		SSHFP:  []string{"4 2 abc123"},
	}

	var buf bytes.Buffer
	err := r.WriteText(&buf)
	require.NoError(t, err)

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// Canonical order: NS → SOA → CNAME → A → AAAA → MX → SRV → TXT → CAA → DNSKEY → HTTPS → SSHFP
	assert.Equal(t, "NS ns1.example.com.", lines[0])
	assert.Equal(t, "SOA ns1.example.com. admin.example.com. 2024010100 3600 900 604800 300", lines[1])
	assert.Equal(t, "CNAME alias.example.com.", lines[2])
	assert.Equal(t, "A 93.184.216.34", lines[3])
	assert.Equal(t, "AAAA 2606:2800::1", lines[4])
	assert.Equal(t, "MX 0 .", lines[5])
	assert.Equal(t, "SRV 10 20 5060 sip.example.com.", lines[6])
	assert.Equal(t, "TXT v=spf1 -all", lines[7])
	assert.Equal(t, `CAA 0 issue "letsencrypt.org"`, lines[8])
	assert.Equal(t, "DNSKEY 257 3 13 abc123==", lines[9])
	assert.Equal(t, "HTTPS 1 h3pool.example.com.", lines[10])
	assert.Equal(t, "SSHFP 4 2 abc123", lines[11])
}

func TestResolveResult_WriteTable(t *testing.T) {
	r := &quad9.ResolveResult{
		Input:  "example.com",
		NS:     []string{"ns1.example.com."},
		SOA:    []string{"ns1.example.com. admin.example.com. 2024010100 3600 900 604800 300"},
		CNAME:  []string{"alias.example.com."},
		A:      []string{"93.184.216.34"},
		SRV:    []string{"10 20 5060 sip.example.com."},
		CAA:    []string{`0 issue "letsencrypt.org"`},
		DNSKEY: []string{"257 3 13 abc123=="},
		HTTPS:  []string{"1 h3pool.example.com."},
		SSHFP:  []string{"4 2 abc123"},
	}

	var buf bytes.Buffer
	err := r.WriteTable(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "NS")
	assert.Contains(t, out, "ns1.example.com.")
	assert.Contains(t, out, "SOA")
	assert.Contains(t, out, "CNAME")
	assert.Contains(t, out, "alias.example.com.")
	assert.Contains(t, out, "A")
	assert.Contains(t, out, "93.184.216.34")
	assert.Contains(t, out, "SRV")
	assert.Contains(t, out, "CAA")
	assert.Contains(t, out, "DNSKEY")
	assert.Contains(t, out, "HTTPS")
	assert.Contains(t, out, "SSHFP")
}
