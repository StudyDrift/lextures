package atriskscoring

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/atrisk"
)

func TestComputeWeightedScore_highRisk(t *testing.T) {
	cfg := atrisk.DefaultConfig(uuid.Nil)
	avg := float32(40)
	in := SignalInputs{
		MissingPct:   80,
		QuizAvg:      &avg,
		DaysInactive: 10,
		GradeTrend:   50,
	}
	score, comp := ComputeWeightedScore(in, cfg)
	if score < 60 {
		t.Fatalf("score = %v, want >= 60", score)
	}
	if comp.TopFactor == "" {
		t.Fatal("expected top factor")
	}
}

func TestComputeWeightedScore_healthy(t *testing.T) {
	cfg := atrisk.DefaultConfig(uuid.Nil)
	avg := float32(90)
	in := SignalInputs{
		MissingPct:   0,
		QuizAvg:      &avg,
		DaysInactive: 1,
		GradeTrend:   0,
	}
	score, _ := ComputeWeightedScore(in, cfg)
	if score >= 60 {
		t.Fatalf("score = %v, want < 60", score)
	}
}

func TestQuizComponent_belowThreshold(t *testing.T) {
	got := quizComponent(ptrF32(40), 60)
	if got < 30 || got > 40 {
		t.Fatalf("quiz component = %v, want ~33", got)
	}
}

func ptrF32(v float32) *float32 { return &v }
