package ratelimit

import (
	"context"
	"math/rand/v2"
	"time"

	"golang.org/x/time/rate"
)

// Limiter wraps a token-bucket rate limiter and adds ±20% jitter to wait intervals.
type Limiter struct {
	inner *rate.Limiter
}

// New creates a Limiter with the given requests-per-second rate and burst capacity.
func New(rps float64, burst int) *Limiter {
	return &Limiter{inner: rate.NewLimiter(rate.Limit(rps), burst)}
}

// Wait reserves a token from the limiter and waits for the token to become available,
// adding ±20% random jitter to the wait duration. Returns ctx.Err() if the context
// is cancelled before the token is granted.
func (l *Limiter) Wait(ctx context.Context) error {
	res := l.inner.Reserve()
	if !res.OK() {
		// Burst exceeded beyond what the limiter can accommodate.
		return ctx.Err()
	}

	delay := res.Delay()
	if delay <= 0 {
		return nil
	}

	// Apply ±20% jitter to prevent detectable request patterns.
	factor := 0.20
	jitter := time.Duration(float64(delay) * factor * (rand.Float64()*2 - 1)) //nolint:gosec // non-cryptographic random is fine for jitter
	delay = max(0, delay+jitter)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		res.Cancel()
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
