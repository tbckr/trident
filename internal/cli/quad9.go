package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
	quad9svc "github.com/tbckr/trident/internal/services/quad9"
)

func newQuad9Cmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "quad9 <subcommand> [domain...]",
		Short:   "Query domains via Quad9 DNS-over-HTTPS",
		GroupID: "osint",
		Long: `Query domains via the Quad9 DNS-over-HTTPS (DoH) resolver.

Quad9 is a security-focused DNS resolver that integrates threat intelligence
from 19+ partners. Two subcommands are available:

  resolve  Standard DNS record lookups (A, AAAA, NS, MX, TXT) via Quad9 DoH.
  blocked  Detect whether Quad9 has flagged a domain as malicious.

PAP level: AMBER (queries go to Quad9 third-party servers).`,
	}
	cmd.AddCommand(newQuad9ResolveCmd(d), newQuad9BlockedCmd(d))
	return cmd
}

func newQuad9ResolveCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve [domain...]",
		Short: "Resolve DNS records for a domain via Quad9 DoH",
		Long: `Resolve DNS records (A, AAAA, NS, MX, TXT) for one or more domains using
the Quad9 DNS-over-HTTPS endpoint (https://dns.quad9.net/dns-query).

Results are grouped by record type in canonical order: NS → A → AAAA → MX → TXT.

PAP level: AMBER (queries go to Quad9 third-party servers).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Resolve DNS records for a domain
  trident quad9 resolve example.com

  # Multiple domains as arguments
  trident quad9 resolve example.com example.org

  # Bulk input from stdin
  echo -e "example.com\nexample.org" | trident quad9 resolve

  # JSON output
  trident quad9 resolve --output json example.com

  # Plain text output
  trident quad9 resolve --output plain example.com`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httpclient.New(d.cfg.Proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
			if err != nil {
				return fmt.Errorf("creating HTTP client: %w", err)
			}
			client.EnableForceHTTP2()
			httpclient.AttachRateLimit(client, ratelimit.New(quad9svc.DefaultRPS, quad9svc.DefaultBurst))
			svc := quad9svc.NewResolveService(client, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}

func newQuad9BlockedCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "blocked [domain...]",
		Short: "Check whether Quad9 has blocked a domain as malicious",
		Long: `Check whether Quad9 has flagged a domain as malicious using threat intelligence
from 19+ security partners.

Quad9 returns NXDOMAIN with an empty authority section for known-malicious domains,
providing a passive threat-intelligence verdict without revealing query origin
to the target domain.

PAP level: AMBER (queries go to Quad9 third-party servers).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Check whether a domain is blocked
  trident quad9 blocked malicious.example.com

  # Multiple domains
  trident quad9 blocked example.com malicious.example.com

  # Bulk input from stdin
  cat domains.txt | trident quad9 blocked

  # JSON output
  trident quad9 blocked --output json example.com`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httpclient.New(d.cfg.Proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
			if err != nil {
				return fmt.Errorf("creating HTTP client: %w", err)
			}
			client.EnableForceHTTP2()
			httpclient.AttachRateLimit(client, ratelimit.New(quad9svc.DefaultRPS, quad9svc.DefaultBurst))
			svc := quad9svc.NewBlockedService(client, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
