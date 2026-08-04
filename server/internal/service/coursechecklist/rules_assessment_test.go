package coursechecklist

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	uuidPattern  = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	emailPattern = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
)

func evidenceLooksPrivate(rows []EvidenceRow) bool {
	for _, r := range rows {
		blob := r.Label + " " + r.Sublabel
		if uuidPattern.MatchString(blob) || emailPattern.MatchString(blob) {
			return true
		}
	}
	return false
}

func TestAssessmentCopyContract(t *testing.T) {
	for _, it := range append(assessmentRules(), append(gradingRules(), append(feedbackRules(), interactionRules()...)...)...) {
		if utf8.RuneCountInString(it.TitleDefault) > 60 {
			t.Errorf("%s title too long", it.ID)
		}
		low := strings.ToLower(it.TitleDefault + " " + it.WhyDefault)
		for _, ban := range []string{"failed", "should have", "!"} {
			if strings.Contains(low, ban) {
				t.Errorf("%s contains banned %q", it.ID, ban)
			}
		}
	}
}

func TestGradingGroupWeightsAC1(t *testing.T) {
	// AC-1: 40/30/17 → todo with "Weights add up to 87%, not 100%"
	w1, w2, w3 := 40.0, 30.0, 17.0
	f, err := findRule(t, ItemGradingGroupWeights).Evaluate(context.Background(), CourseSnapshot{
		AssignmentGroups: []AssignmentGroupSnap{
			{ID: uuid.New(), Name: "A", Weight: &w1},
			{ID: uuid.New(), Name: "B", Weight: &w2},
			{ID: uuid.New(), Name: "C", Weight: &w3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("status=%s", f.Status)
	}
	if !strings.Contains(f.DetailDefault, "87%") || !strings.Contains(f.DetailDefault, "100%") {
		t.Fatalf("detail=%q", f.DetailDefault)
	}
}

func TestGradingGroupWeightsAC2(t *testing.T) {
	// AC-2: sum 100 → done; unweighted → N/A
	w1, w2 := 60.0, 40.0
	f, err := findRule(t, ItemGradingGroupWeights).Evaluate(context.Background(), CourseSnapshot{
		AssignmentGroups: []AssignmentGroupSnap{
			{ID: uuid.New(), Name: "A", Weight: &w1},
			{ID: uuid.New(), Name: "B", Weight: &w2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusDone {
		t.Fatalf("status=%s detail=%q", f.Status, f.DetailDefault)
	}

	reg := MustDefault()
	res := Evaluate(context.Background(), CourseSnapshot{
		AssignmentGroups: []AssignmentGroupSnap{{ID: uuid.New(), Name: "A", Weight: floatPtr(0)}},
	}, EvaluateOptions{Registry: reg, Only: []ItemID{ItemGradingGroupWeights}})
	if len(res.Findings) != 1 || res.Findings[0].Finding.Status != StatusNotApplicable {
		t.Fatalf("unweighted should be N/A: %+v", res.Findings)
	}
}

func TestGradingEmptyGroupsAC3(t *testing.T) {
	gid := uuid.New()
	w := 20.0
	f, err := findRule(t, ItemGradingEmptyGroups).Evaluate(context.Background(), CourseSnapshot{
		AssignmentGroups: []AssignmentGroupSnap{{ID: gid, Name: "Essays", Weight: &w}},
		AssessmentItems:  []AssessmentItemSnap{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 || f.Evidence[0].Label != "Essays" {
		t.Fatalf("got %+v", f)
	}
}

func TestGradingDropRulesAC4(t *testing.T) {
	gid := uuid.New()
	w := 100.0
	items := make([]AssessmentItemSnap, 3)
	for i := range items {
		items[i] = AssessmentItemSnap{
			ID: uuid.New(), Kind: "assignment", Title: "A", AssignmentGroupID: &gid, Points: intPtr(10),
		}
	}
	f, err := findRule(t, ItemGradingDropRules).Evaluate(context.Background(), CourseSnapshot{
		AssignmentGroups: []AssignmentGroupSnap{
			{ID: gid, Name: "Quizzes", Weight: &w, DropLowest: 2, DropHighest: 1},
		},
		AssessmentItems: items,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 || f.Evidence[0].Label != "Quizzes" {
		t.Fatalf("got %+v", f)
	}
}

func TestFeedbackRubricsOnHighStakesAC5(t *testing.T) {
	// AC-5: 30% assignment, no rubric, empty description
	id := uuid.New()
	pts := 30
	other := 70
	f, err := findRule(t, ItemFeedbackRubricsOnHighStakes).Evaluate(context.Background(), CourseSnapshot{
		AssessmentItems: []AssessmentItemSnap{
			{ID: id, Kind: "assignment", Title: "Capstone", Points: &pts, HasRubric: false, HasBody: false},
			{ID: uuid.New(), Kind: "quiz", Title: "Quiz", Points: &other, HasBody: true},
		},
		AssignmentGroups: []AssignmentGroupSnap{{ID: uuid.New(), Name: "All", Weight: floatPtr(100)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("got %+v", f)
	}
	if f.Evidence[0].TargetOverride == nil || f.Evidence[0].TargetOverride.Anchor != "assignment.rubric" {
		t.Fatalf("target=%+v", f.Evidence[0].TargetOverride)
	}
}

func TestFeedbackFormativePerModuleAC6(t *testing.T) {
	mod := uuid.New()
	quiz := uuid.New()
	zero := 0
	f, err := findRule(t, ItemFeedbackFormativePerModule).Evaluate(context.Background(), CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "M1", SortOrder: 0},
			{ID: quiz, Kind: "quiz", Title: "Practice", ParentID: &mod, SortOrder: 1},
		},
		ItemMeta: map[uuid.UUID]ItemMeta{
			quiz: {Kind: "quiz", PointsWorth: &zero, HasBody: true},
		},
		AssessmentItems: []AssessmentItemSnap{}, // practice excluded from gradable set
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusDone {
		t.Fatalf("got %+v", f)
	}
}

func TestAssessmentDatesWithinTermAC7(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	late := end.Add(14 * 24 * time.Hour)
	f, err := findRule(t, ItemAssessmentDatesWithinTerm).Evaluate(context.Background(), CourseSnapshot{
		StartsAt: &start, EndsAt: &end, ScheduleMode: "fixed",
		AssessmentItems: []AssessmentItemSnap{
			{ID: uuid.New(), Kind: "assignment", Title: "Late", DueAt: &late, Points: intPtr(10)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("got %+v", f)
	}
}

func TestAssessmentSpreadAC8(t *testing.T) {
	// AC-8: 60% of points in final week
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	finalDue := end.Add(-24 * time.Hour)
	early := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	f, err := findRule(t, ItemAssessmentSpread).Evaluate(context.Background(), CourseSnapshot{
		EndsAt: &end,
		AssessmentItems: []AssessmentItemSnap{
			{ID: uuid.New(), Kind: "assignment", Title: "Final", DueAt: &finalDue, Points: intPtr(60)},
			{ID: uuid.New(), Kind: "quiz", Title: "Mid", DueAt: &early, Points: intPtr(40)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("status=%s detail=%q", f.Status, f.DetailDefault)
	}
	if !strings.Contains(f.DetailDefault, "60%") && !strings.Contains(f.DetailDefault, "final week") {
		t.Fatalf("detail should include computed %%: %q", f.DetailDefault)
	}
}

func TestFeedbackQuizReviewSettingsAC9(t *testing.T) {
	until := time.Now().UTC().Add(7 * 24 * time.Hour)
	f, err := findRule(t, ItemFeedbackQuizReviewSettings).Evaluate(context.Background(), CourseSnapshot{
		AssessmentItems: []AssessmentItemSnap{{
			ID: uuid.New(), Kind: "quiz", Title: "Q1", Points: intPtr(10),
			ReviewVisibility: "correct_answers", ReviewWhen: "after_submit",
			ShowScoreTiming: "immediate", AvailableUntil: &until,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("got %+v", f)
	}
}

func TestAccommodationsHonoredAC10(t *testing.T) {
	limit := 30
	f, err := findRule(t, ItemAccommodationsHonored).Evaluate(context.Background(), CourseSnapshot{
		AccommodationCount: 1,
		AccommodationTypeCounts: []AccommodationTypeCount{
			{Type: "extended_time", Count: 1},
		},
		AssessmentItems: []AssessmentItemSnap{{
			ID: uuid.New(), Kind: "quiz", Title: "Timed Quiz", Points: intPtr(10),
			TimeLimitMinutes: &limit,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("status=%s", f.Status)
	}
	if evidenceLooksPrivate(f.Evidence) {
		t.Fatalf("evidence leaked identifiers: %+v", f.Evidence)
	}
	blob := f.DetailDefault
	for _, row := range f.Evidence {
		blob += row.Label + row.Sublabel
	}
	if uuidPattern.MatchString(blob) || emailPattern.MatchString(blob) {
		t.Fatalf("serialized evidence contains id/email: %q", blob)
	}
}

func TestInteractionNAAC11AC12(t *testing.T) {
	// AC-11: 1 student → discussion + groups N/A
	reg := MustDefault()
	snap := CourseSnapshot{
		EnrollmentCounts: map[string]int{"student": 1},
		Features:         CourseFeatures{DiscussionsEnabled: true, GroupSpacesEnabled: true},
		EnrollmentGroupsEnabled: true,
	}
	res := Evaluate(context.Background(), snap, EvaluateOptions{
		Registry: reg,
		Only:     []ItemID{ItemInteractionDiscussionExists, ItemInteractionGroupsConfigured},
	})
	for _, it := range res.Findings {
		if it.Finding.Status != StatusNotApplicable {
			t.Fatalf("%s status=%s want N/A", it.ID, it.Finding.Status)
		}
	}

	// AC-12: office_hours_enabled=false → N/A
	res2 := Evaluate(context.Background(), CourseSnapshot{
		Features: CourseFeatures{OfficeHoursEnabled: false},
	}, EvaluateOptions{Registry: reg, Only: []ItemID{ItemInteractionOfficeHours}})
	if len(res2.Findings) != 1 || res2.Findings[0].Finding.Status != StatusNotApplicable {
		t.Fatalf("office hours: %+v", res2.Findings)
	}
}

func TestZeroPointsNANAC13(t *testing.T) {
	// AC-13: 0 total points → spread + rubrics N/A, no panic
	reg := MustDefault()
	snap := CourseSnapshot{
		AssessmentItems: []AssessmentItemSnap{
			{ID: uuid.New(), Kind: "assignment", Title: "A", Points: intPtr(0)},
		},
	}
	res := Evaluate(context.Background(), snap, EvaluateOptions{
		Registry: reg,
		Only:     []ItemID{ItemAssessmentSpread, ItemFeedbackRubricsOnHighStakes},
	})
	for _, it := range res.Findings {
		if it.Finding.Status != StatusNotApplicable {
			t.Fatalf("%s status=%s want N/A", it.ID, it.Finding.Status)
		}
	}
}

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }
