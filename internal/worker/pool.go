package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/tbckr/trident/internal/services"
)

// Result holds the outcome of processing a single input via a Service.
type Result struct {
	Input  string
	Output services.Result
	Err    error
}

// ErrSkipped is the error of every input Run did not start because stopOn
// stopped the run.
var ErrSkipped = errors.New("not started: the run was stopped")

// Run processes inputs through svc using a bounded goroutine pool of size concurrency.
// Results are returned in the same order as inputs regardless of completion order.
//
// stopOn, if not nil, is asked about every error svc.Run returns. Once it
// reports true, no worker starts another input: each input not yet started
// gets ErrSkipped. Lookups already running keep ctx and finish.
func Run(ctx context.Context, svc services.Service, inputs []string, concurrency int, stopOn func(error) bool) []Result {
	results := make([]Result, len(inputs))
	var stopped atomic.Bool

	type job struct {
		index int
		input string
	}

	jobs := make(chan job, len(inputs))
	for i, input := range inputs {
		jobs <- job{index: i, input: input}
	}
	close(jobs)

	var wg sync.WaitGroup
	for range concurrency {
		wg.Go(func() {
			for j := range jobs {
				if stopped.Load() {
					results[j.index] = Result{Input: j.input, Err: ErrSkipped}
					continue
				}
				if err := ctx.Err(); err != nil {
					results[j.index] = Result{Input: j.input, Err: err}
					continue
				}
				out, err := svc.Run(ctx, j.input)
				results[j.index] = Result{Input: j.input, Output: out, Err: err}
				if err != nil && stopOn != nil && stopOn(err) {
					stopped.Store(true)
				}
			}
		})
	}
	wg.Wait()

	return results
}
