package cli

import (
	"github.com/spf13/cobra"

	cymrusvc "github.com/tbckr/trident/internal/services/cymru"
)

func newCymruCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "cymru [ip|ASN...]",
		Short:   "Look up ASN information for an IP address or ASN (e.g. AS15169)",
		GroupID: "services",
		Long: `Look up ASN (Autonomous System Number) information for an IP address or ASN.

For IP addresses, resolves the originating ASN via Team Cymru's DNS-based
service (origin.asn.cymru.com). Supports both IPv4 and IPv6.
For ASN identifiers (e.g. AS15169), retrieves the AS name and description.

PAP level: AMBER (queries Team Cymru's third-party DNS service).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # IP address to ASN
  trident cymru 8.8.8.8

  # IPv6 address
  trident cymru 2001:4860:4860::8888

  # ASN details by number
  trident cymru AS15169

  # Bulk input from stdin
  echo -e "8.8.8.8\n1.1.1.1" | trident cymru

  # JSON output
  trident cymru --output json 8.8.8.8`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := d.newResolver()
			if err != nil {
				return err
			}
			svc := cymrusvc.NewService(r, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
