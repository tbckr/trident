package services

import (
	"context"
	"net"
)

// DNSResolverInterface abstracts net.Resolver for the dns and detect services.
// *net.Resolver satisfies this interface directly. The cymru and apex services
// need only LookupTXT and depend on cymru.TXTResolver instead.
// NOTE: No HTTP interface — *req.Client is a hard dependency for crt.sh, not abstracted.
type DNSResolverInterface interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupNS(ctx context.Context, name string) ([]*net.NS, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupAddr(ctx context.Context, addr string) ([]string, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
}
