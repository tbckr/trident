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

package report

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/tbckr/trident/pkg/api"
	"github.com/tbckr/trident/pkg/api/certspotter"
	"github.com/tbckr/trident/pkg/api/crtsh"
	"github.com/tbckr/trident/pkg/api/hackertarget"
	"github.com/tbckr/trident/pkg/api/securitytrails"
	"github.com/tbckr/trident/pkg/api/wayback"
)

func Generate(domains io.Reader) error {
	_, err := GetDomains(domains)
	return err
}

func GetDomains(domains io.Reader) ([]string, error) {
	subsOnly := true
	// get the list of domain fetchers
	sources := getDomainFetchers()

	out := make(chan string)
	var wg sync.WaitGroup
	sc := bufio.NewScanner(domains)
	for sc.Scan() {
		domain := strings.ToLower(sc.Text())

		// call each of the source workers in a goroutine
		for _, source := range sources {
			wg.Add(1)
			fn := source

			go func() {
				defer wg.Done()

				names, retrievalErr := fn(domain)
				if retrievalErr != nil {
					var httpAPIError api.HTTPError
					if errors.As(retrievalErr, &httpAPIError) {
						slog.Error("Error retrieving domains",
							"source", httpAPIError.Source, "status-code", httpAPIError.StatusCode)
						return
					}
					slog.Error("Error retrieving domains",
						"source", fmt.Sprintf("%T", fn),
						"error", retrievalErr,
					)
					return
				}
				for _, n := range names {
					if subsOnly && !strings.HasSuffix(n, domain) {
						continue
					}
					out <- n
				}
			}()
		}
	}

	// close the output channel when all the workers are done
	go func() {
		wg.Wait()
		close(out)
	}()

	// track what we've already printed to avoid duplicates
	domainTracker := make(map[string]bool)
	receivedDomains := make([]string, 0)
	for elem := range out {
		if _, ok := domainTracker[elem]; ok {
			continue
		}
		domainTracker[elem] = true
		receivedDomains = append(receivedDomains, elem)
	}
	return receivedDomains, nil
}

func getDomainFetchers() []api.DomainFetcherFn {
	return []api.DomainFetcherFn{
		crtsh.FetchDomains,
		certspotter.FetchDomains,
		hackertarget.FetchDomains,
		wayback.FetchDomains,
		securitytrails.FetchDomains,
	}
}
