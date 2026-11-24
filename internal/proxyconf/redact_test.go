package proxyconf_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/proxyconf"
)

func TestRedact(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "no userinfo is unchanged", raw: "socks5h://127.0.0.1:9050", want: "socks5h://127.0.0.1:9050"},
		{name: "username only", raw: "socks5h://alice@127.0.0.1:9050", want: "socks5h://***@127.0.0.1:9050"},
		{name: "username and password, path dropped", raw: "http://alice:S3cr3t@proxy:3128/", want: "http://***@proxy:3128"},
		{name: "empty password", raw: "http://APIKEY:@h", want: "http://***@h"},
		{name: "ipv6 host keeps its brackets", raw: "socks5h://alice:S3cr3t@[::1]:9050", want: "socks5h://***@[::1]:9050"},
		{name: "ipv6 host without userinfo", raw: "socks5h://[::1]:9050", want: "socks5h://[::1]:9050"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)
			got := proxyconf.Redact(u)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "S3cr3t")
			assert.NotContains(t, got, "alice")
			assert.NotContains(t, got, "APIKEY")
		})
	}

	t.Run("nil", func(t *testing.T) {
		assert.Empty(t, proxyconf.Redact(nil))
	})
}

func TestRedactText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "flag value after a space",
			in:   "crtsh --proxy socks5h://alice:S3cr3t@127.0.0.1:9050",
			want: "crtsh --proxy socks5h://***@127.0.0.1:9050",
		},
		{
			name: "flag=value, @ inside the password, path kept",
			in:   "--proxy=http://u:p@ss@h:1/x",
			want: "--proxy=http://***@h:1/x",
		},
		{name: "no URL", in: "dns example.com", want: "dns example.com"},
		{name: "@ only after the host", in: "see https://example.com/a@b", want: "see https://example.com/a@b"},
		{name: "URL without userinfo", in: "socks5h://127.0.0.1:9050", want: "socks5h://127.0.0.1:9050"},
		{
			name: "upper-case scheme",
			in:   "HTTP://alice:S3cr3t@h:1",
			want: "HTTP://***@h:1",
		},
		{
			name: "two URLs",
			in:   "a socks5://u:p@h:1 b https://x:y@h2:2",
			want: "a socks5://***@h:1 b https://***@h2:2",
		},
		{
			name: "inside quotes",
			in:   `invalid value "http://alice:S3cr3t@h:1" for flag`,
			want: `invalid value "http://***@h:1" for flag`,
		},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proxyconf.RedactText(tt.in)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "S3cr3t")
		})
	}
}
