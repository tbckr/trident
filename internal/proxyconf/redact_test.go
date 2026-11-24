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
		{
			name: "scheme glued to a word character",
			in:   "X_socks5h://alice:S3cr3t@h:1",
			want: "X_socks5h://***@h:1",
		},
		{
			name: "scheme-less --proxy value after a space",
			in:   "crtsh --proxy alice:S3cr3t@127.0.0.1:9050",
			want: "crtsh --proxy ***@127.0.0.1:9050",
		},
		{
			name: "scheme-less --proxy=value",
			in:   "--proxy=alice:S3cr3t@h:1",
			want: "--proxy=***@h:1",
		},
		{
			name: "scheme-less --proxy value, @ inside the password",
			in:   "--proxy=u:p@ss@h:1 dns example.com",
			want: "--proxy=***@h:1 dns example.com",
		},
		{
			name: "scheme-less --proxy value, user name only",
			in:   "--proxy alice@h:1",
			want: "--proxy ***@h:1",
		},
		{
			name: "scheme-less --proxy value in quotes",
			in:   `--proxy "alice:S3cr3t@h:1" --pap-limit=red`,
			want: `--proxy "***@h:1" --pap-limit=red`,
		},
		{
			name: "scheme-less --proxy value, tab as separator",
			in:   "--proxy\talice:S3cr3t@h:1",
			want: "--proxy\t***@h:1",
		},
		{
			name: "--proxy with a URL value keeps its scheme",
			in:   "--proxy socks5h://alice:S3cr3t@h:1 x",
			want: "--proxy socks5h://***@h:1 x",
		},
		// A "/" in the password ends the authority for the generic pattern,
		// so the --proxy value is masked up to its last "@".
		{
			name: "--proxy URL value, / in the password, after a space",
			in:   "crtsh --proxy socks5h://alice:S3/cr3t@127.0.0.1:9050",
			want: "crtsh --proxy socks5h://***@127.0.0.1:9050",
		},
		{
			name: "--proxy=URL value, / in the password",
			in:   "--proxy=http://alice:S3/cr3t@h:1 dns example.com",
			want: "--proxy=http://***@h:1 dns example.com",
		},
		{
			name: "--proxy URL value, / in the password, tab as separator",
			in:   "--proxy\tsocks5h://alice:S3/cr3t@h:1",
			want: "--proxy\tsocks5h://***@h:1",
		},
		{
			name: "--proxy URL value, @ before and / after in the password",
			in:   "--proxy socks5h://alice@x:S3/cr3t@h:1",
			want: "--proxy socks5h://***@h:1",
		},
		{
			name: "--proxy URL value in quotes, / in the password",
			in:   `--proxy 'socks5h://alice:S3/cr3t@h:1' x`,
			want: `--proxy 'socks5h://***@h:1' x`,
		},
		// A quoted value may hold a space before its "@".
		{
			name: "--proxy URL value in quotes, space in the password",
			in:   `--proxy "http://alice:S3 cr3t@h:1" --pap-limit=red`,
			want: `--proxy "http://***@h:1" --pap-limit=red`,
		},
		{
			name: "scheme-less --proxy=value in quotes, space in the password",
			in:   `--proxy="alice:S3 cr3t@h:1" x`,
			want: `--proxy="***@h:1" x`,
		},
		{
			name: "--proxy value, text glued to the closing quote",
			in:   `--proxy "alice"S3cr3t@h:1`,
			want: `--proxy "***@h:1`,
		},
		{name: "--proxy value without userinfo", in: "--proxy 127.0.0.1:9050", want: "--proxy 127.0.0.1:9050"},
		{name: "quoted --proxy value without userinfo", in: `--proxy "127.0.0.1:9050" a@b`, want: `--proxy "127.0.0.1:9050" a@b`},
		{name: "--proxy URL without userinfo", in: "--proxy=socks5h://[::1]:9050", want: "--proxy=socks5h://[::1]:9050"},
		{name: "another flag that starts with --proxy", in: "--proxy-foo a@b", want: "--proxy-foo a@b"},
		{name: "e-mail address elsewhere", in: "mail bob@example.com about --proxy x", want: "mail bob@example.com about --proxy x"},
		{name: "--proxy at the end", in: "crtsh --proxy", want: "crtsh --proxy"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proxyconf.RedactText(tt.in)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "cr3t")
			assert.NotContains(t, got, "alice")
		})
	}
}
