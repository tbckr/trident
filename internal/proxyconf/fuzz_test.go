package proxyconf_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tbckr/trident/internal/proxyconf"
)

// FuzzNoCredentialLeak checks that neither the deferred error nor any
// description of a proxy value that holds a user name and a numeric password
// ever shows either of them, whichever way the value is rejected or accepted.
func FuzzNoCredentialLeak(f *testing.F) {
	for _, s := range [][2]string{{"123456", "127.0.0.1:9050"}, {"987654321", "h:abc"}, {"4444444", "h:99999/x"}, {"55555555", "[::1"}, {"777777", "h:1 x"}} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, pw, tail string) {
		// A tail that holds the password or the user name makes them legitimately
		// visible (a host called "alice"), which is not a leak.
		if len(pw) < 6 || strings.Trim(pw, "0123456789") != "" || strings.Contains(tail, pw) || strings.Contains(tail, "alice") {
			t.Skip()
		}
		value := "socks5h://alice:" + pw + "@" + tail
		for _, s := range []*proxyconf.Settings{
			proxyconf.Resolve(value, nil),
			proxyconf.Resolve("", env("HTTPS_PROXY", value)),
			proxyconf.Resolve("", env("ALL_PROXY", value)),
		} {
			var out []string
			if err := s.Err(); err != nil {
				out = append(out, err.Error())
			}
			for _, r := range s.Describe() {
				out = append(out, r.Value)
			}
			for _, o := range out {
				assert.NotContains(t, o, pw)
				assert.NotContains(t, o, "alice")
			}
		}
	})
}
