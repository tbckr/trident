package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/pap"
	identifysvc "github.com/tbckr/trident/internal/services/identify"
)

func newIdentifyCmd(d *deps) *cobra.Command {
	var (
		domain  string
		cnames  []string
		mxHosts []string
		nsHosts []string
	)
	cmd := &cobra.Command{
		Use:     "identify",
		Short:   "Identify CDN, email, and DNS hosting providers from known DNS records",
		GroupID: "services",
		Long: `Matches CNAME, MX, and NS record values against known provider patterns to
identify CDN, email, and DNS hosting providers.

Unlike the detect command, identify does not make any DNS queries — it operates
entirely on record values you already have. This makes it suitable for offline
use and PAP RED environments.

PAP level: RED (no network calls — pure pattern matching).`,
		Example: `  trident identify --cname abc.cloudfront.net
  trident identify --domain example.com --ns ns1.cloudflare.com
  trident identify --domain example.com --cname abc.cloudfront.net --mx aspmx.l.google.com --ns ns1.cloudflare.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := identifysvc.NewService(d.logger)
			if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), svc.PAP()) {
				return fmt.Errorf("service %s requires PAP level %s, but limit is %s", svc.Name(), svc.PAP(), d.cfg.PAPLimit)
			}
			if len(cnames)+len(mxHosts)+len(nsHosts) == 0 {
				return fmt.Errorf("no records provided: specify at least one --cname, --mx, or --ns value")
			}
			result, err := svc.Run(cnames, mxHosts, nsHosts)
			if err != nil {
				return err
			}
			result.Input = domain
			if result.IsEmpty() {
				d.logger.Info("no providers detected")
				return nil
			}
			return writeResult(cmd.OutOrStdout(), d, result)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "optional label for output (e.g. the domain these records belong to)")
	cmd.Flags().StringArrayVar(&cnames, "cname", nil, "CNAME target value (repeatable)")
	cmd.Flags().StringArrayVar(&mxHosts, "mx", nil, "MX exchange hostname (repeatable)")
	cmd.Flags().StringArrayVar(&nsHosts, "ns", nil, "NS server hostname (repeatable)")
	return cmd
}
