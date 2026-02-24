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

// namer is the minimal interface needed to list a service.
// It is satisfied by both services.Service implementors and identify.Service
// (which has a custom Run signature).
type namer interface {
	Name() string
	PAP() pap.Level
}

type serviceEntry struct {
	Name  string `json:"name"`
	Group string `json:"group"`
	PAP   string `json:"pap"`
}

// allServices returns a fixed-order list of every service and aggregate command.
// Services are ordered alphabetically within each group; "services" precedes "aggregate".
// Constructors receive nil for clients/resolvers: Name() and PAP() are pure
// receivers that never dereference those fields.
func allServices() []serviceEntry {
	type item struct {
		svc   namer
		group string
	}
	items := []item{
		// services group — alphabetical
		{cymrusvc.NewService(nil, nil), "services"},
		{crtshsvc.NewService(nil, nil), "services"},
		{detectsvc.NewService(nil, nil), "services"},
		{dnssvc.NewService(nil, nil), "services"},
		{identifysvc.NewService(nil), "services"},
		{pgpsvc.NewService(nil, nil), "services"},
		{quad9svc.NewService(nil, nil), "services"},
		{threatsvc.NewService(nil, nil), "services"},
		// aggregate group — alphabetical
		{apexsvc.NewService(nil, nil, nil), "aggregate"},
	}
	entries := make([]serviceEntry, len(items))
	for i, it := range items {
		entries[i] = serviceEntry{
			Name:  it.svc.Name(),
			Group: it.group,
			PAP:   it.svc.PAP().String(),
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
