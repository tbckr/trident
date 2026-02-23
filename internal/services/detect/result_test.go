package detect_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/detect"
)

func TestResult_IsEmpty(t *testing.T) {
	assert.True(t, (&detect.Result{}).IsEmpty())
	assert.False(t, (&detect.Result{
		Detections: []detect.Detection{{Type: "CDN", Provider: "AWS CloudFront", Evidence: "foo.cloudfront.net."}},
	}).IsEmpty())
}

func TestResult_WriteText(t *testing.T) {
	result := &detect.Result{
		Input: "example.com",
		Detections: []detect.Detection{
			{Type: "CDN", Provider: "AWS CloudFront", Evidence: "foo.cloudfront.net."},
		},
	}

	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	assert.Equal(t, "CDN AWS CloudFront (cname: foo.cloudfront.net.)\n", buf.String())
}

func TestResult_WriteText_AllTypes(t *testing.T) {
	result := &detect.Result{
		Input: "example.com",
		Detections: []detect.Detection{
			{Type: "CDN", Provider: "AWS CloudFront", Evidence: "foo.cloudfront.net."},
			{Type: "Email", Provider: "Google Workspace", Evidence: "aspmx.l.google.com."},
			{Type: "DNS", Provider: "AWS Route 53", Evidence: "ns-1.awsdns-1.com."},
		},
	}

	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "CDN AWS CloudFront (cname: foo.cloudfront.net.)")
	assert.Contains(t, out, "Email Google Workspace (mx: aspmx.l.google.com.)")
	assert.Contains(t, out, "DNS AWS Route 53 (ns: ns-1.awsdns-1.com.)")
}

func TestResult_WriteTable(t *testing.T) {
	result := &detect.Result{
		Input: "example.com",
		Detections: []detect.Detection{
			{Type: "CDN", Provider: "AWS CloudFront", Evidence: "foo.cloudfront.net."},
		},
	}

	var buf bytes.Buffer
	err := result.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "PROVIDER")
	assert.Contains(t, out, "EVIDENCE")
	assert.Contains(t, out, "CDN")
	assert.Contains(t, out, "AWS CloudFront")
	assert.Contains(t, out, "foo.cloudfront.net.")
}
