package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	tmsvc "github.com/tbckr/trident/internal/services/threatminer"
	"github.com/tbckr/trident/internal/worker"
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

			if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), svc.PAP()) {
				return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
					services.ErrPAPBlocked, svc.Name(), svc.PAP(), pap.MustParse(d.cfg.PAPLimit))
			}

			inputs, err := resolveInputs(cmd, args)
			if err != nil {
				return err
			}
			if len(inputs) == 0 {
				return fmt.Errorf("no input: supply a domain, IP, or hash as argument or pipe via stdin")
			}

			if len(inputs) == 1 {
				result, err := svc.Run(cmd.Context(), inputs[0])
				if err != nil {
					return err
				}
				if r, ok := result.(*tmsvc.Result); ok && r.IsEmpty() {
					d.logger.Info("no ThreatMiner data found", "input", inputs[0])
					return nil
				}
				return writeResult(cmd.OutOrStdout(), d, result)
			}

			// Bulk mode
			results := worker.Run(cmd.Context(), svc, inputs, d.cfg.Concurrency)
			var valid []*tmsvc.Result
			for _, r := range results {
				if r.Err != nil {
					d.logger.Error("threatminer lookup failed", "input", r.Input, "error", r.Err)
					continue
				}
				if tr, ok := r.Output.(*tmsvc.Result); ok && tr.IsEmpty() {
					d.logger.Info("no ThreatMiner data found", "input", r.Input)
					continue
				}
				if tr, ok := r.Output.(*tmsvc.Result); ok {
					valid = append(valid, tr)
				}
			}
			switch len(valid) {
			case 0:
				return nil
			case 1:
				return writeResult(cmd.OutOrStdout(), d, valid[0])
			default:
				return writeResult(cmd.OutOrStdout(), d, &tmsvc.MultiResult{Results: valid})
			}
		},
	}
}
