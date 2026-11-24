package worker_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

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

	results := worker.Run(context.Background(), &echoService{}, inputs, 5, nil)
	require.Len(t, results, len(inputs))

	for i, r := range results {
		assert.Equal(t, inputs[i], r.Input)
		assert.Equal(t, stringResult(inputs[i]), r.Output)
		assert.NoError(t, r.Err)
	}
}

func TestRun_ErrorPerInput(t *testing.T) {
	inputs := []string{"good", "bad", "good"}
	results := worker.Run(context.Background(), &perInputService{}, inputs, 3, nil)
	require.Len(t, results, 3)

	assert.NoError(t, results[0].Err)
	assert.Error(t, results[1].Err)
	assert.NoError(t, results[2].Err)
}

func TestRun_AllErrors(t *testing.T) {
	sentinel := errors.New("service error")
	inputs := []string{"a", "b", "c"}
	results := worker.Run(context.Background(), &errorService{err: sentinel}, inputs, 2, nil)
	require.Len(t, results, 3)
	for _, r := range results {
		assert.ErrorIs(t, r.Err, sentinel)
	}
}

func TestRun_SingleInput(t *testing.T) {
	results := worker.Run(context.Background(), &echoService{}, []string{"only"}, 10, nil)
	require.Len(t, results, 1)
	assert.Equal(t, stringResult("only"), results[0].Output)
	assert.NoError(t, results[0].Err)
}

func TestRun_EmptyInputs(t *testing.T) {
	results := worker.Run(context.Background(), &echoService{}, []string{}, 5, nil)
	assert.Empty(t, results)
}

func TestRun_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// echoService ignores ctx, but real services should respect it.
	// Verify Run doesn't panic or hang when ctx is already canceled.
	results := worker.Run(ctx, &echoService{}, []string{"a", "b"}, 2, nil)
	assert.Len(t, results, 2)
}

func TestRun_ConcurrencyOne(t *testing.T) {
	inputs := []string{"x", "y", "z"}
	results := worker.Run(context.Background(), &echoService{}, inputs, 1, nil)
	require.Len(t, results, 3)
	for i, r := range results {
		assert.Equal(t, stringResult(inputs[i]), r.Output)
	}
}

// funcService runs fn for every input and records the inputs it was asked to run.
type funcService struct {
	fn   func(ctx context.Context, input string) (services.Result, error)
	mu   sync.Mutex
	seen []string
}

func (f *funcService) Name() string   { return "func" }
func (f *funcService) PAP() pap.Level { return pap.GREEN }
func (f *funcService) Run(ctx context.Context, input string) (services.Result, error) {
	f.mu.Lock()
	f.seen = append(f.seen, input)
	f.mu.Unlock()
	return f.fn(ctx, input)
}
func (f *funcService) AggregateResults(results []services.Result) services.Result { return results[0] }

func (f *funcService) ran() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.seen...)
}

var errStop = errors.New("stop the run")

func stopOnErrStop(err error) bool { return errors.Is(err, errStop) }

func TestRun_StopOnSkipsRemainingInputs(t *testing.T) {
	svc := &funcService{fn: func(_ context.Context, input string) (services.Result, error) {
		if input == "stop" {
			return nil, fmt.Errorf("x: %w", errStop)
		}
		return stringResult(input), nil
	}}
	results := worker.Run(context.Background(), svc, []string{"a", "stop", "c", "d"}, 1, stopOnErrStop)
	require.Len(t, results, 4)

	assert.NoError(t, results[0].Err)
	assert.Equal(t, stringResult("a"), results[0].Output)
	assert.ErrorIs(t, results[1].Err, errStop)
	for _, r := range results[2:] {
		assert.ErrorIs(t, r.Err, worker.ErrSkipped, r.Input)
		assert.Nil(t, r.Output, r.Input)
	}
	assert.Equal(t, []string{"a", "stop"}, svc.ran())
}

func TestRun_StopOnLetsInFlightFinish(t *testing.T) {
	stopped := make(chan struct{})
	svc := &funcService{fn: func(ctx context.Context, input string) (services.Result, error) {
		switch input {
		case "slow":
			<-stopped // still running when "stop" fails
			// Give the other worker time to record the stop; it then skips
			// x1…x5 without running them.
			time.Sleep(25 * time.Millisecond)
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return stringResult("slow"), nil
		case "stop":
			defer close(stopped)
			return nil, errStop
		}
		return stringResult(input), nil
	}}
	inputs := []string{"slow", "stop", "x1", "x2", "x3", "x4", "x5"}
	results := worker.Run(context.Background(), svc, inputs, 2, stopOnErrStop)
	require.Len(t, results, len(inputs))

	assert.NoError(t, results[0].Err)
	assert.Equal(t, stringResult("slow"), results[0].Output)
	assert.ErrorIs(t, results[1].Err, errStop)
	for _, r := range results[2:] {
		assert.ErrorIs(t, r.Err, worker.ErrSkipped, r.Input)
	}
	assert.ElementsMatch(t, []string{"slow", "stop"}, svc.ran())
}

func TestRun_NilStopOnNeverStops(t *testing.T) {
	svc := &funcService{fn: func(context.Context, string) (services.Result, error) {
		return nil, errStop
	}}
	results := worker.Run(context.Background(), svc, []string{"a", "b", "c"}, 1, nil)
	for _, r := range results {
		assert.ErrorIs(t, r.Err, errStop)
	}
	assert.Equal(t, []string{"a", "b", "c"}, svc.ran())
}
