package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandWithWWW(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string
		want   []string
	}{
		{"SLD gets www appended", []string{"example.com"}, []string{"example.com", "www.example.com"}},
		{"already www unchanged", []string{"www.example.com"}, []string{"www.example.com"}},
		{"subdomain 2 dots unchanged", []string{"sub.example.com"}, []string{"sub.example.com"}},
		{"IPv4 unchanged", []string{"8.8.8.8"}, []string{"8.8.8.8"}},
		{"SLD + explicit www not duplicated", []string{"example.com", "www.example.com"}, []string{"example.com", "www.example.com"}},
		{"multiple SLDs each get www", []string{"example.com", "example.org"}, []string{"example.com", "www.example.com", "example.org", "www.example.org"}},
		{"mixed SLD and subdomain", []string{"example.com", "api.example.com"}, []string{"example.com", "www.example.com", "api.example.com"}},
		{"empty slice", []string{}, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, expandWithWWW(tc.inputs))
		})
	}
}
