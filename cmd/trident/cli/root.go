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
	"context"
	"github.com/tbckr/trident/cmd/trident/cli/certspotter"
	"github.com/tbckr/trident/cmd/trident/cli/hackertarget"
	"github.com/tbckr/trident/cmd/trident/cli/securitytrails"
	"io"
	"log/slog"
	"time"

	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/cmd/trident/cli/bracket"
	"github.com/tbckr/trident/cmd/trident/cli/crtsh"
	"github.com/tbckr/trident/cmd/trident/cli/unbracket"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/client"
	"github.com/tbckr/trident/pkg/config"
	"github.com/tbckr/trident/pkg/pap"
	"github.com/tbckr/trident/pkg/ratelimit"
	"github.com/tbckr/trident/pkg/shell"
)

type RootCmd struct {
	Cmd *cobra.Command

	environmentPapLevel string
	noDomainBrackets    bool
	verbose             bool
}

const (
	rootCmdShortDescription = "trident is a CLI tool for Security practitioners and researchers"
	rootCmdLongDescription  = `trident is a CLI tool for Security practitioners and researchers. It is designed to be used in a variety of security-related tasks, such as penetration testing, digital forensics, and malware analysis.`
)

func Run(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, userconfigdir func() (string, error), args []string) (error, int) {
	// Create config instance
	viperConfig, err := config.New(userconfigdir)
	if err != nil {
		return err, 1
	}

	// Create RateLimiter
	rl := ratelimit.NewRateLimiter(time.Second)
	// Create http client
	reqClient := client.NewHTTPClient(rl)

	// Run CLI
	var root *RootCmd
	root, err = NewRootCmd(
		ctx,
		stdin,
		stdout,
		stderr,
		shell.PipedShell,
		viperConfig,
		reqClient,
	)
	if err != nil {
		return err, 1
	}
	if err = root.Execute(args); err != nil {
		return err, 1
	}
	return nil, 0
}

func NewRootCmd(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, pipedShell func() (bool, error), viperConfig *config.Config, reqClient *req.Client) (*RootCmd, error) {
	root := &RootCmd{}

	cmd := &cobra.Command{
		Use:                   "trident",
		Short:                 rootCmdShortDescription,
		Long:                  rootCmdLongDescription,
		SilenceErrors:         true,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			setupLogging(cmd.OutOrStderr(), root.verbose)

			// Check, if the shell is piped and set value in context
			piped, err := pipedShell()
			if err != nil {
				return err
			}
			newCtx := context.WithValue(ctx, cli.ContextKeyPipedShell, piped)
			cmd.SetContext(newCtx)

			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.SetContext(ctx)
	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	// Add Groups
	pluginsGroup := &cobra.Group{
		ID:    cli.GroupPlugins,
		Title: "Plugin Commands",
	}
	cmd.AddGroup(pluginsGroup)

	// verbosity level
	cmd.PersistentFlags().BoolVarP(&root.verbose, "verbose", "v", false,
		"enable more verbose output for debugging")

	// PAP level
	cmd.PersistentFlags().StringVar(&root.environmentPapLevel, "pap-level", pap.DefaultPapLevelString, "set the environment PAP level")
	if err := viperConfig.BindPFlag(config.ConfigKeyPapLevel, cmd.PersistentFlags().Lookup("pap-level")); err != nil {
		return nil, err
	}
	if err := viperConfig.BindEnv(config.ConfigKeyPapLevel, "TRIDENT_PAP_LEVEL"); err != nil {
		return nil, err
	}
	viperConfig.SetDefault(config.ConfigKeyPapLevel, pap.DefaultPapLevelString)

	// Disable domain brackets
	cmd.PersistentFlags().Bool("no-domain-brackets", false, "disable domain brackets even if PAP level would enforce them")
	if err := viperConfig.BindPFlag(config.ConfigKeyDisableDomainBrackets, cmd.PersistentFlags().Lookup("no-domain-brackets")); err != nil {
		return nil, err
	}
	if err := viperConfig.BindEnv(config.ConfigKeyDisableDomainBrackets, "TRIDENT_DISABLE_DOMAIN_BRACKETS"); err != nil {
		return nil, err
	}
	viperConfig.SetDefault(config.ConfigKeyDisableDomainBrackets, false)

	// Add Subcommands
	cmd.AddCommand(
		bracket.NewBracketCmd().Cmd,
		unbracket.NewBracketCmd().Cmd,
		crtsh.NewCrtShCmd(viperConfig, reqClient).Cmd,
		certspotter.NewCertspotterCmd(viperConfig, reqClient).Cmd,
		hackertarget.NewHackerTargetCmd(viperConfig, reqClient).Cmd,
		securitytrails.NewSecurityTrailsCmd(viperConfig, reqClient).Cmd,
	)

	root.Cmd = cmd
	return root, nil
}

func setupLogging(out io.Writer, verbose bool) {
	if verbose {
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		handler := slog.NewTextHandler(out, opts)
		slog.SetDefault(slog.New(handler))
	}
}

func (r *RootCmd) Execute(args []string) error {
	r.Cmd.SetArgs(args)
	err := r.Cmd.Execute()
	return err
}
