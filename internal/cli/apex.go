package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/doh"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
	"github.com/tbckr/trident/internal/resolver"
	apexsvc "github.com/tbckr/trident/internal/services/apex"
)

func newApexCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "apex [domain...]",
		Short:   "Aggregate DNS recon for an apex domain",
		GroupID: "aggregate",
		Long: `Perform parallel DNS reconnaissance for an apex domain via Quad9 DoH.

apex fans out queries for the apex domain and well-known derived hostnames
(www, mail, autodiscover, _dmarc, _domainkey, _mta-sts, DKIM selectors, BIMI)
and consolidates the results into a single output. CNAME chains are followed
automatically and CDN providers are detected from CNAME targets. Results are
returned in a deterministic order matching the query list.

Queried record types: A, AAAA, CAA, DNSKEY, HTTPS, MX, NS, SOA, SRV, SSHFP,
TXT, CNAME (direct + chain).
SRV services: _sip._tls, _sipfederationtls._tcp, _xmpp-client._tcp,
_xmpp-server._tcp.
Derived hostnames: www, mail, autodiscover, _dmarc, _domainkey, _mta-sts,
_smtp._tls, default._bimi, google._domainkey, selector1._domainkey,
selector2._domainkey.

PAP level: AMBER (queries go to Quad9 and Cymru third-party servers).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Full DNS recon for an apex domain
  trident apex example.com

  # Multiple domains
  trident apex example.com example.org

  # JSON output
  trident apex --output json example.com

  # Bulk input from stdin
  cat domains.txt | trident apex`,
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
			httpclient.AttachRateLimit(client, ratelimit.New(doh.DefaultRPS, doh.DefaultBurst))
			r, err := resolver.NewResolver(d.cfg.Proxy)
			if err != nil {
				return fmt.Errorf("creating DNS resolver: %w", err)
			}
			paths, err := providers.DefaultPatternPaths()
			if err != nil {
				return fmt.Errorf("resolving pattern paths: %w", err)
			}
			if d.cfg.DetectPatterns.File != "" {
				paths = append([]string{d.cfg.DetectPatterns.File}, paths...)
			}
			patterns, err := providers.LoadPatterns(paths...)
			if err != nil {
				return fmt.Errorf("loading detect patterns: %w", err)
			}
			svc := apexsvc.NewService(client, r, d.logger, patterns)
			return runAggregateCmd(cmd, d, svc, args)
		},
	}
}
