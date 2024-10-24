// Copyright (c) 2023 Tim <tbckr>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/imroc/req/v3"
	"github.com/tbckr/trident/pkg/ratelimit"
)

const (
	requestTimeout = 10 * time.Second
)

type DomainFetcher interface {
	FetchDomains(ctx context.Context, domain string) ([]string, error)
}

type DomainFetcherOptions struct {
	OnlyUnique     bool
	OnlySubdomains bool
}

func NewHTTPClient(rl *ratelimit.RateLimiter) *req.Client {
	c := req.C().
		ImpersonateChrome().
		SetTLSFingerprintChrome().
		SetTimeout(requestTimeout).
		SetCommonRetryCount(2).
		SetCommonRetryBackoffInterval(1*time.Second, 10*time.Second).
		AddCommonRetryHook(func(resp *req.Response, err error) {
			if err != nil {
				slog.Warn("Retrying request", "reason", "error", "error", err)
				// RawRequest may be nil at this point, so we quit here
				return
			}
			r := resp.Request.RawRequest
			slog.Warn("Retrying request", "reason", "responseCode", "method", r.Method, "url", r.URL.Hostname(), "responseCode", resp.StatusCode)
		})
	if rl != nil {
		c.GetTransport().WrapRoundTripFunc(ratelimit.ReqRateLimiterMiddleware(rl))
	}
	return c
}

type HTTPError struct {
	Source     string
	StatusCode int
}

func NewHTTPError(source string, statusCode int) HTTPError {
	return HTTPError{
		Source:     source,
		StatusCode: statusCode,
	}
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("request for %s failed with status code %d", e.Source, e.StatusCode)
}
