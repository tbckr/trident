package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/version"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func newVersionCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Print the trident version",
		Args:    cobra.NoArgs,
		GroupID: "utility",
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := versionInfo{Version: version.Version, Commit: version.Commit, Date: version.Date}
			if output.Format(d.cfg.Output) == output.FormatJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(),
				"trident version %s (commit: %s, built: %s)\n",
				info.Version, info.Commit, info.Date)
			return err
		},
	}
}
