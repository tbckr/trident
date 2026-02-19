package services

import (
	"context"
	"errors"
)

// ErrInvalidInput is returned by any service when the provided input fails validation.
// Use errors.Is(err, services.ErrInvalidInput) to detect validation failures uniformly
// across all services.
var ErrInvalidInput = errors.New("invalid input")

// Service is the contract every Trident service must implement.
type Service interface {
	Name() string
	Run(ctx context.Context, input string) (any, error)
}
