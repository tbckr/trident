package detect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/detect"
)

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
	for _, tt := range tests {
		t.Run(tt.cname, func(t *testing.T) {
			detections := detect.CDN([]string{tt.cname})
			require.Len(t, detections, 1)
			assert.Equal(t, detect.TypeCDN, detections[0].Type)
			assert.Equal(t, tt.provider, detections[0].Provider)
			assert.Equal(t, tt.cname, detections[0].Evidence)
		})
	}
}

func TestDetectCDN_UnknownSuffix(t *testing.T) {
	detections := detect.CDN([]string{"foo.unknown-cdn.example.com."})
	assert.Empty(t, detections)
}

func TestDetectCDN_EmptyInput(t *testing.T) {
	assert.Empty(t, detect.CDN(nil))
	assert.Empty(t, detect.CDN([]string{}))
}

func TestDetectCDN_DuplicateSuppression(t *testing.T) {
	// Same CNAME twice — should produce only one detection.
	detections := detect.CDN([]string{"abc.cloudfront.net.", "abc.cloudfront.net."})
	require.Len(t, detections, 1)
	assert.Equal(t, "AWS CloudFront", detections[0].Provider)
}

func TestDetectCDN_MultipleProviders(t *testing.T) {
	cnames := []string{"abc.cloudfront.net.", "edge.akamaiedge.net."}
	detections := detect.CDN(cnames)
	require.Len(t, detections, 2)
	providers := []string{detections[0].Provider, detections[1].Provider}
	assert.Contains(t, providers, "AWS CloudFront")
	assert.Contains(t, providers, "Akamai")
}
