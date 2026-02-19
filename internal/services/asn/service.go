// Package asn implements ASN lookups via the Team Cymru DNS service.
package asn

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services"
)

// cymruIPv4Template is the Team Cymru DNS suffix for IPv4 → ASN lookups.
const cymruIPv4Template = "%s.origin.asn.cymru.com"

// cymruIPv6Template is the Team Cymru DNS suffix for IPv6 → ASN lookups.
const cymruIPv6Template = "%s.origin6.asn.cymru.com"

// cymruASNTemplate is the Team Cymru DNS suffix for ASN → info lookups.
// Format: AS<number>.asn.cymru.com
const cymruASNTemplate = "%s.asn.cymru.com"

// Service performs ASN lookups via the Team Cymru DNS service.
type Service struct {
	resolver services.DNSResolverInterface
	logger   *slog.Logger
}

// NewService creates a new ASN service.
func NewService(resolver services.DNSResolverInterface, logger *slog.Logger) *Service {
	return &Service{resolver: resolver, logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "asn" }

// Result holds the ASN lookup result for a single IP or ASN input.
type Result struct {
	Input       string `json:"input"`
	ASN         string `json:"asn,omitempty"`
	Prefix      string `json:"prefix,omitempty"`
	Country     string `json:"country,omitempty"`
	Registry    string `json:"registry,omitempty"`
	Description string `json:"description,omitempty"`
}

// WriteText renders the result as an ASCII table.
func (r *Result) WriteText(w io.Writer) error {
	rows := [][]string{
		{"ASN", r.ASN},
		{"Prefix", r.Prefix},
		{"Country", r.Country},
		{"Registry", r.Registry},
		{"Description", r.Description},
	}
	table := output.NewWrappingTable(w, 20, 20)
	table.Header([]string{"Field", "Value"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}

// Run performs an ASN lookup for the given IP address or ASN string.
// IP input  → reverse-maps via Team Cymru to find the originating ASN.
// ASN input → queries Team Cymru for the ASN's description.
func (s *Service) Run(ctx context.Context, input string) (any, error) {
	result := &Result{Input: output.StripANSI(input)}

	if net.ParseIP(input) != nil {
		return s.lookupByIP(ctx, result, input)
	}
	upper := strings.ToUpper(input)
	if strings.HasPrefix(upper, "AS") {
		num := upper[2:]
		if num == "" {
			return nil, fmt.Errorf("%w: must be an IP address or ASN (e.g. AS15169): %q", services.ErrInvalidInput, input)
		}
		for _, c := range num {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("%w: must be an IP address or ASN (e.g. AS15169): %q", services.ErrInvalidInput, input)
			}
		}
		return s.lookupByASN(ctx, result, upper)
	}
	return nil, fmt.Errorf("%w: must be an IP address or ASN (e.g. AS15169): %q", services.ErrInvalidInput, input)
}

// lookupByIP resolves ASN info for the given IP using Team Cymru's DNS service.
func (s *Service) lookupByIP(ctx context.Context, result *Result, ip string) (*Result, error) {
	reversed, isV6, err := reverseIP(ip)
	if err != nil {
		return nil, fmt.Errorf("reversing IP: %w", err)
	}
	tmpl := cymruIPv4Template
	if isV6 {
		tmpl = cymruIPv6Template
	}
	query := fmt.Sprintf(tmpl, reversed)
	txts, err := s.resolver.LookupTXT(ctx, query)
	if err != nil {
		s.logger.Debug("Cymru IP lookup failed", "ip", ip, "error", err)
		return result, nil
	}
	if len(txts) > 0 {
		parseCymruIPRecord(result, txts[0])
	}
	if result.ASN != "" {
		s.enrichASN(ctx, result, result.ASN)
	}
	return result, nil
}

// lookupByASN fetches description info for the given ASN from Team Cymru.
func (s *Service) lookupByASN(ctx context.Context, result *Result, asn string) (*Result, error) {
	result.ASN = asn
	query := fmt.Sprintf(cymruASNTemplate, asn)
	txts, err := s.resolver.LookupTXT(ctx, query)
	if err != nil {
		s.logger.Debug("Cymru ASN lookup failed", "asn", asn, "error", err)
		return result, nil
	}
	if len(txts) > 0 {
		parseCymruASNRecord(result, txts[0])
	}
	return result, nil
}

// enrichASN fetches the description for an already-known ASN.
func (s *Service) enrichASN(ctx context.Context, result *Result, asn string) {
	query := fmt.Sprintf(cymruASNTemplate, asn)
	txts, err := s.resolver.LookupTXT(ctx, query)
	if err != nil {
		s.logger.Debug("Cymru ASN enrich failed", "asn", asn, "error", err)
		return
	}
	if len(txts) > 0 {
		parseCymruASNRecord(result, txts[0])
	}
}

// parseCymruIPRecord parses a Team Cymru TXT record of the form:
// "15169 | 8.8.8.0/24 | US | arin | 1992-12-01"
func parseCymruIPRecord(result *Result, txt string) {
	parts := strings.Split(txt, "|")
	if len(parts) >= 1 {
		result.ASN = "AS" + strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		result.Prefix = output.StripANSI(strings.TrimSpace(parts[1]))
	}
	if len(parts) >= 3 {
		result.Country = output.StripANSI(strings.TrimSpace(parts[2]))
	}
	if len(parts) >= 4 {
		result.Registry = output.StripANSI(strings.TrimSpace(parts[3]))
	}
}

// parseCymruASNRecord parses a Team Cymru TXT record of the form:
// "15169 | US | arin | 2000-03-30 | GOOGLE, US"
func parseCymruASNRecord(result *Result, txt string) {
	parts := strings.Split(txt, "|")
	if len(parts) >= 2 {
		result.Country = output.StripANSI(strings.TrimSpace(parts[1]))
	}
	if len(parts) >= 3 {
		result.Registry = output.StripANSI(strings.TrimSpace(parts[2]))
	}
	if len(parts) >= 5 {
		result.Description = output.StripANSI(strings.TrimSpace(parts[4]))
	}
}

// reverseIP reverses an IP address for Team Cymru DNS queries.
// Returns the reversed form, whether it is IPv6, and any error.
// IPv4: "8.8.8.8" → "8.8.8.8" (octets reversed), false
// IPv6: full expansion reversed by nibble, true
func reverseIP(ip string) (string, bool, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", false, fmt.Errorf("invalid IP: %q", ip)
	}
	if v4 := parsed.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d", v4[3], v4[2], v4[1], v4[0]), false, nil
	}
	// IPv6: expand to full hex, build nibbles in big-endian order then reverse in place.
	// This avoids the O(n²) allocations of the prepend pattern.
	v6 := parsed.To16()
	nibbles := make([]string, 0, 32)
	for _, b := range v6 {
		nibbles = append(nibbles, fmt.Sprintf("%x", b>>4), fmt.Sprintf("%x", b&0xf))
	}
	for i, j := 0, len(nibbles)-1; i < j; i, j = i+1, j-1 {
		nibbles[i], nibbles[j] = nibbles[j], nibbles[i]
	}
	return strings.Join(nibbles, "."), true, nil
}
