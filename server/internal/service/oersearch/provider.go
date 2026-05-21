package oersearch

import "context"

// Provider searches a single OER catalog.
type Provider interface {
	ID() string
	Search(ctx context.Context, params SearchParams) ([]Result, error)
}
