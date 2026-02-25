package pgp_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/pgp"
	"github.com/tbckr/trident/internal/testutil"
)

// mrindexFixture is a synthetic HKP MRINDEX response for testing.
const mrindexFixture = `info:1:2
pub:0x1234567890ABCDEF:17:2048:1609459200::
uid:Alice Example <alice@example.com>:1609459200::
pub:0xFEDCBA0987654321:1:4096:1577836800:1893456000:
uid:Alice Example (Work) <alice@example.org>:1577836800:1893456000:
uid:Alice at Work <awork@example.org>:1577836800::
`

func newTestClient(t *testing.T) *req.Client {
	t.Helper()
	client := req.NewClient()
	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)
	return client
}

func TestRun_ValidQuery(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://keys.openpgp.org/pks/lookup?op=index&search=alice%40example.com&options=mr",
		httpmock.NewStringResponder(http.StatusOK, mrindexFixture),
	)

	svc := pgp.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "alice@example.com")
	require.NoError(t, err)

	result, ok := raw.(*pgp.Result)
	require.True(t, ok, "expected *pgp.Result")

	assert.Equal(t, "alice@example.com", result.Input)
	assert.Len(t, result.Keys, 2)

	first := result.Keys[0]
	assert.Equal(t, "0x1234567890ABCDEF", first.KeyID)
	assert.Equal(t, "DSA", first.Algorithm)
	assert.Equal(t, 2048, first.Bits)
	assert.Equal(t, "2021-01-01", first.CreatedAt)
	assert.Equal(t, "", first.ExpiresAt)
	assert.Equal(t, []string{"Alice Example <alice@example.com>"}, first.UIDs)

	second := result.Keys[1]
	assert.Equal(t, "0xFEDCBA0987654321", second.KeyID)
	assert.Equal(t, "RSA", second.Algorithm)
	assert.Len(t, second.UIDs, 2)
	assert.NotEmpty(t, second.ExpiresAt)
}

func TestRun_NotFound(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://keys.openpgp.org/pks/lookup?op=index&search=nobody%40example.com&options=mr",
		httpmock.NewStringResponder(http.StatusNotFound, ""),
	)

	svc := pgp.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "nobody@example.com")
	require.NoError(t, err)

	result, ok := raw.(*pgp.Result)
	require.True(t, ok)
	assert.True(t, result.IsEmpty())
}

func TestRun_InvalidInput(t *testing.T) {
	client := newTestClient(t)
	svc := pgp.NewService(client, testutil.NopLogger())

	_, err := svc.Run(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrInvalidInput)
}

func TestRun_HTTPFailure(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://keys.openpgp.org/pks/lookup?op=index&search=alice%40example.com&options=mr",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""),
	)

	svc := pgp.NewService(client, testutil.NopLogger())
	_, err := svc.Run(context.Background(), "alice@example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
}

func TestRun_NetworkError(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://keys.openpgp.org/pks/lookup?op=index&search=alice%40example.com&options=mr",
		httpmock.NewErrorResponder(fmt.Errorf("connection refused")),
	)

	svc := pgp.NewService(client, testutil.NopLogger())
	_, err := svc.Run(context.Background(), "alice@example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
}

func TestRun_ANSISanitization(t *testing.T) {
	body := "info:1:1\npub:\x1b[31m0xABCD\x1b[0m:1:2048:1609459200::\nuid:\x1b[31malice@example.com\x1b[0m:1609459200::\n"
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://keys.openpgp.org/pks/lookup?op=index&search=alice%40example.com&options=mr",
		httpmock.NewStringResponder(http.StatusOK, body),
	)

	svc := pgp.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "alice@example.com")
	require.NoError(t, err)
	result, ok := raw.(*pgp.Result)
	require.True(t, ok)
	for _, k := range result.Keys {
		assert.NotContains(t, k.KeyID, "\x1b")
		for _, uid := range k.UIDs {
			assert.NotContains(t, uid, "\x1b")
		}
	}
}

func TestRun_MalformedMRINDEX(t *testing.T) {
	// Lines that don't match expected format should be silently skipped.
	malformed := "info:1:1\nnotavalidline\npub:0xABCD1234:1:2048:1609459200::\nuid:Alice <alice@example.com>:::\n"
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://keys.openpgp.org/pks/lookup?op=index&search=alice%40example.com&options=mr",
		httpmock.NewStringResponder(http.StatusOK, malformed),
	)

	svc := pgp.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "alice@example.com")
	require.NoError(t, err)
	result, ok := raw.(*pgp.Result)
	require.True(t, ok, "expected *pgp.Result")
	// Valid pub/uid lines still parsed; invalid line skipped.
	assert.Len(t, result.Keys, 1)
	assert.Equal(t, "0xABCD1234", result.Keys[0].KeyID)
}

func TestService_AggregateResults(t *testing.T) {
	svc := pgp.NewService(req.NewClient(), testutil.NopLogger())

	r1 := &pgp.Result{Input: "alice@example.com", Keys: []pgp.Key{{KeyID: "0x1234ABCD"}}}
	r2 := &pgp.Result{Input: "bob@example.com", Keys: []pgp.Key{{KeyID: "0xDEADBEEF"}}}

	agg := svc.AggregateResults([]services.Result{r1, r2})
	mr, ok := agg.(*pgp.MultiResult)
	require.True(t, ok, "expected *pgp.MultiResult")
	assert.Len(t, mr.Results, 2)
	assert.Equal(t, "alice@example.com", mr.Results[0].Input)
	assert.Equal(t, "bob@example.com", mr.Results[1].Input)
}

func TestService_PAP(t *testing.T) {
	svc := pgp.NewService(req.NewClient(), testutil.NopLogger())
	assert.Equal(t, "amber", svc.PAP().String())
}
