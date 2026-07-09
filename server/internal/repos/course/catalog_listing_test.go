package course

import "testing"

func TestNormalizePriceCurrency(t *testing.T) {
	if got := NormalizePriceCurrency(""); got != "usd" {
		t.Fatalf("empty: got %q want usd", got)
	}
	if got := NormalizePriceCurrency("EUR"); got != "eur" {
		t.Fatalf("upper: got %q want eur", got)
	}
}

func TestValidPriceCurrency(t *testing.T) {
	if !ValidPriceCurrency("usd") {
		t.Fatal("usd should be valid")
	}
	if ValidPriceCurrency("xxx") {
		t.Fatal("xxx should be invalid")
	}
}

func TestPublishStateFromBool(t *testing.T) {
	if PublishStateFromBool(true) != "published" {
		t.Fatal("published")
	}
	if PublishStateFromBool(false) != "draft" {
		t.Fatal("draft")
	}
}

func TestSlugify(t *testing.T) {
	if Slugify("Hello World!") != "hello-world" {
		t.Fatalf("slugify: %q", Slugify("Hello World!"))
	}
}
