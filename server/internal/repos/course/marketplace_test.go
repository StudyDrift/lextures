package course

import "testing"

func TestIsMarketplaceListable(t *testing.T) {
	if IsMarketplaceListable(nil) {
		t.Fatal("nil listing is not listable")
	}
	if IsMarketplaceListable(&MarketplaceListing{Published: false}) {
		t.Fatal("unpublished course is not listable")
	}
	if !IsMarketplaceListable(&MarketplaceListing{Published: true}) {
		t.Fatal("published course is listable")
	}
}

func TestIsFree(t *testing.T) {
	if !IsFree(0) {
		t.Fatal("price_cents=0 is free")
	}
	if !IsFree(-1) {
		t.Fatal("negative treated as free")
	}
	if IsFree(1) {
		t.Fatal("positive price is not free")
	}
}
