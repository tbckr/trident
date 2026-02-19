// Package services defines shared interfaces and types used across service implementations.
package services

import (
	"context"
	"net"
)

// DNSResolverInterface abstracts net.Resolver for DNS and ASN lookups.
// *net.Resolver satisfies this interface directly.
// NOTE: No HTTP interface — *req.Client is a hard dependency for crt.sh, not abstracted.
type DNSResolverInterface interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupNS(ctx context.Context, name string) ([]*net.NS, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupAddr(ctx context.Context, addr string) ([]string, error)
}
