package ratelimit

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"golang.org/x/time/rate"
)

type Limiter struct {
	limiter *rate.Limiter
	logger  *slog.Logger
	jitter  float64
}

func NewLimiter(rps float64, burst int, logger *slog.Logger) *Limiter {
	return &Limiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		logger:  logger,
		jitter:  0.2, // 20% jitter
	}
}

func (l *Limiter) Wait(ctx context.Context) error {
	if err := l.limiter.Wait(ctx); err != nil {
		return err
	}

	// Add jitter
	jitterDuration := time.Duration(float64(time.Second) * l.jitter * (rand.Float64()*2 - 1) / float64(l.limiter.Limit()))
	if jitterDuration > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitterDuration):
		}
	}

	return nil
}
