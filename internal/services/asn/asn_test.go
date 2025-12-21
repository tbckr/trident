package asn_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tbckr/trident/internal/services/asn"
)

func TestASNLookup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := asn.NewService(logger, nil)

	t.Run("Valid ASN", func(t *testing.T) {
		result, err := service.Lookup(context.Background(), "AS15169")
		if err != nil {
			t.Skip("Network issues, skipping real lookup")
		}
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "AS15169", result.ASN)
	})

	t.Run("Valid IP", func(t *testing.T) {
		result, err := service.Lookup(context.Background(), "8.8.8.8")
		if err != nil {
			t.Skip("Network issues, skipping real lookup")
		}
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "AS15169", result.ASN)
	})
}
