package cli

import (
	"github.com/spf13/cobra"

	detectsvc "github.com/tbckr/trident/internal/services/detect"
)

func newDetectCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "detect [domain...]",
		Short:   "Detect CDN, email, and DNS hosting providers",
		GroupID: "services",
		Long: `Detect CDN, email, and DNS hosting providers for one or more domains.

Queries CNAME (apex and www), MX, NS, and TXT records and matches them against
known provider patterns to identify cloud services in use.

PAP level: GREEN (direct interaction with the target's DNS servers).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  trident detect example.com`,
		Args:    cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := d.newResolver()
			if err != nil {
				return err
			}
			patterns, err := d.loadPatterns()
			if err != nil {
				return err
			}
			svc := detectsvc.NewService(r, d.logger, patterns)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
