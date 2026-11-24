package proxyconf_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/proxyconf"
)

func rows(s *proxyconf.Settings) map[string]string {
	m := map[string]string{}
	for _, r := range s.Describe() {
		m[r.Key] = r.Value
	}
	return m
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		env     []string
		want    []proxyconf.Row
	}{
		{
			name:    "setting with credentials",
			setting: "socks5h://alice:pw@127.0.0.1:9050",
			want: []proxyconf.Row{
				{Key: "proxy", Value: "socks5h://***@127.0.0.1:9050"},
				{Key: "proxy_dns", Value: "socks5h://***@127.0.0.1:9050 (proxy), nameserver 9.9.9.10:53"},
				{Key: "proxy_http", Value: "socks5h://***@127.0.0.1:9050 (proxy)"},
				{Key: "proxy_https", Value: "socks5h://***@127.0.0.1:9050 (proxy)"},
				{Key: "proxy_no_proxy", Value: ""},
			},
		},
		{
			name:    "setting ignores the environment and its NO_PROXY",
			setting: "http://corp:3128",
			env:     []string{"HTTPS_PROXY", "socks5h://tor:9050", "NO_PROXY", ".corp"},
			want: []proxyconf.Row{
				{Key: "proxy", Value: "http://corp:3128"},
				{Key: "proxy_dns", Value: "system resolver, not proxied (proxy is not a SOCKS5 proxy)"},
				{Key: "proxy_http", Value: "http://corp:3128 (proxy)"},
				{Key: "proxy_https", Value: "http://corp:3128 (proxy)"},
				{Key: "proxy_no_proxy", Value: ""},
			},
		},
		{
			name: "http proxy from the environment",
			env:  []string{"HTTPS_PROXY", "http://corp:3128", "HTTP_PROXY", "http://corp:3128", "NO_PROXY", ".corp,localhost"},
			want: []proxyconf.Row{
				{Key: "proxy", Value: ""},
				{Key: "proxy_dns", Value: "system resolver, not proxied (HTTPS_PROXY is not a SOCKS5 proxy)"},
				{Key: "proxy_http", Value: "http://corp:3128 (HTTP_PROXY)"},
				{Key: "proxy_https", Value: "http://corp:3128 (HTTPS_PROXY)"},
				{Key: "proxy_no_proxy", Value: ".corp,localhost"},
			},
		},
		{
			name: "socks proxy from ALL_PROXY",
			env:  []string{"ALL_PROXY", "socks5h://tor:9050"},
			want: []proxyconf.Row{
				{Key: "proxy", Value: ""},
				{Key: "proxy_dns", Value: "socks5h://tor:9050 (ALL_PROXY), nameserver 9.9.9.10:53"},
				{Key: "proxy_http", Value: "socks5h://tor:9050 (ALL_PROXY)"},
				{Key: "proxy_https", Value: "socks5h://tor:9050 (ALL_PROXY)"},
				{Key: "proxy_no_proxy", Value: ""},
			},
		},
		{
			name: "https and http from different variables",
			env:  []string{"https_proxy", "http://corp:3128", "ALL_PROXY", "socks5h://tor:9050"},
			want: []proxyconf.Row{
				{Key: "proxy", Value: ""},
				{Key: "proxy_dns", Value: "system resolver, not proxied (https_proxy is not a SOCKS5 proxy)"},
				{Key: "proxy_http", Value: "socks5h://tor:9050 (ALL_PROXY)"},
				{Key: "proxy_https", Value: "http://corp:3128 (https_proxy)"},
				{Key: "proxy_no_proxy", Value: ""},
			},
		},
		{
			name: "both cases set and equal name the lowercase variable",
			env:  []string{"https_proxy", "http://corp:3128", "HTTPS_PROXY", "http://corp:3128", "http_proxy", "http://corp:3128", "HTTP_PROXY", "http://corp:3128"},
			want: []proxyconf.Row{
				{Key: "proxy", Value: ""},
				{Key: "proxy_dns", Value: "system resolver, not proxied (https_proxy is not a SOCKS5 proxy)"},
				{Key: "proxy_http", Value: "http://corp:3128 (http_proxy)"},
				{Key: "proxy_https", Value: "http://corp:3128 (https_proxy)"},
				{Key: "proxy_no_proxy", Value: ""},
			},
		},
		{
			name: "nothing set",
			want: []proxyconf.Row{
				{Key: "proxy", Value: ""},
				{Key: "proxy_dns", Value: "system resolver"},
				{Key: "proxy_http", Value: "direct"},
				{Key: "proxy_https", Value: "direct"},
				{Key: "proxy_no_proxy", Value: ""},
			},
		},
		{
			name: "NO_PROXY=* with proxy variables set",
			env:  []string{"HTTPS_PROXY", "garbage value", "NO_PROXY", " * "},
			want: []proxyconf.Row{
				{Key: "proxy", Value: ""},
				{Key: "proxy_dns", Value: "system resolver (NO_PROXY=*)"},
				{Key: "proxy_http", Value: "direct (NO_PROXY=*)"},
				{Key: "proxy_https", Value: "direct (NO_PROXY=*)"},
				{Key: "proxy_no_proxy", Value: "*"},
			},
		},
		{
			name:    "invalid setting",
			setting: "http://h:99999",
			want: []proxyconf.Row{
				{Key: "proxy", Value: "error: invalid proxy configuration: proxy: port must be a number from 1 to 65535"},
				{Key: "proxy_dns", Value: "error: invalid proxy configuration: proxy: port must be a number from 1 to 65535"},
				{Key: "proxy_http", Value: "error: invalid proxy configuration: proxy: port must be a number from 1 to 65535"},
				{Key: "proxy_https", Value: "error: invalid proxy configuration: proxy: port must be a number from 1 to 65535"},
				{Key: "proxy_no_proxy", Value: ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := proxyconf.Resolve(tt.setting, env(tt.env...))
			assert.Equal(t, tt.want, s.Describe())
		})
	}
}

