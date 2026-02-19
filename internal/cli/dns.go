package cli

import (
	"io"
	"net"

	"github.com/spf13/cobra"

	dnssvc "github.com/tbckr/trident/internal/services/dns"
)

func newDNSCmd(stdout, stderr io.Writer, configFile *string, verbose *bool, outputFmt *string) *cobra.Command {
	return &cobra.Command{
		Use:   "dns <domain|ip>",
		Short: "Perform DNS lookups for a domain or reverse lookup for an IP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// cfg is unused in Phase 1; proxy and rate-limiting config will be consumed in Phase 2.
			_, logger, format, err := buildDeps(stderr, configFile, verbose, outputFmt)
			if err != nil {
				return err
			}

			resolver := &net.Resolver{}
			svc := dnssvc.NewService(resolver, logger)

			result, err := svc.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeResult(stdout, format, result)
		},
	}
}
