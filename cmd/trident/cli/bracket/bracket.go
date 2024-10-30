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

package bracket

import (
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/opsec"
)

type BracketCmd struct {
	Cmd *cobra.Command
}

func NewBracketCmd() *BracketCmd {
	cmdStruct := &BracketCmd{}
	cmd := &cobra.Command{
		Use:                   "bracket [domains...]",
		Short:                 "Bracket domains",
		Long:                  `Bracket domains for Operational Security (OPSEC)`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.PipeCliCommand(cmd, args, opsec.BracketDomain)
		},
	}
	cmdStruct.Cmd = cmd
	return cmdStruct
}
