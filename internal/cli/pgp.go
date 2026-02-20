package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	pgpsvc "github.com/tbckr/trident/internal/services/pgp"
	"github.com/tbckr/trident/internal/worker"
)

func newPGPCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "pgp [query...]",
		Short:   "Search keys.openpgp.org for PGP keys by email or name",
		GroupID: "osint",
		Long: `Search keys.openpgp.org for PGP public keys by email address or name.

Queries the HKP (HTTP Keyserver Protocol) machine-readable index at
keys.openpgp.org. Returns key fingerprints, UIDs (email addresses / names),
and key flags (e.g. encryption, signing). A 404 response means no keys found.

PAP level: AMBER (queries the keys.openpgp.org third-party service).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Search by email address
  trident pgp user@example.com

  # Search by name
  trident pgp "Alice Smith"

  # Bulk input from stdin
  echo -e "alice@example.com\nbob@example.com" | trident pgp

  # JSON output
  trident pgp --output json user@example.com`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httpclient.New(d.cfg.Proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
			if err != nil {
				return fmt.Errorf("creating HTTP client: %w", err)
			}
			svc := pgpsvc.NewService(client, d.logger)

			if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), svc.PAP()) {
				return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
					services.ErrPAPBlocked, svc.Name(), svc.PAP(), pap.MustParse(d.cfg.PAPLimit))
			}

			inputs, err := resolveInputs(cmd, args)
			if err != nil {
				return err
			}
			if len(inputs) == 0 {
				return fmt.Errorf("no input: supply an email or name as argument or pipe via stdin")
			}

			if len(inputs) == 1 {
				result, err := svc.Run(cmd.Context(), inputs[0])
				if err != nil {
					return err
				}
				if r, ok := result.(*pgpsvc.Result); ok && r.IsEmpty() {
					d.logger.Info("no PGP keys found", "input", inputs[0])
					return nil
				}
				return writeResult(cmd.OutOrStdout(), d, result)
			}

			// Bulk mode
			results := worker.Run(cmd.Context(), svc, inputs, d.cfg.Concurrency)
			var valid []*pgpsvc.Result
			for _, r := range results {
				if r.Err != nil {
					d.logger.Error("PGP lookup failed", "input", r.Input, "error", r.Err)
					continue
				}
				if pr, ok := r.Output.(*pgpsvc.Result); ok && pr.IsEmpty() {
					d.logger.Info("no PGP keys found", "input", r.Input)
					continue
				}
				if pr, ok := r.Output.(*pgpsvc.Result); ok {
					valid = append(valid, pr)
				}
			}
			switch len(valid) {
			case 0:
				return nil
			case 1:
				return writeResult(cmd.OutOrStdout(), d, valid[0])
			default:
				return writeResult(cmd.OutOrStdout(), d, &pgpsvc.MultiResult{Results: valid})
			}
		},
	}
}
