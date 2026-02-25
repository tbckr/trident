package threatminer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

const (
	baseURL      = "https://api.threatminer.org/v2"
	domainPDNS   = baseURL + "/domain.php?q=%s&rt=2"
	domainSubs   = baseURL + "/domain.php?q=%s&rt=5"
	ipPDNS       = baseURL + "/host.php?q=%s&rt=2"
	hashMetadata = baseURL + "/sample.php?q=%s&rt=1"

	// DefaultRPS is the target request rate for the ThreatMiner service.
	DefaultRPS float64 = 1.0
	// DefaultBurst is the burst capacity above DefaultRPS.
	DefaultBurst = 1

	// Name is the service identifier.
	Name = "threatminer"
	// PAP is the PAP activity level for the ThreatMiner service.
	PAP = pap.AMBER
)

// inputType classifies the kind of input accepted by the service.
type inputType string

const (
	inputDomain inputType = "domain"
	inputIP     inputType = "ip"
	inputHash   inputType = "hash"
)

// apiResponse is the JSON envelope returned by ThreatMiner.
type apiResponse struct {
	StatusCode    string          `json:"status_code"`
	StatusMessage string          `json:"status_message"`
	Results       json.RawMessage `json:"results"`
}

// Service queries the ThreatMiner API.
type Service struct {
	client *req.Client
	logger *slog.Logger
}

// NewService creates a new ThreatMiner Service.
func NewService(client *req.Client, logger *slog.Logger) *Service {
	return &Service{client: client, logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return Name }

// PAP returns the PAP classification for this service.
func (s *Service) PAP() pap.Level { return PAP }

// AggregateResults combines multiple ThreatMiner results into a MultiResult.
func (s *Service) AggregateResults(results []services.Result) services.Result {
	mr := &MultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*Result))
	}
	return mr
}

// Run queries ThreatMiner for the given input (domain, IP, or hash).
func (s *Service) Run(ctx context.Context, input string) (services.Result, error) {
	itype, err := classify(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", services.ErrInvalidInput, input)
	}

	result := &Result{Input: output.StripANSI(input), InputType: string(itype)}

	switch itype {
	case inputDomain:
		if err := s.fetchDomainData(ctx, input, result); err != nil {
			return nil, err
		}
	case inputIP:
		if err := s.fetchIPData(ctx, input, result); err != nil {
			return nil, err
		}
	case inputHash:
		if err := s.fetchHashData(ctx, input, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *Service) fetchDomainData(ctx context.Context, domain string, result *Result) error {
	pdns, err := s.fetch(ctx, fmt.Sprintf(domainPDNS, domain))
	if err != nil {
		return err
	}
	if pdns != nil {
		var entries []PDNSEntry
		if err := json.Unmarshal(pdns, &entries); err == nil {
			for i := range entries {
				entries[i].IP = output.StripANSI(entries[i].IP)
				entries[i].Domain = output.StripANSI(entries[i].Domain)
			}
			result.PassiveDNS = entries
		}
	}

	subs, err := s.fetch(ctx, fmt.Sprintf(domainSubs, domain))
	if err != nil {
		return err
	}
	if subs != nil {
		var subList []string
		if err := json.Unmarshal(subs, &subList); err == nil {
			cleaned := make([]string, 0, len(subList))
			for _, sub := range subList {
				cleaned = append(cleaned, output.StripANSI(sub))
			}
			result.Subdomains = cleaned
		}
	}
	return nil
}

func (s *Service) fetchIPData(ctx context.Context, ip string, result *Result) error {
	pdns, err := s.fetch(ctx, fmt.Sprintf(ipPDNS, ip))
	if err != nil {
		return err
	}
	if pdns != nil {
		var entries []PDNSEntry
		if err := json.Unmarshal(pdns, &entries); err == nil {
			for i := range entries {
				entries[i].IP = output.StripANSI(entries[i].IP)
				entries[i].Domain = output.StripANSI(entries[i].Domain)
			}
			result.PassiveDNS = entries
		}
	}
	return nil
}

func (s *Service) fetchHashData(ctx context.Context, hash string, result *Result) error {
	raw, err := s.fetch(ctx, fmt.Sprintf(hashMetadata, hash))
	if err != nil {
		return err
	}
	if raw != nil {
		var metas []HashMetadata
		if err := json.Unmarshal(raw, &metas); err == nil && len(metas) > 0 {
			m := metas[0]
			m.MD5 = output.StripANSI(m.MD5)
			m.SHA1 = output.StripANSI(m.SHA1)
			m.SHA256 = output.StripANSI(m.SHA256)
			m.FileType = output.StripANSI(m.FileType)
			m.FileName = output.StripANSI(m.FileName)
			result.HashInfo = &m
		}
	}
	return nil
}

// fetch performs a GET request and returns the raw results JSON, or nil on 404/no-results.
func (s *Service) fetch(ctx context.Context, url string) (json.RawMessage, error) {
	resp, err := s.client.R().SetContext(ctx).Get(url)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %s", services.ErrRequestFailed, err)
	}
	if resp.Response == nil || !resp.IsSuccessState() {
		body := resp.String()
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return nil, fmt.Errorf("%w: HTTP %d: %q", services.ErrRequestFailed, resp.StatusCode, body)
	}

	var envelope apiResponse
	if err := resp.UnmarshalJson(&envelope); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// status_code "404" means no results — treat as empty, not an error.
	if envelope.StatusCode == "404" {
		return nil, nil
	}

	return envelope.Results, nil
}

// classify determines the input type.
func classify(input string) (inputType, error) {
	if input == "" {
		return "", fmt.Errorf("empty input")
	}
	if net.ParseIP(input) != nil {
		return inputIP, nil
	}
	if isHash(input) {
		return inputHash, nil
	}
	if services.IsDomain(input) {
		return inputDomain, nil
	}
	return "", fmt.Errorf("unrecognised input")
}

// isHash returns true for 32-char (MD5), 40-char (SHA1), or 64-char (SHA256) hex strings.
func isHash(s string) bool {
	switch len(s) {
	case 32, 40, 64:
		return isHex(s)
	}
	return false
}

func isHex(s string) bool {
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
