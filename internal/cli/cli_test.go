package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
