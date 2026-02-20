// Package cli provides the Cobra command tree and output wiring for trident.
package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/worker"
)

// newRootCmd builds the top-level Cobra command for trident.
// Callers must set stdout/stderr via cmd.SetOut / cmd.SetErr before Execute.
func newRootCmd() *cobra.Command {
	// d is populated by PersistentPreRunE before any subcommand's RunE runs.
	// INVARIANT: Cobra only executes the innermost PersistentPreRunE in the
	// command chain. If a future subcommand defines its own PersistentPreRunE,
	// the root hook will NOT run and d will be zero-valued. Do not add
	// PersistentPreRunE to any subcommand without also re-calling buildDeps.
	var d deps

	cmd := &cobra.Command{
		Use:   "trident",
		Short: "Trident — keyless OSINT reconnaissance tool",
		Long: `Trident is a fast OSINT CLI for DNS, ASN, certificate transparency, ThreatMiner, and PGP lookups.

No API keys required for Phase 2 services (dns, asn, crtsh, threatminer, pgp).
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
	config.RegisterFlagCompletions(cmd)

	cmd.Version = version
	cmd.SetVersionTemplate("trident version {{.Version}}\n")

	cmd.AddGroup(
		&cobra.Group{ID: "osint", Title: "OSINT Services:"},
		&cobra.Group{ID: "utility", Title: "Utility Commands:"},
	)

	cmd.AddCommand(
		newDNSCmd(&d),
		newASNCmd(&d),
		newCrtshCmd(&d),
		newThreatMinerCmd(&d),
		newPGPCmd(&d),
		newCompletionCmd(),
		newVersionCmd(&d),
	)

	cmd.MarkFlagsMutuallyExclusive("defang", "no-defang")

	return cmd
}

// Execute builds the root command and runs it with os.Args.
func Execute(stdout, stderr io.Writer) error {
	cmd := newRootCmd()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	return cmd.Execute()
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
	case output.FormatText, output.FormatJSON, output.FormatPlain:
	default:
		return nil, fmt.Errorf("invalid output format %q: must be \"text\", \"json\", or \"plain\"", cfg.Output)
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
	return worker.ReadInputs(r)
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
