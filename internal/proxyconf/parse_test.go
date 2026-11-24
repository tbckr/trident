package proxyconf

import (
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
)

func TestParseProxyURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "socks5h with port", raw: "socks5h://127.0.0.1:9050", want: "socks5h://127.0.0.1:9050"},
		{name: "scheme lowercased, host case kept", raw: "SOCKS5://Host:1080", want: "socks5://Host:1080"},
		{name: "trailing slash dropped", raw: "socks5h://127.0.0.1:9050/", want: "socks5h://127.0.0.1:9050"},
		{name: "userinfo redacted", raw: "socks5h://alice:S3cr3t@127.0.0.1:9050", want: "socks5h://***@127.0.0.1:9050"},
		{name: "ipv6 literal", raw: "socks5h://[::1]:9050", want: "socks5h://[::1]:9050"},
		{name: "ipv6 literal with trailing slash", raw: "socks5h://[::1]:9050/", want: "socks5h://[::1]:9050"},
		{name: "host:port without scheme is http", raw: "127.0.0.1:9050", want: "http://127.0.0.1:9050"},
		{name: "localhost:port is not a scheme", raw: "localhost:9050", want: "http://localhost:9050"},
		{name: "userinfo without scheme", raw: "alice:S3cr3t@proxy:3128", want: "http://***@proxy:3128"},
		{name: "no port", raw: "http://proxy.corp", want: "http://proxy.corp"},
		{name: "https", raw: "https://proxy.corp:8443", want: "https://proxy.corp:8443"},
		{name: "port 1", raw: "http://h:1", want: "http://h:1"},
		{name: "port 65535", raw: "http://h:65535", want: "http://h:65535"},
		{name: "empty port is no port", raw: "http://h:", want: "http://h"},
		{name: "empty port on ipv6", raw: "http://[::1]:", want: "http://[::1]"},
		{name: "empty value", raw: "", wantErr: "missing host"},

		{name: "leading space", raw: " socks5://x:1", wantErr: "contains whitespace or control characters"},
		{name: "trailing newline", raw: "socks5h://127.0.0.1:9050\n", wantErr: "contains whitespace or control characters"},
		{name: "carriage return", raw: "socks5h://127.0.0.1:9050\r", wantErr: "contains whitespace or control characters"},
		{name: "tab", raw: "socks5h://127.0.0.1\t:9050", wantErr: "contains whitespace or control characters"},
		{name: "inner space", raw: "socks5h://127.0.0.1 :9050", wantErr: "contains whitespace or control characters"},
		{name: "vertical tab", raw: "socks5h://127.0.0.1:9050\v", wantErr: "contains whitespace or control characters"},
		{name: "form feed", raw: "\fsocks5h://127.0.0.1:9050", wantErr: "contains whitespace or control characters"},
		{name: "no-break space", raw: "socks5h://127.0.0.1:9050\u00a0", wantErr: "contains whitespace or control characters"},
		{name: "next line (C1)", raw: "socks5h://127.0.0.1:9050\u0085", wantErr: "contains whitespace or control characters"},
		{name: "ideographic space", raw: "\u3000socks5h://127.0.0.1:9050", wantErr: "contains whitespace or control characters"},
		{name: "line separator", raw: "socks5h://127.0.0.1:9050\u2028", wantErr: "contains whitespace or control characters"},
		{name: "narrow no-break space", raw: "socks5h://127.0.0.1\u202f:9050", wantErr: "contains whitespace or control characters"},
		{name: "NUL", raw: "socks5h://127.0.0.1:9050\x00", wantErr: "contains whitespace or control characters"},
		{name: "escape", raw: "socks5h://\x1b[31mh:1", wantErr: "contains whitespace or control characters"},
		{name: "DEL", raw: "socks5h://h\x7f:1", wantErr: "contains whitespace or control characters"},
		{name: "C1 control", raw: "socks5h://h\u009b:1", wantErr: "contains whitespace or control characters"},
		{name: "zero-width space", raw: "socks5h://h\u200b:1", wantErr: "contains whitespace or control characters"},
		{name: "bidi override", raw: "socks5h://\u202eh:1", wantErr: "contains whitespace or control characters"},
		{name: "byte order mark", raw: "\ufeffsocks5h://h:1", wantErr: "contains whitespace or control characters"},
		{name: "invalid utf-8", raw: "socks5h://h\xff:1", wantErr: "not a valid URL"},
		{name: "percent-encoded C1 control in host", raw: "http://%C2%9B31m:1", wantErr: "contains whitespace or control characters"},
		{name: "percent-encoded bidi override in host", raw: "http://%E2%80%AE.moc.live:1", wantErr: "contains whitespace or control characters"},
		{name: "percent-encoded no-break space in host", raw: "http://%C2%A0h:1", wantErr: "contains whitespace or control characters"},
		{name: "percent-encoded zero-width space in host", raw: "socks5h://h%E2%80%8B:1", wantErr: "contains whitespace or control characters"},
		{name: "percent-encoded invalid utf-8 in host", raw: "http://%ff:1", wantErr: "not a valid URL"},
		{name: "percent-encoded invalid utf-8 without scheme", raw: "%ff:1", wantErr: "not a valid URL"},

		{name: "scheme only", raw: "socks5://", wantErr: "missing host"},
		{name: "authority-less double slash", raw: "//h:1", wantErr: "missing host"},
		{name: "port only", raw: ":9050", wantErr: "missing host"},
		{name: "userinfo and port only", raw: "socks5://alice:S3cr3t@:9050", wantErr: "missing host"},
		{name: "unclosed bracket", raw: "http://[::1", wantErr: "not a valid URL"},
		{name: "non-numeric port", raw: "http://h:abc", wantErr: "not a valid URL"},
		{name: "bare percent", raw: "%", wantErr: "not a valid URL"},
		{name: "non-numeric port with secret", raw: "socks5://alice:S3cr3t@h:abc", wantErr: "not a valid URL"},
		{name: "unbracketed ipv6", raw: "socks5h://::1", wantErr: "not a valid URL"},
		{name: "unbracketed ipv6 with port", raw: "socks5h://::1:9050", wantErr: "not a valid URL"},
		{name: "bad escape in userinfo", raw: "socks5://alice:S3cr3t%zz@h:1", wantErr: "not a valid URL"},
		{name: "port too large", raw: "http://h:99999", wantErr: "port must be a number from 1 to 65535"},
		{name: "port just too large", raw: "http://h:65536", wantErr: "port must be a number from 1 to 65535"},
		{name: "port overflows int", raw: "http://h:99999999999999999999", wantErr: "port must be a number from 1 to 65535"},
		{name: "port zero", raw: "http://h:0", wantErr: "port must be a number from 1 to 65535"},
		{name: "port zero with secret", raw: "socks5://alice:S3cr3t@h:0", wantErr: "port must be a number from 1 to 65535"},
		{name: "single-slash typo", raw: "socks5:/127.0.0.1:9050", wantErr: "must not contain a path, query or fragment"},
		{name: "unix socket path", raw: "socks5h://localhost/run/tor/socks", wantErr: "must not contain a path, query or fragment"},
		{name: "query", raw: "http://h:1/?x=1", wantErr: "must not contain a path, query or fragment"},
		{name: "query without slash", raw: "http://h:1?x=1", wantErr: "must not contain a path, query or fragment"},
		{name: "empty query", raw: "http://h:1?", wantErr: "must not contain a path, query or fragment"},
		{name: "fragment", raw: "http://h:1#frag", wantErr: "must not contain a path, query or fragment"},
		{name: "empty fragment", raw: "http://h:1#", wantErr: "must not contain a path, query or fragment"},
		{name: "double slash path", raw: "http://h:1//", wantErr: "must not contain a path, query or fragment"},
		{name: "path with secret", raw: "socks5://alice:S3cr3t@h:1/S3cr3t", wantErr: "must not contain a path, query or fragment"},
		{name: "socks4", raw: "socks4://h:1080", wantErr: "socks4:// is not supported; use socks5h://"},
		{name: "socks4 mixed case", raw: "SOCKS4://h:1080", wantErr: "socks4:// is not supported; use socks5h://"},
		{name: "socks4a", raw: "socks4a://h:1080", wantErr: "socks4a:// is not supported; use socks5h://"},
		{name: "socks", raw: "socks://h:1080/", wantErr: "socks:// is ambiguous; use socks5h://"},
		{name: "ftp", raw: "ftp://h:1", wantErr: `unsupported scheme "ftp"; use http://, https://, socks5:// or socks5h://`},
		{name: "scheme lowercased in message", raw: "FTP://h:1", wantErr: `unsupported scheme "ftp"; use http://, https://, socks5:// or socks5h://`},
		{name: "scheme that is not a token is omitted", raw: "S3cr3t_x://h:1", wantErr: "unsupported scheme; use http://, https://, socks5:// or socks5h://"},
		{name: "userinfo in scheme position is omitted", raw: "alice:S3cr3t@h://x", wantErr: "unsupported scheme; use http://, https://, socks5:// or socks5h://"},
		{name: "path before :// is omitted", raw: "h:1/S3cr3t://x", wantErr: "unsupported scheme; use http://, https://, socks5:// or socks5h://"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := parseProxyURL(tt.raw, "HTTPS_PROXY")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Nil(t, u)
				assert.ErrorIs(t, err, apperr.ErrProxyConfig)
				assert.Contains(t, err.Error(), "HTTPS_PROXY: "+tt.wantErr)
				assert.NotContains(t, err.Error(), "S3cr3t")
				assert.NotContains(t, err.Error(), "alice")
				if tt.raw != "" {
					assert.NotContains(t, err.Error(), tt.raw)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, u)
			assert.Equal(t, tt.want, Redact(u))
			assert.Empty(t, u.Path)
			assert.Empty(t, u.RawPath)
			assert.Empty(t, u.RawQuery)
			assert.Empty(t, u.Fragment)
			assert.Equal(t, strings.ToLower(u.Scheme), u.Scheme)
		})
	}
}

