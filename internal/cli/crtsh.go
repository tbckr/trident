package cli

import (
	"io"

	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"

	crtshsvc "github.com/tbckr/trident/internal/services/crtsh"
)

func newCrtshCmd(stdout, stderr io.Writer, configFile *string, verbose *bool, outputFmt *string) *cobra.Command {
	return &cobra.Command{
		Use:   "crtsh <domain>",
		Short: "Search crt.sh certificate transparency logs for subdomains",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, logger, format, err := buildDeps(stderr, configFile, verbose, outputFmt)
			if err != nil {
				return err
			}

			client := req.NewClient()
			svc := crtshsvc.NewService(client, logger)

			result, err := svc.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeResult(stdout, format, result)
		},
	}
}
