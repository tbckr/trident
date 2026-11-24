package httpclient_test

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/proxyconf"
)

// settings resolves trident's proxy setting and an environment given as
// name/value pairs, as buildDeps does with os.Getenv. Tests never read the
// process environment: it may export proxy variables of its own.
func settings(proxy string, kv ...string) *proxyconf.Settings {
	if len(kv)%2 != 0 {
		panic("settings: odd number of environment arguments")
	}
	env := make(map[string]string, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		env[kv[i]] = kv[i+1]
	}
	return proxyconf.Resolve(proxy, func(name string) string { return env[name] })
}

func TestNew_NoProxy(t *testing.T) {
	client, err := httpclient.New(settings(""), "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithUserAgent(t *testing.T) {
	client, err := httpclient.New(settings(""), "MyBot/1.0", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithHTTPProxy(t *testing.T) {
	client, err := httpclient.New(settings("http://proxy.example.com:8080"), "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithHTTPSProxy(t *testing.T) {
	client, err := httpclient.New(settings("https://proxy.example.com:8080"), "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithSocks5Proxy(t *testing.T) {
	client, err := httpclient.New(settings("socks5://127.0.0.1:9050"), "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_InvalidProxyScheme(t *testing.T) {
	client, err := httpclient.New(settings("ftp://proxy.example.com:8080"), "", nil, false)
	require.ErrorIs(t, err, apperr.ErrProxyConfig)
	assert.Nil(t, client)
}

func TestNew_DefaultUserAgent(t *testing.T) {
	client, err := httpclient.New(settings(""), "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_WithDebugLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := httpclient.New(settings(""), "", logger, true)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_DebugHookSurvivesTransportError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close()) // nothing listens: connection refused

	var buf bytes.Buffer
	client, err := httpclient.New(settings(""), "", slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), true)
	require.NoError(t, err)
	client.SetCommonRetryCount(0)
	assert.NotPanics(t, func() {
		_, err = client.R().Get("https://" + addr + "/")
	})
	require.Error(t, err)
}

func TestNew_WithNilLoggerDebugFalse(t *testing.T) {
	client, err := httpclient.New(settings(""), "", nil, false)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_TransportsNeverKeepTheDefaultProxyFunc(t *testing.T) {
	// The transports' default proxy function reads the process environment;
	// both transports must use the resolved settings' instead, also when no
	// proxy applies.
	defaults := []uintptr{
		reflect.ValueOf(req.C().GetTransport().Proxy).Pointer(),
		// nosemgrep: trident-no-implicit-proxy-env -- reads the default proxy function only to assert that no transport keeps it
		reflect.ValueOf(http.DefaultTransport.(*http.Transport).Proxy).Pointer(),
	}
	cases := map[string]*proxyconf.Settings{
		"no proxy":   settings(""),
		"NO_PROXY=*": settings("", "HTTPS_PROXY", "http://proxy.corp:3128", "NO_PROXY", "*"),
		"proxy":      settings("socks5h://127.0.0.1:9050"),
	}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) {
			client, err := httpclient.New(s, "", nil, false)
			require.NoError(t, err)
			require.NotNil(t, client.GetTransport().Proxy)
			assert.NotContains(t, defaults, reflect.ValueOf(client.GetTransport().Proxy).Pointer(), "req's transport")

			doh, _, err := httpclient.NewDoH(s, "https://dns.example/dns-query", "", nil, false)
			require.NoError(t, err)
			require.NotNil(t, doh.GetTransport().Proxy)
			assert.NotContains(t, defaults, reflect.ValueOf(doh.GetTransport().Proxy).Pointer(), "the DoH client's req transport")
			tr := dohTransport(t, doh)
			require.NotNil(t, tr.Proxy)
			assert.NotContains(t, defaults, reflect.ValueOf(tr.Proxy).Pointer(), "the DoH client's Go transport")
		})
	}
}

const exemptionMsg = "NO_PROXY exempts this host from the proxy; connecting directly"

func TestExemptionWarner_NilLogger(t *testing.T) {
	assert.Nil(t, httpclient.ExemptionWarner(nil))
}

func TestExemptionWarner_OncePerNormalisedHost(t *testing.T) {
	var buf bytes.Buffer
	warn := httpclient.ExemptionWarner(slog.New(slog.NewTextHandler(&buf, nil)))
	require.NotNil(t, warn)

	for _, host := range []string{"A.CORP.", "a.corp", "a.corp.", "A.Corp", "b.corp", "b.corp"} {
		warn(host)
	}
	out := buf.String()
	assert.Equal(t, 2, strings.Count(out, exemptionMsg), out)
	assert.Equal(t, 2, strings.Count(out, "level=WARN"), out)
	assert.Contains(t, out, "host=a.corp\n", "the host is logged lowercase and without its trailing dot")
	assert.Contains(t, out, "host=b.corp\n")
}

func TestExemptionWarner_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	warn := httpclient.ExemptionWarner(slog.New(slog.NewTextHandler(&buf, nil)))

	var wg sync.WaitGroup
	for i := range 64 {
		wg.Go(func() {
			if i%2 == 0 {
				warn("x.corp")
			} else {
				warn("X.CORP.")
			}
		})
	}
	wg.Wait()
	assert.Equal(t, 1, strings.Count(buf.String(), exemptionMsg), "parallel requests to one host must warn once")
}
