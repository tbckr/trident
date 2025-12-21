package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/input"
	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/services/threatminer"
	"github.com/tbckr/trident/internal/validation"
	"github.com/tbckr/trident/internal/worker"
)

type threatminerOptions struct {
	output     string
	reportType int
}

func NewThreatMinerCmd(logger *slog.Logger, cfg *config.Config) *cobra.Command {
	opts := &threatminerOptions{}

	cmd := &cobra.Command{
		Use:   "threatminer [domain|ip|hash]",
		Short: "Search ThreatMiner for contextual intelligence",
		Long: `Search for Passive DNS history, associated malware hashes, and WHOIS data on ThreatMiner.
This command automatically detects the input type (domain, IP, or hash).

PAP Level: AMBER (Passive query - interacts with 3rd party infrastructure)`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := input.GetInputs(args, cmd.InOrStdin())
			if err != nil {
				return err
			}

			httpClient := http.NewClient(logger, cfg)
			service := threatminer.NewService(httpClient, logger, nil)

			pool := worker.NewPool(cfg.Concurrency, logger)
			formatter := output.NewFormatter(opts.output, []string{"Target", "Type", "Result"})

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
				if err := validation.ValidateDomain(target); err == nil {
					return service.LookupDomain(ctx, target, opts.reportType)
				}
				if err := validation.ValidateIP(target); err == nil {
					return service.LookupIP(ctx, target, opts.reportType)
				}
				// Default to hash
				return service.LookupHash(ctx, target, opts.reportType)
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
	cmd.Flags().IntVarP(&opts.reportType, "report-type", "r", 1, "report type (1=metadata/pDNS, 2=malware/uri, etc.)")

	return cmd
}
