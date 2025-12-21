package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/imroc/req/v3"
	"github.com/tbckr/trident/internal/config"
)

type Client interface {
	Get(ctx context.Context, url string) (*req.Response, error)
}

type httpClient struct {
	client *req.Client
	logger *slog.Logger
}

func NewClient(logger *slog.Logger, cfg *config.Config) Client {
	c := req.C().
		SetUserAgent(cfg.UserAgent)

	if cfg.Proxy != "" {
		c.SetProxyURL(cfg.Proxy)
		logger.Debug("proxy configured", "url", cfg.Proxy)
	}

	return &httpClient{
		client: c,
		logger: logger,
	}
}

func NewClientWithReqClient(logger *slog.Logger, client *req.Client) Client {
	return &httpClient{
		client: client,
		logger: logger,
	}
}

func (h *httpClient) Get(ctx context.Context, url string) (*req.Response, error) {
	resp, err := h.client.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, nil
}
