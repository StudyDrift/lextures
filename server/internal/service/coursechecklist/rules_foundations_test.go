package coursechecklist

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func findRule(t *testing.T, id ItemID) ItemDescriptor {
	t.Helper()
	it := MustDefault().Get(id)
	if it == nil {
		t.Fatalf("missing rule %s", id)
	}
	return *it
}

func TestFoundationsCopyContract(t *testing.T) {
	for _, it := range foundationsRules() {
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

func TestCourseDatesAC1Inverted(t *testing.T) {
	// AC-1
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f, err := findRule(t, ItemCourseDates).Evaluate(context.Background(), CourseSnapshot{
		ScheduleMode: "fixed", StartsAt: &start, EndsAt: &end,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || !strings.Contains(f.DetailDefault, "End date is before the start date") {
		t.Fatalf("got %+v", f)
	}
}

func TestCourseDatesAC2Relative(t *testing.T) {
	// AC-2
	reg := MustDefault()
	res := Evaluate(context.Background(), CourseSnapshot{ScheduleMode: "relative"}, EvaluateOptions{
		Registry: reg,
		Only:     []ItemID{ItemCourseDates, ItemCourseRelativeSchedule},
	})
	var dates, rel *ItemResult
	for i := range res.Findings {
		switch res.Findings[i].ID {
		case ItemCourseDates:
			dates = &res.Findings[i]
		case ItemCourseRelativeSchedule:
			rel = &res.Findings[i]
		}
	}
	if dates == nil || dates.Finding.Status != StatusNotApplicable {
		t.Fatalf("dates=%+v", dates)
	}
	if rel == nil || rel.Finding.Status == StatusNotApplicable {
		t.Fatalf("relative should apply: %+v", rel)
	}
}

func TestCoursePublishedAC3Urgent(t *testing.T) {
	// AC-3
	start := time.Now().UTC().Add(3 * 24 * time.Hour)
	f, err := findRule(t, ItemCoursePublished).Evaluate(context.Background(), CourseSnapshot{
		Published: false, StartsAt: &start,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusInProgress {
		t.Fatalf("status=%s", f.Status)
	}
	if !strings.Contains(f.DetailDefault, "days") {
		t.Fatalf("detail=%q", f.DetailDefault)
	}
}

func TestCourseFeaturesReviewedAC10(t *testing.T) {
	// AC-10
	rule := findRule(t, ItemCourseFeaturesReviewed)
	f, err := rule.Evaluate(context.Background(), CourseSnapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("unreviewed=%s", f.Status)
	}
	now := time.Now().UTC()
	f, err = rule.Evaluate(context.Background(), CourseSnapshot{FeaturesReviewedAt: &now})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusDone {
		t.Fatalf("reviewed=%s", f.Status)
	}
}

func TestCourseTitleDescriptionAndTimezone(t *testing.T) {
	rule := findRule(t, ItemCourseTitleAndDescription)
	f, _ := rule.Evaluate(context.Background(), CourseSnapshot{Title: "Untitled", Description: "short"})
	if f.Status != StatusTodo {
		t.Fatalf("placeholder=%s", f.Status)
	}
	f, _ = rule.Evaluate(context.Background(), CourseSnapshot{
		Title: "Biology 101", Description: strings.Repeat("a", 120),
	})
	if f.Status != StatusDone {
		t.Fatalf("ok=%s", f.Status)
	}

	tz := "America/New_York"
	f, _ = findRule(t, ItemCourseTimezone).Evaluate(context.Background(), CourseSnapshot{CourseTimezone: &tz})
	if f.Status != StatusDone {
		t.Fatalf("tz=%s", f.Status)
	}
	local := "LOCAL"
	f, _ = findRule(t, ItemCourseTimezone).Evaluate(context.Background(), CourseSnapshot{CourseTimezone: &local})
	if f.Status != StatusDone {
		t.Fatalf("local tz=%s", f.Status)
	}
}

func TestCourseRelativeScheduleMissingDue(t *testing.T) {
	rule := findRule(t, ItemCourseRelativeSchedule)
	f, _ := rule.Evaluate(context.Background(), CourseSnapshot{
		ScheduleMode: "relative",
		StructureItems: []StructureItem{
			{ID: uuid.New(), Kind: "assignment", Title: "A1"},
			{ID: uuid.New(), Kind: "quiz", Title: "Q1", DueAt: ptrTime(time.Now())},
		},
	})
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("got %+v", f)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
