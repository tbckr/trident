package cli

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	dnssvc "github.com/tbckr/trident/internal/services/dns"
	"github.com/tbckr/trident/internal/worker"
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
			resolver := &net.Resolver{}
			svc := dnssvc.NewService(resolver, d.logger)

			if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), svc.PAP()) {
				return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
					services.ErrPAPBlocked, svc.Name(), svc.PAP(), pap.MustParse(d.cfg.PAPLimit))
			}

			inputs, err := resolveInputs(cmd, args)
			if err != nil {
				return err
			}
			if len(inputs) == 0 {
				return fmt.Errorf("no input: supply a domain or IP as argument or pipe via stdin")
			}

			if len(inputs) == 1 {
				result, err := svc.Run(cmd.Context(), inputs[0])
				if err != nil {
					return err
				}
				if r, ok := result.(*dnssvc.Result); ok && r.IsEmpty() {
					d.logger.Info("no DNS records found", "input", inputs[0])
					return nil
				}
				return writeResult(cmd.OutOrStdout(), d, result)
			}

			// Bulk mode
			results := worker.Run(cmd.Context(), svc, inputs, d.cfg.Concurrency)
			var valid []*dnssvc.Result
			for _, r := range results {
				if r.Err != nil {
					d.logger.Error("dns lookup failed", "input", r.Input, "error", r.Err)
					continue
				}
				if dr, ok := r.Output.(*dnssvc.Result); ok && dr.IsEmpty() {
					d.logger.Info("no DNS records found", "input", r.Input)
					continue
				}
				if dr, ok := r.Output.(*dnssvc.Result); ok {
					valid = append(valid, dr)
				}
			}
			switch len(valid) {
			case 0:
				return nil
			case 1:
				return writeResult(cmd.OutOrStdout(), d, valid[0])
			default:
				return writeResult(cmd.OutOrStdout(), d, &dnssvc.MultiResult{Results: valid})
			}
		},
	}
}
