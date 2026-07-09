package course

import "testing"

func TestCatalogCursorRoundTrip(t *testing.T) {
	for _, off := range []int{0, 20, 40, 1000} {
		c := EncodeCatalogCursor(off)
		got, err := DecodeCatalogCursor(c)
		if err != nil {
			t.Fatalf("decode(%q): %v", c, err)
		}
		if got != off {
			t.Fatalf("round trip: got %d want %d", got, off)
		}
	}
}

func TestDecodeCatalogCursor_Empty(t *testing.T) {
	got, err := DecodeCatalogCursor("")
	if err != nil || got != 0 {
		t.Fatalf("empty cursor: got %d err %v", got, err)
	}
}

func TestDecodeCatalogCursor_Invalid(t *testing.T) {
	for _, bad := range []string{"not-base64!!", "Zm9v", "bzpub3Q="} {
		if _, err := DecodeCatalogCursor(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestValidCatalogSort(t *testing.T) {
	for _, s := range []string{"popular", "rating", "newest", "relevance", "price"} {
		if !ValidCatalogSort(s) {
			t.Fatalf("%q should be valid", s)
		}
	}
	if ValidCatalogSort("bogus") {
		t.Fatalf("bogus should be invalid")
	}
}

func TestValidDifficultyLevel(t *testing.T) {
	for _, s := range []string{"beginner", "intermediate", "advanced"} {
		if !ValidDifficultyLevel(s) {
			t.Fatalf("%q should be valid", s)
		}
	}
	if ValidDifficultyLevel("expert") {
		t.Fatalf("expert should be invalid")
	}
}
