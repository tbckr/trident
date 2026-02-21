package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
	pgpsvc "github.com/tbckr/trident/internal/services/pgp"
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
			httpclient.AttachRateLimit(client, ratelimit.New(pgpsvc.DefaultRPS, pgpsvc.DefaultBurst))
			svc := pgpsvc.NewService(client, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
