package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/services"
)

// clearedVars are the variables runTrident empties before every run, so that
// neither the developer's nor the CI runner's proxy settings reach a test.
// proxyconf and viper treat an empty variable as unset.
var clearedVars = []string{
	"HTTPS_PROXY", "https_proxy",
	"HTTP_PROXY", "http_proxy",
	"ALL_PROXY", "all_proxy",
	"NO_PROXY", "no_proxy",
	"REQUEST_METHOD",
	"TRIDENT_PROXY",
}

// runTrident runs the trident command line args in an isolated environment:
// the proxy variables cleared, then env applied, HOME and XDG_CONFIG_HOME in
// temporary directories, and an empty config file passed as --config ahead of
// args (a later --config in args wins). stdout and stderr hold everything the
// command wrote, the deps logger included. Tests using it must not call
// t.Parallel (t.Setenv), and their commands must not reach the network: pass
// a cancelled context, a PAP limit that blocks the service, or loopback-only
// proxies.
func runTrident(t *testing.T, ctx context.Context, env map[string]string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	for _, name := range clearedVars {
		t.Setenv(name, "")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for name, value := range env {
		t.Setenv(name, value)
	}
	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, nil, 0o600))

	var out, errOut bytes.Buffer
	cmd := newRootCmd(nil)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(append([]string{"--config", cfgFile}, args...))
	err = cmd.ExecuteContext(ctx)
	return out.String(), errOut.String(), err
}

// cancelledContext returns a context that is already done, so a command stops
// before it sends anything.
func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestApex_ProxiedDoHSkipsSystemResolver(t *testing.T) {
	_, stderr, _ := runTrident(t, cancelledContext(), nil, "--proxy", "http://127.0.0.1:1", "apex", "example.com")
	assert.NotContains(t, stderr, "DNS lookups cannot go through")
}

func TestApex_UnproxiedDoHFallsBackToSystemResolverAndWarns(t *testing.T) {
	// NO_PROXY exempts the DoH endpoint, so apex's ASN lookups use the system
	// resolver while an HTTP proxy is configured: the warn case. The PAP limit
	// stops apex before it sends anything.
	env := map[string]string{"HTTPS_PROXY": "http://127.0.0.1:1", "NO_PROXY": "dns.quad9.net"}
	_, stderr, err := runTrident(t, cancelledContext(), env, "--pap-limit", "red", "apex", "example.com")
	require.ErrorIs(t, err, services.ErrPAPBlocked)
	assert.Contains(t, stderr, "DNS lookups cannot go through")
}

func TestDNSWarning_OnlyInDNSCommands(t *testing.T) {
	// The PAP limit blocks every one of these services after the command has
	// built its clients, so nothing is sent.
	tests := []struct {
		command string
		warns   bool
	}{
		{"dns", true},
		{"detect", true},
		{"cymru", true},
		{"crtsh", false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			_, stderr, err := runTrident(t, cancelledContext(), nil,
				"--proxy", "http://127.0.0.1:1", "--pap-limit", "red", tt.command, "example.com")
			require.ErrorIs(t, err, services.ErrPAPBlocked)
			if tt.warns {
				assert.Contains(t, stderr, "DNS lookups cannot go through")
				assert.Equal(t, 1, strings.Count(stderr, "DNS lookups cannot go through"), "warned once")
			} else {
				assert.NotContains(t, stderr, "DNS lookups cannot go through")
			}
		})
	}
}

func TestConfigShow_ProxyRows(t *testing.T) {
	env := map[string]string{
		"HTTPS_PROXY": "http://alice:S3cr3t@corp:3128",
		"HTTP_PROXY":  "http://alice:S3cr3t@corp:3128",
		"NO_PROXY":    ".corp",
	}
	stdout, _, err := runTrident(t, context.Background(), env, "--output", "text", "config", "show")
	require.NoError(t, err)
	assert.Contains(t, stdout, "proxy_https=http://***@corp:3128 (HTTPS_PROXY)\n")
	assert.Contains(t, stdout, "proxy_http=http://***@corp:3128 (HTTP_PROXY)\n")
	assert.Contains(t, stdout, "proxy_dns=system resolver, not proxied (HTTPS_PROXY is not a SOCKS5 proxy)\n")
	assert.Contains(t, stdout, "proxy_no_proxy=.corp\n")
	assert.Contains(t, stdout, "proxy=\n", "trident's own setting is empty when only the environment is set")
	assert.NotContains(t, stdout, "S3cr3t")
	assert.NotContains(t, stdout, "alice")
}

