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

package crtsh

import (
	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/client"
	"github.com/tbckr/trident/pkg/config"
	"github.com/tbckr/trident/pkg/pap"
	"github.com/tbckr/trident/pkg/plugins/crtsh"
)

type CrtShCmd struct {
	Cmd *cobra.Command
}

type DomainCmd struct {
	Cmd *cobra.Command

	opts client.DomainFetcherOptions
}

func NewCrtShCmd(viperConfig *config.Config, reqClient *req.Client) *CrtShCmd {
	cmdStruct := &CrtShCmd{}
	cmd := &cobra.Command{
		Use:   "crtsh",
		Short: "Fetch data from crt.sh",
		Long: `PAP Level: AMBER

Fetch domains from crt.sh`,
		GroupID:               cli.GroupPlugins,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newDomainCmd(viperConfig, reqClient).Cmd,
	)

	cmdStruct.Cmd = cmd
	return cmdStruct
}

func newDomainCmd(viperConfig *config.Config, reqClient *req.Client) *DomainCmd {
	cmdStruct := &DomainCmd{
		opts: client.DomainFetcherOptions{},
	}
	cmd := &cobra.Command{
		Use:     "domain [domains...]",
		Aliases: []string{"d", "domains"},
		Short:   "Fetch domains from crt.sh",
		Long: `PAP Level: AMBER

Fetch domains from crt.sh`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		PreRunE:               cli.PapPreRunCheck(viperConfig, pap.LevelAmber),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.DomainFetcherCliCommand(cmd, args, viperConfig, crtsh.NewCrtShClient(reqClient), cmdStruct.opts)
		},
	}

	cmd.Flags().BoolVarP(&cmdStruct.opts.OnlyUnique, "only-unique", "u", false, "Only unique domains")
	cmd.Flags().BoolVarP(&cmdStruct.opts.OnlySubdomains, "only-subdomains", "s", false, "Only subdomains")

	cmdStruct.Cmd = cmd
	return cmdStruct
}
