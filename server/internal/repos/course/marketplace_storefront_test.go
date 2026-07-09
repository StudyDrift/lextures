package course

import "testing"

func TestMarketplaceFilter_FreeOnlySetsPriceMax(t *testing.T) {
	f := MarketplaceFilter{FreeOnly: true, Q: "algo"}
	pf := f.ToPublicCatalogFilter()
	if pf.PriceMax == nil || *pf.PriceMax != 0 {
		t.Fatalf("expected price_max=0 for free_only, got %#v", pf.PriceMax)
	}
	if pf.Q != "algo" {
		t.Fatalf("q: got %q", pf.Q)
	}
}

func TestMarketplaceOrderBy_Price(t *testing.T) {
	add := func(v any) string { return "$1" }
	got := marketplaceOrderBy(CatalogSortPrice, "", add)
	if got == "" || got[:12] != "c.price_cent" {
		t.Fatalf("unexpected price order: %q", got)
	}
}
