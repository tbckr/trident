package proxyconf_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/proxyconf"
)

// env returns a getenv over the given name, value pairs. Tests never read the
// process environment.
func env(kv ...string) func(string) string {
	m := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

// probe asks the proxy function for the proxy of rawURL; it requires that the
// function does not fail.
func probe(t *testing.T, s *proxyconf.Settings, rawURL string) *url.URL {
	t.Helper()
	got, err := s.ProxyFunc(nil)(&http.Request{URL: mustParse(t, rawURL)})
	require.NoError(t, err)
	return got
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name         string
		setting      string
		env          []string
		wantErr      string
		https        string
		http         string
		dns          string
		dnsUnproxied bool
		configured   bool
	}{
		{
			name:    "setting wins, NO_PROXY and environment ignored",
			setting: "socks5h://127.0.0.1:9050",
			env:     []string{"HTTPS_PROXY", "http://corp:3128", "NO_PROXY", "crt.sh"},
			https:   "socks5h://127.0.0.1:9050", http: "socks5h://127.0.0.1:9050", dns: "socks5h://127.0.0.1:9050",
			configured: true,
		},
		{
			name:  "ALL_PROXY alone",
			env:   []string{"ALL_PROXY", "socks5h://127.0.0.1:9050"},
			https: "socks5h://127.0.0.1:9050", http: "socks5h://127.0.0.1:9050", dns: "socks5h://127.0.0.1:9050",
			configured: true,
		},
		{
			name:  "https chain decides DNS",
			env:   []string{"HTTPS_PROXY", "http://corp:3128", "ALL_PROXY", "socks5h://tor:9050"},
			https: "http://corp:3128", http: "socks5h://tor:9050", dns: "",
			dnsUnproxied: true, configured: true,
		},
		{
			name:       "lowercase and uppercase differ",
			env:        []string{"https_proxy", "socks5h://a:1", "HTTPS_PROXY", "socks5h://b:1"},
			wantErr:    "https_proxy and HTTPS_PROXY are both set and differ; unset one",
			configured: true,
		},
		{
			name:  "same value in both cases",
			env:   []string{"https_proxy", "socks5h://a:1", "HTTPS_PROXY", "socks5h://a:1"},
			https: "socks5h://a:1", http: "", dns: "socks5h://a:1",
			configured: true,
		},
		{
			name:       "http only",
			env:        []string{"http_proxy", "http://127.0.0.1:8118"},
			wantErr:    "http_proxy applies only to http:// URLs",
			configured: true,
		},
		{
			name: "http only with NO_PROXY=*",
			env:  []string{"http_proxy", "http://127.0.0.1:8118", "NO_PROXY", "*"},
		},
		{
			name:  "CGI drops uppercase HTTP_PROXY",
			env:   []string{"REQUEST_METHOD", "GET", "HTTP_PROXY", "http://evil:1", "HTTPS_PROXY", "http://corp:3128"},
			https: "http://corp:3128", http: "", dns: "",
			dnsUnproxied: true, configured: true,
		},
		{
			name: "CGI junk is not validated",
			env:  []string{"REQUEST_METHOD", "GET", "HTTP_PROXY", " junk"},
		},
		{
			name:  "CGI keeps lowercase http_proxy",
			env:   []string{"REQUEST_METHOD", "GET", "http_proxy", "http://corp:3128", "https_proxy", "http://corp:3128"},
			https: "http://corp:3128", http: "http://corp:3128", dns: "",
			dnsUnproxied: true, configured: true,
		},
		{
			name:       "CGI keeps lowercase http_proxy, which then is http only",
			env:        []string{"REQUEST_METHOD", "GET", "http_proxy", "http://corp:3128"},
			wantErr:    "http_proxy applies only to http:// URLs",
			configured: true,
		},
		{
			name:  "shadowed socks:// is ignored",
			env:   []string{"all_proxy", "socks://h:1080/", "https_proxy", "http://a:1", "http_proxy", "http://a:2"},
			https: "http://a:1", http: "http://a:2", dns: "",
			dnsUnproxied: true, configured: true,
		},
		{
			name:       "socks:// alone",
			env:        []string{"all_proxy", "socks://h:1080/"},
			wantErr:    "all_proxy: socks:// is ambiguous; use socks5h://",
			configured: true,
		},
		{
			name:  "shadowed ALL_PROXY conflict is ignored",
			env:   []string{"all_proxy", "x", "ALL_PROXY", "y", "HTTPS_PROXY", "http://a:1", "HTTP_PROXY", "http://a:2"},
			https: "http://a:1", http: "http://a:2", dns: "",
			dnsUnproxied: true, configured: true,
		},
		{
			name:       "used ALL_PROXY conflict",
			env:        []string{"all_proxy", "socks5h://a:1", "ALL_PROXY", "socks5h://b:1", "HTTPS_PROXY", "http://a:1"},
			wantErr:    "all_proxy and ALL_PROXY are both set and differ; unset one",
			configured: true,
		},
		{
			name:       "http pair conflict",
			env:        []string{"https_proxy", "http://a:1", "http_proxy", "http://a:2", "HTTP_PROXY", "http://a:3"},
			wantErr:    "http_proxy and HTTP_PROXY are both set and differ; unset one",
			configured: true,
		},
		{
			name:       "NO_PROXY pair conflict",
			env:        []string{"HTTPS_PROXY", "http://a:1", "no_proxy", "a.corp", "NO_PROXY", "b.corp"},
			wantErr:    "no_proxy and NO_PROXY are both set and differ; unset one",
			configured: true,
		},
		{
			name:       "broken value",
			env:        []string{"HTTPS_PROXY", " socks5://x:1"},
			wantErr:    "HTTPS_PROXY: contains whitespace",
			configured: true,
		},
		{
			name:       "star among NO_PROXY entries",
			env:        []string{"HTTPS_PROXY", "http://corp:3128", "NO_PROXY", "localhost,*"},
			wantErr:    "NO_PROXY",
			configured: true,
		},
		{
			name: "odd NO_PROXY is ignored when no proxy is selected",
			env:  []string{"NO_PROXY", "localhost,*"},
		},
		{
			name: "NO_PROXY pair conflict is ignored when no proxy is selected",
			env:  []string{"no_proxy", "a", "NO_PROXY", "b"},
		},
		{
			name: "NO_PROXY=* disables everything and validates nothing",
			env:  []string{"HTTPS_PROXY", "garbage value", "NO_PROXY", " * "},
		},
		{
			name: "nothing set",
		},
		{
			name: "empty variables count as unset",
			env:  []string{"https_proxy", "", "HTTPS_PROXY", "", "ALL_PROXY", ""},
		},
		{
			name:  "empty lowercase next to a value",
			env:   []string{"https_proxy", "", "HTTPS_PROXY", "socks5h://a:1"},
			https: "socks5h://a:1", http: "", dns: "socks5h://a:1",
			configured: true,
		},
		{
			name:       "invalid setting",
			setting:    "socks5://u:S3cr3t@h:abc",
			wantErr:    "proxy: not a valid URL",
			configured: true,
		},
		{
			name:       "invalid setting masks a valid environment",
			setting:    " socks5h://x:1",
			env:        []string{"HTTPS_PROXY", "http://corp:3128"},
			wantErr:    "proxy: contains whitespace",
			configured: true,
		},
		{
			name:  "scheme-less environment value is http",
			env:   []string{"HTTPS_PROXY", "corp:3128", "HTTP_PROXY", "corp:3128"},
			https: "http://corp:3128", http: "http://corp:3128", dns: "",
			dnsUnproxied: true, configured: true,
		},
		{
			name:  "uppercase scheme is accepted",
			env:   []string{"ALL_PROXY", "SOCKS5H://tor:9050"},
			https: "socks5h://tor:9050", http: "socks5h://tor:9050", dns: "socks5h://tor:9050",
			configured: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := proxyconf.Resolve(tt.setting, env(tt.env...))
			require.NotNil(t, s)
			assert.Equal(t, tt.configured, s.Configured(), "Configured")

			if tt.wantErr != "" {
				require.Error(t, s.Err())
				assert.ErrorIs(t, s.Err(), apperr.ErrProxyConfig)
				assert.Contains(t, s.Err().Error(), tt.wantErr)
				assert.NotContains(t, s.Err().Error(), "S3cr3t")
				for _, target := range []string{"https://crt.sh/", "http://p.example/"} {
					got, err := s.ProxyFunc(nil)(&http.Request{URL: mustParse(t, target)})
					assert.Nil(t, got, target)
					assert.Equal(t, s.Err(), err, target)
				}
				assert.Nil(t, s.HTTPSProxy())
				assert.Nil(t, s.DNSProxy())
				assert.False(t, s.DNSUnproxied())
				return
			}

			require.NoError(t, s.Err())
			assert.Equal(t, tt.https, proxyconf.Redact(probe(t, s, "https://crt.sh/")), "https")
			assert.Equal(t, tt.http, proxyconf.Redact(probe(t, s, "http://p.example/")), "http")
			assert.Equal(t, tt.https, proxyconf.Redact(s.HTTPSProxy()), "HTTPSProxy")
			assert.Equal(t, tt.dns, proxyconf.Redact(s.DNSProxy()), "DNSProxy")
			assert.Equal(t, tt.dnsUnproxied, s.DNSUnproxied(), "DNSUnproxied")
		})
	}
}

