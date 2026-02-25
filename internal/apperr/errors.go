package apperr

import "errors"

// ErrInvalidInput is returned by any service when the provided input fails validation.
// Use errors.Is(err, apperr.ErrInvalidInput) to detect validation failures uniformly
// across all services.
var ErrInvalidInput = errors.New("invalid input")

// ErrRequestFailed is returned by any HTTP-based service when the request fails at the
// transport level or the server responds with a non-2xx status code.
// Use errors.Is(err, apperr.ErrRequestFailed) to detect request failures uniformly
// across all services.
var ErrRequestFailed = errors.New("request failed")

// ErrPAPBlocked is returned when a service's PAP level exceeds the user-defined limit.
var ErrPAPBlocked = errors.New("PAP limit exceeded")
