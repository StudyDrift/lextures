package atriskscoring

import (
	"math"

	"github.com/lextures/lextures/server/internal/repos/atrisk"
)

// SignalInputs are raw engagement metrics for one enrollment.
type SignalInputs struct {
	MissingPct   float32 // 0–100 percent of overdue assignments without submission
	QuizAvg      *float32
	DaysInactive int
	GradeTrend   float32 // positive = declining (worse)
}

// ComponentScores are per-signal 0–100 subscores.
type ComponentScores struct {
	Missing   float32
	Quiz      float32
	Inactive  float32
	Trend     float32
	TopFactor string
}

// ComputeWeightedScore returns the 0–100 at-risk score and top risk factor key.
func ComputeWeightedScore(in SignalInputs, cfg atrisk.Config) (float32, ComponentScores) {
	comp := componentScores(in, cfg)
	total := comp.Missing*cfg.WeightMissing +
		comp.Quiz*cfg.WeightQuiz +
		comp.Inactive*cfg.WeightInactive +
		comp.Trend*cfg.WeightTrend
	score := float32(math.Round(float64(total)*10) / 10)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score, comp
}

func componentScores(in SignalInputs, cfg atrisk.Config) ComponentScores {
	missing := clamp100(in.MissingPct)
	quiz := quizComponent(in.QuizAvg, cfg.QuizAvgThreshold)
	inactive := inactiveComponent(in.DaysInactive)
	trend := clamp100(in.GradeTrend)

	top := "missing"
	topVal := missing * cfg.WeightMissing
	if quiz*cfg.WeightQuiz > topVal {
		top = "quiz"
		topVal = quiz * cfg.WeightQuiz
	}
	if inactive*cfg.WeightInactive > topVal {
		top = "inactive"
		topVal = inactive * cfg.WeightInactive
	}
	if trend*cfg.WeightTrend > topVal {
		top = "trend"
	}

	return ComponentScores{
		Missing:   missing,
		Quiz:      quiz,
		Inactive:  inactive,
		Trend:     trend,
		TopFactor: top,
	}
}

func quizComponent(avg *float32, threshold float32) float32 {
	if avg == nil {
		return 0
	}
	if *avg >= threshold {
		return 0
	}
	// Linear scale: 0% avg -> 100, threshold avg -> 0
	return clamp100((threshold - *avg) / threshold * 100)
}

func inactiveComponent(days int) float32 {
	if days <= 0 {
		return 0
	}
	if days >= 7 {
		return 100
	}
	return clamp100(float32(days) / 7 * 100)
}

func clamp100(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
