package services

import (
	"context"
	"errors"
)

// ErrInvalidInput is returned by any service when the provided input fails validation.
// Use errors.Is(err, services.ErrInvalidInput) to detect validation failures uniformly
// across all services.
var ErrInvalidInput = errors.New("invalid input")

// ErrRequestFailed is returned by any HTTP-based service when the request fails at the
// transport level or the server responds with a non-2xx status code.
// Use errors.Is(err, services.ErrRequestFailed) to detect request failures uniformly
// across all services.
var ErrRequestFailed = errors.New("request failed")

// Service is the contract every Trident service must implement.
type Service interface {
	Name() string
	Run(ctx context.Context, input string) (any, error)
}
