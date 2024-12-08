package certspotter

import (
	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/client"
	"github.com/tbckr/trident/pkg/config"
	"github.com/tbckr/trident/pkg/pap"
	"github.com/tbckr/trident/pkg/plugins/certspotter"
)

type CertspotterCmd struct {
	Cmd *cobra.Command
}

type DomainCmd struct {
	Cmd *cobra.Command

	opts client.DomainFetcherOptions
}

func NewCertspotterCmd(viperConfig *config.Config, reqClient *req.Client) *CertspotterCmd {
	cmdStruct := &CertspotterCmd{}
	cmd := &cobra.Command{
		Use:   "certspotter",
		Short: "Fetch data from certspotter",
		Long: `PAP Level: AMBER

Fetch domains from certspotter`,
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
	cmdStruct := &DomainCmd{
		opts: client.DomainFetcherOptions{
			OnlyUnique:     false,
			OnlySubdomains: false,
		},
	}
	cmd := &cobra.Command{
		Use:     "domain [domains...]",
		Aliases: []string{"d", "domains"},
		Short:   "Fetch domains from certspotter",
		Long: `PAP Level: AMBER

Fetch domains from certspotter`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		PreRunE:               cli.PapPreRunCheck(viperConfig, pap.LevelAmber),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.DomainFetcherCliCommand(cmd, args, viperConfig, certspotter.NewCertspotterClient(reqClient), cmdStruct.opts)
		},
	}

	cmd.Flags().BoolVarP(&cmdStruct.opts.OnlyUnique, "only-unique", "u", false, "Only unique domains")
	cmd.Flags().BoolVarP(&cmdStruct.opts.OnlySubdomains, "only-subdomains", "s", false, "Only subdomains")

	cmdStruct.Cmd = cmd
	return cmdStruct
}
