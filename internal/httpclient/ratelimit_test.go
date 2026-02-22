package httpclient_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/httpclient"
	"github.com/tbckr/trident/internal/ratelimit"
)

// TestAttachRateLimit_TransportError verifies that the retry condition does not
// panic when a transport-level error leaves resp.Response (the embedded
// *http.Response) nil. This was the root cause of the panic reported for
// "quad9 blocked <domain>" when the underlying HTTP transport failed.
func TestAttachRateLimit_TransportError_NoPanic(t *testing.T) {
	client, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)

	// High rate/burst so the limiter never blocks during the test.
	httpclient.AttachRateLimit(client, ratelimit.New(1000, 1000))

	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)

	httpmock.RegisterResponder(http.MethodGet, "https://example.com/",
		httpmock.NewErrorResponder(errors.New("connection refused")))

	// Must not panic. The retry condition receives a non-nil *req.Response
	// whose embedded *http.Response is nil — the scenario that caused the crash.
	_, err = client.R().Get("https://example.com/")
	assert.Error(t, err)
}

// TestAttachRateLimit_TransportError_Retries verifies that transient transport
// errors are retried and that the client succeeds once the error clears.
func TestAttachRateLimit_TransportError_Retries(t *testing.T) {
	client, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)

	httpclient.AttachRateLimit(client, ratelimit.New(1000, 1000))

	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)

	callCount := 0
	httpmock.RegisterResponder(http.MethodGet, "https://example.com/",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("connection reset by peer")
			}
			return httpmock.NewStringResponse(http.StatusOK, "ok"), nil
		})

	resp, err := client.R().Get("https://example.com/")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, callCount, "expected 2 failures then 1 success")
}

// TestAttachRateLimit_ContextCancel_NoRetry verifies that context.Canceled
// errors are not retried.
func TestAttachRateLimit_ContextCancel_NoRetry(t *testing.T) {
	client, err := httpclient.New("", "", nil, false)
	require.NoError(t, err)

	httpclient.AttachRateLimit(client, ratelimit.New(1000, 1000))

	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)

	callCount := 0
	httpmock.RegisterResponder(http.MethodGet, "https://example.com/",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			return nil, context.Canceled
		})

	ctx := context.Background()
	_, err = client.R().SetContext(ctx).Get("https://example.com/")
	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "context.Canceled must not be retried")
}
