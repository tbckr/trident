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

package hackertarget

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tbckr/trident/pkg/api"
	"github.com/tbckr/trident/pkg/domainutils"
)

const source = "hackertarget"

func FetchDomains(domain string) ([]string, error) {
	// Params
	fetchURL := fmt.Sprintf("https://api.hackertarget.com/hostsearch/?q=%s", domain)
	// Log
	slog.Debug("Fetching domains",
		"source", source,
		"domain", domain,
		slog.Group("params",
			"url", fetchURL,
		),
	)

	// Request
	resp, err := api.Request().
		Get(fetchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check response
	if !resp.IsSuccessState() {
		return []string{}, api.NewHTTPError(source, resp.StatusCode)
	}

	// Extract domains
	var output []string
	sc := bufio.NewScanner(bytes.NewReader(resp.Bytes()))
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ",", 2)
		if len(parts) != 2 {
			continue
		}
		output = append(output, domainutils.Clean(parts[0]))
	}
	slog.Debug("Retrieved domains",
		"source", source,
		"domain", domain,
		"count", len(output),
	)
	return output, nil
}
