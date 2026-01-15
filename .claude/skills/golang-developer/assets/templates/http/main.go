package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/myapp/internal/server"
)

func main() {
	ctx := context.Background()

	// Setup structured logging with dynamic level
	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelInfo)

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: levelVar,
	}))

	if err := run(ctx, os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, logger, levelVar); err != nil {
		logger.Error("fatal error", slog.String("error", err.Error()))
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
	levelVar *slog.LevelVar,
) error {
	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize server
	srv := server.New(logger)

	// Start server
	return srv.Start(ctx, ":8080")
}
