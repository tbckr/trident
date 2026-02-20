package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/httpclient"
	tmsvc "github.com/tbckr/trident/internal/services/threatminer"
)

func newThreatMinerCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "threatminer [domain|ip|hash...]",
		Short:   "Query ThreatMiner for passive DNS, subdomains, or file hash metadata",
		GroupID: "osint",
		Long: `Query the ThreatMiner API for threat intelligence data.

Automatically detects the input type and queries the appropriate endpoint:
  - Domain: passive DNS, subdomains, related samples (domain.php)
  - IP address: passive DNS, URI/domain associations (host.php)
  - File hash (MD5/SHA1/SHA256): metadata, AV detections (sample.php)

A 404-status response from ThreatMiner is treated as "no data found" (not an
error). Results vary by input type.

PAP level: AMBER (queries the ThreatMiner third-party API).

Multiple inputs can be supplied as arguments or piped via stdin (one per line).
Bulk stdin input is processed concurrently (see --concurrency).`,
		Example: `  # Domain lookup (passive DNS, subdomains)
  trident threatminer example.com

  # IP address lookup (passive DNS)
  trident threatminer 1.2.3.4

  # File hash lookup (SHA256)
  trident threatminer e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

  # Bulk input from stdin
  echo -e "example.com\n1.2.3.4" | trident threatminer

  # JSON output
  trident threatminer --output json example.com`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httpclient.New(d.cfg.Proxy, d.cfg.UserAgent, d.logger, d.cfg.Verbose)
			if err != nil {
				return fmt.Errorf("creating HTTP client: %w", err)
			}
			svc := tmsvc.NewService(client, d.logger)
			return runServiceCmd(cmd, d, svc, args)
		},
	}
}
