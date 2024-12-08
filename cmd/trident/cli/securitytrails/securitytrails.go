package securitytrails

import (
	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/config"
	"github.com/tbckr/trident/pkg/pap"
)

type SecurityTrailsCmd struct {
	Cmd *cobra.Command
}

type DomainCmd struct {
	Cmd *cobra.Command
}

type SubdomainCmd struct {
	Cmd *cobra.Command

	subdomainsOnly  bool
	includeInactive bool
}

func NewSecurityTrailsCmd(viperConfig *config.Config, reqClient *req.Client) *SecurityTrailsCmd {
	cmdStruct := &SecurityTrailsCmd{}
	cmd := &cobra.Command{
		Use:   "securitytrails",
		Short: "Fetch data from securitytrails",
		Long: `PAP Level: AMBER

Fetch data from securitytrails`,
		GroupID:               cli.GroupPlugins,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newSubdomainCmd(viperConfig, reqClient).Cmd,
	)

	cmdStruct.Cmd = cmd
	return cmdStruct
}

func newSubdomainCmd(viperConfig *config.Config, reqClient *req.Client) *SubdomainCmd {
	cmdStruct := &SubdomainCmd{}
	cmd := &cobra.Command{
		Use:     "subdomain [domains...]",
		Aliases: []string{"s", "subdomains"},
		Short:   "Fetch subdomains from securitytrails",
		Long: `PAP Level: AMBER

Fetch domains from securitytrails`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		PreRunE:               cli.PapPreRunCheck(viperConfig, pap.LevelAmber),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO Implement
			return nil
		},
	}
	cmd.Flags().BoolVarP(&cmdStruct.subdomainsOnly, "subdomains-only", "s", false, "Only subdomains")
	cmd.Flags().BoolVarP(&cmdStruct.includeInactive, "include-inactive", "i", false, "Include inactive subdomains")

	cmdStruct.Cmd = cmd
	return cmdStruct
}
