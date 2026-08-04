package coursechecklist

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	ccrepo "github.com/lextures/lextures/server/internal/repos/coursechecklist"
)

func TestAssembleChecklist_DismissedPileAndSummary(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	actor := uuid.New()
	dismissedAt := now.Add(-time.Hour)
	res := Result{
		EngineVersion:  1,
		CatalogVersion: "cat",
		Findings: []ItemResult{
			{
				ID: ItemCourseDates, Category: CategoryReference, Tier: TierEssential,
				TitleKey: "t", TitleDefault: "Dates", WhyKey: "w", WhyDefault: "why",
				Sources: []string{"OSCQR 1"},
				Finding: Finding{Status: StatusTodo},
			},
			{
				ID: ItemCourseSections, Category: CategoryReference, Tier: TierEssential,
				TitleKey: "t2", TitleDefault: "Sections", WhyKey: "w2", WhyDefault: "why2",
				Sources: []string{"OSCQR 2"},
				Finding: Finding{Status: StatusTodo},
			},
			{
				ID: "course.na", Category: CategoryReference, Tier: TierRecommended,
				TitleKey: "t3", TitleDefault: "NA", WhyKey: "w3", WhyDefault: "why3",
				Sources: []string{"x"},
				Finding: Finding{Status: StatusNotApplicable},
			},
		},
	}
	resp := AssembleChecklist(res, AssembleOptions{
		CourseCode: "C-1",
		ComputedAt: now,
		Dismissed: []ccrepo.ItemState{{
			ItemID: string(ItemCourseSections), DismissedAt: &dismissedAt,
			DismissedByUserID: &actor, DismissReason: "not_applicable",
		}},
		DisplayNames: map[uuid.UUID]string{actor: "Teacher"},
	})
	if len(resp.Dismissed) != 1 || resp.Dismissed[0].ID != string(ItemCourseSections) {
		t.Fatalf("dismissed pile: %+v", resp.Dismissed)
	}
	if resp.Dismissed[0].Dismissal == nil || resp.Dismissed[0].Dismissal.ByDisplayName != "Teacher" {
		t.Fatalf("dismissal meta: %+v", resp.Dismissed[0].Dismissal)
	}
	if resp.Summary.Dismissed != 1 || resp.Summary.OutstandingEssential != 1 || resp.Summary.Total != 1 {
		t.Fatalf("summary: %+v", resp.Summary)
	}
	// N/A omitted by default; dismissed not inline.
	for _, cat := range resp.Categories {
		for _, it := range cat.Items {
			if it.ID == string(ItemCourseSections) || it.Status == StatusNotApplicable {
				t.Fatalf("unexpected inline item %s", it.ID)
			}
		}
	}
}

func TestDropEvidence_FitsPayload(t *testing.T) {
	t.Parallel()
	big := strings.Repeat("x", 4000)
	rows := make([]EvidenceRow, 0, 80)
	for i := 0; i < 80; i++ {
		rows = append(rows, EvidenceRow{Label: big})
	}
	res := Result{
		EngineVersion: 1, CatalogVersion: "c",
		Findings: []ItemResult{{
			ID: ItemCourseDates, Category: CategoryReference, Tier: TierEssential,
			TitleKey: "t", TitleDefault: "Dates", WhyKey: "w", WhyDefault: "why",
			Sources: []string{"a"},
			Finding: Finding{Status: StatusTodo, Evidence: rows},
		}},
	}
	fitted, truncated := fitPayload(res)
	if !truncated {
		t.Fatal("expected truncation")
	}
	if len(fitted.Findings[0].Finding.Evidence) != 0 {
		t.Fatal("evidence should be dropped")
	}
	if _, err := encodeSnapshotPayload(fitted, true); err != nil {
		t.Fatalf("encode after drop: %v", err)
	}
}
