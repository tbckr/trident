package cli

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/proxyconf"
)

// dnsLeakWarning is the start of the warning newResolver logs when DNS cannot
// use the configured proxy.
const dnsLeakWarning = "DNS lookups cannot go through the configured proxy"

// newBufferedDeps returns deps for the proxy setting p in an empty environment,
// and the buffer its logger writes to.
func newBufferedDeps(p string) (*deps, *bytes.Buffer) {
	var buf bytes.Buffer
	return &deps{
		logger: slog.New(slog.NewTextHandler(&buf, nil)),
		cfg:    &config.Config{},
		proxy:  proxyconf.Resolve(p, nil),
	}, &buf
}

func TestNewResolver_WarnsOnlyWhenDNSIsUnproxied(t *testing.T) {
	d, buf := newBufferedDeps("http://127.0.0.1:3128")
	_, err := d.newResolver()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), dnsLeakWarning)
	assert.Contains(t, buf.String(), "proxy=http://127.0.0.1:3128")

	for _, p := range []string{"socks5h://127.0.0.1:9050", ""} {
		d, buf := newBufferedDeps(p)
		_, err := d.newResolver()
		require.NoError(t, err, "proxy=%q", p)
		assert.NotContains(t, buf.String(), dnsLeakWarning, "proxy=%q", p)
	}
}

func TestNewResolver_WarningHidesCredentials(t *testing.T) {
	d, buf := newBufferedDeps("http://alice:S3cr3t@127.0.0.1:3128")
	_, err := d.newResolver()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "proxy=http://***@127.0.0.1:3128")
	assert.NotContains(t, buf.String(), "S3cr3t")
	assert.NotContains(t, buf.String(), "alice")
}

func TestNewHTTPClient_DoesNotWarnAboutDNS(t *testing.T) {
	d, buf := newBufferedDeps("http://127.0.0.1:3128")
	_, err := d.newHTTPClient()
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), dnsLeakWarning)
}
