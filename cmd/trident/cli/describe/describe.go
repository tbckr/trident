package describe

import (
	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/config"
	descriptor "github.com/tbckr/trident/pkg/describe/securitytrails"
	"github.com/tbckr/trident/pkg/opsec"
	"github.com/tbckr/trident/pkg/pap"
	plugin "github.com/tbckr/trident/pkg/plugins/securitytrails"
	"github.com/tbckr/trident/pkg/report"
	"github.com/tbckr/trident/pkg/writer/shell"
	"strings"
)

type DescribeCmd struct {
	Cmd *cobra.Command
}

type DomainCmd struct {
	Cmd *cobra.Command
}

func NewDescribeCmd(viperConfig *config.Config, reqClient *req.Client) *DescribeCmd {
	cmdStruct := &DescribeCmd{}
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe a target",
		Long: `PAP Level: depends on the strategy

Fetches data from several sources and generates a report in a opinionated way`,
		GroupID:               cli.GroupPlugins,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newDomainCmd(viperConfig, reqClient).Cmd,
	)

	cmdStruct.Cmd = cmd
	return cmdStruct
}

func newDomainCmd(viperConfig *config.Config, reqClient *req.Client) *DomainCmd {
	cmdStruct := &DomainCmd{}
	cmd := &cobra.Command{
		Use:     "domain [domain]",
		Aliases: []string{"d"},
		Short:   "Describe a target based on a domain",
		Long: `PAP Level: depends on the strategy

Fetches data from several sources and generates a report for a domain in a opinionated way`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// TODO check pap level based on strategy
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get domain
			domain := strings.ToLower(args[0])
			domain = opsec.UnbracketDomain(domain)

			// Get PAP level
			environmentPapLevel, err := viperConfig.GetEnvironmentPapLevel()
			if err != nil {
				return err
			}
			escapeDomain := pap.IsEscapeData(environmentPapLevel) && !viperConfig.GetDisableDomainBrackets()

			// Get api key
			var apiKey string
			apiKey, err = viperConfig.GetSecurityTrailsApiKey()
			if err != nil {
				return err
			}

			// Build client
			client := plugin.NewSecurityTrailsClient(reqClient, apiKey)

			// Describe domain
			strategy := descriptor.NewSecuritytrailsStrategy(client, escapeDomain)
			var domainDescription report.DomainDescriptionReport
			domainDescription, err = strategy.DescribeDomain(cmd.Context(), domain)

			// Print report
			var w *shell.Writer
			w, err = shell.NewShellWriter()
			if err != nil {
				return err
			}
			err = w.WriteDomainDescriptionReport(cmd.OutOrStdout(), domainDescription)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmdStruct.Cmd = cmd
	return cmdStruct
}
