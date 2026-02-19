// Package testutil provides shared test helpers for service unit tests.
package testutil

import (
	"context"
	"io"
	"log/slog"
	"net"

	"github.com/tbckr/trident/internal/services"
)

// MockResolver implements services.DNSResolverInterface for testing.
// Each field is a function so tests can set only the methods they need.
type MockResolver struct {
	LookupIPAddrFn func(ctx context.Context, host string) ([]net.IPAddr, error)
	LookupMXFn     func(ctx context.Context, name string) ([]*net.MX, error)
	LookupNSFn     func(ctx context.Context, name string) ([]*net.NS, error)
	LookupTXTFn    func(ctx context.Context, name string) ([]string, error)
	LookupAddrFn   func(ctx context.Context, addr string) ([]string, error)
}

var _ services.DNSResolverInterface = (*MockResolver)(nil)

// LookupIPAddr implements DNSResolverInterface.
func (m *MockResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	if m.LookupIPAddrFn != nil {
		return m.LookupIPAddrFn(ctx, host)
	}
	return nil, nil
}

// LookupMX implements DNSResolverInterface.
func (m *MockResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	if m.LookupMXFn != nil {
		return m.LookupMXFn(ctx, name)
	}
	return nil, nil
}

// LookupNS implements DNSResolverInterface.
func (m *MockResolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	if m.LookupNSFn != nil {
		return m.LookupNSFn(ctx, name)
	}
	return nil, nil
}

// LookupTXT implements DNSResolverInterface.
func (m *MockResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	if m.LookupTXTFn != nil {
		return m.LookupTXTFn(ctx, name)
	}
	return nil, nil
}

// LookupAddr implements DNSResolverInterface.
func (m *MockResolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	if m.LookupAddrFn != nil {
		return m.LookupAddrFn(ctx, addr)
	}
	return nil, nil
}

// NopLogger returns a logger that discards all output.
func NopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
