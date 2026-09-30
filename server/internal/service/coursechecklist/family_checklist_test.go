package coursechecklist

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFamilyChecklist_ReplacesInstitutionalPack(t *testing.T) {
	t.Parallel()
	audiences := []CourseSnapshot{
		{HomeschoolMode: true},
		{OrgIsK12: true},
		{CreatorIsParent: true},
		{HasManagedLearner: true},
	}
	for i, snap := range audiences {
		res := Evaluate(context.Background(), snap, EvaluateOptions{})
		if len(res.Findings) != len(familyItemOrder) {
			t.Fatalf("audience %d findings=%d", i, len(res.Findings))
		}
		if res.Counts.Total != len(familyItemOrder) || res.Counts.Todo != len(familyItemOrder) {
			t.Fatalf("audience %d counts=%+v", i, res.Counts)
		}
		for j, id := range familyItemOrder {
			fr := res.Findings[j]
			if fr.ID != id {
				t.Fatalf("audience %d index %d id=%s want %s", i, j, fr.ID, id)
			}
			if fr.Category != CategoryFamily {
				t.Fatalf("audience %d %s category=%s", i, id, fr.Category)
			}
			if fr.Tier != TierEssential {
				t.Fatalf("audience %d %s tier=%s", i, id, fr.Tier)
			}
			if fr.TitleDefault == "" || fr.Finding.Status != StatusTodo {
				t.Fatalf("audience %d %s title=%q status=%s", i, id, fr.TitleDefault, fr.Finding.Status)
			}
			for _, src := range fr.Sources {
				if isCampusRubricSource(src) {
					t.Fatalf("audience %d %s kept campus source %q", i, id, src)
				}
			}
		}
		if res.Findings[1].TitleDefault != "Enroll your kids" {
			t.Fatalf("audience %d enroll title=%q", i, res.Findings[1].TitleDefault)
		}
	}
}

func TestFamilyChecklist_HigherEdKeepsInstitutionalPack(t *testing.T) {
	t.Parallel()
	res := Evaluate(context.Background(), CourseSnapshot{}, EvaluateOptions{})
	if len(res.Findings) <= len(familyItemOrder) {
		t.Fatalf("higher-ed findings=%d", len(res.Findings))
	}
	var welcome *ItemResult
	for i := range res.Findings {
		if res.Findings[i].ID == ItemOrientationWelcomeMessage {
			welcome = &res.Findings[i]
			break
		}
	}
	if welcome == nil || welcome.Category == CategoryFamily {
		t.Fatalf("welcome=%+v", welcome)
	}
	if welcome.TitleDefault == "Enroll your kids" {
		t.Fatal("higher-ed item was rewritten")
	}
}

func TestFamilyChecklist_ReadyCourseIsDone(t *testing.T) {
	t.Parallel()
	mod := uuid.New()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(90 * 24 * time.Hour)
	snap := CourseSnapshot{
		HomeschoolMode: true,
		StartsAt:       &start,
		EndsAt:         &end,
		People: []PersonSnap{
			{UserID: uuid.New(), DisplayName: "Ada", Role: "student", Active: true},
		},
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "Unit 1"},
			{ID: uuid.New(), Kind: "content_page", Title: "Lesson 1", ParentID: &mod},
		},
	}
	res := Evaluate(context.Background(), snap, EvaluateOptions{})
	if res.Counts.Done != len(familyItemOrder) || res.Counts.Todo != 0 || res.Counts.OutstandingEssential != 0 {
		t.Fatalf("counts=%+v", res.Counts)
	}
	if res.Findings[3].Target.Route != "/courses/{courseCode}/reports" {
		t.Fatalf("progress route=%s", res.Findings[3].Target.Route)
	}
}

func TestFamilyChecklist_RecheckRewritesFamilyItemOnly(t *testing.T) {
	t.Parallel()
	res := Evaluate(context.Background(), CourseSnapshot{OrgIsK12: true}, EvaluateOptions{
		Only: []ItemID{ItemPeopleStudentsEnrolled},
	})
	if len(res.Findings) != 1 {
		t.Fatalf("findings=%d", len(res.Findings))
	}
	fr := res.Findings[0]
	if fr.TitleDefault != "Enroll your kids" || fr.Category != CategoryFamily || fr.Finding.Status != StatusTodo {
		t.Fatalf("recheck=%+v finding=%+v", fr.TitleDefault, fr.Finding)
	}

	campus := Evaluate(context.Background(), CourseSnapshot{HomeschoolMode: true}, EvaluateOptions{
		Only: []ItemID{ItemPeopleStaffRoles},
	})
	if campus.Findings[0].Finding.Status != StatusNotApplicable {
		t.Fatalf("staff=%s", campus.Findings[0].Finding.Status)
	}
}
