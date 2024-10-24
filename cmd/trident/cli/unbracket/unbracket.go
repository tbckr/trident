package unbracket

import (
	"github.com/spf13/cobra"
	"github.com/tbckr/trident/pkg/cli"
	"github.com/tbckr/trident/pkg/opsec"
)

type UnbracketCmd struct {
	Cmd *cobra.Command
}

func NewBracketCmd() *UnbracketCmd {
	cmdStruct := &UnbracketCmd{}
	cmd := &cobra.Command{
		Use:                   "unbracket [domains...]",
		Short:                 "Unbracket domains",
		Long:                  `Unbracket domains in context of Operational Security (OPSEC)`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.PipeCliCommand(cmd, args, opsec.UnbracketDomain)
		},
	}
	cmdStruct.Cmd = cmd
	return cmdStruct
}