func TestResolve_NilGetenvIsAnEmptyEnvironment(t *testing.T) {
	s := proxyconf.Resolve("", nil)
	require.NoError(t, s.Err())
	assert.False(t, s.Configured())
	assert.Nil(t, probe(t, s, "https://crt.sh/"))
}

func TestResolve_CredentialsReachTheProxyFunction(t *testing.T) {
	// The userinfo drives SOCKS authentication, so the proxy function keeps it;
	// every display goes through Redact.
	s := proxyconf.Resolve("socks5h://alice:S3cr3t@127.0.0.1:9050", nil)
	require.NoError(t, s.Err())
	got := probe(t, s, "https://crt.sh/")
	require.NotNil(t, got)
	pw, _ := got.User.Password()
	assert.Equal(t, "alice", got.User.Username())
	assert.Equal(t, "S3cr3t", pw)
	assert.Equal(t, "socks5h://***@127.0.0.1:9050", proxyconf.Redact(got))
}

func TestResolve_ReturnedURLsAreCopies(t *testing.T) {
	s := proxyconf.Resolve("socks5h://alice:S3cr3t@127.0.0.1:9050", nil)
	for _, u := range []*url.URL{probe(t, s, "https://crt.sh/"), s.HTTPSProxy(), s.DNSProxy()} {
		require.NotNil(t, u)
		u.Host = "evil:1"
		u.User = nil
	}
	assert.Equal(t, "socks5h://***@127.0.0.1:9050", proxyconf.Redact(probe(t, s, "https://crt.sh/")))
	assert.Equal(t, "socks5h://***@127.0.0.1:9050", proxyconf.Redact(probe(t, s, "http://p.example/")))
	assert.Equal(t, "socks5h://***@127.0.0.1:9050", proxyconf.Redact(s.HTTPSProxy()))
	assert.Equal(t, "socks5h://***@127.0.0.1:9050", proxyconf.Redact(s.DNSProxy()))
}

