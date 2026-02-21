package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
	crtshsvc "github.com/tbckr/trident/internal/services/crtsh"
)

func newCrtshCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "crtsh [domain...]",
		Short:   "Search crt.sh certificate transparency logs for subdomains",
		GroupID: "osint",
		Long: `Search crt.sh certificate transparency logs for subdomains of a domain.

Queries the crt.sh API for all TLS certificates that contain the target domain
as a subject or SAN. Wildcard entries and the root domain itself are filtered
from the results; only valid subdomains are returned.

PAP level: AMBER (queries the crt.sh third-party API).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Find subdomains for a domain
  trident crtsh example.com

  # Bulk input from stdin
  echo -e "example.com\nexample.org" | trident crtsh

  # Plain text output (one subdomain per line)
  trident crtsh --output plain example.com

  # JSON output
  trident crtsh --output json example.com`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httpclient.New(d.cfg.Proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
			if err != nil {
				return fmt.Errorf("creating HTTP client: %w", err)
			}
			httpclient.AttachRateLimit(client, ratelimit.New(crtshsvc.DefaultRPS, crtshsvc.DefaultBurst))
			svc := crtshsvc.NewService(client, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