// TestParseProxyURL_KeepsUserinfo checks that redaction is a display concern:
// the validated URL still carries the credentials the proxy needs.
func TestParseProxyURL_KeepsUserinfo(t *testing.T) {
	u, err := parseProxyURL("socks5h://alice:S3cr3t@127.0.0.1:9050", "proxy")
	require.NoError(t, err)
	require.NotNil(t, u.User)
	assert.Equal(t, "alice", u.User.Username())
	pw, ok := u.User.Password()
	assert.True(t, ok)
	assert.Equal(t, "S3cr3t", pw)
	assert.Equal(t, "127.0.0.1", u.Hostname())
	assert.Equal(t, "9050", u.Port())
}

func TestParseProxyURL_ErrorShape(t *testing.T) {
	_, err := parseProxyURL("socks5://", "https_proxy")
	require.Error(t, err)
	assert.EqualError(t, err, "invalid proxy configuration: https_proxy: missing host")
}

func TestValidate(t *testing.T) {
	require.NoError(t, Validate("socks5h://127.0.0.1:9050"))

	err := Validate("socks4://h:1")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrProxyConfig)
	assert.Contains(t, err.Error(), "invalid proxy configuration: proxy: socks4://")
}

// reasonPattern matches every reason parseProxyURL may give. Anything else in
// an error message would be text derived from the value.
var reasonPattern = regexp.MustCompile(`^(contains whitespace or control characters` +
	`|missing host` +
	`|not a valid URL` +
	`|port must be a number from 1 to 65535` +
	`|must not contain a path, query or fragment` +
	`|socks4:// is not supported; use socks5h://` +
	`|socks4a:// is not supported; use socks5h://` +
	`|socks:// is ambiguous; use socks5h://` +
	`|unsupported scheme( "[a-z][a-z0-9+.-]*")?; use http://, https://, socks5:// or socks5h://)$`)

