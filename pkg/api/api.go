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

package api

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/imroc/req/v3"
)

const (
	userAgent = "secscan (https://github.com/tbckr/secscan)"

	requestTimeout = 5 * time.Second
)

type DomainFetcherFn func(domain string) ([]string, error)

var r *req.Client

func init() {
	r = req.C().
		SetUserAgent(userAgent).
		SetTimeout(requestTimeout).
		SetCommonRetryCount(2).
		// Set the retry sleep interval with a commonly used algorithm: capped exponential backoff with jitter (https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/).
		SetCommonRetryBackoffInterval(1*time.Second, 5*time.Second).
		// Add a retry hook that will be called if a retry occurs
		AddCommonRetryHook(func(resp *req.Response, err error) {
			rawReq := resp.Request.RawRequest
			slog.Warn("Retrying request", "method", rawReq.Method, "url", rawReq.URL.String())
		})
}

func Request() *req.Request {
	return r.R()
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
