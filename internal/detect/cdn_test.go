package detect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/detect"
)

// allCDNPatterns is the full set of CDN patterns used across CDN tests.
var allCDNPatterns = []detect.CDNPattern{
	{Suffix: "cloudfront.net", Provider: "AWS CloudFront"},
	{Suffix: "akamaiedge.net", Provider: "Akamai"},
	{Suffix: "edgekey.net", Provider: "Akamai"},
	{Suffix: "edgesuite.net", Provider: "Akamai"},
	{Suffix: "fastly.net", Provider: "Fastly"},
	{Suffix: "cloudflare.net", Provider: "Cloudflare"},
	{Suffix: "azureedge.net", Provider: "Azure CDN"},
	{Suffix: "azurefd.net", Provider: "Azure Front Door"},
	{Suffix: "googleplex.com", Provider: "Google Cloud CDN"},
	{Suffix: "l.google.com", Provider: "Google Cloud CDN"},
	{Suffix: "b-cdn.net", Provider: "Bunny CDN"},
	{Suffix: "incapdns.net", Provider: "Imperva"},
	{Suffix: "sucuri.net", Provider: "Sucuri"},
	{Suffix: "stackpathcdn.com", Provider: "StackPath"},
	{Suffix: "netdna-cdn.com", Provider: "StackPath"},
	{Suffix: "llnwd.net", Provider: "Edgio"},
	{Suffix: "edgio.net", Provider: "Edgio"},
	{Suffix: "cdn77.org", Provider: "CDN77"},
	{Suffix: "kxcdn.com", Provider: "KeyCDN"},
	{Suffix: "edgecastcdn.net", Provider: "Verizon EdgeCast"},
	{Suffix: "cachefly.net", Provider: "CacheFly"},
	{Suffix: "gcdn.co", Provider: "G-Core"},
	{Suffix: "alikunlun.com", Provider: "Alibaba Cloud CDN"},
}

func newCDNDetector() *detect.Detector {
	return detect.NewDetector(detect.Patterns{CDN: allCDNPatterns})
}

func TestDetectCDN_KnownProviders(t *testing.T) {
	tests := []struct {
		cname    string
		provider string
	}{
		{"abc.cloudfront.net.", "AWS CloudFront"},
		{"abc.cloudfront.net", "AWS CloudFront"},
		{"edge.akamaiedge.net.", "Akamai"},
		{"edge.edgekey.net.", "Akamai"},
		{"x.edgesuite.net.", "Akamai"},
		{"cache.fastly.net.", "Fastly"},
		{"x.cloudflare.net.", "Cloudflare"},
		{"foo.azureedge.net.", "Azure CDN"},
		{"foo.azurefd.net.", "Azure Front Door"},
		{"foo.googleplex.com.", "Google Cloud CDN"},
		{"foo.l.google.com.", "Google Cloud CDN"},
		{"foo.b-cdn.net.", "Bunny CDN"},
		{"foo.incapdns.net.", "Imperva"},
		{"foo.sucuri.net.", "Sucuri"},
		{"foo.stackpathcdn.com.", "StackPath"},
		{"foo.netdna-cdn.com.", "StackPath"},
		{"foo.llnwd.net.", "Edgio"},
		{"foo.edgio.net.", "Edgio"},
		{"foo.cdn77.org.", "CDN77"},
		{"foo.kxcdn.com.", "KeyCDN"},
		{"foo.edgecastcdn.net.", "Verizon EdgeCast"},
		{"foo.cachefly.net.", "CacheFly"},
		{"foo.gcdn.co.", "G-Core"},
		{"foo.alikunlun.com.", "Alibaba Cloud CDN"},
	}
	d := newCDNDetector()
	for _, tt := range tests {
		t.Run(tt.cname, func(t *testing.T) {
			detections := d.CDN([]string{tt.cname})
			require.Len(t, detections, 1)
			assert.Equal(t, detect.TypeCDN, detections[0].Type)
			assert.Equal(t, tt.provider, detections[0].Provider)
			assert.Equal(t, tt.cname, detections[0].Evidence)
			assert.Equal(t, "cname", detections[0].Source)
		})
	}
}

func TestDetectCDN_UnknownSuffix(t *testing.T) {
	detections := newCDNDetector().CDN([]string{"foo.unknown-cdn.example.com."})
	assert.Empty(t, detections)
}

func TestDetectCDN_EmptyInput(t *testing.T) {
	d := newCDNDetector()
	assert.Empty(t, d.CDN(nil))
	assert.Empty(t, d.CDN([]string{}))
}

func TestDetectCDN_DuplicateSuppression(t *testing.T) {
	// Same CNAME twice — should produce only one detection.
	detections := newCDNDetector().CDN([]string{"abc.cloudfront.net.", "abc.cloudfront.net."})
	require.Len(t, detections, 1)
	assert.Equal(t, "AWS CloudFront", detections[0].Provider)
}

func TestDetectCDN_MultipleProviders(t *testing.T) {
	cnames := []string{"abc.cloudfront.net.", "edge.akamaiedge.net."}
	detections := newCDNDetector().CDN(cnames)
	require.Len(t, detections, 2)
	providers := []string{detections[0].Provider, detections[1].Provider}
	assert.Contains(t, providers, "AWS CloudFront")
	assert.Contains(t, providers, "Akamai")
}
