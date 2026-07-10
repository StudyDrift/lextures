package productfeedback

import (
	"testing"
)

func TestDecodeCursor(t *testing.T) {
	off, err := DecodeCursor("")
	if err != nil || off != 0 {
		t.Fatalf("empty: %d %v", off, err)
	}
	c := EncodeCursor(25)
	off, err = DecodeCursor(c)
	if err != nil || off != 25 {
		t.Fatalf("round trip: %d %v", off, err)
	}
	if _, err := DecodeCursor("!!!"); err == nil {
		t.Fatal("expected invalid cursor error")
	}
}
