package bracket

import (
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/opsec"
)

type BracketCmd struct {
	Cmd *cobra.Command
}

func NewBracketCmd() *BracketCmd {
	cmdStruct := &BracketCmd{}
	cmd := &cobra.Command{
		Use:                   "bracket [domains...]",
		Short:                 "Bracket domains",
		Long:                  `Bracket domains for Operational Security (OPSEC)`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.PipeCliCommand(cmd, args, opsec.BracketDomain)
		},
	}
	cmdStruct.Cmd = cmd
	return cmdStruct
}
