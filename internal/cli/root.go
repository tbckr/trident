package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/input"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/version"
	"github.com/tbckr/trident/internal/worker"
)

// newRootCmd builds the top-level Cobra command for trident.
// Callers must set stdout/stderr via cmd.SetOut / cmd.SetErr before Execute.
// aliases is the map loaded from config; pass nil when aliases are unavailable.
func newRootCmd(aliases map[string]string) *cobra.Command {
	// d is populated by PersistentPreRunE before any subcommand's RunE runs.
	// INVARIANT: Cobra only executes the innermost PersistentPreRunE in the
	// command chain. If a future subcommand defines its own PersistentPreRunE,
	// the root hook will NOT run and d will be zero-valued. Do not add
	// PersistentPreRunE to any subcommand without also re-calling buildDeps.
	var d deps

	cmd := &cobra.Command{
		Use:   "trident",
		Short: "trident — keyless OSINT reconnaissance tool",
		Long: `trident is a fast, keyless OSINT CLI for DNS, ASN, certificate transparency, threat intelligence, PGP, Quad9, provider detection, and aggregate DNS recon.

No API keys required for any service (dns, cymru, crtsh, threatminer, pgp, quad9, detect, identify, apex).
PAP levels (least to most active intrusion): red < amber < green < white.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := buildDeps(cmd, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			d = *resolved
			return nil
		},
	}

	config.RegisterFlags(cmd.PersistentFlags())
	_ = cmd.RegisterFlagCompletionFunc("output", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "text"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("pap-limit", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"red", "amber", "green", "white"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("tls-fingerprint", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"chrome", "firefox", "edge", "safari", "ios", "android", "randomized"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("user-agent", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return httpclient.PresetNames(), cobra.ShellCompDirectiveNoFileComp
	})

	cmd.Version = version.Version
	cmd.SetVersionTemplate("trident {{.Version}}\n")

	cmd.AddGroup(&cobra.Group{ID: "services", Title: "Services:"})
	cmd.AddGroup(&cobra.Group{ID: "aggregate", Title: "Aggregate Commands:"})

	if len(aliases) > 0 {
		cmd.AddGroup(&cobra.Group{ID: "aliases", Title: "Aliases:"})
		for name, expansion := range aliases {
			cmd.AddCommand(&cobra.Command{
				Use:     name,
				Short:   fmt.Sprintf("Alias for %q", expansion),
				GroupID: "aliases",
				RunE: func(aliasCmd *cobra.Command, extraArgs []string) error {
					parts := append(strings.Fields(expansion), extraArgs...)
					aliasCmd.Root().SetArgs(parts)
					return aliasCmd.Root().ExecuteContext(aliasCmd.Context())
				},
			})
		}
	}

	cmd.AddGroup(&cobra.Group{ID: "utility", Title: "Utility Commands:"})

	cmd.AddCommand(
		newDNSCmd(&d),
		newCymruCmd(&d),
		newCrtshCmd(&d),
		newThreatMinerCmd(&d),
		newPGPCmd(&d),
		newQuad9Cmd(&d),
		newDetectCmd(&d),
		newIdentifyCmd(&d),
		newApexCmd(&d),
		newCompletionCmd(),
		newVersionCmd(&d),
		newConfigCmd(&d),
		newAliasCmd(&d),
		newServicesCmd(&d),
		newDownloadCmd(&d),
	)

	cmd.SetHelpCommandGroupID("utility")
	cmd.MarkFlagsMutuallyExclusive("defang", "no-defang")

	return cmd
}

// NewRootCmd returns the top-level Cobra command for doc generation.
// Callers must not execute the returned command — use it only for
// tree traversal (man pages, shell completions).
func NewRootCmd() *cobra.Command {
	return newRootCmd(nil)
}

// Execute builds the root command and runs it with os.Args.
func Execute(ctx context.Context, stdout, stderr io.Writer) error {
	// Load aliases before constructing the command so the "Aliases:" group and
	// stub commands are registered in the correct position (after "OSINT Services:"
	// and before "Utility Commands:") and only when aliases actually exist.
	aliases := map[string]string{}
	cfgPath, err := config.DefaultConfigPath()
	if err == nil {
		cfgPath = peekConfigFlag(os.Args[1:], cfgPath)
		if loaded, aErr := config.LoadAliases(cfgPath); aErr == nil {
			aliases = loaded
		}
	}

	cmd := newRootCmd(aliases)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	// Arg rewriting: if the first positional arg matches an alias name, expand it.
	if len(aliases) > 0 {
		args := os.Args[1:]
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			if expansion, ok := aliases[args[0]]; ok {
				expanded := append(strings.Fields(expansion), args[1:]...)
				cmd.SetArgs(expanded)
			}
		}
	}

	return cmd.ExecuteContext(ctx)
}

// peekConfigFlag scans args for --config <path> or --config=<path> and returns
// the explicit path when present, or defaultPath otherwise.
func peekConfigFlag(args []string, defaultPath string) string {
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(arg, "--config="); ok {
			return v
		}
	}
	return defaultPath
}

// resolveInputs returns positional args, or reads non-empty lines from stdin when
// no args are provided. Returns an error if stdin is an interactive terminal with
// no args (i.e. the user forgot to pass an argument or pipe input).
func resolveInputs(cmd *cobra.Command, args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}
	r := cmd.InOrStdin()
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) { //nolint:gosec // uintptr→int is safe for file descriptors; they fit in int on all supported platforms
		return nil, fmt.Errorf("no input: pass an argument or pipe stdin")
	}
	return input.Read(r)
}

// runCmdBody is the shared execution body for all OSINT subcommands after PAP enforcement.
// It handles input resolution, single-result and bulk paths.
func runCmdBody(cmd *cobra.Command, d *deps, svc services.Service, args []string) error {
	inputs, err := resolveInputs(cmd, args)
	if err != nil {
		return err
	}

	if len(inputs) == 1 {
		result, err := svc.Run(cmd.Context(), inputs[0])
		if err != nil {
			return err
		}
		if result.IsEmpty() {
			d.logger.Info("no results found", "service", svc.Name(), "input", inputs[0])
			return nil
		}
		return writeResult(cmd.OutOrStdout(), d, result)
	}

	// Bulk path
	workerResults := worker.Run(cmd.Context(), svc, inputs, d.cfg.Concurrency)
	var valid []services.Result
	for _, r := range workerResults {
		if r.Err != nil {
			d.logger.Error("lookup failed", "service", svc.Name(), "input", r.Input, "error", r.Err)
			continue
		}
		if r.Output.IsEmpty() {
			d.logger.Info("no results found", "service", svc.Name(), "input", r.Input)
			continue
		}
		valid = append(valid, r.Output)
	}
	switch len(valid) {
	case 0:
		return nil
	case 1:
		return writeResult(cmd.OutOrStdout(), d, valid[0])
	default:
		return writeResult(cmd.OutOrStdout(), d, svc.AggregateResults(valid))
	}
}

// runServiceCmd is the shared RunE body for all OSINT subcommands.
// It handles PAP enforcement, input resolution, single-result and bulk paths.
func runServiceCmd(cmd *cobra.Command, d *deps, svc services.Service, args []string) error {
	if !pap.Allows(d.papLevel, svc.PAP()) {
		return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
			services.ErrPAPBlocked, svc.Name(), svc.PAP(), d.papLevel)
	}
	return runCmdBody(cmd, d, svc, args)
}

// runAggregateCmd is the shared RunE body for aggregate commands that orchestrate multiple sub-services.
// It enforces the minimum PAP level required for any useful output; sub-services exceeding the limit
// are skipped at the service level.
func runAggregateCmd(cmd *cobra.Command, d *deps, svc services.AggregateService, args []string) error {
	if !pap.Allows(d.papLevel, svc.MinPAP()) {
		return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
			services.ErrPAPBlocked, svc.Name(), svc.MinPAP(), d.papLevel)
	}
	return runCmdBody(cmd, d, svc, args)
}
