package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/apperr"
	"github.com/tbckr/trident/internal/config"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/crtsh"
	"github.com/tbckr/trident/internal/testutil"
)

// bulkResult is a JSON-encodable services.Result holding the inputs it came from.
type bulkResult []string

func (b bulkResult) IsEmpty() bool { return len(b) == 0 }

// bulkService runs fn for every input and records which inputs it ran.
type bulkService struct {
	fn   func(ctx context.Context, input string) (services.Result, error)
	mu   sync.Mutex
	seen []string
}

func (s *bulkService) Name() string   { return "bulk" }
func (s *bulkService) PAP() pap.Level { return pap.GREEN }
func (s *bulkService) Run(ctx context.Context, input string) (services.Result, error) {
	s.mu.Lock()
	s.seen = append(s.seen, input)
	s.mu.Unlock()
	return s.fn(ctx, input)
}

func (s *bulkService) AggregateResults(results []services.Result) services.Result {
	var all bulkResult
	for _, r := range results {
		all = append(all, r.(bulkResult)...)
	}
	return all
}

func (s *bulkService) ran(input string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Contains(s.seen, input)
}

// runBulk runs runCmdBody for inputs with the given concurrency and returns
// stdout, the log output and the returned error.
func runBulk(t *testing.T, svc services.Service, concurrency int, inputs ...string) (string, string, error) {
	t.Helper()
	var stdout, logs bytes.Buffer
	d := &deps{
		logger: slog.New(slog.NewTextHandler(&logs, nil)),
		cfg:    &config.Config{Concurrency: concurrency, Output: "json"},
	}
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetContext(context.Background())
	err := runCmdBody(cmd, d, svc, inputs)
	return stdout.String(), logs.String(), err
}

func TestRunCmdBody_StopsAtProxyError(t *testing.T) {
	ok := func(input string) (services.Result, error) { return bulkResult{input}, nil }

	t.Run("stops and prints what was collected", func(t *testing.T) {
		svc := &bulkService{fn: func(_ context.Context, input string) (services.Result, error) {
			if input == "b" {
				return nil, fmt.Errorf("crt.sh: %w", apperr.ErrProxyBypass)
			}
			return ok(input)
		}}
		stdout, logs, err := runBulk(t, svc, 1, "a", "b", "c")
		require.ErrorIs(t, err, apperr.ErrProxyBypass)
		assert.JSONEq(t, `["a"]`, stdout)
		assert.False(t, svc.ran("c"), "no input may start after the proxy error")
		assert.Contains(t, logs, "skipped=1")
		assert.NotContains(t, logs, "input=c", "a skipped input is not logged as a failed lookup")
		assert.NotContains(t, logs, "lookup failed", "the returned error is not logged as well")
	})

	t.Run("the first proxy error in input order is returned", func(t *testing.T) {
		bDone := make(chan struct{})
		svc := &bulkService{fn: func(_ context.Context, input string) (services.Result, error) {
			switch input {
			case "a":
				<-bDone
				return nil, fmt.Errorf("doh: %w", apperr.ErrUnsupportedProxy)
			case "b":
				defer close(bDone)
				return nil, fmt.Errorf("crt.sh: %w", apperr.ErrProxyBypass)
			}
			return ok(input)
		}}
		_, _, err := runBulk(t, svc, 2, "a", "b")
		require.ErrorIs(t, err, apperr.ErrUnsupportedProxy)
	})

	t.Run("all inputs fail", func(t *testing.T) {
		svc := &bulkService{fn: func(context.Context, string) (services.Result, error) {
			return nil, fmt.Errorf("resolver: %w", apperr.ErrProxyConfig)
		}}
		stdout, _, err := runBulk(t, svc, 1, "a", "b")
		require.ErrorIs(t, err, apperr.ErrProxyConfig)
		assert.Empty(t, stdout)
	})

	t.Run("other errors keep the run going", func(t *testing.T) {
		svc := &bulkService{fn: func(_ context.Context, input string) (services.Result, error) {
			if input == "a" {
				return nil, errors.New("timeout")
			}
			return ok(input)
		}}
		stdout, logs, err := runBulk(t, svc, 1, "a", "b")
		require.NoError(t, err)
		assert.JSONEq(t, `["b"]`, stdout)
		assert.Contains(t, logs, "lookup failed")
		assert.Contains(t, logs, "input=a")
	})

	t.Run("a cancelled context is not a proxy error", func(t *testing.T) {
		svc := &bulkService{fn: func(_ context.Context, input string) (services.Result, error) {
			if input == "a" {
				return nil, context.Canceled
			}
			return ok(input)
		}}
		stdout, _, err := runBulk(t, svc, 1, "a", "b")
		require.NoError(t, err)
		assert.JSONEq(t, `["b"]`, stdout)
	})

	t.Run("a lookup in flight finishes and is printed", func(t *testing.T) {
		bDone := make(chan struct{})
		svc := &bulkService{fn: func(ctx context.Context, input string) (services.Result, error) {
			switch input {
			case "slow":
				<-bDone
				// Give the other worker time to record the stop; it then
				// skips "c" without running it.
				time.Sleep(25 * time.Millisecond)
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return ok(input)
			case "b":
				defer close(bDone)
				return nil, fmt.Errorf("crt.sh: %w", apperr.ErrProxyBypass)
			}
			return ok(input)
		}}
		stdout, _, err := runBulk(t, svc, 2, "slow", "b", "c")
		require.ErrorIs(t, err, apperr.ErrProxyBypass)
		assert.JSONEq(t, `["slow"]`, stdout)
		assert.False(t, svc.ran("c"), "no input may start after the proxy error")
	})
}

// The stop only works if the service keeps the proxy sentinel in its error:
// run a real service whose transport refuses the direct connection.
func TestRunCmdBody_StopsAtProxyErrorFromARealService(t *testing.T) {
	client := req.NewClient()
	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)
	httpmock.RegisterResponder(http.MethodGet, "=~^https://crt\\.sh/",
		httpmock.NewErrorResponder(fmt.Errorf("dial tcp: %w", apperr.ErrProxyBypass)))

	svc := crtsh.NewService(client, testutil.NopLogger())
	stdout, logs, err := runBulk(t, svc, 1, "a.example", "b.example")
	require.ErrorIs(t, err, apperr.ErrProxyBypass)
	assert.Empty(t, stdout)
	assert.Contains(t, logs, "skipped=1")
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "the second domain must not be queried")
}
