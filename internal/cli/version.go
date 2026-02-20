package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/output"
)

// Build-time variables injected via ldflags:
//
//	-X github.com/tbckr/trident/internal/cli.version=v1.0.0
//	-X github.com/tbckr/trident/internal/cli.commit=abc1234
//	-X github.com/tbckr/trident/internal/cli.date=2024-01-01
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
			info := versionInfo{Version: version, Commit: commit, Date: date}
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
