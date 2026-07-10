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

func TestParseHeroFileIDFromURL(t *testing.T) {
	id := "75782c7e-8410-4ac5-8f88-61a3290b938e"
	got, ok := ParseHeroFileIDFromURL("/api/v1/courses/C-AIESS1/course-files/" + id + "/content")
	if !ok {
		t.Fatal("expected parse ok")
	}
	if got.String() != id {
		t.Fatalf("got %s want %s", got, id)
	}
	_, ok = ParseHeroFileIDFromURL("/course-card-hero.png")
	if ok {
		t.Fatal("expected non-course-file URL to fail")
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
