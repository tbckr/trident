package identify

import (
	"log/slog"

	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/pap"
)

const (
	// Name is the service identifier.
	Name = "identify"
	// PAP is the PAP activity level for the identify service.
	PAP = pap.RED
)

// Service performs provider detection from known DNS record values without
// making any network calls.
type Service struct {
	logger *slog.Logger
}

// NewService creates a new identify service with the given logger.
func NewService(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return Name }

// PAP returns the PAP activity level for the identify service.
func (s *Service) PAP() pap.Level { return PAP }

// Run matches CNAME, MX, NS, and TXT record values against known provider patterns.
// No network calls are made — this is pure local computation.
func (s *Service) Run(cnames, mxHosts, nsHosts, txtRecords []string) (*Result, error) {
	result := &Result{}

	for _, d := range providers.CDN(cnames) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
		})
	}
	for _, d := range providers.EmailProvider(mxHosts) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
		})
	}
	for _, d := range providers.DNSHost(nsHosts) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
		})
	}
	for _, d := range providers.TXTRecord(txtRecords) {
		result.Detections = append(result.Detections, Detection{
			Type:     string(d.Type),
			Provider: d.Provider,
			Evidence: d.Evidence,
		})
	}

	return result, nil
}
