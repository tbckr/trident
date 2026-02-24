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

// Result is the common interface every service's Run output must satisfy.
type Result interface {
	IsEmpty() bool
}

// Service is the contract every trident service must implement.
type Service interface {
	Name() string
	PAP() pap.Level
	Run(ctx context.Context, input string) (Result, error)
	AggregateResults(results []Result) Result
}

// AggregateService is implemented by commands that orchestrate multiple sub-services.
// MinPAP returns the minimum PAP level required to produce any useful results.
// Sub-services whose PAP exceeds the user's limit are skipped with a log message.
type AggregateService interface {
	Service
	MinPAP() pap.Level
}
