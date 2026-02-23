package cymru_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/cymru"
	"github.com/tbckr/trident/internal/testutil"
)

func TestRun_IPv4(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, host string) ([]string, error) {
			switch host {
			case "8.8.8.8.origin.asn.cymru.com":
				return []string{"15169 | 8.8.8.0/24 | US | arin | 1992-12-01"}, nil
			case "AS15169.asn.cymru.com":
				return []string{"15169 | US | arin | 2000-03-30 | GOOGLE, US"}, nil
			}
			return nil, errors.New("unexpected host")
		},
	}

	svc := cymru.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "8.8.8.8")
	require.NoError(t, err)

	result, ok := raw.(*cymru.Result)
	require.True(t, ok)

	assert.Equal(t, "8.8.8.8", result.Input)
	assert.Equal(t, "AS15169", result.ASN)
	assert.Equal(t, "8.8.8.0/24", result.Prefix)
	assert.Equal(t, "US", result.Country)
	assert.Equal(t, "arin", result.Registry)
	assert.Equal(t, "GOOGLE, US", result.Description)
}

func TestRun_ASN(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, host string) ([]string, error) {
			if host == "AS15169.asn.cymru.com" {
				return []string{"15169 | US | arin | 2000-03-30 | GOOGLE, US"}, nil
			}
			return nil, errors.New("unexpected host")
		},
	}

	svc := cymru.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "AS15169")
	require.NoError(t, err)

	result, ok := raw.(*cymru.Result)
	require.True(t, ok)

	assert.Equal(t, "AS15169", result.Input)
	assert.Equal(t, "AS15169", result.ASN)
	assert.Equal(t, "GOOGLE, US", result.Description)
}

func TestRun_ASN_LowercaseInput(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, host string) ([]string, error) {
			if host == "AS15169.asn.cymru.com" {
				return []string{"15169 | US | arin | 2000-03-30 | GOOGLE, US"}, nil
			}
			return nil, errors.New("unexpected host")
		},
	}

	svc := cymru.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "as15169")
	require.NoError(t, err)
	result, ok := raw.(*cymru.Result)
	require.True(t, ok, "expected *cymru.Result")
	assert.Equal(t, "AS15169", result.ASN)
}

func TestRun_InvalidInput(t *testing.T) {
	svc := cymru.NewService(&testutil.MockResolver{}, testutil.NopLogger())
	for _, bad := range []string{"", "notanip", "AS", "AS_BAD", "example.com"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestRun_LookupFailure(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("DNS failure")
		},
	}

	svc := cymru.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "8.8.8.8")
	require.NoError(t, err)
	result, ok := raw.(*cymru.Result)
	require.True(t, ok, "expected *cymru.Result")
	assert.Equal(t, "8.8.8.8", result.Input)
	assert.Empty(t, result.ASN)
	assert.True(t, result.IsEmpty())
}

func TestRun_ANSISanitization(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, host string) ([]string, error) {
			if host == "AS15169.asn.cymru.com" {
				return []string{"15169 | US | arin | 2000-03-30 | \x1b[31mGOOGLE\x1b[0m, US"}, nil
			}
			return nil, nil
		},
	}

	svc := cymru.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "AS15169")
	require.NoError(t, err)
	result, ok := raw.(*cymru.Result)
	require.True(t, ok, "expected *cymru.Result")
	assert.Equal(t, "GOOGLE, US", result.Description)
}

func TestRun_IPv6(t *testing.T) {
	// 2001:4860:4860::8888 nibbles reversed:
	// Full: 2001:4860:4860:0000:0000:0000:0000:8888
	// Reversed nibble form: 8.8.8.8.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.6.8.4.0.6.8.4.1.0.0.2
	resolver := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, host string) ([]string, error) {
			if host == "8.8.8.8.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.6.8.4.0.6.8.4.1.0.0.2.origin6.asn.cymru.com" {
				return []string{"15169 | 2001:4860::/32 | US | arin | 2005-03-14"}, nil
			}
			if host == "AS15169.asn.cymru.com" {
				return []string{"15169 | US | arin | 2000-03-30 | GOOGLE, US"}, nil
			}
			return nil, errors.New("unexpected host: " + host)
		},
	}

	svc := cymru.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "2001:4860:4860::8888")
	require.NoError(t, err)

	result, ok := raw.(*cymru.Result)
	require.True(t, ok)

	assert.Equal(t, "AS15169", result.ASN)
	assert.Equal(t, "2001:4860::/32", result.Prefix)
}

func TestService_PAP(t *testing.T) {
	svc := cymru.NewService(&testutil.MockResolver{}, testutil.NopLogger())
	assert.Equal(t, "amber", svc.PAP().String())
}
