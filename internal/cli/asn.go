package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/resolver"
	asnsvc "github.com/tbckr/trident/internal/services/asn"
)

func newASNCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "asn [ip|ASN...]",
		Short:   "Look up ASN information for an IP address or ASN (e.g. AS15169)",
		GroupID: "osint",
		Long: `Look up ASN (Autonomous System Number) information for an IP address or ASN.

For IP addresses, resolves the originating ASN via Team Cymru's DNS-based
service (origin.asn.cymru.com). Supports both IPv4 and IPv6.
For ASN identifiers (e.g. AS15169), retrieves the AS name and description.

PAP level: AMBER (queries Team Cymru's third-party DNS service).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # IP address to ASN
  trident asn 8.8.8.8

  # IPv6 address
  trident asn 2001:4860:4860::8888

  # ASN details by number
  trident asn AS15169

  # Bulk input from stdin
  echo -e "8.8.8.8\n1.1.1.1" | trident asn

  # JSON output
  trident asn --output json 8.8.8.8`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolver.NewResolver(d.cfg.Proxy)
			if err != nil {
				return fmt.Errorf("creating DNS resolver: %w", err)
			}
			svc := asnsvc.NewService(r, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
