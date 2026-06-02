package sbgaggregation

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/sbgreport"
)

var (
	stu1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	std1 = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	crs1 = uuid.MustParse("00000000-0000-0000-0000-000000000100")
)

func makeScore(student, standard uuid.UUID, val int, offset int) sbgreport.MasteryScore {
	return sbgreport.MasteryScore{
		ID:            uuid.New(),
		StudentID:     student,
		StandardID:    standard,
		CourseID:      crs1,
		GradingPeriod: "Q1-2026",
		ScoreValue:    val,
		Source:        "observation",
		AssessedAt:    time.Now().Add(time.Duration(offset) * time.Minute),
	}
}

func TestAggregate_MostRecent(t *testing.T) {
	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 1, 0),
		makeScore(stu1, std1, 3, 1),
		makeScore(stu1, std1, 2, 2),
	}
	agg := Aggregate(scores, MostRecent)
	if len(agg) != 1 {
		t.Fatalf("expected 1 aggregated score, got %d", len(agg))
	}
	if agg[0].ScoreValue != 2 {
		t.Errorf("most-recent: expected 2, got %d", agg[0].ScoreValue)
	}
}

func TestAggregate_Highest(t *testing.T) {
	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 2, 0),
		makeScore(stu1, std1, 4, 1),
		makeScore(stu1, std1, 3, 2),
	}
	agg := Aggregate(scores, Highest)
	if agg[0].ScoreValue != 4 {
		t.Errorf("highest: expected 4, got %d", agg[0].ScoreValue)
	}
}

func TestAggregate_Mode(t *testing.T) {
	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 3, 0),
		makeScore(stu1, std1, 2, 1),
		makeScore(stu1, std1, 3, 2),
		makeScore(stu1, std1, 4, 3),
	}
	agg := Aggregate(scores, Mode)
	if agg[0].ScoreValue != 3 {
		t.Errorf("mode: expected 3, got %d", agg[0].ScoreValue)
	}
}

func TestAggregate_Mode_TieBreakHighest(t *testing.T) {
	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 2, 0),
		makeScore(stu1, std1, 4, 1),
	}
	agg := Aggregate(scores, Mode)
	if agg[0].ScoreValue != 4 {
		t.Errorf("mode tie: expected 4 (highest), got %d", agg[0].ScoreValue)
	}
}

func TestAggregate_Trend(t *testing.T) {
	// Trend = avg of last 3 → (2+3+4)/3 = 3
	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 1, 0),
		makeScore(stu1, std1, 2, 1),
		makeScore(stu1, std1, 3, 2),
		makeScore(stu1, std1, 4, 3),
	}
	agg := Aggregate(scores, Trend)
	if agg[0].ScoreValue != 3 {
		t.Errorf("trend: expected 3, got %d", agg[0].ScoreValue)
	}
}

func TestAggregate_Trend_FewScores(t *testing.T) {
	// Only 2 scores: (2+4)/2 = 3
	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 2, 0),
		makeScore(stu1, std1, 4, 1),
	}
	agg := Aggregate(scores, Trend)
	if agg[0].ScoreValue != 3 {
		t.Errorf("trend (2 scores): expected 3, got %d", agg[0].ScoreValue)
	}
}

func TestAggregate_Empty(t *testing.T) {
	agg := Aggregate(nil, MostRecent)
	if len(agg) != 0 {
		t.Errorf("empty input: expected 0 results, got %d", len(agg))
	}
}

func TestAggregate_MultipleStudentsAndStandards(t *testing.T) {
	stu2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	std2 := uuid.MustParse("00000000-0000-0000-0000-000000000020")

	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 4, 0),
		makeScore(stu1, std2, 2, 0),
		makeScore(stu2, std1, 3, 0),
	}
	agg := Aggregate(scores, MostRecent)
	if len(agg) != 3 {
		t.Fatalf("expected 3 distinct pairs, got %d", len(agg))
	}
}

func TestAggregateForReport(t *testing.T) {
	scores := []sbgreport.MasteryScore{
		makeScore(stu1, std1, 3, 0),
		makeScore(stu1, std1, 4, 1),
	}
	m := AggregateForReport(scores, MostRecent)
	if m[stu1][std1] != 4 {
		t.Errorf("AggregateForReport: expected 4, got %d", m[stu1][std1])
	}
}

func TestParseMethod(t *testing.T) {
	if ParseMethod("highest") != Highest {
		t.Error("ParseMethod(highest) failed")
	}
	if ParseMethod("mode") != Mode {
		t.Error("ParseMethod(mode) failed")
	}
	if ParseMethod("trend") != Trend {
		t.Error("ParseMethod(trend) failed")
	}
	if ParseMethod("unknown") != MostRecent {
		t.Error("ParseMethod(unknown) should default to MostRecent")
	}
}
