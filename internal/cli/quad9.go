package cli

import (
	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/doh"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
	quad9svc "github.com/tbckr/trident/internal/services/quad9"
)

func newQuad9Cmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "quad9 [domain...]",
		Short:   "Check whether Quad9 has blocked a domain as malicious",
		GroupID: "services",
		Long: `Check whether Quad9 has flagged a domain as malicious using threat intelligence
from 19+ security partners.

Quad9 returns NXDOMAIN with an empty authority section for known-malicious domains,
providing a passive threat-intelligence verdict without revealing query origin
to the target domain.

PAP level: AMBER (queries go to Quad9 third-party servers).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Check whether a domain is blocked
  trident quad9 malicious.example.com

  # Multiple domains
  trident quad9 example.com malicious.example.com

  # Bulk input from stdin
  cat domains.txt | trident quad9

  # JSON output
  trident quad9 --output json example.com`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := d.newHTTPClient()
			if err != nil {
				return err
			}
			client.EnableForceHTTP2()
			httpclient.AttachRateLimit(client, ratelimit.New(doh.DefaultRPS, doh.DefaultBurst))
			svc := quad9svc.NewService(client, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
