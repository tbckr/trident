// Copyright (c) 2024 Tim <tbckr>
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

package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/client"
	"github.com/tbckr/trident/pkg/config"
	"github.com/tbckr/trident/pkg/opsec"
	"github.com/tbckr/trident/pkg/pap"
)

func InputFromCli(cmd *cobra.Command, args []string) (io.Reader, error) {
	pipedShell := cmd.Context().Value(ContextKeyPipedShell)
	if pipedShell == nil {
		return nil, fmt.Errorf("pipedShell not set in context")
	}
	var input io.Reader
	if pipedShell.(bool) {
		input = cmd.InOrStdin()
	} else {
		// We treat the args as lines to be read, so that we can process them individually
		lines := strings.Join(args, "\n")
		input = strings.NewReader(lines)
	}
	return input, nil
}

func PapPreRunCheck(viperConfig *config.Config, papLevel pap.PapLevel) func(*cobra.Command, []string) error {
	return PapPreRunCheckWrapper(viperConfig, papLevel, nil)
}

func PapPreRunCheckWrapper(viperConfig *config.Config, pluginPapLevel pap.PapLevel, wrappedFn func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		environmentPapLevel, err := viperConfig.GetEnvironmentPapLevel()
		if err != nil {
			return err
		}
		if !pap.IsAllowed(environmentPapLevel, pluginPapLevel) {
			return pap.NewPapLevelConstraintError(environmentPapLevel, pluginPapLevel)
		}
		if wrappedFn != nil {
			return wrappedFn(cmd, args)
		}
		return nil
	}
}

func PipeCliCommand(cmd *cobra.Command, args []string, textModifier func(string) string) error {
	input, err := InputFromCli(cmd, args)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(input)
	for sc.Scan() {
		outString := textModifier(sc.Text())
		_, err = fmt.Fprintln(cmd.OutOrStdout(), outString)
		if err != nil {
			return err
		}
	}
	return nil
}

func DomainFetcherCliCommand(cmd *cobra.Command, args []string, viperConfig *config.Config, df client.DomainFetcher, opts client.DomainFetcherOptions) error {
	// Get input
	input, err := InputFromCli(cmd, args)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(input)

	// Get PAP level
	var environmentPapLevel pap.PapLevel
	environmentPapLevel, err = viperConfig.GetEnvironmentPapLevel()
	if err != nil {
		return err
	}

	var fetchedDomains []string
	domainTracker := make(map[string]bool)

	var domain string
	for sc.Scan() {
		domain = strings.ToLower(sc.Text())
		domain = opsec.UnbracketDomain(domain)

		fetchedDomains, err = df.FetchDomains(cmd.Context(), domain)
		if err != nil {
			return err
		}

		for _, d := range fetchedDomains {
			if opts.OnlyUnique {
				if _, ok := domainTracker[d]; ok {
					continue
				}
				domainTracker[d] = true
			}
			if opts.OnlySubdomains && !strings.HasSuffix(d, domain) {
				continue
			}
			if pap.IsEscapeData(environmentPapLevel) && !viperConfig.GetDisableDomainBrackets() {
				d = opsec.BracketDomain(d)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), d)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
