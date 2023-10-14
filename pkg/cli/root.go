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

package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	dnsquery "github.com/tbckr/trident/pkg/dnsutils"
	"github.com/tbckr/trident/pkg/geoip2utils"
	"github.com/tbckr/trident/pkg/report"
)

type rootCmd struct {
	cmd     *cobra.Command
	exit    func(int)
	verbose bool
}

func Execute(args []string) {
	newRootCmd(os.Exit).Execute(args)
}

func (r *rootCmd) Execute(args []string) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("Panic occured", "error", err)
		}
	}()

	// Set args for root command
	r.cmd.SetArgs(args)

	if err := r.cmd.Execute(); err != nil {
		// Defaults
		code := 1
		msg := "command failed"

		// Override defaults if possible
		exitErr := &ExitError{}
		if errors.As(err, &exitErr) {
			code = exitErr.Code()
			if exitErr.Details() != "" {
				msg = exitErr.Details()
			}
		}

		// Log error with details and exit
		slog.Debug(msg, "error", err)
		r.exit(code)
		return
	}
	r.exit(0)
}

func newRootCmd(exit func(int)) *rootCmd {
	root := &rootCmd{
		exit: exit,
	}
	cmd := &cobra.Command{
		Use:                   "secscan [domain]",
		Short:                 "",
		DisableFlagsInUseLine: true,
		SilenceUsage:          true,
		Args:                  cobra.RangeArgs(0, 1),
		ValidArgsFunction:     cobra.NoFileCompletions,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if root.verbose {
				opts := &slog.HandlerOptions{
					Level: slog.LevelDebug,
				}
				handler := slog.NewTextHandler(os.Stdout, opts)
				slog.SetDefault(slog.New(handler))
			}
		},
		RunE: runSubdomains,
	}

	cmd.PersistentFlags().BoolVarP(&root.verbose, "verbose", "v", false,
		"enable more verbose output for debugging")

	cmd.AddCommand(
		NewLicensesCmd(),
		NewVersionCmd(),
	)
	root.cmd = cmd
	return root
}

func runSubdomains(cmd *cobra.Command, args []string) error {
	var domains io.Reader
	domains = cmd.InOrStdin()
	// if a domain is provided as an argument, use that instead of stdin
	if len(args) == 1 {
		domains = strings.NewReader(args[0])
	}

	receivedDomains, err := report.GetDomains(domains)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(receivedDomains, "\n"))
	return err
}

func runDNS(cmd *cobra.Command, args []string) error {
	var domains io.Reader
	domains = cmd.InOrStdin()
	// if a domain is provided as an argument, use that instead of stdin
	if len(args) == 1 {
		domains = strings.NewReader(args[0])
	}

	reports := dnsquery.Retrieve(domains)
	for _, report := range reports {
		_, err := cmd.OutOrStdout().Write([]byte(report.String()))
		if err != nil {
			return err
		}
	}
	return nil
}

func runIPInfo(cmd *cobra.Command, args []string) error {
	ipAdr := args[0]

	data, err := geoip2utils.Info(ipAdr)
	if err != nil {
		return err
	}

	_, err = cmd.OutOrStdout().Write([]byte(data.String()))
	return err
}

//func runSecuritytrails(cmd *cobra.Command, args []string) error {
//	domain := args[0]
//
//	c := securitytrails.NewClient(os.Getenv("SECURITYTRAILS_API_KEY"))
//
//	resp, err := c.Subdomains(domain, true, false)
//	if err != nil {
//		return err
//	}
//
//	var marshalled []byte
//	marshalled, err = json.Marshal(resp)
//	if err != nil {
//		return err
//	}
//
//	_, err = cmd.OutOrStdout().Write(marshalled)
//	return err
//}
