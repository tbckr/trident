package cli

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/output"
)

// rootCmd is the top-level cobra command for trident.
func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	var (
		configFile string
		verbose    bool
		outputFmt  string
	)

	cmd := &cobra.Command{
		Use:   "trident",
		Short: "Trident — keyless OSINT reconnaissance tool",
		Long: `Trident is a fast, keyless OSINT CLI for DNS, ASN, and certificate transparency lookups.

No API keys required for Phase 1 services (dns, asn, crtsh).`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default: $XDG_CONFIG_HOME/trident/config.yaml)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose (debug) logging")
	cmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "text", "output format: text or json")

	cmd.AddCommand(
		newDNSCmd(stdout, stderr, &configFile, &verbose, &outputFmt),
		newASNCmd(stdout, stderr, &configFile, &verbose, &outputFmt),
		newCrtshCmd(stdout, stderr, &configFile, &verbose, &outputFmt),
	)

	return cmd
}

// Execute builds the root command and runs it with os.Args.
func Execute(stdout, stderr io.Writer) error {
	return newRootCmd(stdout, stderr).Execute()
}

// buildDeps resolves config and logger for a subcommand.
func buildDeps(stderr io.Writer, configFile *string, verbose *bool, outputFmt *string) (*config.Config, *slog.Logger, output.Format, error) {
	cfg, err := config.Load(*configFile, *verbose, *outputFmt)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading config: %w", err)
	}

	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))

	format := output.Format(cfg.Output)
	if format != output.FormatText && format != output.FormatJSON {
		return nil, nil, "", fmt.Errorf("invalid output format %q: must be \"text\" or \"json\"", cfg.Output)
	}

	return cfg, logger, format, nil
}

// writeResult formats and writes the service result to stdout.
func writeResult(stdout io.Writer, format output.Format, result any) error {
	if err := output.Write(stdout, format, result); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}

