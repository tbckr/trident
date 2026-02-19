package dns_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/dns"
	"github.com/tbckr/trident/internal/testutil"
)

func TestRun_ValidDomain(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupIPAddrFn: func(_ context.Context, _ string) ([]net.IPAddr, error) {
			return []net.IPAddr{
				{IP: net.ParseIP("93.184.216.34")},
				{IP: net.ParseIP("2606:2800:21f:cb07:6820:80da:af6b:8b2c")},
			}, nil
		},
		LookupMXFn: func(_ context.Context, _ string) ([]*net.MX, error) {
			return []*net.MX{{Host: "mail.example.com.", Pref: 10}}, nil
		},
		LookupNSFn: func(_ context.Context, _ string) ([]*net.NS, error) {
			return []*net.NS{{Host: "ns1.example.com."}, {Host: "ns2.example.com."}}, nil
		},
		LookupTXTFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"v=spf1 -all"}, nil
		},
	}

	svc := dns.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*dns.Result)
	require.True(t, ok, "expected *dns.Result")

	assert.Equal(t, "example.com", result.Input)
	assert.Equal(t, []string{"93.184.216.34"}, result.A)
	assert.Len(t, result.AAAA, 1)
	assert.Equal(t, []string{"mail.example.com."}, result.MX)
	assert.Equal(t, []string{"ns1.example.com.", "ns2.example.com."}, result.NS)
	assert.Equal(t, []string{"v=spf1 -all"}, result.TXT)
	assert.Nil(t, result.PTR)
}

func TestRun_IP(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupAddrFn: func(_ context.Context, addr string) ([]string, error) {
			assert.Equal(t, "8.8.8.8", addr)
			return []string{"dns.google."}, nil
		},
	}

	svc := dns.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "8.8.8.8")
	require.NoError(t, err)

	result, ok := raw.(*dns.Result)
	require.True(t, ok)

	assert.Equal(t, "8.8.8.8", result.Input)
	assert.Equal(t, []string{"dns.google."}, result.PTR)
	assert.Nil(t, result.A)
}

func TestRun_InvalidInput(t *testing.T) {
	svc := dns.NewService(&testutil.MockResolver{}, testutil.NopLogger())

	for _, bad := range []string{"", "not_a_domain", "has space.com", "$(injection)"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestRun_ContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resolver := &testutil.MockResolver{
		LookupIPAddrFn: func(ctx context.Context, _ string) ([]net.IPAddr, error) {
			return nil, ctx.Err()
		},
	}
	svc := dns.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(ctx, "example.com")
	require.NoError(t, err) // partial results are OK
	result, ok := raw.(*dns.Result)
	require.True(t, ok, "expected *dns.Result")
	assert.Nil(t, result.A)
}

func TestRun_PartialFailure(t *testing.T) {
	dnsErr := errors.New("lookup failed")
	resolver := &testutil.MockResolver{
		LookupIPAddrFn: func(_ context.Context, _ string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("1.2.3.4")}}, nil
		},
		LookupMXFn: func(_ context.Context, _ string) ([]*net.MX, error) {
			return nil, dnsErr
		},
		LookupNSFn: func(_ context.Context, _ string) ([]*net.NS, error) {
			return nil, dnsErr
		},
		LookupTXTFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, dnsErr
		},
	}

	svc := dns.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*dns.Result)
	require.True(t, ok, "expected *dns.Result")
	assert.Equal(t, []string{"1.2.3.4"}, result.A)
	assert.Nil(t, result.MX)
	assert.Nil(t, result.NS)
	assert.Nil(t, result.TXT)
}

func TestRun_ANSISanitization(t *testing.T) {
	resolver := &testutil.MockResolver{
		LookupTXTFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"\x1b[31mmalicious\x1b[0m"}, nil
		},
	}

	svc := dns.NewService(resolver, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*dns.Result)
	require.True(t, ok, "expected *dns.Result")
	assert.Equal(t, []string{"malicious"}, result.TXT)
}

func TestResult_WriteText(t *testing.T) {
	result := &dns.Result{
		Input: "example.com",
		A:     []string{"1.2.3.4"},
		MX:    []string{"mail.example.com."},
		TXT:   []string{"v=spf1 -all"},
	}

	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "mail.example.com.")
}
