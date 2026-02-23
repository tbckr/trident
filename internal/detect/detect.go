package detect

import "strings"

// ServiceType identifies the category of a detected cloud service.
type ServiceType string

// ServiceType constants for each detection category.
const (
	TypeCDN   ServiceType = "CDN"
	TypeEmail ServiceType = "Email"
	TypeDNS   ServiceType = "DNS"
)

// Detection holds the result of matching a DNS record against known provider patterns.
type Detection struct {
	Type     ServiceType
	Provider string
	Evidence string // e.g. CNAME target, MX exchange, NS server
}

// pattern maps a DNS suffix to a provider name.
type pattern struct {
	suffix   string
	provider string
}

// matchSuffix returns true when host == suffix or host ends with "."+suffix.
func matchSuffix(host, suffix string) bool {
	h := strings.TrimSuffix(host, ".")
	return h == suffix || strings.HasSuffix(h, "."+suffix)
}
