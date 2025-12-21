package pgp_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	tridenthttp "github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/services/pgp"
)

func TestPGPSearch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	client := req.C()
	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpClient := tridenthttp.NewClientWithReqClient(logger, client)
	service := pgp.NewService(httpClient, logger, nil)

	t.Run("Found", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://keys.openpgp.org/vks/v1/by-email/test@example.com",
			httpmock.NewStringResponder(200, `{"fingerprint":"123","key_id":"ABC","user_ids":["test@example.com"]}`))

		result, err := service.Search(context.Background(), "test@example.com")
		assert.NoError(t, err)
		assert.Equal(t, "Found", result.Status)
		assert.Equal(t, "123", result.Fingerprint)
	})

	t.Run("Not Found", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://keys.openpgp.org/vks/v1/by-email/missing@example.com",
			httpmock.NewStringResponder(404, "Not Found"))

		result, err := service.Search(context.Background(), "missing@example.com")
		assert.NoError(t, err)
		assert.Equal(t, "Not Found", result.Status)
	})
}
