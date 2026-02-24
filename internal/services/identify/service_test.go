package identify_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/identify"
	"github.com/tbckr/trident/internal/testutil"
)

func TestRun_CDNDetected(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run([]string{"abc.cloudfront.net."}, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "CDN", result.Detections[0].Type)
	assert.Equal(t, "AWS CloudFront", result.Detections[0].Provider)
	assert.Equal(t, "abc.cloudfront.net.", result.Detections[0].Evidence)
}

func TestRun_EmailDetected(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run(nil, []string{"aspmx.l.google.com."}, nil, nil)
	require.NoError(t, err)
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "Email", result.Detections[0].Type)
	assert.Equal(t, "Google Workspace", result.Detections[0].Provider)
}

func TestRun_DNSDetected(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run(nil, nil, []string{"ns-123.awsdns-45.com."}, nil)
	require.NoError(t, err)
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "DNS", result.Detections[0].Type)
	assert.Equal(t, "AWS Route 53", result.Detections[0].Provider)
}

func TestRun_TXTEmailDetected(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run(nil, nil, nil, []string{"v=spf1 include:_spf.google.com ~all"})
	require.NoError(t, err)
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "Email", result.Detections[0].Type)
	assert.Equal(t, "Google Workspace", result.Detections[0].Provider)
}

func TestRun_TXTVerificationDetected(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run(nil, nil, nil, []string{"google-site-verification=abc123"})
	require.NoError(t, err)
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "Verification", result.Detections[0].Type)
	assert.Equal(t, "Google", result.Detections[0].Provider)
}

func TestRun_MultipleTypes(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run(
		[]string{"abc.cloudfront.net."},
		[]string{"aspmx.l.google.com."},
		[]string{"diana.ns.cloudflare.com."},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, result.Detections, 3)
}

func TestRun_NoDetections(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run(
		[]string{"unknown.example.invalid."},
		[]string{"mail.unknown.invalid."},
		[]string{"ns1.unknown.invalid."},
		nil,
	)
	require.NoError(t, err)
	assert.True(t, result.IsEmpty())
}

func TestRun_EmptyInputs(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	result, err := svc.Run(nil, nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, result.IsEmpty())
}

func TestService_PAP(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	assert.Equal(t, "red", svc.PAP().String())
}

func TestService_Name(t *testing.T) {
	svc := identify.NewService(testutil.NopLogger())
	assert.Equal(t, "identify", svc.Name())
}
