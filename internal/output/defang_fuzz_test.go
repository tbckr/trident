package output_test

import (
	"testing"

	"github.com/tbckr/trident/internal/output"
)

func FuzzDefangURL(f *testing.F) {
	f.Add("http://example.com/path?q=1")
	f.Add("https://foo.bar.baz")
	f.Add("HTTP://EXAMPLE.COM")
	f.Add("ftp://files.example.com")
	f.Add("example.com")
	f.Add("")
	f.Add("http://")
	f.Add("://broken")
	f.Add("https://host:8080/path")
	f.Add("https://user:pass@host.com/path")

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic on any input.
		output.DefangURL(input)
	})
}

func FuzzDefangDomain(f *testing.F) {
	f.Add("example.com")
	f.Add("sub.example.co.uk")
	f.Add("")
	f.Add("nodots")
	f.Add("...")

	f.Fuzz(func(t *testing.T, input string) {
		output.DefangDomain(input)
	})
}

func FuzzDefangIP(f *testing.F) {
	f.Add("192.168.1.1")
	f.Add("10.0.0.1")
	f.Add("::1")
	f.Add("2001:db8::1")
	f.Add("")
	f.Add("not-an-ip")
	f.Add("999.999.999.999")

	f.Fuzz(func(t *testing.T, input string) {
		output.DefangIP(input)
	})
}
