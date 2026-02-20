// Package threatminer provides a service for querying the ThreatMiner threat intelligence API.
package threatminer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/validate"
)

const (
	baseURL      = "https://api.threatminer.org/v2"
	domainPDNS   = baseURL + "/domain.php?q=%s&rt=2"
	domainSubs   = baseURL + "/domain.php?q=%s&rt=5"
	ipPDNS       = baseURL + "/host.php?q=%s&rt=2"
	hashMetadata = baseURL + "/sample.php?q=%s&rt=1"
)

// inputType classifies the kind of input accepted by the service.
type inputType string

const (
	inputDomain inputType = "domain"
	inputIP     inputType = "ip"
	inputHash   inputType = "hash"
)

// PDNSEntry represents a single passive DNS record.
type PDNSEntry struct {
	IP        string `json:"ip"`
	Domain    string `json:"domain"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

// HashMetadata holds metadata returned for a file hash query.
type HashMetadata struct {
	MD5      string `json:"md5"`
	SHA1     string `json:"sha1"`
	SHA256   string `json:"sha256"`
	FileType string `json:"file_type"`
	FileName string `json:"file_name"`
	FileSize string `json:"file_size"`
}

// Result holds the output of a ThreatMiner query.
type Result struct {
	Input      string      `json:"input"`
	InputType  string      `json:"input_type"`
	PassiveDNS []PDNSEntry `json:"passive_dns,omitempty"`
	Subdomains []string    `json:"subdomains,omitempty"`
	// Hash-specific fields — non-nil only for hash queries
	HashInfo *HashMetadata `json:"hash_info,omitempty"`
}

// IsEmpty returns true when the result contains no data.
func (r *Result) IsEmpty() bool {
	return len(r.PassiveDNS) == 0 && len(r.Subdomains) == 0 && r.HashInfo == nil
}

// WriteText writes a human-readable table to w.
func (r *Result) WriteText(w io.Writer) error {
	if r.InputType == string(inputHash) && r.HashInfo != nil {
		h := r.HashInfo
		tbl := output.NewWrappingTable(w, 20, 20)
		tbl.Header([]string{"Field", "Value"})
		rows := [][]string{
			{"MD5", h.MD5},
			{"SHA1", h.SHA1},
			{"SHA256", h.SHA256},
			{"File Type", h.FileType},
			{"File Name", h.FileName},
			{"File Size", h.FileSize},
		}
		if err := tbl.Bulk(rows); err != nil {
			return err
		}
		return tbl.Render()
	}

	if len(r.PassiveDNS) > 0 {
		tbl := output.NewWrappingTable(w, 20, 30)
		tbl.Header([]string{"IP", "Domain", "First Seen", "Last Seen"})
		rows := make([][]string, 0, len(r.PassiveDNS))
		for _, e := range r.PassiveDNS {
			rows = append(rows, []string{e.IP, e.Domain, e.FirstSeen, e.LastSeen})
		}
		if err := tbl.Bulk(rows); err != nil {
			return err
		}
		if err := tbl.Render(); err != nil {
			return err
		}
	}

	if len(r.Subdomains) > 0 {
		tbl := output.NewWrappingTable(w, 30, 6)
		tbl.Header([]string{"Subdomain"})
		rows := make([][]string, 0, len(r.Subdomains))
		for _, s := range r.Subdomains {
			rows = append(rows, []string{s})
		}
		if err := tbl.Bulk(rows); err != nil {
			return err
		}
		if err := tbl.Render(); err != nil {
			return err
		}
	}

	return nil
}

// WritePlain writes one record per line to w.
//
// For passive DNS: "<ip> <domain>" per entry.
// For subdomains: one subdomain per line.
// For hashes: "<field>: <value>" per field.
func (r *Result) WritePlain(w io.Writer) error {
	if r.InputType == string(inputHash) && r.HashInfo != nil {
		h := r.HashInfo
		fields := [][2]string{
			{"MD5", h.MD5},
			{"SHA1", h.SHA1},
			{"SHA256", h.SHA256},
			{"FileType", h.FileType},
			{"FileName", h.FileName},
			{"FileSize", h.FileSize},
		}
		for _, f := range fields {
			if _, err := fmt.Fprintf(w, "%s: %s\n", f[0], f[1]); err != nil {
				return err
			}
		}
		return nil
	}

	for _, e := range r.PassiveDNS {
		if _, err := fmt.Fprintf(w, "%s %s\n", e.IP, e.Domain); err != nil {
			return err
		}
	}
	for _, s := range r.Subdomains {
		if _, err := fmt.Fprintln(w, s); err != nil {
			return err
		}
	}
	return nil
}

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
func (s *Service) Name() string { return "threatminer" }

// PAP returns the PAP classification for this service.
func (s *Service) PAP() pap.Level { return pap.AMBER }

// Run queries ThreatMiner for the given input (domain, IP, or hash).
func (s *Service) Run(ctx context.Context, input string) (any, error) {
	itype, err := classify(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", services.ErrInvalidInput, input)
	}

	result := &Result{Input: input, InputType: string(itype)}

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
		return nil, fmt.Errorf("%w: %s", services.ErrRequestFailed, err)
	}
	if !resp.IsSuccessState() {
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
	if validate.IsDomain(input) {
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
