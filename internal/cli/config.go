package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"

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

// readOnlyKeys are the rows of `config show` that report the resolved proxy per
// path. They are not config keys: `config get` reads them, `config set` rejects
// them (change proxy or the proxy environment instead).
var readOnlyKeys = []string{"proxy_dns", "proxy_http", "proxy_https", "proxy_no_proxy"}

// buildConfigRows returns all config key–value pairs and the read-only proxy
// rows, sorted alphabetically.
// Values are sourced from the fully-resolved d.cfg (includes defaults, env vars,
// flag overrides) and d.proxy. This is intentional for show/get — they display
// effective state, not just what is written to the file.
func buildConfigRows(d *deps) []configRow {
	keys := slices.Concat(config.ValidKeys(), readOnlyKeys)
	slices.Sort(keys)
	rows := make([]configRow, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, configRow{key: k, value: effectiveValue(d, k)})
	}
	return rows
}

// effectiveValue returns the current effective value for key from d.cfg and,
// for the proxy rows, from d.proxy. It never returns credentials.
func effectiveValue(d *deps, key string) string {
	switch key {
	case "verbose":
		return fmt.Sprintf("%v", d.cfg.Verbose)
	case "output":
		return d.cfg.Output
	case "proxy", "proxy_dns", "proxy_http", "proxy_https", "proxy_no_proxy":
		return proxyRow(d, key)
	case "user_agent":
		if d.cfg.UserAgent != "" {
			return d.cfg.UserAgent
		}
		return httpclient.DefaultUserAgent
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
	default:
		return ""
	}
}

// proxyRow returns the value of the proxy report row key (proxy, proxy_dns,
// proxy_http, proxy_https or proxy_no_proxy; see proxyconf.Settings.Describe),
// or "" for any other key. Proxy URLs in it are redacted.
func proxyRow(d *deps, key string) string {
	for _, r := range d.proxy.Describe() {
		if r.Key == key {
			return r.Value
		}
	}
	return ""
}

func newConfigShowCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Aliases: []string{"cat"},
		Short:   "Display all effective config settings",
		Long: `Display all effective config settings.

Values reflect the fully resolved configuration — defaults, config file, environment
variables, and flags are all merged before display.

user_agent: shows the configured User-Agent string.
If not explicitly configured, the built-in default is used:
  trident/<version> (+https://github.com/tbckr/trident)

proxy rows: five rows describe the proxy trident actually uses, per path. The four
proxy_* rows are read-only (config set rejects them; change proxy or the environment
instead). No proxy row shows credentials: a proxy URL with a user name or password
appears as scheme://***@host:port.
  proxy           trident's own setting (--proxy, config file or TRIDENT_PROXY);
                  empty when only the environment is set
  proxy_https     the proxy for https requests and where it comes from, or direct
  proxy_http      the same for http requests; "refused (no http proxy; https is
                  proxied)" when only https has a proxy: http:// URLs then fail
                  instead of going direct
  proxy_dns       how DNS lookups are made: through a SOCKS5 proxy (to the nameserver
                  9.9.9.10:53); by the system resolver, not proxied, behind an
                  HTTP(S) proxy; else "system resolver", or "system resolver
                  (NO_PROXY=*)"
  proxy_no_proxy  the NO_PROXY value in effect (empty when proxy is set)
Rule: when proxy is set it applies to every path and the proxy environment
variables, NO_PROXY included, are ignored. Otherwise https and DNS use https_proxy
(HTTPS_PROXY), then all_proxy (ALL_PROXY); http uses http_proxy (HTTP_PROXY), then
all_proxy. A lowercase variable and its uppercase twin must agree, and http_proxy
(HTTP_PROXY) without an https or all proxy is refused (https requests would bypass
it). NO_PROXY exempts hosts from the https and http proxy only: DNS is not exempted,
except by NO_PROXY=*, which turns the proxy off for every path. A value that cannot
be honoured is not hidden: the affected rows read "error: <reason>", and only the
commands that use the network abort with it; config show itself still works.

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
				keys := slices.Concat(config.ValidKeys(), readOnlyKeys)
				slices.Sort(keys)
				return keys, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := config.NormalizeKey(args[0])
			if !slices.Contains(readOnlyKeys, key) {
				if err := config.ValidateKey(key); err != nil {
					return err
				}
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
			if slices.Contains(readOnlyKeys, key) {
				return fmt.Errorf("%w: %q (read-only: it reports the resolved proxy; set proxy instead)",
					config.ErrUnknownKey, key)
			}
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
