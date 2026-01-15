package cli

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func NewRootCmd(logger *slog.Logger, levelVar *slog.LevelVar) *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "myapp",
		Short: "A brief description of your application",
		Long:  `A longer description that spans multiple lines...`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Setup that runs before all commands
			if configFile != "" {
				// Load config file
			}
			return nil
		},
	}

	// Persistent flags available to all subcommands
	cmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&levelVar.Level().String(), "log-level", "info", "log level (debug|info|warn|error)")

	// Add subcommands
	cmd.AddCommand(NewVersionCmd())

	return cmd
}
