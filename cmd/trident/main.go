package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"

	"github.com/tbckr/trident/internal/cmd"
	"golang.org/x/exp/slog"
)

func main() {
	ctx := context.Background()

	// Use LevelVar for dynamic log level switching
	programLevel := &slog.LevelVar{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel}))

	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr, logger, programLevel); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	logger *slog.Logger,
	programLevel *slog.LevelVar,
) error {
	// Handle signal cancellation
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// Initialize root command with dependency injection
	rootCmd := cmd.NewRootCmd(logger, programLevel, stdout, stderr)
	rootCmd.SetArgs(args[1:])
	rootCmd.SetIn(stdin)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	return rootCmd.ExecuteContext(ctx)
}
