package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/pap"
	identifysvc "github.com/tbckr/trident/internal/services/identify"
)

func newIdentifyCmd(d *deps) *cobra.Command {
	var (
		domain     string
		cnames     []string
		mxHosts    []string
		nsHosts    []string
		txtRecords []string
	)
	cmd := &cobra.Command{
		Use:     "identify",
		Short:   "Identify CDN, email, DNS, and verification providers from known DNS records",
		GroupID: "services",
		Long: `Matches CNAME, MX, NS, and TXT record values against known provider patterns to
identify CDN, email, DNS hosting, and domain verification providers.

Unlike the detect command, identify does not make any DNS queries — it operates
entirely on record values you already have. This makes it suitable for offline
use and PAP RED environments.

PAP level: RED (no network calls — pure pattern matching).`,
		Example: `  trident identify --cname abc.cloudfront.net
  trident identify --domain example.com --ns ns1.cloudflare.com
  trident identify --domain example.com --cname abc.cloudfront.net --mx aspmx.l.google.com --ns ns1.cloudflare.com
  trident identify --txt "google-site-verification=abc123" --txt "v=spf1 include:_spf.google.com ~all"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := providers.DefaultPatternPaths()
			if err != nil {
				return fmt.Errorf("resolving pattern paths: %w", err)
			}
			patterns, err := providers.LoadPatterns(paths...)
			if err != nil {
				return fmt.Errorf("loading detect patterns: %w", err)
			}
			svc := identifysvc.NewService(d.logger, patterns)
			if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), svc.PAP()) {
				return fmt.Errorf("service %s requires PAP level %s, but limit is %s", svc.Name(), svc.PAP(), d.cfg.PAPLimit)
			}
			if len(cnames)+len(mxHosts)+len(nsHosts)+len(txtRecords) == 0 {
				return fmt.Errorf("no records provided: specify at least one --cname, --mx, --ns, or --txt value")
			}
			result, err := svc.Run(cnames, mxHosts, nsHosts, txtRecords)
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
	cmd.Flags().StringArrayVar(&txtRecords, "txt", nil, "TXT record value (repeatable)")
	return cmd
}
