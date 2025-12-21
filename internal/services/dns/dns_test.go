package dns_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tbckr/trident/internal/services/dns"
)

func TestDNSLookup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := dns.NewService(logger, nil)

	t.Run("Valid domain", func(t *testing.T) {
		result, err := service.Lookup(context.Background(), "google.com")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.A)
	})

	t.Run("Invalid domain", func(t *testing.T) {
		result, err := service.Lookup(context.Background(), "invalid.domain.that.does.not.exist.example.com")
		assert.NoError(t, err) // net.Resolver usually returns empty result, not error for non-existent domains
		assert.NotNil(t, result)
		assert.Empty(t, result.A)
	})
}
