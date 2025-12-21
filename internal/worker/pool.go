package worker

import (
	"context"
	"log/slog"
	"sync"
)

type Pool struct {
	size   int
	logger *slog.Logger
}

type Input interface{}

type JobResult struct {
	Input Input
	Value interface{}
	Error error
}

func NewPool(size int, logger *slog.Logger) *Pool {
	return &Pool{
		size:   size,
		logger: logger,
	}
}

func (p *Pool) Process(ctx context.Context, inputs <-chan Input, fn func(context.Context, Input) (interface{}, error)) <-chan JobResult {
	results := make(chan JobResult)
	var wg sync.WaitGroup

	for i := 0; i < p.size; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case input, ok := <-inputs:
					if !ok {
						return
					}
					val, err := fn(ctx, input)
					results <- JobResult{
						Input: input,
						Value: val,
						Error: err,
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