func TestConfigShow_ProxyRowsInJSON(t *testing.T) {
	stdout, _, err := runTrident(t, context.Background(), nil,
		"--proxy", "socks5h://alice:S3cr3t@127.0.0.1:9050", "--output", "json", "config", "show")
	require.NoError(t, err)
	var rows map[string]string
	require.NoError(t, json.Unmarshal([]byte(stdout), &rows))
	assert.Equal(t, "socks5h://***@127.0.0.1:9050", rows["proxy"])
	assert.Equal(t, "socks5h://***@127.0.0.1:9050 (proxy), nameserver 9.9.9.10:53", rows["proxy_dns"])
	assert.Equal(t, "socks5h://***@127.0.0.1:9050 (proxy)", rows["proxy_http"])
	assert.Equal(t, "socks5h://***@127.0.0.1:9050 (proxy)", rows["proxy_https"])
	assert.Equal(t, "", rows["proxy_no_proxy"])
}

func TestConfigShow_ProxyRowsAreSorted(t *testing.T) {
	stdout, _, err := runTrident(t, context.Background(), nil, "--output", "text", "config", "show")
	require.NoError(t, err)
	var keys []string
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		key, _, _ := strings.Cut(line, "=")
		keys = append(keys, key)
	}
	assert.True(t, slices.IsSorted(keys), "keys not sorted: %v", keys)
	assert.Subset(t, keys, []string{"proxy", "proxy_dns", "proxy_http", "proxy_https", "proxy_no_proxy"})
}

func TestConfigGet_ReadOnlyProxyKey(t *testing.T) {
	stdout, _, err := runTrident(t, context.Background(), nil,
		"--proxy", "socks5h://127.0.0.1:9050", "config", "get", "proxy_dns")
	require.NoError(t, err)
	assert.Equal(t, "socks5h://127.0.0.1:9050 (proxy), nameserver 9.9.9.10:53\n", stdout)

	// The hyphenated spelling works like it does for the other keys.
	stdout, _, err = runTrident(t, context.Background(), nil,
		"--proxy", "socks5h://127.0.0.1:9050", "config", "get", "proxy-https")
	require.NoError(t, err)
	assert.Equal(t, "socks5h://127.0.0.1:9050 (proxy)\n", stdout)
}

// proxyKeys are the five rows that report the proxy.
var proxyKeys = []string{"proxy", "proxy_dns", "proxy_http", "proxy_https", "proxy_no_proxy"}

func TestConfig_NeverPrintsCredentials(t *testing.T) {
	cfgWithProxy := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgWithProxy, []byte("proxy: socks5://alice:S3cr3t@corp:1080\n"), 0o600))

	tests := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{name: "flag", args: []string{"--proxy", "http://alice:S3cr3t@corp:3128"}},
		{name: "flag, token-only userinfo", args: []string{"--proxy", "http://T0kenOnly@corp:3128"}},
		{name: "config file", args: []string{"--config", cfgWithProxy}},
		{name: "TRIDENT_PROXY", env: map[string]string{"TRIDENT_PROXY": "socks5h://alice:S3cr3t@corp:1080"}},
		{name: "HTTPS_PROXY", env: map[string]string{"HTTPS_PROXY": "http://alice:S3cr3t@corp:3128"}},
		{name: "https_proxy", env: map[string]string{"https_proxy": "http://alice:S3cr3t@corp:3128"}},
		{name: "HTTP_PROXY only", env: map[string]string{"HTTP_PROXY": "http://alice:S3cr3t@corp:3128"}},
		{name: "ALL_PROXY", env: map[string]string{"ALL_PROXY": "socks5://alice:S3cr3t@corp:1080"}},
		{name: "invalid scheme in flag", args: []string{"--proxy", "socks4://alice:S3cr3t@corp:1"}},
		{name: "invalid port in flag", args: []string{"--proxy", "http://alice:S3cr3t@corp:abc"}},
		{name: "invalid scheme in TRIDENT_PROXY", env: map[string]string{"TRIDENT_PROXY": "socks4://alice:S3cr3t@corp:1"}},
		{name: "invalid HTTPS_PROXY", env: map[string]string{"HTTPS_PROXY": "ftp://alice:S3cr3t@corp:3128"}},
		{name: "invalid ALL_PROXY", env: map[string]string{"ALL_PROXY": "http://alice:S3cr3t@corp:abc"}},
		{name: "disagreeing cases", env: map[string]string{
			"HTTPS_PROXY": "http://alice:S3cr3t@corp:3128", "https_proxy": "http://bob:Pa55w0rd@corp:3129",
		}},
		{name: "HTTP_PROXY alone with credentials", env: map[string]string{"HTTP_PROXY": "ftp://alice:S3cr3t@corp:1"}},
	}
	secrets := []string{"alice", "S3cr3t", "T0kenOnly", "bob", "Pa55w0rd"}
	assertClean := func(t *testing.T, stdout, stderr string, err error) {
		t.Helper()
		all := stdout + stderr
		if err != nil {
			all += err.Error()
		}
		for _, secret := range secrets {
			assert.NotContains(t, all, secret)
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, format := range []string{"text", "json"} {
				stdout, stderr, err := runTrident(t, context.Background(), tt.env,
					append([]string{"--output", format, "config", "show"}, tt.args...)...)
				require.NoError(t, err, "config show never fails because of a proxy problem")
				assertClean(t, stdout, stderr, err)
			}
			for _, key := range proxyKeys {
				stdout, stderr, err := runTrident(t, context.Background(), tt.env,
					append([]string{"config", "get", key}, tt.args...)...)
				require.NoError(t, err)
				assertClean(t, stdout, stderr, err)
			}
		})
	}
}

