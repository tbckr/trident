package worker_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/worker"
)

// stringResult is a minimal services.Result implementation for use in tests.
type stringResult string

func (s stringResult) IsEmpty() bool { return s == "" }

// echoService is a minimal services.Service that echoes the input as output.
type echoService struct{}

func (e *echoService) Name() string   { return "echo" }
func (e *echoService) PAP() pap.Level { return pap.GREEN }
func (e *echoService) Run(_ context.Context, input string) (services.Result, error) {
	return stringResult(input), nil
}
func (e *echoService) AggregateResults(results []services.Result) services.Result { return results[0] }

// errorService returns an error for every Run call.
type errorService struct{ err error }

func (e *errorService) Name() string   { return "error" }
func (e *errorService) PAP() pap.Level { return pap.GREEN }
func (e *errorService) Run(_ context.Context, _ string) (services.Result, error) {
	return nil, e.err
}
func (e *errorService) AggregateResults(_ []services.Result) services.Result { return nil }

// perInputService succeeds for even-indexed inputs and errors for odd.
type perInputService struct{}

func (p *perInputService) Name() string   { return "per-input" }
func (p *perInputService) PAP() pap.Level { return pap.GREEN }
func (p *perInputService) Run(_ context.Context, input string) (services.Result, error) {
	if input == "bad" {
		return nil, errors.New("bad input")
	}
	return stringResult(input), nil
}
func (p *perInputService) AggregateResults(results []services.Result) services.Result {
	return results[0]
}

func TestRun_OrderPreserved(t *testing.T) {
	inputs := make([]string, 20)
	for i := range inputs {
		inputs[i] = fmt.Sprintf("input-%d", i)
	}

	results := worker.Run(context.Background(), &echoService{}, inputs, 5)
	require.Len(t, results, len(inputs))

	for i, r := range results {
		assert.Equal(t, inputs[i], r.Input)
		assert.Equal(t, stringResult(inputs[i]), r.Output)
		assert.NoError(t, r.Err)
	}
}

func TestRun_ErrorPerInput(t *testing.T) {
	inputs := []string{"good", "bad", "good"}
	results := worker.Run(context.Background(), &perInputService{}, inputs, 3)
	require.Len(t, results, 3)

	assert.NoError(t, results[0].Err)
	assert.Error(t, results[1].Err)
	assert.NoError(t, results[2].Err)
}

func TestRun_AllErrors(t *testing.T) {
	sentinel := errors.New("service error")
	inputs := []string{"a", "b", "c"}
	results := worker.Run(context.Background(), &errorService{err: sentinel}, inputs, 2)
	require.Len(t, results, 3)
	for _, r := range results {
		assert.ErrorIs(t, r.Err, sentinel)
	}
}

func TestRun_SingleInput(t *testing.T) {
	results := worker.Run(context.Background(), &echoService{}, []string{"only"}, 10)
	require.Len(t, results, 1)
	assert.Equal(t, stringResult("only"), results[0].Output)
	assert.NoError(t, results[0].Err)
}

func TestRun_EmptyInputs(t *testing.T) {
	results := worker.Run(context.Background(), &echoService{}, []string{}, 5)
	assert.Empty(t, results)
}

func TestRun_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// echoService ignores ctx, but real services should respect it.
	// Verify Run doesn't panic or hang when ctx is already canceled.
	results := worker.Run(ctx, &echoService{}, []string{"a", "b"}, 2)
	assert.Len(t, results, 2)
}

func TestRun_ConcurrencyOne(t *testing.T) {
	inputs := []string{"x", "y", "z"}
	results := worker.Run(context.Background(), &echoService{}, inputs, 1)
	require.Len(t, results, 3)
	for i, r := range results {
		assert.Equal(t, stringResult(inputs[i]), r.Output)
	}
}
