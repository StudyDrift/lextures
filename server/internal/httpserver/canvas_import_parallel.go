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

