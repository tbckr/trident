package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/imroc/req/v3"
)

// RateLimiter allows you to defaultDelay operations
// on a per-key basis. I.e. only one operation for
// a given key can be done within the defaultDelay time
type RateLimiter struct {
	sync.Mutex
	defaultDelay time.Duration
	ops          map[string]time.Time
}

// NewRateLimiter returns a new *RateLimiter for the
// provided defaultDelay
func NewRateLimiter(defaultDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		defaultDelay: defaultDelay,
		ops:          make(map[string]time.Time),
	}
}

// ReqRateLimiterMiddleware implements the Middleware interface for the req library
// and blocks requests based on the RateLimiter configuration. The Host of the request
// is used as the key for the RateLimiter.
func ReqRateLimiterMiddleware(rl *RateLimiter) func(rt http.RoundTripper) req.HttpRoundTripFunc {
	return func(rt http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (resp *http.Response, err error) {
			rl.Block(r.Host)
			return rt.RoundTrip(r)
		}
	}
}

// Block blocks until an operation for key is
// allowed to proceed
func (r *RateLimiter) Block(key string) {
	now := time.Now()

	r.Lock()

	// if there's nothing in the map we can
	// return straight away
	if _, ok := r.ops[key]; !ok {
		r.ops[key] = now
		r.Unlock()
		return
	}

	// if time is up we can return straight away
	t := r.ops[key]
	deadline := t.Add(r.defaultDelay)
	if now.After(deadline) {
		r.ops[key] = now
		r.Unlock()
		return
	}

	remaining := deadline.Sub(now)

	// Set the time of the operation
	r.ops[key] = now.Add(remaining)
	r.Unlock()

	// Block for the remaining time
	<-time.After(remaining)
}
