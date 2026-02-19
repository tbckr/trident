package main

import (
	"os"

	"github.com/tbckr/trident/internal/cli"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	return cli.Execute(os.Stdout, os.Stderr)
}
