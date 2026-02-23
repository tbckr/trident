package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/input"
	"github.com/tbckr/trident/internal/output"
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
		Long: `trident is a fast OSINT CLI for DNS, ASN, certificate transparency, ThreatMiner, and PGP lookups.

No API keys required for any service (dns, asn, crtsh, threatminer, pgp).
PAP levels (least to most active intrusion): white < green < amber < red.`,
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

	cmd.Version = version.Version
	cmd.SetVersionTemplate("trident {{.Version}}\n")

	cmd.AddGroup(&cobra.Group{ID: "services", Title: "Services:"})
	cmd.AddGroup(&cobra.Group{ID: "aggregate", Title: "Aggregate Commands:"})

	if len(aliases) > 0 {
		cmd.AddGroup(&cobra.Group{ID: "aliases", Title: "Aliases:"})
		for name, expansion := range aliases {
			n, e := name, expansion
			cmd.AddCommand(&cobra.Command{
				Use:     n,
				Short:   fmt.Sprintf("Alias for %q", e),
				GroupID: "aliases",
				RunE: func(aliasCmd *cobra.Command, extraArgs []string) error {
					parts := append(strings.Fields(e), extraArgs...)
					aliasCmd.Root().SetArgs(parts)
					return aliasCmd.Root().ExecuteContext(aliasCmd.Context())
				},
			})
		}
	}

	cmd.AddGroup(&cobra.Group{ID: "utility", Title: "Utility Commands:"})

	cmd.AddCommand(
		newDNSCmd(&d),
		newASNCmd(&d),
		newCrtshCmd(&d),
		newThreatMinerCmd(&d),
		newPGPCmd(&d),
		newQuad9Cmd(&d),
		newApexCmd(&d),
		newCompletionCmd(),
		newVersionCmd(&d),
		newConfigCmd(&d),
		newAliasCmd(&d),
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

// deps holds fully-resolved runtime dependencies for a subcommand.
type deps struct {
	logger   *slog.Logger
	cfg      *config.Config
	doDefang bool
}

// buildDeps resolves config, logger, output format, PAP level, and defang flag.
func buildDeps(cmd *cobra.Command, stderr io.Writer) (*deps, error) {
	cfg, err := config.Load(cmd.Flags())
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if cfg.Defang && cfg.NoDefang {
		return nil, fmt.Errorf("--defang and --no-defang are mutually exclusive")
	}

	if cfg.Concurrency < 1 {
		return nil, fmt.Errorf("--concurrency must be at least 1, got %d", cfg.Concurrency)
	}

	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))

	format := output.Format(cfg.Output)
	switch format {
	case output.FormatTable, output.FormatJSON, output.FormatText:
	default:
		return nil, fmt.Errorf("invalid output format %q: must be \"table\", \"json\", or \"text\"", cfg.Output)
	}

	papLevel, err := pap.Parse(cfg.PAPLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid PAP limit %q: %w", cfg.PAPLimit, err)
	}

	doDefang := output.ResolveDefang(papLevel, format, cfg.Defang, cfg.NoDefang)

	return &deps{cfg: cfg, logger: logger, doDefang: doDefang}, nil
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

// runServiceCmd is the shared RunE body for all OSINT subcommands.
// It handles PAP enforcement, input resolution, single-result and bulk paths.
func runServiceCmd(cmd *cobra.Command, d *deps, svc services.Service, args []string) error {
	if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), svc.PAP()) {
		return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
			services.ErrPAPBlocked, svc.Name(), svc.PAP(), pap.MustParse(d.cfg.PAPLimit))
	}

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

// writeResult formats and writes a service result to stdout.
// When d.doDefang is true the writer is wrapped with DefangWriter.
func writeResult(stdout io.Writer, d *deps, result any) error {
	w := stdout
	if d.doDefang {
		w = &output.DefangWriter{Inner: stdout}
	}
	if err := output.Write(w, output.Format(d.cfg.Output), result); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}
