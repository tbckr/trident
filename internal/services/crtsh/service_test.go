package crtsh_test

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/crtsh"
	"github.com/tbckr/trident/internal/testutil"
)

func newTestClient(t *testing.T) *req.Client {
	t.Helper()
	client := req.NewClient()
	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)
	return client
}

func TestRun_ValidDomain(t *testing.T) {
	fixture, err := os.ReadFile("testdata/crtsh_response.json")
	require.NoError(t, err)

	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://crt.sh/?q=%.example.com&output=json",
		httpmock.NewBytesResponder(http.StatusOK, fixture),
	)

	svc := crtsh.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*crtsh.Result)
	require.True(t, ok)

	assert.Equal(t, "example.com", result.Input)
	assert.Contains(t, result.Subdomains, "example.com")
	assert.Contains(t, result.Subdomains, "www.example.com")
	// Deduplication: www.example.com appears twice in fixture but once in result
	count := 0
	for _, sub := range result.Subdomains {
		if sub == "www.example.com" {
			count++
		}
	}
	assert.Equal(t, 1, count, "www.example.com should be deduplicated")
}

func TestRun_InvalidInput(t *testing.T) {
	client := newTestClient(t)
	svc := crtsh.NewService(client, testutil.NopLogger())

	for _, bad := range []string{"", "not_a_domain", "has space.com", "$(injection)"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

func TestRun_HTTPFailure(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://crt.sh/?q=%.example.com&output=json",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""),
	)

	svc := crtsh.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)
	result, ok := raw.(*crtsh.Result)
	require.True(t, ok, "expected *crtsh.Result")
	assert.Nil(t, result.Subdomains)
}

func TestRun_EmptyResponse(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://crt.sh/?q=%.example.com&output=json",
		httpmock.NewStringResponder(http.StatusOK, "[]"),
	)

	svc := crtsh.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)
	result, ok := raw.(*crtsh.Result)
	require.True(t, ok, "expected *crtsh.Result")
	assert.Nil(t, result.Subdomains)
}

func TestRun_ANSISanitization(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://crt.sh/?q=%.example.com&output=json",
		httpmock.NewStringResponder(http.StatusOK,
			`[{"common_name":"\x1b[31mmalicious\x1b[0m","name_value":"clean.example.com"}]`),
	)

	svc := crtsh.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)
	result, ok := raw.(*crtsh.Result)
	require.True(t, ok, "expected *crtsh.Result")
	for _, sub := range result.Subdomains {
		assert.NotContains(t, sub, "\x1b")
	}
}

func TestRun_ContextCanceled(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://crt.sh/?q=%.example.com&output=json",
		httpmock.NewStringResponder(http.StatusOK, "[]"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := crtsh.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(ctx, "example.com")
	require.NoError(t, err)
	result, ok := raw.(*crtsh.Result)
	require.True(t, ok, "expected *crtsh.Result")
	assert.Equal(t, "example.com", result.Input)
}

func TestResult_WriteText(t *testing.T) {
	result := &crtsh.Result{
		Input:      "example.com",
		Subdomains: []string{"example.com", "www.example.com"},
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "www.example.com")
}
