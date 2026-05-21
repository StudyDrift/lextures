package oersearch

import "context"

type openStaxProvider struct{}

func newOpenStaxProvider() Provider {
	return openStaxProvider{}
}

func (openStaxProvider) ID() string { return "openstax" }

// Search uses the embedded catalog (OpenStax API can be wired when OER_STUB=false and keys are set).
func (openStaxProvider) Search(ctx context.Context, params SearchParams) ([]Result, error) {
	_ = ctx
	return filterStubResults(stubCatalog("openstax"), params), nil
}
