package outcomesreport

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
)

func TestWeightedAvgForStudentLinks_Weighted(t *testing.T) {
	sid1 := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	sid2 := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")
	links := []courseoutcomes.OutcomeLinkWithItemRow{
		{
			OutcomeLinkRow: courseoutcomes.OutcomeLinkRow{
				StructureItemID: sid1,
				TargetKind:      "assignment",
				Weight:          2,
			},
			ItemKind: "assignment",
		},
		{
			OutcomeLinkRow: courseoutcomes.OutcomeLinkRow{
				StructureItemID: sid2,
				TargetKind:      "quiz",
				Weight:          1,
			},
			ItemKind: "quiz",
		},
	}
	scores := map[evidenceKey]float32{
		{structureItemID: sid1, targetKind: "assignment"}: 80,
		{structureItemID: sid2, targetKind: "quiz"}:       50,
	}
	avg := WeightedAvgForStudentLinks(links, scores)
	if avg == nil {
		t.Fatal("expected avg")
	}
	// (80*2 + 50*1) / 3 = 70
	if *avg < 69.9 || *avg > 70.1 {
		t.Fatalf("expected ~70, got %v", *avg)
	}
}

func TestStudentMet(t *testing.T) {
	v := float32(80)
	if !studentMet(&v, 70) {
		t.Fatal("expected met")
	}
	if studentMet(&v, 85) {
		t.Fatal("expected not met")
	}
	if studentMet(nil, 70) {
		t.Fatal("nil avg not met")
	}
}

func TestPctMetNotMet(t *testing.T) {
	met, notMet := pctMetNotMet(10, 7)
	if met != 70 || notMet != 30 {
		t.Fatalf("got met=%v notMet=%v", met, notMet)
	}
	met, notMet = pctMetNotMet(0, 0)
	if met != 0 || notMet != 0 {
		t.Fatalf("empty: met=%v notMet=%v", met, notMet)
	}
}
