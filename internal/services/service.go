package services

import (
	"context"
	"errors"

	"github.com/tbckr/trident/internal/pap"
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

// ErrPAPBlocked is returned when a service's PAP level exceeds the user-defined limit.
var ErrPAPBlocked = errors.New("PAP limit exceeded")

// Service is the contract every Trident service must implement.
type Service interface {
	Name() string
	PAP() pap.Level
	Run(ctx context.Context, input string) (any, error)
}