func TestConfigSet_RejectsReadOnlyKeysAndInvalidProxy(t *testing.T) {
	original := []byte("concurrency: 5\n")
	tests := []struct {
		name    string
		args    []string
		wantErr error
		leak    string
	}{
		{name: "read-only proxy_https", args: []string{"proxy_https", "x"}, wantErr: config.ErrUnknownKey},
		{name: "read-only proxy_dns", args: []string{"proxy_dns", "socks5h://h:1"}, wantErr: config.ErrUnknownKey},
		{name: "read-only proxy_http", args: []string{"proxy_http", "http://h:1"}, wantErr: config.ErrUnknownKey},
		{name: "read-only hyphenated", args: []string{"proxy-no-proxy", ".corp"}, wantErr: config.ErrUnknownKey},
		{name: "unsupported scheme", args: []string{"proxy", "socks4://h:1"}, wantErr: apperr.ErrProxyConfig},
		{name: "credentials and bad port", args: []string{"proxy", "socks5://alice:S3cr3t@h:abc"}, wantErr: apperr.ErrProxyConfig, leak: "S3cr3t"},
		{name: "credentials and bad scheme", args: []string{"proxy", "ftp://alice:S3cr3t@h:1"}, wantErr: apperr.ErrProxyConfig, leak: "alice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgFile := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(cfgFile, original, 0o600))

			stdout, stderr, err := runTrident(t, context.Background(), nil,
				append([]string{"--config", cfgFile, "config", "set"}, tt.args...)...)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, stdout)
			if tt.leak != "" {
				assert.NotContains(t, err.Error(), tt.leak)
				assert.NotContains(t, stderr, tt.leak)
			}
			if tt.wantErr == apperr.ErrProxyConfig {
				assert.NotContains(t, err.Error(), tt.args[1], "the error must not echo the value")
			}
			after, readErr := os.ReadFile(cfgFile)
			require.NoError(t, readErr)
			assert.Equal(t, original, after, "a rejected set leaves the file untouched")
		})
	}
}

func TestConfigSet_AcceptsValidProxy(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, nil, 0o600))
	_, _, err := runTrident(t, context.Background(), nil,
		"--config", cfgFile, "config", "set", "proxy", "SOCKS5H://127.0.0.1:9050")
	require.NoError(t, err)
	data, err := os.ReadFile(cfgFile)
	require.NoError(t, err)
	assert.Equal(t, "proxy: SOCKS5H://127.0.0.1:9050\n", string(data))

	// An empty value unsets the proxy and is not validated.
	_, _, err = runTrident(t, context.Background(), nil, "--config", cfgFile, "config", "set", "proxy", "")
	require.NoError(t, err)
}

