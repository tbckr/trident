package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tbckr/trident/internal/cli"
	"github.com/tbckr/trident/internal/output"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n\nRun 'trident --help' for usage.\n", output.SanitizeTerminal(err.Error()))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	return cli.Execute(ctx, os.Stdout, os.Stderr)
}
