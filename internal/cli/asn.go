package cli

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	asnsvc "github.com/tbckr/trident/internal/services/asn"
	"github.com/tbckr/trident/internal/worker"
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
			resolver := &net.Resolver{}
			svc := asnsvc.NewService(resolver, d.logger)

			if !pap.Allows(pap.MustParse(d.cfg.PAPLimit), svc.PAP()) {
				return fmt.Errorf("%w: %q requires PAP %s but limit is %s",
					services.ErrPAPBlocked, svc.Name(), svc.PAP(), pap.MustParse(d.cfg.PAPLimit))
			}

			inputs, err := resolveInputs(cmd, args)
			if err != nil {
				return err
			}
			if len(inputs) == 0 {
				return fmt.Errorf("no input: supply an IP or ASN as argument or pipe via stdin")
			}

			if len(inputs) == 1 {
				result, err := svc.Run(cmd.Context(), inputs[0])
				if err != nil {
					return err
				}
				if r, ok := result.(*asnsvc.Result); ok && r.IsEmpty() {
					d.logger.Info("no ASN data found", "input", inputs[0])
					return nil
				}
				return writeResult(cmd.OutOrStdout(), d, result)
			}

			// Bulk mode
			results := worker.Run(cmd.Context(), svc, inputs, d.cfg.Concurrency)
			var valid []*asnsvc.Result
			for _, r := range results {
				if r.Err != nil {
					d.logger.Error("ASN lookup failed", "input", r.Input, "error", r.Err)
					continue
				}
				if ar, ok := r.Output.(*asnsvc.Result); ok && ar.IsEmpty() {
					d.logger.Info("no ASN data found", "input", r.Input)
					continue
				}
				if ar, ok := r.Output.(*asnsvc.Result); ok {
					valid = append(valid, ar)
				}
			}
			switch len(valid) {
			case 0:
				return nil
			case 1:
				return writeResult(cmd.OutOrStdout(), d, valid[0])
			default:
				return writeResult(cmd.OutOrStdout(), d, &asnsvc.MultiResult{Results: valid})
			}
		},
	}
}
