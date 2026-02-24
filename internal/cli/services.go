package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	apexsvc "github.com/tbckr/trident/internal/services/apex"
	crtshsvc "github.com/tbckr/trident/internal/services/crtsh"
	cymrusvc "github.com/tbckr/trident/internal/services/cymru"
	detectsvc "github.com/tbckr/trident/internal/services/detect"
	dnssvc "github.com/tbckr/trident/internal/services/dns"
	identifysvc "github.com/tbckr/trident/internal/services/identify"
	pgpsvc "github.com/tbckr/trident/internal/services/pgp"
	quad9svc "github.com/tbckr/trident/internal/services/quad9"
	threatsvc "github.com/tbckr/trident/internal/services/threatminer"
)

type serviceEntry struct {
	Name  string `json:"name"`
	Group string `json:"group"`
	PAP   string `json:"pap"`
}

// allServices returns a fixed-order list of every service and aggregate command.
// Services are ordered alphabetically within each group; "services" precedes "aggregate".
func allServices() []serviceEntry {
	type meta struct {
		name  string
		pap   pap.Level
		group string
	}
	metas := []meta{
		// services group — alphabetical
		{cymrusvc.Name, cymrusvc.PAP, "services"},
		{crtshsvc.Name, crtshsvc.PAP, "services"},
		{detectsvc.Name, detectsvc.PAP, "services"},
		{dnssvc.Name, dnssvc.PAP, "services"},
		{identifysvc.Name, identifysvc.PAP, "services"},
		{pgpsvc.Name, pgpsvc.PAP, "services"},
		{quad9svc.Name, quad9svc.PAP, "services"},
		{threatsvc.Name, threatsvc.PAP, "services"},
		// aggregate group — alphabetical
		{apexsvc.Name, apexsvc.PAP, "aggregate"},
	}
	entries := make([]serviceEntry, len(metas))
	for i, m := range metas {
		entries[i] = serviceEntry{
			Name:  m.name,
			Group: m.group,
			PAP:   m.pap.String(),
		}
	}
	return entries
}

func newServicesCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "services",
		Short:   "List all implemented services and their PAP levels",
		Args:    cobra.NoArgs,
		GroupID: "utility",
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries := allServices()
			w := cmd.OutOrStdout()
			switch output.Format(d.cfg.Output) {
			case output.FormatJSON:
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			case output.FormatTable:
				return writeServicesTable(w, entries)
			default: // text
				return writeServicesText(w, entries)
			}
		},
	}
}

func writeServicesTable(w io.Writer, entries []serviceEntry) error {
	rows := make([][]string, len(entries))
	for i, e := range entries {
		rows[i] = []string{e.Group, e.Name, e.PAP}
	}
	table := output.NewGroupedWrappingTable(w, 20, 30)
	table.Header([]string{"Group", "Command", "PAP"})
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}

func writeServicesText(w io.Writer, entries []serviceEntry) error {
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", e.Group, e.Name, e.PAP); err != nil {
			return err
		}
	}
	return nil
}
