package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/output"
)

func newConfigCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Read and write trident config file values",
		GroupID: "utility",
	}
	cmd.AddCommand(
		newConfigPathCmd(d),
		newConfigShowCmd(d),
		newConfigGetCmd(d),
		newConfigSetCmd(d),
		newConfigEditCmd(d),
	)
	return cmd
}

func newConfigPathCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), d.cfg.ConfigFile)
			return err
		},
	}
}

// configRow holds one key–value pair for display.
type configRow struct {
	key   string
	value string
}

// buildConfigRows returns all config key–value pairs sorted alphabetically.
// Values are sourced from the fully-resolved d.cfg (includes defaults, env vars,
// flag overrides). This is intentional for show/get — they display effective state,
// not just what is written to the file.
func buildConfigRows(d *deps) []configRow {
	keys := config.ValidKeys()
	sort.Strings(keys)

	rows := make([]configRow, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, configRow{key: k, value: effectiveValue(d, k)})
	}
	return rows
}

// effectiveValue returns the current effective value for key from d.cfg.
func effectiveValue(d *deps, key string) string {
	switch key {
	case "verbose":
		return fmt.Sprintf("%v", d.cfg.Verbose)
	case "output":
		return d.cfg.Output
	case "proxy":
		return d.cfg.Proxy
	case "user_agent":
		return d.cfg.UserAgent
	case "pap_limit":
		return d.cfg.PAPLimit
	case "defang":
		return fmt.Sprintf("%v", d.cfg.Defang)
	case "no_defang":
		return fmt.Sprintf("%v", d.cfg.NoDefang)
	case "concurrency":
		return fmt.Sprintf("%d", d.cfg.Concurrency)
	default:
		return ""
	}
}

func newConfigShowCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Aliases: []string{"cat"},
		Short:   "Display all effective config settings",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			rows := buildConfigRows(d)
			switch output.Format(d.cfg.Output) {
			case output.FormatJSON:
				m := make(map[string]string, len(rows))
				for _, r := range rows {
					m[r.key] = r.value
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(m)
			case output.FormatText:
				for _, r := range rows {
					if _, err := fmt.Fprintf(w, "%s=%s\n", r.key, r.value); err != nil {
						return err
					}
				}
				return nil
			default: // text
				table := output.NewWrappingTable(w, 20, 6)
				table.Header([]string{"KEY", "VALUE"})
				tableRows := make([][]string, len(rows))
				for i, r := range rows {
					tableRows[i] = []string{r.key, r.value}
				}
				if err := table.Bulk(tableRows); err != nil {
					return err
				}
				return table.Render()
			}
		},
	}
}

func newConfigGetCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print the value of a config key",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return config.ValidKeys(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeConfigKey(args[0])
			if err := config.ValidateKey(key); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), effectiveValue(d, key))
			return err
		},
	}
	return cmd
}

func newConfigSetCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value and persist it to the config file",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0:
				return config.ValidKeys(), cobra.ShellCompDirectiveNoFileComp
			case 1:
				return config.KeyCompletions(normalizeConfigKey(args[0])), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeConfigKey(args[0])
			if err := config.ValidateKey(key); err != nil {
				return err
			}
			typedValue, err := config.ParseValue(key, args[1])
			if err != nil {
				return err
			}

			// Read only what is already explicitly in the file — never from d.cfg
			// (which is fully populated with defaults from flags/env vars/code).
			// This ensures a fresh-file set writes only the one requested key.
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

			// Set ONLY the single specified key; leave every other key untouched.
			raw[key] = typedValue

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
	return cmd
}

func newConfigEditCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the config file in $EDITOR",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vi"
			}
			c := exec.CommandContext(cmd.Context(), editor, d.cfg.ConfigFile) //nolint:gosec // editor is sourced from user's $EDITOR/$VISUAL env var
			c.Stdin = cmd.InOrStdin()
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			return c.Run()
		},
	}
}

// normalizeConfigKey converts hyphenated flag names to their viper key equivalents
// (e.g. "pap-limit" → "pap_limit").
func normalizeConfigKey(key string) string {
	result := make([]byte, len(key))
	for i := range key {
		if key[i] == '-' {
			result[i] = '_'
		} else {
			result[i] = key[i]
		}
	}
	return string(result)
}
