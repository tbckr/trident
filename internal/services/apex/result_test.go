package apex_test

import (
	"bytes"
	"strings"
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

func TestResult_WriteTable_SortOrder(t *testing.T) {
	// Records are intentionally out of natural order to verify sorting:
	// apex domain first, other hosts alphabetically, sentinel rows (detected) next, ASN rows last.
	r := &apex.Result{
		Input: "example.com",
		Records: []apex.Record{
			{Host: "detected", Type: "CDN", Value: "CloudFront"},
			{Host: "detected", Type: "Email", Value: "Google Workspace (mx: aspmx.l.google.com.)"},
			{Host: "detected", Type: "DNS", Value: "Cloudflare DNS (ns: liz.ns.cloudflare.com.)"},
			{Host: "www.example.com", Type: "A", Value: "9.9.9.9"},
			{Host: "example.com", Type: "TXT", Value: "v=spf1"},
			{Host: "autodiscover.example.com", Type: "A", Value: "5.6.7.8"},
			{Host: "example.com", Type: "A", Value: "1.2.3.4"},
			{Host: "detected", Type: "ASN", Value: "15169 / 1.2.0.0/16 / US / arin / GOOGLE"},
		},
	}

	var buf bytes.Buffer
	err := r.WriteTable(&buf)
	require.NoError(t, err)

	out := buf.String()

	// apex domain rows appear before subdomain rows.
	assert.Less(t, strings.Index(out, "1.2.3.4"), strings.Index(out, "5.6.7.8"),
		"apex domain rows should appear before subdomain rows")
	// autodiscover appears before www (alphabetical among non-apex).
	assert.Less(t, strings.Index(out, "autodiscover.example.com"), strings.Index(out, "www.example.com"),
		"autodiscover should appear before www alphabetically")
	// cdn appears after all non-sentinel hosts.
	assert.Greater(t, strings.Index(out, "CloudFront"), strings.Index(out, "9.9.9.9"),
		"cdn row should appear after all other hosts")
	// email appears after all non-sentinel hosts.
	assert.Greater(t, strings.Index(out, "Google Workspace"), strings.Index(out, "9.9.9.9"),
		"email row should appear after all other hosts")
	// dns appears after all non-sentinel hosts.
	assert.Greater(t, strings.Index(out, "Cloudflare DNS"), strings.Index(out, "9.9.9.9"),
		"dns row should appear after all other hosts")
	// ASN rows appear after detected rows.
	assert.Greater(t, strings.Index(out, "GOOGLE"), strings.Index(out, "CloudFront"),
		"ASN rows should appear after detected rows")
}
