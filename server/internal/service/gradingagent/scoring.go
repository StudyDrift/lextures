package gradingagent

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

// GradeOutput is the parsed model response after server-side validation.
type GradeOutput struct {
	TotalPoints  float64
	RubricScores map[string]float64
	Comment      string
	Confidence   float64
}

type rawModelGrade struct {
	Total      float64                       `json:"total"`
	Rubric     map[string]rawRubricCriterion `json:"rubric"`
	Comment    string                        `json:"comment"`
	Confidence float64                       `json:"confidence"`
}

type rawRubricCriterion struct {
	Score     float64 `json:"score"`
	Rationale string  `json:"rationale"`
}

// ParseAndClampModelOutput parses strict JSON from the model and clamps scores to rubric bounds.
func ParseAndClampModelOutput(raw string, rubric *assignmentrubric.RubricDefinition, maxPoints float64) (GradeOutput, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed rawModelGrade
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return GradeOutput{}, fmt.Errorf("invalid model JSON: %w", err)
	}
	out := GradeOutput{
		TotalPoints:  parsed.Total,
		RubricScores: make(map[string]float64),
		Comment:      strings.TrimSpace(parsed.Comment),
		Confidence:   parsed.Confidence,
	}
	if math.IsNaN(out.Confidence) || out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 1 {
		out.Confidence = 1
	}

	if rubric != nil && len(parsed.Rubric) > 0 {
		levelsByCriterion := make(map[uuid.UUID][]assignmentrubric.RubricLevel, len(rubric.Criteria))
		for _, c := range rubric.Criteria {
			levelsByCriterion[c.ID] = c.Levels
		}
		scores := make(map[uuid.UUID]float64)
		for k, v := range parsed.Rubric {
			id, err := uuid.Parse(strings.TrimSpace(k))
			if err != nil {
				continue
			}
			levels, ok := levelsByCriterion[id]
			if !ok {
				continue
			}
			score := snapScoreToRubricLevel(v.Score, levels)
			scores[id] = score
			out.RubricScores[id.String()] = score
		}
		total, err := assignmentrubric.ValidateRubricScoresForGrade(rubric, scores)
		if err != nil {
			// Model returned partial/invalid rubric breakdown; keep total + comment when possible.
			if parsed.Total >= 0 {
				out.TotalPoints = parsed.Total
				if maxPoints > 0 && out.TotalPoints > maxPoints {
					out.TotalPoints = maxPoints
				}
				if out.TotalPoints < 0 {
					out.TotalPoints = 0
				}
				out.RubricScores = nil
				return out, nil
			}
			return GradeOutput{}, fmt.Errorf("invalid rubric scores: %w", err)
		}
		out.TotalPoints = total
		for _, c := range rubric.Criteria {
			if s, ok := scores[c.ID]; ok {
				out.RubricScores[c.ID.String()] = s
			}
		}
	} else {
		if out.TotalPoints < 0 {
			out.TotalPoints = 0
		}
		if maxPoints > 0 && out.TotalPoints > maxPoints {
			out.TotalPoints = maxPoints
		}
	}
	return out, nil
}

func snapScoreToRubricLevel(score float64, levels []assignmentrubric.RubricLevel) float64 {
	if len(levels) == 0 {
		return score
	}
	best := levels[0].Points
	bestDist := math.Abs(score - best)
	for _, lvl := range levels[1:] {
		d := math.Abs(score - lvl.Points)
		if d < bestDist {
			bestDist = d
			best = lvl.Points
		}
	}
	return best
}