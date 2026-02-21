package httpclient

import (
	"net/http"
	"strconv"
	"time"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/ratelimit"
)

const (
	// retryAfterFallback is used when Retry-After header is absent or unparseable.
	retryAfterFallback = 5 * time.Second
	// retryAfterCap is the maximum sleep duration honoured from a Retry-After header.
	retryAfterCap = 60 * time.Second
)

// AttachRateLimit hooks a Limiter onto the client's request pipeline.
//
// OnBeforeRequest: calls limiter.Wait(ctx), gating every outbound request.
// Retry: up to 3 retries on HTTP 429, using the Retry-After header (or
// retryAfterFallback when absent). Capped at retryAfterCap to prevent
// unbounded waits.
func AttachRateLimit(client *req.Client, limiter *ratelimit.Limiter) {
	client.OnBeforeRequest(func(_ *req.Client, r *req.Request) error {
		return limiter.Wait(r.Context())
	})

	client.SetCommonRetryCount(3)
	client.AddCommonRetryCondition(func(resp *req.Response, _ error) bool {
		return resp != nil && resp.StatusCode == http.StatusTooManyRequests
	})
	client.SetCommonRetryInterval(func(resp *req.Response, _ int) time.Duration {
		if resp == nil {
			return retryAfterFallback
		}
		return parseRetryAfter(resp.Header.Get("Retry-After"))
	})
}

// parseRetryAfter parses a Retry-After header value (integer seconds or HTTP-date)
// and returns a capped sleep duration.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return retryAfterFallback
	}
	// Try integer seconds first.
	if secs, err := strconv.Atoi(header); err == nil {
		d := time.Duration(secs) * time.Second
		return min(d, retryAfterCap)
	}
	// Try HTTP-date format.
	if t, err := http.ParseTime(header); err == nil {
		d := max(time.Until(t), 0)
		return min(d, retryAfterCap)
	}
	return retryAfterFallback
}
