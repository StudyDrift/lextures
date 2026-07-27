package sort_sequence_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/sort_sequence"
)

func categorizeConfig() sort_sequence.Config {
	return sort_sequence.Config{
		Mode:   sort_sequence.ModeCategorize,
		Prompt: "Sort organelles",
		Items: []sort_sequence.Item{
			{ID: "mito", Text: "Mitochondrion"},
			{ID: "ribo", Text: "Ribosome"},
			{ID: "nucleus", Text: "Nucleus"},
		},
		Buckets: []sort_sequence.Bucket{
			{ID: "energy", Label: "Energy"},
			{ID: "protein", Label: "Protein"},
			{ID: "control", Label: "Control"},
		},
		CorrectBucketByItem: map[string]json.RawMessage{
			"mito":    json.RawMessage(`"energy"`),
			"ribo":    json.RawMessage(`"protein"`),
			"nucleus": json.RawMessage(`["control","protein"]`),
		},
		ItemFeedback: map[string]string{
			"mito": "Powerhouse of the cell.",
		},
		ScoreMode: sort_sequence.ScorePerItem,
		Attempts:  3,
	}
}

func TestGradeCategorizeExactAndPartial(t *testing.T) {
	cfg := categorizeConfig()
	perfect := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito":    strPtr("energy"),
		"ribo":    strPtr("protein"),
		"nucleus": strPtr("control"),
	})
	g := sort_sequence.GradePlacement(cfg, perfect)
	if g.ScorePct != 100 || len(g.CorrectItemIDs) != 3 {
		t.Fatalf("perfect: %+v", g)
	}
	if !g.PerItem["mito"].Correct || g.PerItem["mito"].Feedback == "" {
		t.Fatalf("mito feedback: %+v", g.PerItem["mito"])
	}

	// Multi-bucket allowance for nucleus.
	alt := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito":    strPtr("energy"),
		"ribo":    strPtr("protein"),
		"nucleus": strPtr("protein"),
	})
	g2 := sort_sequence.GradePlacement(cfg, alt)
	if !g2.PerItem["nucleus"].Correct || g2.ScorePct != 100 {
		t.Fatalf("multi-bucket: %+v", g2)
	}

	partial := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito":    strPtr("protein"),
		"ribo":    strPtr("protein"),
		"nucleus": strPtr("control"),
	})
	g3 := sort_sequence.GradePlacement(cfg, partial)
	if g3.ScorePct < 66 || g3.ScorePct > 67 {
		t.Fatalf("partial pct: %v", g3.ScorePct)
	}
	if g3.PerItem["mito"].Correct {
		t.Fatal("mito should be wrong")
	}
}

func TestGradeAllOrNothing(t *testing.T) {
	cfg := categorizeConfig()
	cfg.ScoreMode = sort_sequence.ScoreAllOrNothing
	partial := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito":    strPtr("energy"),
		"ribo":    strPtr("protein"),
		"nucleus": strPtr("energy"),
	})
	g := sort_sequence.GradePlacement(cfg, partial)
	if g.ScorePct != 0 || g.ScoreRaw != 0 {
		t.Fatalf("all-or-nothing partial should be 0: %+v", g)
	}
}

func TestGradeOrderWithTieGroups(t *testing.T) {
	cfg := sort_sequence.Config{
		Mode:   sort_sequence.ModeOrder,
		Prompt: "Order events",
		Items: []sort_sequence.Item{
			{ID: "a", Text: "A"},
			{ID: "b", Text: "B"},
			{ID: "c", Text: "C"},
			{ID: "d", Text: "D"},
		},
		CorrectOrder: []string{"a", "b", "c", "d"},
		TieGroups:    [][]string{{"b", "c"}},
		ScoreMode:    sort_sequence.ScorePerItem,
	}
	// Swap b and c within tie group — all should be correct.
	swapped := sort_sequence.MarshalOrderPlacement([]string{"a", "c", "b", "d"})
	g := sort_sequence.GradePlacement(cfg, swapped)
	if g.ScorePct != 100 {
		t.Fatalf("tie swap should be 100: %+v correct=%v", g.ScorePct, g.CorrectItemIDs)
	}
	// Move a after d — a and d wrong; b/c may still be in tie slots.
	bad := sort_sequence.MarshalOrderPlacement([]string{"b", "c", "d", "a"})
	g2 := sort_sequence.GradePlacement(cfg, bad)
	if g2.PerItem["a"].Correct {
		t.Fatal("a should be incorrect")
	}
	if g2.ScorePct >= 100 {
		t.Fatalf("expected partial, got %v", g2.ScorePct)
	}
}

func TestValidatePlacementRejectsUnknown(t *testing.T) {
	cfg := categorizeConfig()
	bad := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito": strPtr("nope"),
	})
	if err := sort_sequence.ValidatePlacementIDs(cfg, bad); err == nil {
		t.Fatal("expected unknown bucket error")
	}
	badItem := json.RawMessage(`{"ghost":"energy"}`)
	if err := sort_sequence.ValidatePlacementIDs(cfg, badItem); err == nil {
		t.Fatal("expected unknown item error")
	}
}

func TestAllItemsPlaced(t *testing.T) {
	cfg := categorizeConfig()
	incomplete := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito": strPtr("energy"),
		"ribo": nil,
	})
	if sort_sequence.AllItemsPlaced(cfg, incomplete) {
		t.Fatal("should be incomplete")
	}
	complete := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito":    strPtr("energy"),
		"ribo":    strPtr("protein"),
		"nucleus": strPtr("control"),
	})
	if !sort_sequence.AllItemsPlaced(cfg, complete) {
		t.Fatal("should be complete")
	}
}

func TestResetUnlockedToTray(t *testing.T) {
	cfg := categorizeConfig()
	st := sort_sequence.EmptyState()
	st.LockedItemIDs = []string{"mito"}
	st.Placement = sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"mito":    strPtr("energy"),
		"ribo":    strPtr("protein"),
		"nucleus": strPtr("control"),
	})
	st = sort_sequence.ResetUnlockedToTray(cfg, st)
	place, ok := sort_sequence.ParseCategorizePlacement(st.Placement)
	if !ok {
		t.Fatal("parse")
	}
	if place["mito"] == nil || *place["mito"] != "energy" {
		t.Fatalf("locked mito should stay: %v", place["mito"])
	}
	if place["ribo"] != nil {
		t.Fatalf("ribo should be tray: %v", place["ribo"])
	}
}

func strPtr(s string) *string { return &s }
