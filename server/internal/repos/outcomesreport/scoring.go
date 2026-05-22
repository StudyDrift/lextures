package outcomesreport

import (
	"math"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
)

type evidenceKey struct {
	structureItemID uuid.UUID
	targetKind      string
	quizQuestionID  string
	subOutcomeID    *uuid.UUID
}

type weightedScore struct {
	score  float32
	weight float32
}

// WeightedAvgForStudentLinks computes a student's weighted average across aligned evidence.
// Duplicate evidence keys (same item + target) are deduped; last score wins per key.
func WeightedAvgForStudentLinks(links []courseoutcomes.OutcomeLinkWithItemRow, scores map[evidenceKey]float32) *float32 {
	by := map[evidenceKey]weightedScore{}
	for _, link := range links {
		k := evidenceKey{
			structureItemID: link.StructureItemID,
			targetKind:      link.TargetKind,
			quizQuestionID:  link.QuizQuestionID,
			subOutcomeID:    link.SubOutcomeID,
		}
		sc, ok := scores[k]
		if !ok {
			continue
		}
		w := link.Weight
		if w <= 0 || !isFiniteF32(w) {
			w = 1
		}
		by[k] = weightedScore{score: sc, weight: w}
	}
	if len(by) == 0 {
		return nil
	}
	var sumW, sum float32
	for _, ws := range by {
		sumW += ws.weight
		sum += ws.score * ws.weight
	}
	if sumW <= 0 {
		return nil
	}
	avg := sum / sumW
	return &avg
}

func studentMet(avg *float32, threshold float32) bool {
	if avg == nil {
		return false
	}
	return *avg >= threshold
}

func isFiniteF32(v float32) bool {
	f := float64(v)
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
