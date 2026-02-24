package detect_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/detect"
	"github.com/tbckr/trident/internal/testutil"
)

// embeddedPatterns loads the embedded defaults for use in service tests.
func embeddedPatterns(t *testing.T) providers.Patterns {
	t.Helper()
	p, err := providers.LoadPatterns()
	require.NoError(t, err)
	return p
}

func TestRun_CDNDetected(t *testing.T) {
	r := &testutil.MockResolver{
		LookupCNAMEFn: func(_ context.Context, host string) (string, error) {
			if host == "example.com" {
				return "example.cloudfront.net.", nil
			}
			// www.example.com → identity, no alias
			return host + ".", nil
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "CDN", result.Detections[0].Type)
	assert.Equal(t, "AWS CloudFront", result.Detections[0].Provider)
	assert.Equal(t, "example.cloudfront.net.", result.Detections[0].Evidence)
	assert.Equal(t, "cname", result.Detections[0].Source)
}

func TestRun_EmailDetected(t *testing.T) {
	r := &testutil.MockResolver{
		LookupMXFn: func(_ context.Context, _ string) ([]*net.MX, error) {
			return []*net.MX{
				{Host: "aspmx.l.google.com.", Pref: 1},
			}, nil
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "Email", result.Detections[0].Type)
	assert.Equal(t, "Google Workspace", result.Detections[0].Provider)
	assert.Equal(t, "mx", result.Detections[0].Source)
}

func TestRun_DNSDetected(t *testing.T) {
	r := &testutil.MockResolver{
		LookupNSFn: func(_ context.Context, _ string) ([]*net.NS, error) {
			return []*net.NS{
				{Host: "ns-123.awsdns-45.com."},
			}, nil
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "DNS", result.Detections[0].Type)
	assert.Equal(t, "AWS Route 53", result.Detections[0].Provider)
	assert.Equal(t, "ns", result.Detections[0].Source)
}

func TestRun_NoDetections(t *testing.T) {
	r := &testutil.MockResolver{
		LookupCNAMEFn: func(_ context.Context, host string) (string, error) {
			// identity — no alias
			return host + ".", nil
		},
		LookupMXFn: func(_ context.Context, _ string) ([]*net.MX, error) {
			return []*net.MX{{Host: "mail.unknown-provider.com.", Pref: 10}}, nil
		},
		LookupNSFn: func(_ context.Context, _ string) ([]*net.NS, error) {
			return []*net.NS{{Host: "ns1.unknown-provider.com."}}, nil
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	assert.True(t, result.IsEmpty())
}

func TestRun_TXTEmailDetected(t *testing.T) {
	r := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"v=spf1 include:_spf.google.com ~all"}, nil
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "Email", result.Detections[0].Type)
	assert.Equal(t, "Google Workspace", result.Detections[0].Provider)
	assert.Equal(t, "txt", result.Detections[0].Source)
}

func TestRun_TXTVerificationDetected(t *testing.T) {
	r := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"google-site-verification=abc123"}, nil
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	require.Len(t, result.Detections, 1)
	assert.Equal(t, "Verification", result.Detections[0].Type)
	assert.Equal(t, "Google", result.Detections[0].Provider)
	assert.Equal(t, "txt", result.Detections[0].Source)
}

func TestRun_TXTLookupError(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	r := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, lookupErr
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	assert.True(t, result.IsEmpty())
}

func TestRun_InvalidInput(t *testing.T) {
	svc := detect.NewService(&testutil.MockResolver{}, testutil.NopLogger(), embeddedPatterns(t))

	for _, bad := range []string{"", "not_a_domain", "has space.com", "$(injection)"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestRun_LookupErrors(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	r := &testutil.MockResolver{
		LookupCNAMEFn: func(_ context.Context, _ string) (string, error) {
			return "", lookupErr
		},
		LookupMXFn: func(_ context.Context, _ string) ([]*net.MX, error) {
			return nil, lookupErr
		},
		LookupNSFn: func(_ context.Context, _ string) ([]*net.NS, error) {
			return nil, lookupErr
		},
	}

	svc := detect.NewService(r, testutil.NopLogger(), embeddedPatterns(t))
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*detect.Result)
	require.True(t, ok, "expected *detect.Result")
	assert.True(t, result.IsEmpty())
}

func TestService_PAP(t *testing.T) {
	svc := detect.NewService(&testutil.MockResolver{}, testutil.NopLogger(), embeddedPatterns(t))
	assert.Equal(t, "green", svc.PAP().String())
}
