package httpserver

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// canvasImportMaxConcurrency bounds parallel Canvas HTTP work during import.
const canvasImportMaxConcurrency = 8

func canvasImportConcurrencyLimit(taskCount int) int {
	if taskCount <= 0 {
		return 1
	}
	if taskCount < canvasImportMaxConcurrency {
		return taskCount
	}
	return canvasImportMaxConcurrency
}

func canvasImportParallelGroup(ctx context.Context, taskCount int) (*errgroup.Group, context.Context) {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(canvasImportConcurrencyLimit(taskCount))
	return g, gctx
}

func canvasImportParallelEach[T any](
	ctx context.Context,
	items []T,
	fn func(context.Context, T) error,
) error {
	if len(items) == 0 {
		return nil
	}
	g, gctx := canvasImportParallelGroup(ctx, len(items))
	for _, item := range items {
		item := item
		g.Go(func() error {
			return fn(gctx, item)
		})
	}
	return g.Wait()
}

