package inline_questions

import (
	"encoding/json"
	"strings"
	"time"
)

// ParseConfig unmarshals instructor config with defaults applied.
func ParseConfig(raw json.RawMessage) Config {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg
	}
	var overlay struct {
		Questions          []Question    `json:"questions"`
		Attempts           any           `json:"attempts"`
		RevealCorrectAfter *RevealPolicy `json:"revealCorrectAfter"`
		ShuffleOptions     *bool         `json:"shuffleOptions"`
		Sequential         *bool         `json:"sequential"`
		QuestionsAtATime   any           `json:"questionsAtATime"`
		ScorePolicy        *ScorePolicy  `json:"scorePolicy"`
		Label              *string       `json:"label"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Questions != nil {
		cfg.Questions = overlay.Questions
		if len(cfg.Questions) > 3 {
			cfg.Questions = cfg.Questions[:3]
		}
	}
	if overlay.Attempts != nil {
		cfg.Attempts = overlay.Attempts
	}
	if overlay.RevealCorrectAfter != nil {
		switch *overlay.RevealCorrectAfter {
		case RevealFirstAttempt, RevealLastAttempt, RevealNever:
			cfg.RevealCorrectAfter = *overlay.RevealCorrectAfter
		}
	}
	if overlay.ShuffleOptions != nil {
		cfg.ShuffleOptions = *overlay.ShuffleOptions
	}
	if overlay.Sequential != nil {
		cfg.Sequential = *overlay.Sequential
	}
	if overlay.QuestionsAtATime != nil {
		switch v := overlay.QuestionsAtATime.(type) {
		case string:
			if v == "all" {
				cfg.QuestionsAtATime = "all"
			}
		case float64:
			n := int(v)
			if n >= 1 && n <= 3 {
				cfg.QuestionsAtATime = n
			}
		case int:
			if v >= 1 && v <= 3 {
				cfg.QuestionsAtATime = v
			}
		case int64:
			n := int(v)
			if n >= 1 && n <= 3 {
				cfg.QuestionsAtATime = n
			}
		}
	}
	if overlay.ScorePolicy != nil {
		switch *overlay.ScorePolicy {
		case ScoreLast, ScoreBest, ScoreFirst:
			cfg.ScorePolicy = *overlay.ScorePolicy
		}
	}
	if overlay.Label != nil {
		cfg.Label = *overlay.Label
	}
	return cfg
}

// ParseState unmarshals learner state with defaults.
func ParseState(raw json.RawMessage) State {
	st := EmptyState()
	if len(raw) == 0 {
		return st
	}
	_ = json.Unmarshal(raw, &st)
	if st.V == 0 {
		st.V = 1
	}
	if st.Answers == nil {
		st.Answers = map[string]QuestionAnswer{}
	}
	return st
}

// FindQuestion returns a question by id.
func FindQuestion(cfg Config, id string) *Question {
	id = strings.TrimSpace(id)
	for i := range cfg.Questions {
		if cfg.Questions[i].ID == id {
			return &cfg.Questions[i]
		}
	}
	return nil
}

// AttemptsUsed returns how many attempts the learner has for a question.
func AttemptsUsed(st State, questionID string) int {
	ans, ok := st.Answers[questionID]
	if !ok {
		return 0
	}
	return len(ans.Attempts)
}

// AttemptsRemaining returns remaining attempts, or -1 for unlimited.
func AttemptsRemaining(cfg Config, st State, questionID string) int {
	max := MaxAttempts(cfg)
	if max == 0 {
		return -1
	}
	left := max - AttemptsUsed(st, questionID)
	if left < 0 {
		return 0
	}
	return left
}

// QuestionUnlocked reports whether sequential mode allows answering this question.
func QuestionUnlocked(cfg Config, st State, questionID string) bool {
	if !cfg.Sequential {
		return true
	}
	for _, q := range cfg.Questions {
		if q.ID == questionID {
			return true
		}
		if AttemptsUsed(st, q.ID) == 0 {
			return false
		}
	}
	return false
}

// ShouldReveal reports whether the correct answer should be revealed for this question.
func ShouldReveal(cfg Config, st State, questionID string, justCorrect bool) bool {
	ans := st.Answers[questionID]
	if ans.Revealed {
		return true
	}
	switch cfg.RevealCorrectAfter {
	case RevealNever:
		return false
	case RevealFirstAttempt:
		return AttemptsUsed(st, questionID) >= 1
	case RevealLastAttempt:
		if justCorrect {
			return true
		}
		max := MaxAttempts(cfg)
		if max == 0 {
			return false
		}
		return AttemptsUsed(st, questionID) >= max
	default:
		return false
	}
}

func attemptPoints(q Question, a Attempt) float64 {
	if a.Points > 0 {
		return a.Points
	}
	if a.Correct {
		return PointsFor(q)
	}
	return 0
}

// ComputeScore applies scorePolicy across questions and returns raw/max.
func ComputeScore(cfg Config, st State) (raw, max float64) {
	for _, q := range cfg.Questions {
		max += PointsFor(q)
		ans, ok := st.Answers[q.ID]
		if !ok || len(ans.Attempts) == 0 {
			continue
		}
		switch cfg.ScorePolicy {
		case ScoreFirst:
			raw += attemptPoints(q, ans.Attempts[0])
		case ScoreBest:
			best := 0.0
			for _, a := range ans.Attempts {
				if pts := attemptPoints(q, a); pts > best {
					best = pts
				}
			}
			raw += best
		default:
			raw += attemptPoints(q, ans.Attempts[len(ans.Attempts)-1])
		}
	}
	return raw, max
}

// AllQuestionsExhaustedOrCorrect reports whether every question is done.
func AllQuestionsExhaustedOrCorrect(cfg Config, st State) bool {
	if len(cfg.Questions) == 0 {
		return false
	}
	max := MaxAttempts(cfg)
	for _, q := range cfg.Questions {
		ans, ok := st.Answers[q.ID]
		if !ok || len(ans.Attempts) == 0 {
			return false
		}
		last := ans.Attempts[len(ans.Attempts)-1]
		if last.Correct {
			continue
		}
		if max == 0 || len(ans.Attempts) < max {
			return false
		}
	}
	return true
}

// NowRFC3339 returns UTC now for attempt timestamps.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
