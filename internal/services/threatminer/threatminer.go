package threatminer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/ratelimit"
	"github.com/tbckr/trident/internal/validation"
)

type DomainResult struct {
	Status     string   `json:"status"`
	StatusMsg  string   `json:"status_message"`
	Results    []string `json:"results"`
	ReportType string   `json:"report_type"`
}

type IPResult struct {
	Status     string   `json:"status"`
	StatusMsg  string   `json:"status_message"`
	Results    []string `json:"results"`
	ReportType string   `json:"report_type"`
}

type HashResult struct {
	Status    string `json:"status"`
	StatusMsg string `json:"status_message"`
	Results   []struct {
		SampleID       string `json:"sample_id"`
		FileName       string `json:"file_name"`
		MD5            string `json:"md5"`
		SHA1           string `json:"sha1"`
		SHA256         string `json:"sha256"`
		EntryTimestamp string `json:"entry_timestamp"`
	} `json:"results"`
	ReportType string `json:"report_type"`
}

type Service struct {
	client  http.Client
	logger  *slog.Logger
	limiter *ratelimit.Limiter
}

func NewService(client http.Client, logger *slog.Logger, limiter *ratelimit.Limiter) *Service {
	return &Service{
		client:  client,
		logger:  logger,
		limiter: limiter,
	}
}

func (s *Service) wait(ctx context.Context) error {
	if s.limiter != nil {
		return s.limiter.Wait(ctx)
	}
	return nil
}

func (s *Service) LookupDomain(ctx context.Context, domain string, rt int) (*DomainResult, error) {
	if err := validation.ValidateDomain(domain); err != nil {
		return nil, err
	}
	if err := s.wait(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.threatminer.org/v2/domain.php?q=%s&rt=%d", domain, rt)
	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var result DomainResult
	if err := resp.UnmarshalJson(&result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal threatminer domain response: %w", err)
	}
	result.ReportType = fmt.Sprintf("%d", rt)

	return &result, nil
}

func (s *Service) LookupIP(ctx context.Context, ip string, rt int) (*IPResult, error) {
	if err := validation.ValidateIP(ip); err != nil {
		return nil, err
	}
	if err := s.wait(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.threatminer.org/v2/host.php?q=%s&rt=%d", ip, rt)
	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var result IPResult
	if err := resp.UnmarshalJson(&result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal threatminer host response: %w", err)
	}
	result.ReportType = fmt.Sprintf("%d", rt)

	return &result, nil
}

func (s *Service) LookupHash(ctx context.Context, hash string, rt int) (*HashResult, error) {
	// TODO: Add hash validation to internal/validation
	// For now we assume validator will be updated or we use a basic check
	if err := s.wait(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.threatminer.org/v2/sample.php?q=%s&rt=%d", hash, rt)
	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var result HashResult
	if err := resp.UnmarshalJson(&result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal threatminer sample response: %w", err)
	}
	result.ReportType = fmt.Sprintf("%d", rt)

	return &result, nil
}
