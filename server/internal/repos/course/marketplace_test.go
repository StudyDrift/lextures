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

func TestCourseHeroContentPath(t *testing.T) {
	id := "6bb960af-bc69-478e-8fca-7e8092976eca"
	got := "/api/v1/courses/C-AIESS1/course-files/" + id + "/content"
	want := "/api/v1/courses/C-AIESS1/course-files/6bb960af-bc69-478e-8fca-7e8092976eca/content"
	if got != want {
		t.Fatalf("hero content path shape: %q", got)
	}
}
