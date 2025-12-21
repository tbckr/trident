package pgp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/ratelimit"
)

type PGPResult struct {
	Email       string   `json:"email"`
	Fingerprint string   `json:"fingerprint"`
	KeyID       string   `json:"key_id"`
	UserIDs     []string `json:"user_ids"`
	Created     string   `json:"created"`
	Status      string   `json:"status"`
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

func (s *Service) Search(ctx context.Context, email string) (*PGPResult, error) {
	if s.limiter != nil {
		if err := s.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("https://keys.openpgp.org/vks/v1/by-email/%s", email)
	resp, err := s.client.Get(ctx, url)
	if err != nil {
		// imroc/req returns error on 4xx/5xx if not configured otherwise
		// but trident/internal/http/client.go might handle it.
		// If it reaches here with an error, it might be the 404 handled as error.
		return &PGPResult{Email: email, Status: "Not Found"}, nil
	}

	var result PGPResult
	if err := resp.UnmarshalJson(&result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PGP response: %w", err)
	}

	result.Email = email
	result.Status = "Found"
	return &result, nil
}