func TestDescribe_ErrorsInTheEnvironment(t *testing.T) {
	t.Run("http only", func(t *testing.T) {
		s := proxyconf.Resolve("", env("http_proxy", "http://127.0.0.1:8118"))
		require.Error(t, s.Err())
		want := "error: " + s.Err().Error()
		got := rows(s)
		assert.Equal(t, want, got["proxy_dns"])
		assert.Equal(t, want, got["proxy_http"])
		assert.Equal(t, want, got["proxy_https"])
		// trident's own setting is fine, and so is NO_PROXY.
		assert.Equal(t, "", got["proxy"])
		assert.Equal(t, "", got["proxy_no_proxy"])
	})

	t.Run("broken https value", func(t *testing.T) {
		s := proxyconf.Resolve("", env("HTTPS_PROXY", " socks5://x:1", "NO_PROXY", ".corp"))
		require.Error(t, s.Err())
		got := rows(s)
		assert.Equal(t, "error: invalid proxy configuration: HTTPS_PROXY: contains whitespace or control characters", got["proxy_https"])
		assert.Equal(t, got["proxy_https"], got["proxy_http"])
		assert.Equal(t, got["proxy_https"], got["proxy_dns"])
		assert.Equal(t, "", got["proxy"])
		assert.Equal(t, ".corp", got["proxy_no_proxy"])
	})

	t.Run("NO_PROXY pair conflict", func(t *testing.T) {
		s := proxyconf.Resolve("", env("HTTPS_PROXY", "http://corp:3128", "no_proxy", "a", "NO_PROXY", "b"))
		require.Error(t, s.Err())
		got := rows(s)
		assert.Equal(t, "error: invalid proxy configuration: no_proxy and NO_PROXY are both set and differ; unset one", got["proxy_no_proxy"])
		assert.Equal(t, "error: "+s.Err().Error(), got["proxy_https"])
	})

	t.Run("star among NO_PROXY entries", func(t *testing.T) {
		s := proxyconf.Resolve("", env("HTTPS_PROXY", "http://corp:3128", "NO_PROXY", "localhost,*"))
		require.Error(t, s.Err())
		got := rows(s)
		assert.Contains(t, got["proxy_no_proxy"], "error: invalid proxy configuration: NO_PROXY: ")
		assert.Equal(t, "error: "+s.Err().Error(), got["proxy_dns"])
	})

	t.Run("a bad value does not hide a fine NO_PROXY", func(t *testing.T) {
		s := proxyconf.Resolve("", env("https_proxy", "socks5h://a:1", "HTTPS_PROXY", "socks5h://b:1", "NO_PROXY", ".corp"))
		require.Error(t, s.Err())
		assert.Equal(t, ".corp", rows(s)["proxy_no_proxy"])
	})
}

// Without a selected proxy NO_PROXY is not looked at, so an odd or conflicting
// one is no error and shows without one.
func TestDescribe_NoProxyWithoutASelectedProxy(t *testing.T) {
	odd := proxyconf.Resolve("", env("NO_PROXY", "localhost,*"))
	require.NoError(t, odd.Err())
	assert.Equal(t, []proxyconf.Row{
		{Key: "proxy", Value: ""},
		{Key: "proxy_dns", Value: "system resolver"},
		{Key: "proxy_http", Value: "direct"},
		{Key: "proxy_https", Value: "direct"},
		{Key: "proxy_no_proxy", Value: "localhost,*"},
	}, odd.Describe())

	conflict := proxyconf.Resolve("", env("no_proxy", "a", "NO_PROXY", "b"))
	require.NoError(t, conflict.Err())
	assert.Equal(t, "", rows(conflict)["proxy_no_proxy"])
	assert.Equal(t, "direct", rows(conflict)["proxy_https"])
}

func TestDescribe_RowOrder(t *testing.T) {
	var keys []string
	for _, r := range proxyconf.Resolve("", nil).Describe() {
		keys = append(keys, r.Key)
	}
	assert.Equal(t, []string{"proxy", "proxy_dns", "proxy_http", "proxy_https", "proxy_no_proxy"}, keys)
}

// A NO_PROXY value is shown as typed, but control characters never reach the
// terminal.
func TestDescribe_NoProxyHasNoControlCharacters(t *testing.T) {
	s := proxyconf.Resolve("", env("HTTPS_PROXY", "http://corp:3128", "NO_PROXY", "a.corp\x1b]0;owned\x07,b.corp\u202e"))
	require.NoError(t, s.Err())
	got := rows(s)["proxy_no_proxy"]
	assert.NotContains(t, got, "\x1b")
	assert.NotContains(t, got, "\x07")
	assert.NotContains(t, got, "\u202e")
	assert.Contains(t, got, "a.corp")
	assert.Contains(t, got, "b.corp")
}
