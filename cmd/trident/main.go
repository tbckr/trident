package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"

	"github.com/tbckr/trident/internal/cmd"
	"github.com/tbckr/trident/internal/config"
)

func main() {
	ctx := context.Background()

	// Use LevelVar for dynamic log level switching
	programLevel := &slog.LevelVar{}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel}))

	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr, logger, programLevel); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string,
	stdin io.Reader, stdout, stderr io.Writer,
	logger *slog.Logger, levelVar *slog.LevelVar) error {

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	// Initialize root command
	rootCmd := cmd.NewRootCmd(logger, levelVar, getenv)

	// Set IO
	rootCmd.SetArgs(args[1:])
	rootCmd.SetIn(stdin)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	// Load configuration to pass to subcommands
	// We handle the --config flag manually here or let PersistentPreRunE do it
	// For simplicity, we create a default config or load from common paths
	cfg, err := config.LoadConfig("", getenv)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Add subcommands
	rootCmd.AddCommand(
		cmd.NewDNSCmd(logger, cfg),
		cmd.NewASNCmd(logger, cfg),
		cmd.NewCrtshCmd(logger, cfg),
		cmd.NewThreatMinerCmd(logger, cfg),
		cmd.NewPGPCmd(logger, cfg),
		cmd.NewBurnCmd(),
	)

	return rootCmd.ExecuteContext(ctx)
}
