package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/input"
	"github.com/tbckr/trident/internal/services/dns"
)

func NewDNSCmd(logger *slog.Logger, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns [domain or IP]",
		Short: "Perform DNS lookups (A, AAAA, MX, NS, TXT, CNAME)",
		Long: `Perform DNS lookups for domains or reverse lookups for IP addresses.
This command queries multiple record types (A, AAAA, MX, NS, TXT, CNAME) and 
supports bulk processing via stdin.

PAP Level: GREEN (Active query - interacts with target nameservers via recursion)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := input.GetInputs(args, cmd.InOrStdin())
			if err != nil {
				return err
			}

			service := dns.NewService(logger, nil)

			for _, target := range targets {
				result, err := service.Lookup(cmd.Context(), target)
				if err != nil {
					logger.Error("lookup failed", "target", target, "error", err)
					continue
				}
				logger.Info("dns lookup result", "target", target, "result", result)
			}

			return nil
		},
	}

	return cmd
}
