package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/input"
	"github.com/tbckr/trident/internal/services/asn"
)

func NewASNCmd(logger *slog.Logger, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asn [AS number or IP]",
		Short: "Perform Autonomous System Number (ASN) lookups",
		Long: `Retrieve owner, country, and registry information for an ASN or IP address.
This command queries Team Cymru's DNS TXT record interface and supports 
bulk processing via stdin.

PAP Level: AMBER (Passive query - interacts with 3rd party infrastructure)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := input.GetInputs(args, cmd.InOrStdin())
			if err != nil {
				return err
			}

			service := asn.NewService(logger, nil)

			for _, target := range targets {
				result, err := service.Lookup(cmd.Context(), target)
				if err != nil {
					logger.Error("lookup failed", "target", target, "error", err)
					continue
				}
				logger.Info("asn lookup result", "target", target, "result", result)
			}

			return nil
		},
	}

	return cmd
}
