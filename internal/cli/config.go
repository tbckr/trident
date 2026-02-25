package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/tbckr/trident/internal/config"
	providers "github.com/tbckr/trident/internal/detect"
	"github.com/tbckr/trident/internal/httpclient"
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
		return httpclient.ResolveProxy(d.cfg.Proxy)
	case "user_agent":
		return httpclient.ResolveUserAgent(d.cfg.UserAgent, d.cfg.TLSFingerprint)
	case "pap_limit":
		return d.cfg.PAPLimit
	case "defang":
		return fmt.Sprintf("%v", d.cfg.Defang)
	case "no_defang":
		return fmt.Sprintf("%v", d.cfg.NoDefang)
	case "concurrency":
		return fmt.Sprintf("%d", d.cfg.Concurrency)
	case "detect_patterns.url":
		return d.cfg.DetectPatterns.URL
	case "detect_patterns.file":
		return providers.ResolvePatternFile(d.cfg.DetectPatterns.File)
	case "tls_fingerprint":
		return httpclient.ResolveTLSFingerprint(d.cfg.UserAgent, d.cfg.TLSFingerprint)
	default:
		return ""
	}
}

func newConfigShowCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Aliases: []string{"cat"},
		Short:   "Display all effective config settings",
		Long: `Display all effective config settings.

Values reflect the fully resolved configuration — defaults, config file, environment
variables, and flags are all merged before display.

user_agent and tls_fingerprint are bidirectionally linked via browser presets
(chrome, firefox, safari, edge, ios, android):
  --user-agent=chrome      → Chrome TLS fingerprint derived
  --tls-fingerprint=chrome → Chrome TLS fingerprint + matching browser profile
  chrome, firefox, and safari set a full browser profile (TLS, HTTP/2, User-Agent).
  edge, ios, and android set the TLS fingerprint only.
  Explicit custom strings always win; custom values disable derivation.

user_agent: shows the configured User-Agent.
For chrome, firefox, and safari the preset name is shown; the browser profile
manages the actual User-Agent string. For all other cases the effective string is shown.
If not explicitly configured, the built-in default is used:
  trident/<version> (+https://github.com/tbckr/trident)

tls_fingerprint: shows the resolved TLS fingerprint that will actually be used.
If user_agent is a preset name and no explicit fingerprint is set, the matching
fingerprint is derived and displayed here.

proxy: shows the resolved proxy configuration that will actually be used.
If not explicitly configured, standard environment variables are honoured:
  HTTP client  — HTTP_PROXY / HTTPS_PROXY / NO_PROXY
  DNS resolver — ALL_PROXY / all_proxy (SOCKS5 only)
If any of these variables are set, "<from environment>" is displayed.

detect_patterns.file: shows the resolved patterns file that will actually be used.
If not explicitly configured, trident searches in order:

  1. <config-dir>/detect.yaml          (user-maintained override)
  2. <config-dir>/detect-downloaded.yaml  (downloaded via 'trident download detect')
  3. built-in embedded patterns          (displayed as "<embedded>")

Use 'trident config path' to find <config-dir> on this system.`,
		Args: cobra.NoArgs,
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
			key := config.NormalizeKey(args[0])
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
				return config.KeyCompletions(config.NormalizeKey(args[0])), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := config.NormalizeKey(args[0])
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
