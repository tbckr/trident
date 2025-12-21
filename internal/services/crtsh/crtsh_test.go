package crtsh_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	tridenthttp "github.com/tbckr/trident/internal/http"
	"github.com/tbckr/trident/internal/ratelimit"
	"github.com/tbckr/trident/internal/services/crtsh"
)

func TestCrtshSearch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	client := req.C()
	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpClient := tridenthttp.NewClientWithReqClient(logger, client)
	service := crtsh.NewService(httpClient, logger, nil)

	// Load fixture
	fixturePath := filepath.Join("testdata", "response.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	t.Run("Valid domain", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://crt.sh/?q=%.example.com&output=json",
			httpmock.NewStringResponder(200, string(fixture)))

		result, err := service.Search(context.Background(), "example.com")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, *result, 2)
		assert.Equal(t, "example.com", (*result)[0].CommonName)
		assert.Equal(t, "www.example.com", (*result)[1].CommonName)
	})

	t.Run("Invalid domain format", func(t *testing.T) {
		result, err := service.Search(context.Background(), "invalid_domain")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid domain format")
	})

	t.Run("HTTP 404 Error", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://crt.sh/?q=%.notfound.com&output=json",
			httpmock.NewStringResponder(404, "Not Found"))

		result, err := service.Search(context.Background(), "notfound.com")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unexpected status code: 404")
	})

	t.Run("HTTP 500 Error", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://crt.sh/?q=%.servererror.com&output=json",
			httpmock.NewStringResponder(500, "Internal Server Error"))

		result, err := service.Search(context.Background(), "servererror.com")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unexpected status code: 500")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://crt.sh/?q=%.invalidjson.com&output=json",
			httpmock.NewStringResponder(200, `invalid json`))

		result, err := service.Search(context.Background(), "invalidjson.com")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to unmarshal crt.sh response")
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://crt.sh/?q=%.cancelled.com&output=json",
			httpmock.NewStringResponder(200, `[]`).Delay(100*time.Millisecond))

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		result, err := service.Search(ctx, "cancelled.com")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("Context Timeout", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://crt.sh/?q=%.timeout.com&output=json",
			httpmock.NewStringResponder(200, `[]`).Delay(100*time.Millisecond))

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		result, err := service.Search(ctx, "timeout.com")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("Rate Limiter", func(t *testing.T) {
		limiter := ratelimit.NewLimiter(1, 1, logger)
		serviceWithLimiter := crtsh.NewService(httpClient, logger, limiter)

		httpmock.RegisterResponder("GET", "https://crt.sh/?q=%.ratelimit.com&output=json",
			httpmock.NewStringResponder(200, `[]`))

		// First call should be fast
		result, err := serviceWithLimiter.Search(context.Background(), "ratelimit.com")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Second call should be delayed or fail if context cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		_, err = serviceWithLimiter.Search(ctx, "ratelimit.com")
		assert.Error(t, err)
	})
}
