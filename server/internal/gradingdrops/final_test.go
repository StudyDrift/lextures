package gradingdrops

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func near(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("got nil, want %v", want)
	}
	if math.Abs(*got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", *got, want)
	}
}

// Issue #736: assignment 8/10 plus quiz 1/2 (quiz max from question totals) is 75% (9/12),
// not the 65% unweighted mean of per-item percents and not the 80% assignment-only figure.
func TestCourseFinalPercent_PointsBasedAcrossAssignmentAndQuiz(t *testing.T) {
	a, q := uuid.New(), uuid.New()
	cols := []FinalCol{{ID: a, MaxPoints: 10}, {ID: q, MaxPoints: 2}}
	got := CourseFinalPercent(cols, map[uuid.UUID]string{a: "8", q: "1"}, nil, nil, time.Now())
	near(t, got, 75)
}

func TestCourseFinalPercent_NothingCounts(t *testing.T) {
	a := uuid.New()
	future := time.Now().Add(24 * time.Hour)
	cols := []FinalCol{{ID: a, MaxPoints: 10, DueAt: &future}}
	if got := CourseFinalPercent(cols, nil, nil, nil, time.Now()); got != nil {
		t.Fatalf("want nil, got %v", *got)
	}
	if got := CourseFinalPercent(nil, nil, nil, nil, time.Now()); got != nil {
		t.Fatalf("want nil for no cols")
	}
}

func TestCourseFinalPercent_PastDueMissingCountsAsZeroAndExcusedSkipped(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	past := time.Now().Add(-time.Hour)
	cols := []FinalCol{
		{ID: a, MaxPoints: 10},
		{ID: b, MaxPoints: 10, DueAt: &past},
		{ID: c, MaxPoints: 10, DueAt: &past},
	}
	got := CourseFinalPercent(cols, map[uuid.UUID]string{a: "10"}, nil, map[uuid.UUID]bool{c: true}, time.Now())
	near(t, got, 50)
}

func TestCourseFinalPercent_WeightedGroupsWithDropLowest(t *testing.T) {
	hw1, hw2, hw3, exam := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	hwG, exG := uuid.New(), uuid.New()
	cols := []FinalCol{
		{ID: hw1, MaxPoints: 10, GroupID: &hwG},
		{ID: hw2, MaxPoints: 10, GroupID: &hwG},
		{ID: hw3, MaxPoints: 10, GroupID: &hwG},
		{ID: exam, MaxPoints: 100, GroupID: &exG},
	}
	groups := []FinalGroup{
		{ID: hwG, WeightPercent: 40, Policy: GroupDropPolicy{DropLowest: 1}},
		{ID: exG, WeightPercent: 60},
	}
	grades := map[uuid.UUID]string{hw1: "0", hw2: "10", hw3: "5", exam: "80"}
	// homework: drop the 0 -> 15/20 = 75%; exam 80%; 0.4*75 + 0.6*80 = 78
	near(t, CourseFinalPercent(cols, grades, groups, nil, time.Now()), 78)
}

func TestCourseFinalPercent_UngroupedGetsRemainderWeight(t *testing.T) {
	g1, u1 := uuid.New(), uuid.New()
	gid := uuid.New()
	cols := []FinalCol{{ID: g1, MaxPoints: 10, GroupID: &gid}, {ID: u1, MaxPoints: 10}}
	groups := []FinalGroup{{ID: gid, WeightPercent: 50}}
	// group 100% at 50 weight, ungrouped 0% at remainder 50 -> 50
	near(t, CourseFinalPercent(cols, map[uuid.UUID]string{g1: "10", u1: "0"}, groups, nil, time.Now()), 50)
}

func TestCourseFinalPercent_ReplaceLowestWithFinal(t *testing.T) {
	a, fin := uuid.New(), uuid.New()
	gid := uuid.New()
	cols := []FinalCol{
		{ID: a, MaxPoints: 10, GroupID: &gid},
		{ID: fin, MaxPoints: 10, GroupID: &gid, ReplaceWithFinal: true},
	}
	groups := []FinalGroup{{ID: gid, WeightPercent: 100, Policy: GroupDropPolicy{ReplaceLowestWithFinal: true}}}
	// a=2/10 replaced by final 9/10 -> (9 + 9)/20 = 90
	near(t, CourseFinalPercent(cols, map[uuid.UUID]string{a: "2", fin: "9"}, groups, nil, time.Now()), 90)
}
