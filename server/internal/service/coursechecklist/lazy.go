package coursechecklist

import (
	"context"
	"time"
)

// LazyLoaderID identifies an expensive optional data loader (FR-8).
type LazyLoaderID string

// LazyLoaderBudget is the per-loader timeout; on timeout dependent items become unknown (AC-8).
const LazyLoaderBudget = 5 * time.Second

// LazyLoader loads expensive data into CourseSnapshot.Lazy at most once per evaluation.
type LazyLoader interface {
	ID() LazyLoaderID
	Load(ctx context.Context, snap *CourseSnapshot) error
}

// LazyFunc adapts a function to LazyLoader.
type LazyFunc struct {
	LoaderID LazyLoaderID
	Fn       func(ctx context.Context, snap *CourseSnapshot) error
}

func (l LazyFunc) ID() LazyLoaderID { return l.LoaderID }

func (l LazyFunc) Load(ctx context.Context, snap *CourseSnapshot) error {
	if l.Fn == nil {
		return nil
	}
	return l.Fn(ctx, snap)
}
