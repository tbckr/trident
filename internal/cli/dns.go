package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/resolver"
	dnssvc "github.com/tbckr/trident/internal/services/dns"
)

func newDNSCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "dns [domain|ip...]",
		Short:   "Perform DNS lookups for a domain or reverse lookup for an IP",
		GroupID: "osint",
		Long: `Perform DNS lookups for one or more domains or IP addresses.

Queries A, AAAA, MX, NS, TXT records for domains. For IP addresses, performs a
reverse PTR lookup. Results are grouped by record type.

PAP level: GREEN (direct interaction with the target's DNS servers).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Single domain
  trident dns example.com

  # Reverse PTR lookup for an IP
  trident dns 8.8.8.8

  # Multiple domains as arguments
  trident dns example.com example.org

  # Bulk input from stdin
  echo -e "example.com\nexample.org" | trident dns

  # JSON output
  trident dns --output json example.com`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolver.NewResolver(d.cfg.Proxy)
			if err != nil {
				return fmt.Errorf("creating DNS resolver: %w", err)
			}
			svc := dnssvc.NewService(r, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
