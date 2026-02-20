// Package main is the entry point for the trident CLI.
package main

import (
	"fmt"
	"os"

	"github.com/tbckr/trident/internal/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\nRun 'trident --help' for usage.\n", err)
		os.Exit(1)
	}
}

func run() error {
	return cli.Execute(os.Stdout, os.Stderr)
}