// The first problem found wins: conflict, then http only, then a bad value,
// then a bad NO_PROXY. There is one deferred error per run.
func TestResolve_ErrorPrecedence(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want string
	}{
		{
			name: "conflict beats a bad value",
			env:  []string{"https_proxy", "socks5h://a:1", "HTTPS_PROXY", "socks5h://b:1", "http_proxy", " bad", "NO_PROXY", "*,x"},
			want: "https_proxy and HTTPS_PROXY are both set and differ",
		},
		{
			name: "http only beats a bad NO_PROXY",
			env:  []string{"http_proxy", "http://a:1", "NO_PROXY", "*,x"},
			want: "http_proxy applies only to http:// URLs",
		},
		{
			name: "bad value beats a bad NO_PROXY",
			env:  []string{"HTTPS_PROXY", "socks4://a:1", "NO_PROXY", "*,x"},
			want: "HTTPS_PROXY: socks4:// is not supported",
		},
		{
			name: "bad NO_PROXY last",
			env:  []string{"HTTPS_PROXY", "http://a:1", "NO_PROXY", "*,x"},
			want: "NO_PROXY",
		},
		{
			name: "bad https value before bad http value",
			env:  []string{"HTTPS_PROXY", "socks4://a:1", "HTTP_PROXY", "socks4a://a:1"},
			want: "HTTPS_PROXY: socks4:// is not supported",
		},
		{
			name: "bad http value alone is reported",
			env:  []string{"HTTPS_PROXY", "http://a:1", "HTTP_PROXY", "http://a:abc"},
			want: "HTTP_PROXY: not a valid URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := proxyconf.Resolve("", env(tt.env...)).Err()
			require.Error(t, err)
			assert.ErrorIs(t, err, apperr.ErrProxyConfig)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestResolve_HTTPOnlyNamesTheVariableAndItsCase(t *testing.T) {
	lower := proxyconf.Resolve("", env("http_proxy", "http://127.0.0.1:8118")).Err()
	require.Error(t, lower)
	assert.Equal(t, "invalid proxy configuration: http_proxy applies only to http:// URLs, "+
		"but trident's services use https:// and would bypass the proxy; "+
		"set https_proxy (or all_proxy) to the same value, use --proxy socks5h://…, "+
		"or set NO_PROXY='*' to connect directly (dns, detect and cymru cannot use an HTTP proxy)",
		lower.Error())

	upper := proxyconf.Resolve("", env("HTTP_PROXY", "http://127.0.0.1:8118")).Err()
	require.Error(t, upper)
	assert.Contains(t, upper.Error(), "HTTP_PROXY applies only to http:// URLs")
	assert.Contains(t, upper.Error(), "set HTTPS_PROXY (or ALL_PROXY) to the same value")

	for _, err := range []error{lower, upper} {
		assert.ErrorIs(t, err, apperr.ErrProxyConfig)
		assert.NotContains(t, err.Error(), "127.0.0.1")
	}
}

func TestResolve_ErrorsNeverEchoTheValue(t *testing.T) {
	for _, value := range []string{
		" socks5://u:S3cr3t@h:1", "socks5://u:S3cr3t@h:abc", "socks4://u:S3cr3t@h:1", "socks://u:S3cr3t@h:1",
		"ftp://u:S3cr3t@h:1", "http://u:S3cr3t@h:1/path", "http://u:S3cr3t@:1",
	} {
		for name, s := range map[string]*proxyconf.Settings{
			"setting": proxyconf.Resolve(value, nil),
			"https":   proxyconf.Resolve("", env("HTTPS_PROXY", value)),
			"all":     proxyconf.Resolve("", env("all_proxy", value)),
			"http":    proxyconf.Resolve("", env("https_proxy", "http://ok:1", "HTTP_PROXY", value)),
		} {
			require.Error(t, s.Err(), "%s %q", name, value)
			assert.NotContains(t, s.Err().Error(), "S3cr3t", "%s %q", name, value)
			assert.NotContains(t, s.Err().Error(), "u:", "%s %q", name, value)
			for _, r := range s.Describe() {
				assert.NotContains(t, r.Value, "S3cr3t", "%s %q %s", name, value, r.Key)
			}
		}
	}
}

func TestProxyFunc_SchemeAndPort(t *testing.T) {
	// https goes through the https chain, http through the http chain; every
	// other scheme errs toward proxying and uses the https value.
	s := proxyconf.Resolve("", env("HTTPS_PROXY", "http://sec:1", "HTTP_PROXY", "http://plain:1"))
	require.NoError(t, s.Err())
	assert.Equal(t, "http://sec:1", proxyconf.Redact(probe(t, s, "https://crt.sh/")))
	assert.Equal(t, "http://plain:1", proxyconf.Redact(probe(t, s, "http://crt.sh/")))
	assert.Equal(t, "http://sec:1", proxyconf.Redact(probe(t, s, "wss://crt.sh/")))
	assert.Equal(t, "http://sec:1", proxyconf.Redact(probe(t, s, "ftp://crt.sh/")))

	// A request without a URL cannot be matched against NO_PROXY; it is
	// proxied, not sent direct.
	got, err := s.ProxyFunc(nil)(&http.Request{})
	require.NoError(t, err)
	assert.Equal(t, "http://sec:1", proxyconf.Redact(got))
	got, err = s.ProxyFunc(nil)(nil)
	require.NoError(t, err)
	assert.Equal(t, "http://sec:1", proxyconf.Redact(got))
}

func TestProxyFunc_NoProxyPorts(t *testing.T) {
	s := proxyconf.Resolve("", env(
		"HTTPS_PROXY", "http://corp:3128", "HTTP_PROXY", "http://corp:3128",
		"NO_PROXY", "a.corp:8443,b.corp:80,[::1]:9000",
	))
	require.NoError(t, s.Err())
	// The default port of the scheme counts.
	assert.Nil(t, probe(t, s, "https://a.corp:8443/"))
	assert.NotNil(t, probe(t, s, "https://a.corp/"))
	assert.Nil(t, probe(t, s, "http://b.corp/"))
	assert.NotNil(t, probe(t, s, "https://b.corp/"))
	assert.Nil(t, probe(t, s, "https://[::1]:9000/"))
	assert.NotNil(t, probe(t, s, "https://[::1]:9001/"))
}

func TestResolve_NoProxyNotifies(t *testing.T) {
	var got []string
	notify := func(host string) { got = append(got, host) }

	s := proxyconf.Resolve("", env("HTTPS_PROXY", "http://corp:3128", "NO_PROXY", ".corp"))
	require.NoError(t, s.Err())
	fn := s.ProxyFunc(notify)

	u, err := fn(&http.Request{URL: mustParse(t, "https://a.corp/")})
	require.NoError(t, err)
	assert.Nil(t, u)
	assert.Equal(t, []string{"a.corp"}, got)

	u, err = fn(&http.Request{URL: mustParse(t, "https://crt.sh/")})
	require.NoError(t, err)
	assert.Equal(t, "http://corp:3128", proxyconf.Redact(u))
	assert.Equal(t, []string{"a.corp"}, got, "a proxied request does not notify")

	// Every exempted request notifies; de-duplication per host is the caller's job.
	_, err = fn(&http.Request{URL: mustParse(t, "https://a.corp/again")})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.corp", "a.corp"}, got)

	// A nil notify is fine.
	u, err = s.ProxyFunc(nil)(&http.Request{URL: mustParse(t, "https://a.corp/")})
	require.NoError(t, err)
	assert.Nil(t, u)
}

// With only an https proxy an http:// request has no proxy of its own: the
// proxy function sends it direct and the dial guard refuses it, unless NO_PROXY
// exempts the host, which is then notified like any other exemption.
func TestProxyFunc_HTTPWithOnlyAnHTTPSProxyNotifiesExemptions(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{name: "HTTPS_PROXY only", env: []string{"HTTPS_PROXY", "http://corp:3128", "NO_PROXY", ".corp,b.example:80"}},
		{name: "CGI drops HTTP_PROXY", env: []string{"REQUEST_METHOD", "GET", "HTTP_PROXY", "http://evil:1", "https_proxy", "http://corp:3128", "no_proxy", ".corp,b.example:80"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			s := proxyconf.Resolve("", env(tc.env...))
			require.NoError(t, s.Err())
			fn := s.ProxyFunc(func(host string) { got = append(got, host) })

			for _, target := range []string{"http://a.corp/", "http://b.example/", "http://crt.sh/", "http://b.example:8080/"} {
				u, err := fn(&http.Request{URL: mustParse(t, target)})
				require.NoError(t, err, target)
				assert.Nil(t, u, "%s: no http proxy, the request is not proxied", target)
			}
			assert.Equal(t, []string{"a.corp", "b.example"}, got, "exactly the exempted hosts notify; the scheme's default port counts")

			// The guard's lookups (a nil notify) still work.
			u, err := s.ProxyFunc(nil)(&http.Request{URL: mustParse(t, "http://a.corp/")})
			require.NoError(t, err)
			assert.Nil(t, u)
		})
	}
}

func TestResolve_NoProxyNeverNotifiesWithoutAnExemption(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		env     []string
		want    string
	}{
		{
			name:    "trident's own setting ignores NO_PROXY",
			setting: "http://corp:3128",
			env:     []string{"NO_PROXY", ".corp"},
			want:    "http://corp:3128",
		},
		{
			name: "NO_PROXY=* disables the proxy without a notice",
			env:  []string{"HTTPS_PROXY", "http://corp:3128", "NO_PROXY", "*"},
			want: "",
		},
		{
			name: "no proxy at all",
			env:  []string{"NO_PROXY", ".corp"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := proxyconf.Resolve(tt.setting, env(tt.env...))
			require.NoError(t, s.Err())
			fn := s.ProxyFunc(func(host string) { t.Errorf("unexpected notify for %q", host) })
			u, err := fn(&http.Request{URL: mustParse(t, "https://a.corp/")})
			require.NoError(t, err)
			assert.Equal(t, tt.want, proxyconf.Redact(u))
		})
	}
}

// NO_PROXY does not apply to DNS: only NO_PROXY=* disables it.
func TestResolve_NoProxyDoesNotTouchDNS(t *testing.T) {
	s := proxyconf.Resolve("", env("HTTPS_PROXY", "socks5h://tor:9050", "NO_PROXY", "crt.sh,.corp"))
	require.NoError(t, s.Err())
	assert.Equal(t, "socks5h://tor:9050", proxyconf.Redact(s.DNSProxy()))
}
