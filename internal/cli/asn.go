package cli

import (
	"io"
	"net"

	"github.com/spf13/cobra"

	asnsvc "github.com/tbckr/trident/internal/services/asn"
)

func newASNCmd(stdout, stderr io.Writer, configFile *string, verbose *bool, outputFmt *string) *cobra.Command {
	return &cobra.Command{
		Use:   "asn <ip|ASN>",
		Short: "Look up ASN information for an IP address or ASN (e.g. AS15169)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// cfg is unused in Phase 1; proxy and rate-limiting config will be consumed in Phase 2.
			_, logger, format, err := buildDeps(stderr, configFile, verbose, outputFmt)
			if err != nil {
				return err
			}

			resolver := &net.Resolver{}
			svc := asnsvc.NewService(resolver, logger)

			result, err := svc.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			asnResult, ok := result.(*asnsvc.Result)
			if ok && asnResult.IsEmpty() {
				logger.Info("no ASN data found", "input", args[0])
				return nil
			}
			return writeResult(stdout, format, result)
		},
	}
}
