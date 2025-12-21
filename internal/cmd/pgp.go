package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/input"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services/pgp"
	"github.com/tbckr/trident/internal/worker"
)

type pgpOptions struct {
	output string
}

func NewPGPCmd(logger *slog.Logger, cfg *config.Config) *cobra.Command {
	opts := &pgpOptions{}

	cmd := &cobra.Command{
		Use:   "pgp [email]",
		Short: "Search PGP keys by email or name",
		Long: `Search for PGP keys on keys.openpgp.org (HKP protocol).
Retrieve Key IDs, fingerprints, and creation dates associated with 
an email address or name.

PAP Level: AMBER (Passive query - interacts with 3rd party infrastructure)`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := input.GetInputs(args, cmd.InOrStdin())
			if err != nil {
				return err
			}

			httpClient := http.NewClient(logger, cfg)
			service := pgp.NewService(httpClient, logger, nil)

			pool := worker.NewPool(cfg.Concurrency, logger)
			formatter := output.NewFormatter(opts.output, []string{"Email", "Key ID", "Fingerprint", "Created"})

			inputChan := make(chan worker.Input)
			go func() {
				defer close(inputChan)
				for _, t := range targets {
					inputChan <- t
				}
			}()

			processFn := func(ctx context.Context, input worker.Input) (interface{}, error) {
				target, ok := input.(string)
				if !ok {
					return nil, nil
				}
				return service.Search(ctx, target)
			}

			resultsChan := pool.Process(cmd.Context(), inputChan, processFn)

			var allResults []interface{}
			for res := range resultsChan {
				if res.Error != nil {
					logger.Error("search failed", "target", res.Input, "error", res.Error)
					continue
				}
				allResults = append(allResults, res.Value)
			}

			out, err := formatter.Format(allResults)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(out + "\n"))
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "table", "output format (table, json, plain)")

	return cmd
}
