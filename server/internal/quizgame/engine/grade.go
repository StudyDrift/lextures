package engine

import (
	"encoding/json"
	"math"
	"strings"
)

// GradeAnswer returns whether the submitted answer is correct for the snapshot question.
// Poll and word_cloud are never "correct" (opinion / collection); they still accept submissions.
func GradeAnswer(q SnapshotQuestion, answer json.RawMessage) (correct bool) {
	switch q.QuestionType {
	case "poll", "word_cloud":
		return false
	case "mc_single", "true_false":
		var body struct {
			OptionID string `json:"optionId"`
		}
		if json.Unmarshal(answer, &body) != nil || body.OptionID == "" {
			return false
		}
		for _, o := range q.Options {
			if o.ID == body.OptionID {
				return o.IsCorrect
			}
		}
		return false
	case "mc_multiple":
		var body struct {
			OptionIDs []string `json:"optionIds"`
		}
		if json.Unmarshal(answer, &body) != nil {
			return false
		}
		want := map[string]bool{}
		for _, o := range q.Options {
			if o.IsCorrect {
				want[o.ID] = true
			}
		}
		if len(body.OptionIDs) != len(want) {
			return false
		}
		for _, id := range body.OptionIDs {
			if !want[id] {
				return false
			}
		}
		return true
	case "type_answer":
		var body struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(answer, &body) != nil {
			return false
		}
		got := strings.TrimSpace(body.Text)
		accepted, _ := q.CorrectAnswer["accepted"].([]any)
		for _, raw := range accepted {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			text, _ := m["text"].(string)
			mode, _ := m["matchMode"].(string)
			switch mode {
			case "case_insensitive":
				if strings.EqualFold(strings.TrimSpace(text), got) {
					return true
				}
			case "trim", "exact", "":
				if strings.TrimSpace(text) == got {
					return true
				}
			default:
				if strings.EqualFold(strings.TrimSpace(text), got) {
					return true
				}
			}
		}
		return false
	case "numeric":
		var body struct {
			Value float64 `json:"value"`
		}
		if json.Unmarshal(answer, &body) != nil {
			return false
		}
		target, _ := asFloat(q.CorrectAnswer["value"])
		tol, _ := asFloat(q.CorrectAnswer["tolerance"])
		return math.Abs(body.Value-target) <= tol
	case "ordering":
		var body struct {
			Order []string `json:"order"`
		}
		if json.Unmarshal(answer, &body) != nil {
			return false
		}
		var want []string
		if raw, ok := q.CorrectAnswer["order"].([]any); ok {
			for _, v := range raw {
				if s, ok := v.(string); ok {
					want = append(want, s)
				}
			}
		}
		if len(want) == 0 {
			// Fallback: options marked in correct order via IsCorrect sequence in options array.
			for _, o := range q.Options {
				want = append(want, o.ID)
			}
		}
		if len(body.Order) != len(want) {
			return false
		}
		for i := range want {
			if body.Order[i] != want[i] {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// StubPoints is retained for legacy unit tests; production scoring lives in quizgame/scoring.
// Correct standard = 1000; double = 2000; no_points / incorrect / poll = 0.
func StubPoints(pointsStyle string, correct bool, questionType string) int {
	if questionType == "poll" || questionType == "word_cloud" || !correct {
		return 0
	}
	switch pointsStyle {
	case "double":
		return 2000
	case "no_points":
		return 0
	default:
		return 1000
	}
}
