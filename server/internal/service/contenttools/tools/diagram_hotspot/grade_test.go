package diagram_hotspot_test

import (
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/diagram_hotspot"
)

func TestPointInRectCirclePolygon(t *testing.T) {
	rect := diagram_hotspot.Shape{Kind: "rect", X: 0.1, Y: 0.1, W: 0.3, H: 0.2}
	if !diagram_hotspot.PointInShape(0.2, 0.2, rect) {
		t.Fatal("expected inside rect")
	}
	if diagram_hotspot.PointInShape(0.5, 0.5, rect) {
		t.Fatal("expected outside rect")
	}
	// Edge inclusive
	if !diagram_hotspot.PointInShape(0.1, 0.1, rect) {
		t.Fatal("expected edge inside rect")
	}

	circle := diagram_hotspot.Shape{Kind: "circle", CX: 0.5, CY: 0.5, R: 0.2}
	if !diagram_hotspot.PointInShape(0.5, 0.5, circle) {
		t.Fatal("expected center inside circle")
	}
	if diagram_hotspot.PointInShape(0.9, 0.9, circle) {
		t.Fatal("expected far point outside circle")
	}

	// Concave-ish L via polygon (simple triangle for winding)
	poly := diagram_hotspot.Shape{
		Kind: "polygon",
		Points: [][]float64{
			{0.1, 0.1},
			{0.5, 0.1},
			{0.3, 0.4},
		},
	}
	if !diagram_hotspot.PointInShape(0.3, 0.15, poly) {
		t.Fatal("expected inside polygon")
	}
	if diagram_hotspot.PointInShape(0.9, 0.9, poly) {
		t.Fatal("expected outside polygon")
	}
}

func TestSmallestContainingRegion(t *testing.T) {
	regions := []diagram_hotspot.Region{
		{ID: "outer", Label: "Outer", Description: "big", Shape: diagram_hotspot.Shape{Kind: "rect", X: 0, Y: 0, W: 1, H: 1}},
		{ID: "inner", Label: "Inner", Description: "small", Shape: diagram_hotspot.Shape{Kind: "rect", X: 0.4, Y: 0.4, W: 0.2, H: 0.2}},
	}
	got := diagram_hotspot.SmallestContainingRegion(regions, 0.5, 0.5)
	if got != "inner" {
		t.Fatalf("got %q want inner", got)
	}
}

func TestHeatCellForPoint(t *testing.T) {
	cell := diagram_hotspot.HeatCellForPoint(0, 0)
	if cell != "r0c0" {
		t.Fatalf("got %q", cell)
	}
	cell = diagram_hotspot.HeatCellForPoint(0.99, 0.99)
	if cell != "r7c7" {
		t.Fatalf("got %q want r7c7", cell)
	}
}

func labelConfig() diagram_hotspot.Config {
	return diagram_hotspot.Config{
		Mode:   diagram_hotspot.ModeLabel,
		Prompt: "Label the cell",
		Image:  diagram_hotspot.ImageRef{URL: "/img.png", Alt: "Cell diagram", NaturalWidth: 800, NaturalHeight: 600},
		Regions: []diagram_hotspot.Region{
			{ID: "nuc", Label: "Nucleus", Description: "Dense center of the cell", Shape: diagram_hotspot.Shape{Kind: "circle", CX: 0.5, CY: 0.5, R: 0.15}},
			{ID: "mem", Label: "Membrane", Description: "Outer boundary of the cell", Shape: diagram_hotspot.Shape{Kind: "rect", X: 0.05, Y: 0.05, W: 0.9, H: 0.9}},
		},
		Labels: []diagram_hotspot.LabelChip{
			{ID: "l_nuc", Text: "Nucleus"},
			{ID: "l_mem", Text: "Membrane"},
		},
		CorrectRegionByLabel: map[string]string{
			"l_nuc": "nuc",
			"l_mem": "mem",
		},
		FeedbackByRegion: map[string]string{
			"nuc": "The nucleus holds DNA.",
		},
		Attempts:               3,
		LockCorrect:            true,
		ShowPerItemCorrectness: true,
	}
}

func TestGradeAssignments(t *testing.T) {
	cfg := labelConfig()
	nuc, mem := "nuc", "mem"
	perfect := map[string]*string{"l_nuc": &nuc, "l_mem": &mem}
	g := diagram_hotspot.GradeAssignments(cfg, perfect)
	if g.ScorePct != 100 || len(g.CorrectIDs) != 2 {
		t.Fatalf("perfect: %+v", g)
	}

	wrongMem := "nuc"
	partial := map[string]*string{"l_nuc": &nuc, "l_mem": &wrongMem}
	g2 := diagram_hotspot.GradeAssignments(cfg, partial)
	if g2.ScorePct != 50 {
		t.Fatalf("partial score %v", g2.ScorePct)
	}
	if !g2.PerItem["l_nuc"].Correct || g2.PerItem["l_mem"].Correct {
		t.Fatalf("per-item %+v", g2.PerItem)
	}
}

func TestValidateAndReset(t *testing.T) {
	cfg := labelConfig()
	bad := "nope"
	nuc := "nuc"
	assignments := map[string]*string{"l_nuc": &bad, "l_mem": &nuc}
	if err := diagram_hotspot.ValidateAssignmentIDs(cfg, assignments); err == nil {
		t.Fatal("expected unknown region")
	}
	good := map[string]*string{"l_nuc": &nuc, "l_mem": &nuc}
	if err := diagram_hotspot.ValidateAssignmentIDs(cfg, good); err != nil {
		t.Fatal(err)
	}
	if !diagram_hotspot.AllItemsAssigned(cfg, map[string]*string{"l_nuc": &nuc, "l_mem": &nuc}) {
		t.Fatal("expected all assigned")
	}
	if diagram_hotspot.AllItemsAssigned(cfg, map[string]*string{"l_nuc": &nuc, "l_mem": nil}) {
		t.Fatal("expected incomplete")
	}

	st := diagram_hotspot.EmptyState()
	st.Assignments = map[string]*string{"l_nuc": &nuc, "l_mem": &nuc}
	st.LockedIDs = []string{"l_nuc"}
	st = diagram_hotspot.ResetUnlocked(cfg, st)
	if st.Assignments["l_nuc"] == nil || *st.Assignments["l_nuc"] != "nuc" {
		t.Fatal("locked should remain")
	}
	if st.Assignments["l_mem"] != nil {
		t.Fatal("unlocked should clear")
	}
}

func TestValidateConfigForAuthoring(t *testing.T) {
	cfg := labelConfig()
	if err := diagram_hotspot.ValidateConfigForAuthoring(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Regions[0].Description = ""
	if err := diagram_hotspot.ValidateConfigForAuthoring(cfg); err == nil {
		t.Fatal("expected missing description")
	}
	cfg = labelConfig()
	cfg.Image.Alt = ""
	if err := diagram_hotspot.ValidateConfigForAuthoring(cfg); err == nil {
		t.Fatal("expected missing alt")
	}
}

func TestHeatCellsForAssignments(t *testing.T) {
	cfg := labelConfig()
	nuc := "nuc"
	cells := diagram_hotspot.HeatCellsForAssignments(cfg, map[string]*string{"l_nuc": &nuc})
	if len(cells) != 1 {
		t.Fatalf("got %v", cells)
	}
}
