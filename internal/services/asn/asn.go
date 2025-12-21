package asn

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
)

type ASNResult struct {
	ASN         string
	Description string
	Country     string
	Registry    string
	Prefixes    []string
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

func (s *Service) Lookup(ctx context.Context, target string) (*ASNResult, error) {
	// Parse as IP
	ip := net.ParseIP(target)
	if ip != nil {
		return s.lookupIP(ctx, target)
	}

	// Parse as ASN
	if strings.HasPrefix(strings.ToUpper(target), "AS") {
		return s.lookupASN(ctx, target)
	}

	return nil, fmt.Errorf("invalid target: %s", target)
}

func (s *Service) lookupIP(ctx context.Context, ipStr string) (*ASNResult, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP: %s", ipStr)
	}

	// Reverse IP for Team Cymru DNS query
	var reversedIP string
	if ipv4 := ip.To4(); ipv4 != nil {
		reversedIP = fmt.Sprintf("%d.%d.%d.%d", ipv4[3], ipv4[2], ipv4[1], ipv4[0])
	} else {
		// IPv6 not implemented yet for Team Cymru DNS TXT
		return nil, fmt.Errorf("IPv6 not supported for ASN lookup via DNS")
	}

	query := fmt.Sprintf("%s.origin.asn.cymru.com", reversedIP)
	txts, err := s.resolver.LookupTXT(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ASN origin lookup failed: %w", err)
	}

	if len(txts) == 0 {
		return nil, fmt.Errorf("no ASN info found for IP: %s", ipStr)
	}

	// team cymru format: "AS | IP | Prefix | Country | Registry | Date"
	parts := strings.Split(txts[0], "|")
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected TXT response format: %s", txts[0])
	}

	asn := strings.TrimSpace(parts[0])

	return s.lookupASN(ctx, "AS"+asn)
}

func (s *Service) lookupASN(ctx context.Context, asn string) (*ASNResult, error) {
	asnNum := strings.TrimPrefix(strings.ToUpper(asn), "AS")
	query := fmt.Sprintf("AS%s.asn.cymru.com", asnNum)
	txts, err := s.resolver.LookupTXT(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ASN details lookup failed: %w", err)
	}

	if len(txts) == 0 {
		return nil, fmt.Errorf("no details found for ASN: %s", asn)
	}

	// team cymru format: "AS | Country | Registry | Date | Description"
	parts := strings.Split(txts[0], "|")
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected TXT response format: %s", txts[0])
	}

	return &ASNResult{
		ASN:         asn,
		Country:     strings.TrimSpace(parts[1]),
		Registry:    strings.TrimSpace(parts[2]),
		Description: strings.TrimSpace(parts[4]),
	}, nil
}
