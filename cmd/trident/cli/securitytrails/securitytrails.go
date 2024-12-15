package securitytrails

import (
	"bufio"
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/config"
	"github.com/tbckr/trident/pkg/opsec"
	"github.com/tbckr/trident/pkg/pap"
	plugin "github.com/tbckr/trident/pkg/plugins/securitytrails"
	securitytrailsReport "github.com/tbckr/trident/pkg/report/securitytrails"
	"github.com/tbckr/trident/pkg/writer/shell"
	"strings"
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
		newDomainCmd(viperConfig, reqClient).Cmd,
		newSubdomainCmd(viperConfig, reqClient).Cmd,
	)

	cmdStruct.Cmd = cmd
	return cmdStruct
}

func run(cmd *cobra.Command, args []string, viperConfig *config.Config, reqClient *req.Client, handler func(environmentPapLevel pap.PapLevel, client *plugin.SecurityTrailsClient, domain string) error) error {
	// Get input
	input, err := cli.InputFromCli(cmd, args)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(input)

	// Get PAP level
	var environmentPapLevel pap.PapLevel
	environmentPapLevel, err = viperConfig.GetEnvironmentPapLevel()
	if err != nil {
		return err
	}

	// Get api key
	var apiKey string
	apiKey, err = viperConfig.GetSecurityTrailsApiKey()
	if err != nil {
		return err
	}

	// Build client
	client := plugin.NewSecurityTrailsClient(reqClient, apiKey)

	var domain string
	for sc.Scan() {
		domain = strings.ToLower(sc.Text())
		domain = opsec.UnbracketDomain(domain)
		err = handler(environmentPapLevel, client, domain)
		if err != nil {
			return err
		}
	}
	return nil
}

func newDomainCmd(viperConfig *config.Config, reqClient *req.Client) *DomainCmd {
	cmdStruct := &DomainCmd{}
	cmd := &cobra.Command{
		Use:     "domain [domains...]",
		Aliases: []string{"d", "domains"},
		Short:   "Fetch domain information from securitytrails",
		Long: `PAP Level: AMBER

Fetch domain information from securitytrails`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		PreRunE:               cli.PapPreRunCheck(viperConfig, pap.LevelAmber),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO do we want to support multiple domains?
			return run(cmd, args, viperConfig, reqClient, func(environmentPapLevel pap.PapLevel, client *plugin.SecurityTrailsClient, domain string) error {
				// Query domain details
				resp, err := client.GetDomainDetails(cmd.Context(), domain)
				if err != nil {
					return err
				}

				// Generate report
				rep := securitytrailsReport.GenerateDomainReport(resp, pap.IsEscapeData(environmentPapLevel))

				// Write report to shell
				var w *shell.Writer
				w, err = shell.NewShellWriter()
				if err != nil {
					return err
				}
				err = w.WriteDomainReport(cmd.OutOrStdout(), rep)
				if err != nil {
					return err
				}
				return nil
			})
		},
	}
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
			return run(cmd, args, viperConfig, reqClient, func(environmentPapLevel pap.PapLevel, client *plugin.SecurityTrailsClient, domain string) error {
				resp, err := client.GetSubdomains(cmd.Context(), domain, cmdStruct.subdomainsOnly, cmdStruct.includeInactive)
				for _, d := range resp.Subdomains {
					d = fmt.Sprintf("%s.%s", d, domain)
					if pap.IsEscapeData(environmentPapLevel) && !viperConfig.GetDisableDomainBrackets() {
						d = opsec.BracketDomain(d)
					}
					_, err = fmt.Fprintln(cmd.OutOrStdout(), d)
					if err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVarP(&cmdStruct.subdomainsOnly, "subdomains-only", "s", false, "Only subdomains")
	cmd.Flags().BoolVarP(&cmdStruct.includeInactive, "include-inactive", "i", false, "Include inactive subdomains")

	cmdStruct.Cmd = cmd
	return cmdStruct
}
