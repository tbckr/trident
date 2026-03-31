package services_test

import (
	"testing"

	"github.com/tbckr/trident/internal/services"
)

func FuzzIsDomain(f *testing.F) {
	f.Add("example.com")
	f.Add("sub.example.com")
	f.Add("a.b.c.d.example.co.uk")
	f.Add("")
	f.Add("not a domain")
	f.Add("192.168.1.1")
	f.Add("-invalid.com")
	f.Add("example-.com")
	f.Add("a")
	f.Add(".com")
	f.Add("example.")
	f.Add("ex" + string(make([]byte, 64)) + ".com")

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic on any input.
		services.IsDomain(input)
	})
}