func TestBrokenProxyOnlyBreaksNetworkCommands(t *testing.T) {
	env := map[string]string{"HTTPS_PROXY": "socks4://127.0.0.1:1080"}

	t.Run("commands without network keep working", func(t *testing.T) {
		_, _, err := runTrident(t, context.Background(), env,
			"identify", "--domain", "example.com", "--ns", "ns1.cloudflare.com")
		require.NoError(t, err)

		// The text format: the table wraps long cells.
		stdout, _, err := runTrident(t, context.Background(), env, "--output", "text", "config", "show")
		require.NoError(t, err)
		assert.Contains(t, stdout, "proxy_https=error: invalid proxy configuration: HTTPS_PROXY: socks4:// is not supported")

		_, _, err = runTrident(t, context.Background(), env, "version")
		require.NoError(t, err)
	})

	// These commands must fail while they build their client or resolver,
	// before any request. The cancelled context and --pap-limit red are
	// backstops: should construction ever succeed, the PAP check (which runs
	// after it) blocks the command before it can reach the network.
	for _, command := range []string{"crtsh", "quad9", "dns"} {
		t.Run(command+" fails", func(t *testing.T) {
			_, _, err := runTrident(t, cancelledContext(), env, "--pap-limit", "red", command, "example.com")
			require.ErrorIs(t, err, apperr.ErrProxyConfig)
		})
	}
}

func TestAliases_MaskProxyCredentials(t *testing.T) {
	const secret = "S3cr3t"
	aliases := map[string]string{
		"tor":  "crtsh --proxy socks5h://alice:S3cr3t@127.0.0.1:9050",
		"bare": "crtsh --proxy alice:S3cr3t@127.0.0.1:9050",
	}

	t.Run("help", func(t *testing.T) {
		for _, args := range [][]string{{"--help"}, {"tor", "--help"}, {"bare", "--help"}} {
			var out bytes.Buffer
			cmd := newRootCmd(aliases)
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(args)
			require.NoError(t, cmd.ExecuteContext(context.Background()))

			help := out.String()
			assert.NotContains(t, help, secret, "args %v", args)
			if args[0] == "--help" || args[0] == "tor" {
				assert.Contains(t, help, "socks5h://***@127.0.0.1:9050", "args %v", args)
			}
			if args[0] == "--help" || args[0] == "bare" {
				assert.Contains(t, help, "--proxy ***@127.0.0.1:9050", "args %v", args)
			}
		}
	})

	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	cfg := "alias:\n"
	for name, expansion := range aliases {
		cfg += "  " + name + ": " + expansion + "\n"
	}
	require.NoError(t, os.WriteFile(cfgFile, []byte(cfg), 0o600))

	t.Run("alias list", func(t *testing.T) {
		for _, format := range []string{"table", "text", "json"} {
			stdout, stderr, err := runTrident(t, context.Background(), nil,
				"--config", cfgFile, "--output", format, "alias", "list")
			require.NoError(t, err, format)
			assert.NotContains(t, stdout, secret, format)
			assert.NotContains(t, stderr, secret, format)
			assert.Contains(t, stdout, "socks5h://***@127.0.0.1:9050", format)
			assert.Contains(t, stdout, "--proxy ***@127.0.0.1:9050", format)
		}

		// The JSON stays a name-to-expansion object.
		stdout, _, err := runTrident(t, context.Background(), nil,
			"--config", cfgFile, "--output", "json", "alias", "list")
		require.NoError(t, err)
		var got map[string]string
		require.NoError(t, json.Unmarshal([]byte(stdout), &got))
		assert.Equal(t, map[string]string{
			"tor":  "crtsh --proxy socks5h://***@127.0.0.1:9050",
			"bare": "crtsh --proxy ***@127.0.0.1:9050",
		}, got)
	})

	// The masking is display-only: running the alias must hand the real proxy
	// URL to the command. The PAP limit stops crtsh before it sends anything,
	// and the proxy flag then still holds the alias's value.
	t.Run("alias still expands to the real value", func(t *testing.T) {
		for _, name := range clearedVars {
			t.Setenv(name, "")
		}
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		for name, want := range map[string]string{
			"tor":  "socks5h://alice:S3cr3t@127.0.0.1:9050",
			"bare": "alice:S3cr3t@127.0.0.1:9050",
		} {
			var out bytes.Buffer
			cmd := newRootCmd(aliases)
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs([]string{name, "--config", cfgFile, "--pap-limit", "red", "example.com"})
			err := cmd.ExecuteContext(context.Background())

			require.ErrorIs(t, err, services.ErrPAPBlocked, name)
			assert.Equal(t, want, cmd.PersistentFlags().Lookup("proxy").Value.String(), name)
			assert.NotContains(t, out.String(), secret, name)
		}
	})
}
