package cmd

import (
	"io"
	"log/slog"

	"github.com/spf13/cobra"
)

// RootOptions holds all global flags for the root command.
type RootOptions struct {
	Config      string
	Verbose     bool
	Output      string
	Proxy       string
	UserAgent   string
	PAPLimit    string
	Defang      bool
	NoDefang    bool
	Concurrency int
}

// NewRootCmd creates the root command with dependency injection.
func NewRootCmd(
	logger *slog.Logger,
	levelVar *slog.LevelVar,
	stdout, stderr io.Writer,
) *cobra.Command {
	opts := &RootOptions{}

	cmd := &cobra.Command{
		Use:   "trident",
		Short: "Trident - OSINT tool for threat intelligence gathering",
		Long: `Trident is a high-performance OSINT tool written in Go.
It provides a unified CLI for querying threat intelligence, network, 
and identity/social media platforms.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return preRun(cmd, opts, logger, levelVar)
		},
	}

	// Global flags
	cmd.PersistentFlags().StringVar(
		&opts.Config,
		"config",
		"",
		"config file (default: $HOME/.config/trident/config.yaml)",
	)

	cmd.PersistentFlags().BoolVarP(
		&opts.Verbose,
		"verbose",
		"v",
		false,
		"enable verbose logging (debug level)",
	)

	cmd.PersistentFlags().StringVarP(
		&opts.Output,
		"output",
		"o",
		"text",
		"output format: text, json, plain",
	)

	cmd.PersistentFlags().StringVar(
		&opts.Proxy,
		"proxy",
		"",
		"proxy URL (supports HTTP, HTTPS, SOCKS5, e.g., socks5://127.0.0.1:9050)",
	)

	cmd.PersistentFlags().StringVar(
		&opts.UserAgent,
		"user-agent",
		"",
		"custom User-Agent string (default: rotate browser UA)",
	)

	cmd.PersistentFlags().StringVar(
		&opts.PAPLimit,
		"pap-limit",
		"white",
		"Permissible Actions Protocol limit: red, amber, green, white",
	)

	cmd.PersistentFlags().BoolVar(
		&opts.Defang,
		"defang",
		false,
		"enable output defanging (IOC safety)",
	)

	cmd.PersistentFlags().BoolVar(
		&opts.NoDefang,
		"no-defang",
		false,
		"disable output defanging (overrides automatic defanging)",
	)

	cmd.PersistentFlags().IntVarP(
		&opts.Concurrency,
		"concurrency",
		"c",
		10,
		"number of concurrent workers for bulk processing",
	)

	return cmd
}

// preRun handles persistent pre-run logic for the root command.
func preRun(
	cmd *cobra.Command,
	opts *RootOptions,
	logger *slog.Logger,
	levelVar *slog.LevelVar,
) error {
	// Set log level to debug if verbose flag is set
	if opts.Verbose {
		levelVar.Set(slog.LevelDebug)
		logger.Debug("verbose logging enabled")
	}

	// Validate flag combinations
	if opts.Defang && opts.NoDefang {
		return ErrConflictingFlags("--defang and --no-defang cannot be used together")
	}

	// Validate output format
	if opts.Output != "text" && opts.Output != "json" && opts.Output != "plain" {
		return ErrInvalidOutputFormat(opts.Output)
	}

	// Validate PAP limit
	if opts.PAPLimit != "red" && opts.PAPLimit != "amber" && opts.PAPLimit != "green" && opts.PAPLimit != "white" {
		return ErrInvalidPAPLimit(opts.PAPLimit)
	}

	// Validate concurrency
	if opts.Concurrency < 1 {
		return ErrInvalidConcurrency(opts.Concurrency)
	}

	logger.Debug("configuration validated",
		"output", opts.Output,
		"pap_limit", opts.PAPLimit,
		"concurrency", opts.Concurrency,
		"proxy", opts.Proxy != "",
	)

	return nil
}
