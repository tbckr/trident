package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/input"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services/crtsh"
	"github.com/tbckr/trident/internal/worker"
)

type crtshOptions struct {
	output string
}

func NewCrtshCmd(logger *slog.Logger, cfg *config.Config) *cobra.Command {
	opts := &crtshOptions{}

	cmd := &cobra.Command{
		Use:   "crtsh [domain]",
		Short: "Search certificates on crt.sh (Certificate Transparency)",
		Long: `Search for subdomains and certificates on crt.sh.
Certificate Transparency (CT) logs are a rich source for subdomain discovery 
and infrastructure mapping.

PAP Level: AMBER (Passive query - interacts with 3rd party infrastructure)`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := input.GetInputs(args, cmd.InOrStdin())
			if err != nil {
				return err
			}

			httpClient := http.NewClient(logger, cfg)
			// TODO: Add rate limiter for crt.sh if needed
			service := crtsh.NewService(httpClient, logger, nil)

			pool := worker.NewPool(cfg.Concurrency, logger)
			// Fix: pass a string header slice for formatter
			formatter := output.NewFormatter(opts.output, []string{"Domain", "Issuer", "Common Name", "Not Before", "Not After"})

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
