package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/tbckr/trident/internal/output"
)

func newAliasCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "alias",
		Short:   "Manage command aliases",
		GroupID: "utility",
	}
	cmd.AddCommand(
		newAliasSetCmd(d),
		newAliasListCmd(d),
		newAliasDeleteCmd(d),
	)
	return cmd
}

func newAliasSetCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "set <name> <expansion>",
		Short: "Create or update an alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			expansion := args[1]

			if err := validateAliasName(name); err != nil {
				return err
			}

			// Reject names that shadow built-in (non-alias) commands.
			for _, c := range cmd.Root().Commands() {
				if c.GroupID != "aliases" && c.Name() == name {
					return fmt.Errorf("alias %q shadows a built-in command; choose a different name", name)
				}
			}

			raw := map[string]any{}
			data, err := os.ReadFile(d.cfg.ConfigFile)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("reading config file: %w", err)
			}
			if len(data) > 0 {
				if err := yaml.Unmarshal(data, &raw); err != nil {
					return fmt.Errorf("parsing config file: %w", err)
				}
			}

			aliasMap, _ := raw["alias"].(map[string]any)
			if aliasMap == nil {
				aliasMap = map[string]any{}
			}
			aliasMap[name] = expansion
			raw["alias"] = aliasMap

			out, err := yaml.Marshal(raw)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}
			if err := os.WriteFile(d.cfg.ConfigFile, out, 0o600); err != nil {
				return fmt.Errorf("writing config file: %w", err)
			}
			return nil
		},
	}
}

func newAliasListCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all aliases",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(d.cfg.Aliases) == 0 {
				return nil
			}

			// Sort for stable output.
			names := make([]string, 0, len(d.cfg.Aliases))
			for k := range d.cfg.Aliases {
				names = append(names, k)
			}
			sort.Strings(names)

			w := cmd.OutOrStdout()
			switch output.Format(d.cfg.Output) {
			case output.FormatJSON:
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(d.cfg.Aliases)
			case output.FormatText:
				for _, name := range names {
					if _, err := fmt.Fprintf(w, "%s=%s\n", name, d.cfg.Aliases[name]); err != nil {
						return err
					}
				}
				return nil
			default: // table
				table := output.NewWrappingTable(w, 20, 6)
				table.Header([]string{"ALIAS", "EXPANSION"})
				rows := make([][]string, 0, len(names))
				for _, name := range names {
					rows = append(rows, []string{name, d.cfg.Aliases[name]})
				}
				if err := table.Bulk(rows); err != nil {
					return err
				}
				return table.Render()
			}
		},
	}
}

func newAliasDeleteCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete an alias",
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				names := make([]string, 0, len(d.cfg.Aliases))
				for k := range d.cfg.Aliases {
					names = append(names, k)
				}
				return names, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			if _, ok := d.cfg.Aliases[name]; !ok {
				return fmt.Errorf("alias %q not found", name)
			}

			raw := map[string]any{}
			data, err := os.ReadFile(d.cfg.ConfigFile)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("reading config file: %w", err)
			}
			if len(data) > 0 {
				if err := yaml.Unmarshal(data, &raw); err != nil {
					return fmt.Errorf("parsing config file: %w", err)
				}
			}

			aliasMap, _ := raw["alias"].(map[string]any)
			if aliasMap != nil {
				delete(aliasMap, name)
				if len(aliasMap) == 0 {
					delete(raw, "alias")
				} else {
					raw["alias"] = aliasMap
				}
			}

			out, err := yaml.Marshal(raw)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}
			if err := os.WriteFile(d.cfg.ConfigFile, out, 0o600); err != nil {
				return fmt.Errorf("writing config file: %w", err)
			}
			return nil
		},
	}
}

// validateAliasName rejects names that start with '-' or contain whitespace.
func validateAliasName(name string) error {
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("alias name %q must not start with '-'", name)
	}
	for _, r := range name {
		if unicode.IsSpace(r) {
			return fmt.Errorf("alias name %q must not contain whitespace", name)
		}
	}
	return nil
}
