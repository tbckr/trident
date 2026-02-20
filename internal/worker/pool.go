package worker

import (
	"context"
	"sync"

	"github.com/tbckr/trident/internal/services"
)

// Result holds the outcome of processing a single input via a Service.
type Result struct {
	Input  string
	Output any
	Err    error
}

// Run processes inputs through svc using a bounded goroutine pool of size concurrency.
// Results are returned in the same order as inputs regardless of completion order.
func Run(ctx context.Context, svc services.Service, inputs []string, concurrency int) []Result {
	results := make([]Result, len(inputs))

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
				out, err := svc.Run(ctx, j.input)
				results[j.index] = Result{Input: j.input, Output: out, Err: err}
			}
		})
	}
	wg.Wait()

	return results
}
