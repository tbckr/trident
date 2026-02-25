package services

import (
	"context"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/pap"
)

// ErrInvalidInput is re-exported from apperr for backward compatibility.
// Use errors.Is(err, services.ErrInvalidInput) to detect validation failures uniformly
// across all services.
var ErrInvalidInput = apperr.ErrInvalidInput

// ErrRequestFailed is re-exported from apperr for backward compatibility.
// Use errors.Is(err, services.ErrRequestFailed) to detect request failures uniformly
// across all services.
var ErrRequestFailed = apperr.ErrRequestFailed

// ErrPAPBlocked is re-exported from apperr for backward compatibility.
var ErrPAPBlocked = apperr.ErrPAPBlocked

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
