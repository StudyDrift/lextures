package oersearch

import (
	"context"
	"testing"
)

func TestStubProviderSearch_algebra(t *testing.T) {
	p := newStubProvider("oer_commons")
	results, err := p.Search(context.Background(), SearchParams{Query: "algebra"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 1 {
		t.Fatalf("expected algebra results, got %d", len(results))
	}
}

func TestQueryHash_stable(t *testing.T) {
	a := queryHash(SearchParams{Query: "test", License: "CC-BY"})
	b := queryHash(SearchParams{Query: "test", License: "CC-BY"})
	if a != b {
		t.Fatalf("hash unstable: %s vs %s", a, b)
	}
}
