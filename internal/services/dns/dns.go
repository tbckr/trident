package dns

import (
	"context"
	"log/slog"
	"net"
)

type DNSResult struct {
	A     []string
	AAAA  []string
	MX    []string
	NS    []string
	TXT   []string
	CNAME string
}

type Service struct {
	logger   *slog.Logger
	resolver *net.Resolver
}

func NewService(logger *slog.Logger, resolver *net.Resolver) *Service {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &Service{
		logger:   logger,
		resolver: resolver,
	}
}

func (s *Service) Lookup(ctx context.Context, target string) (*DNSResult, error) {
	result := &DNSResult{}

	// A records
	ips, err := s.resolver.LookupHost(ctx, target)
	if err == nil {
		for _, ip := range ips {
			if net.ParseIP(ip).To4() != nil {
				result.A = append(result.A, ip)
			} else {
				result.AAAA = append(result.AAAA, ip)
			}
		}
	}

	// MX records
	mxs, err := s.resolver.LookupMX(ctx, target)
	if err == nil {
		for _, mx := range mxs {
			result.MX = append(result.MX, mx.Host)
		}
	}

	// NS records
	nss, err := s.resolver.LookupNS(ctx, target)
	if err == nil {
		for _, ns := range nss {
			result.NS = append(result.NS, ns.Host)
		}
	}

	// TXT records
	txts, err := s.resolver.LookupTXT(ctx, target)
	if err == nil {
		result.TXT = txts
	}

	// CNAME
	cname, err := s.resolver.LookupCNAME(ctx, target)
	if err == nil && cname != target+"." {
		result.CNAME = cname
	}

	return result, nil
}
