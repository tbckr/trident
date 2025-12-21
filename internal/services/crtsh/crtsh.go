package crtsh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/ratelimit"
	"github.com/tbckr/trident/internal/validation"
)

type CrtshResult []struct {
	IssuerCaID     int    `json:"issuer_ca_id"`
	IssuerName     string `json:"issuer_name"`
	CommonName     string `json:"common_name"`
	NameValue      string `json:"name_value"`
	ID             int64  `json:"id"`
	EntryTimestamp string `json:"entry_timestamp"`
	NotBefore      string `json:"not_before"`
	NotAfter       string `json:"not_after"`
	SerialNumber   string `json:"serial_number"`
	ResultID       int    `json:"result_id"`
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

func (s *Service) Search(ctx context.Context, domain string) (*CrtshResult, error) {
	if err := validation.ValidateDomain(domain); err != nil {
		return nil, err
	}

	if s.limiter != nil {
		if err := s.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("https://crt.sh/?q=%%.%s&output=json", domain)
	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var result CrtshResult
	if err := resp.UnmarshalJson(&result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal crt.sh response: %w", err)
	}

	return &result, nil
}
