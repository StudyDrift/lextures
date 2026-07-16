package board

import (
	"testing"

	"github.com/google/uuid"
)

func TestMidpointSortIndex(t *testing.T) {
	mid := MidpointSortIndex(0, 2)
	if mid != 1 {
		t.Fatalf("expected 1, got %v", mid)
	}
	mid2 := MidpointSortIndex(1, 2)
	if mid2 <= 1 || mid2 >= 2 {
		t.Fatalf("midpoint not between 1 and 2: %v", mid2)
	}
	after := MidpointSortIndex(5, 5)
	if after <= 5 {
		t.Fatalf("expected append after 5, got %v", after)
	}
}

func TestNextSortIndexBetween(t *testing.T) {
	a, b := 1.0, 3.0
	mid := NextSortIndexBetween(&a, &b)
	if mid <= 1 || mid >= 3 {
		t.Fatalf("expected between 1 and 3, got %v", mid)
	}
	prepend := NextSortIndexBetween(nil, &a)
	if prepend >= a {
		t.Fatalf("expected prepend before 1, got %v", prepend)
	}
	appendIdx := NextSortIndexBetween(&b, nil)
	if appendIdx <= b {
		t.Fatalf("expected append after 3, got %v", appendIdx)
	}
	zero := NextSortIndexBetween(nil, nil)
	if zero != 0 {
		t.Fatalf("expected 0, got %v", zero)
	}
}

func TestRenormalizeSortIndexes(t *testing.T) {
	got := RenormalizeSortIndexes(3)
	if len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("unexpected: %v", got)
	}
	if RenormalizeSortIndexes(0) != nil {
		t.Fatal("expected nil for n=0")
	}
}

func TestNormalizeLayout(t *testing.T) {
	for _, layout := range []string{"wall", "stream", "grid", "columns", "canvas", "timeline", "map"} {
		got, err := NormalizeLayout(layout)
		if err != nil || got != layout {
			t.Fatalf("layout %q: got %q err %v", layout, got, err)
		}
	}
	if _, err := NormalizeLayout("table"); err == nil {
		t.Fatal("expected error for unknown layout")
	}
	got, err := NormalizeLayout("")
	if err != nil || got != LayoutWall {
		t.Fatalf("empty should default to wall, got %q err %v", got, err)
	}
}

func TestCanArrangePost(t *testing.T) {
	viewer := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	author := "11111111-1111-1111-1111-111111111111"
	other := "22222222-2222-2222-2222-222222222222"

	if err := CanArrangePost(false, true, &other, viewer); err != nil {
		t.Fatalf("manager should arrange: %v", err)
	}
	if err := CanArrangePost(true, true, &other, viewer); err != nil {
		t.Fatalf("manager should arrange when locked: %v", err)
	}
	if err := CanArrangePost(false, false, &author, viewer); err != nil {
		t.Fatalf("author should arrange when unlocked: %v", err)
	}
	if err := CanArrangePost(true, false, &author, viewer); err != ErrLayoutLocked {
		t.Fatalf("author blocked when locked, got %v", err)
	}
	if err := CanArrangePost(false, false, &other, viewer); err != ErrArrangeForbidden {
		t.Fatalf("non-author blocked, got %v", err)
	}
}

func TestValidateArrangeCoords(t *testing.T) {
	lat, lng := 91.0, 0.0
	if err := ValidateArrangeCoords(&lat, &lng); err == nil {
		t.Fatal("expected lat error")
	}
	lat, lng = 45.0, 200.0
	if err := ValidateArrangeCoords(&lat, &lng); err == nil {
		t.Fatal("expected lng error")
	}
	lat, lng = 40.7, -74.0
	if err := ValidateArrangeCoords(&lat, &lng); err != nil {
		t.Fatalf("valid coords: %v", err)
	}
}
