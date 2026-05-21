package oersearch

import "context"

type stubProvider struct {
	id string
}

func newStubProvider(id string) Provider {
	return stubProvider{id: id}
}

func (p stubProvider) ID() string { return p.id }

func (p stubProvider) Search(_ context.Context, params SearchParams) ([]Result, error) {
	return filterStubResults(stubCatalog(p.id), params), nil
}
