package detect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/detect"
)

// allDNSPatterns is the full set of DNS patterns used across DNS tests.
var allDNSPatterns = []detect.DNSPattern{
	{Suffix: "ns.cloudflare.com", Provider: "Cloudflare DNS"},
	{Contains: "awsdns", Provider: "AWS Route 53"},
	{Suffix: "azure-dns.com", Provider: "Azure DNS"},
	{Suffix: "azure-dns.net", Provider: "Azure DNS"},
	{Suffix: "azure-dns.org", Provider: "Azure DNS"},
	{Suffix: "azure-dns.info", Provider: "Azure DNS"},
	{Suffix: "googledomains.com", Provider: "Google Cloud DNS"},
	{Suffix: "nsone.net", Provider: "NS1"},
	{Suffix: "dnsimple.com", Provider: "DNSimple"},
	{Suffix: "ultradns.net", Provider: "UltraDNS"},
	{Suffix: "ultradns.com", Provider: "UltraDNS"},
	{Suffix: "dnsmadeeasy.com", Provider: "DNS Made Easy"},
	{Suffix: "cloudns.net", Provider: "ClouDNS"},
	{Suffix: "domaincontrol.com", Provider: "GoDaddy"},
	{Suffix: "registrar-servers.com", Provider: "Namecheap"},
}

func newDNSDetector() *detect.Detector {
	return detect.NewDetector(detect.Patterns{DNS: allDNSPatterns})
}

func TestDetectDNSHost_CloudflareDNS(t *testing.T) {
	detections := newDNSDetector().DNSHost([]string{"liz.ns.cloudflare.com."})
	require.Len(t, detections, 1)
	assert.Equal(t, detect.TypeDNS, detections[0].Type)
	assert.Equal(t, "Cloudflare DNS", detections[0].Provider)
	assert.Equal(t, "liz.ns.cloudflare.com.", detections[0].Evidence)
	assert.Equal(t, "ns", detections[0].Source)
}

func TestDetectDNSHost_AWSRoute53(t *testing.T) {
	// AWS Route 53 NS names contain "awsdns" as a substring.
	detections := newDNSDetector().DNSHost([]string{"ns-123.awsdns-45.com."})
	require.Len(t, detections, 1)
	assert.Equal(t, "AWS Route 53", detections[0].Provider)
	assert.Equal(t, "ns-123.awsdns-45.com.", detections[0].Evidence)
}

func TestDetectDNSHost_UnknownHost(t *testing.T) {
	assert.Empty(t, newDNSDetector().DNSHost([]string{"ns1.unknown-dns.example."}))
}

func TestDetectDNSHost_EmptyInput(t *testing.T) {
	d := newDNSDetector()
	assert.Empty(t, d.DNSHost(nil))
	assert.Empty(t, d.DNSHost([]string{}))
}

func TestDetectDNSHost_KnownProviders(t *testing.T) {
	tests := []struct {
		host     string
		provider string
	}{
		{"liz.ns.cloudflare.com.", "Cloudflare DNS"},
		{"ns-123.awsdns-45.com.", "AWS Route 53"},
		{"ns1-01.azure-dns.com.", "Azure DNS"},
		{"ns1.azure-dns.net.", "Azure DNS"},
		{"ns1.azure-dns.org.", "Azure DNS"},
		{"ns1.azure-dns.info.", "Azure DNS"},
		{"ns1.googledomains.com.", "Google Cloud DNS"},
		{"dns1.p01.nsone.net.", "NS1"},
		{"ns1.dnsimple.com.", "DNSimple"},
		{"pdns1.ultradns.net.", "UltraDNS"},
		{"udns1.ultradns.com.", "UltraDNS"},
		{"ns1.dnsmadeeasy.com.", "DNS Made Easy"},
		{"ns1.cloudns.net.", "ClouDNS"},
		{"ns1.domaincontrol.com.", "GoDaddy"},
		{"dns1.registrar-servers.com.", "Namecheap"},
	}
	d := newDNSDetector()
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			detections := d.DNSHost([]string{tt.host})
			require.Len(t, detections, 1)
			assert.Equal(t, tt.provider, detections[0].Provider)
		})
	}
}

func TestDetectDNSHost_DuplicateSuppression(t *testing.T) {
	hosts := []string{"liz.ns.cloudflare.com.", "liz.ns.cloudflare.com."}
	detections := newDNSDetector().DNSHost(hosts)
	require.Len(t, detections, 1)
	assert.Equal(t, "Cloudflare DNS", detections[0].Provider)
}