// unsafeText is the fuzz test's own notion of "must never be displayed":
// whitespace, control and format characters, and invalid UTF-8.
func unsafeText(s string) bool {
	if !utf8.ValidString(s) {
		return true
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return true
		}
	}
	return false
}

func FuzzParseProxyURL(f *testing.F) {
	for _, seed := range []string{
		"socks5h://127.0.0.1:9050", "SOCKS5://Host:1080", "127.0.0.1:9050", "localhost:9050",
		"socks5h://alice:S3cr3t@127.0.0.1:9050", "socks5h://[::1]:9050", "http://[::1", "http://h:abc",
		"%", "//h:1", ":9050", "socks5:/127.0.0.1:9050", "socks4://h:1080", "socks://h:1080/",
		"ftp://h:1", " socks5://x:1", "http://h:1#", "http://h:1?", "alice:S3cr3t@h://x", "\xff",
		"http://%C2%9B31m:1", "http://%E2%80%AE.moc.live:1", "http://%C2%A0h:1", "http://%ff:1",
	} {
		f.Add(seed)
	}
	const prefix = "invalid proxy configuration: HTTPS_PROXY: "
	f.Fuzz(func(t *testing.T, raw string) {
		u, err := parseProxyURL(raw, "HTTPS_PROXY")
		if err != nil {
			require.Nil(t, u)
			require.ErrorIs(t, err, apperr.ErrProxyConfig)
			reason, ok := strings.CutPrefix(err.Error(), prefix)
			require.True(t, ok, "unexpected error shape: %q", err.Error())
			require.Regexp(t, reasonPattern, reason)
			return
		}
		require.NotNil(t, u)
		require.Contains(t, []string{"http", "https", "socks5", "socks5h"}, u.Scheme)
		require.NotEmpty(t, u.Hostname())
		require.Empty(t, u.Path)
		require.Empty(t, u.RawQuery)
		require.Empty(t, u.Fragment)
		// What is shown must hold nothing invisible, whether it came in raw or
		// percent-encoded (net/url decodes %XX in the host).
		require.False(t, unsafeText(Redact(u)), "Redact shows %q", Redact(u))
		if u.User != nil {
			require.Contains(t, Redact(u), "://***@")
		}
	})
}
