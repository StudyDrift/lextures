package course

import "testing"

func TestLegacyHiddenKanbanPlacementsAreIgnoredInMetaMerge(t *testing.T) {
	placements := []UserKanbanPlacement{
		{ColumnID: "hidden", SortOrder: 0},
		{ColumnID: "todo", SortOrder: 0},
	}
	placementByCourse := map[string]UserKanbanPlacement{}
	for _, p := range placements {
		if p.ColumnID == "hidden" {
			continue
		}
		placementByCourse["course"] = p
	}
	if len(placementByCourse) != 1 {
		t.Fatalf("expected one non-hidden placement, got %d", len(placementByCourse))
	}
	if placementByCourse["course"].ColumnID != "todo" {
		t.Fatalf("expected todo placement, got %q", placementByCourse["course"].ColumnID)
	}
}